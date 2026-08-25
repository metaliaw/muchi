"""Cliente de scry.cl, el agregador chileno que ya indexa ~30 tiendas.

Por que no scrapear cada tienda por separado:
  - scry ya resuelve el problema dificil (matchear "Ragavan, Nimble Pilferer
    (Borderless) [MH2] NM Foil" contra el catalogo de cada tienda),
  - server-rendera cada oferta con data-* limpios (precio numerico, url directa),
  - es 1 request en vez de 30.

Flujo:
    nombre --> /card/{slug}        (card_id + ofertas cacheadas)
    card_id --> /search_stream     (SSE: dispara refresco en las 30 tiendas)
    card_id --> /buscar_cache      (HTML con las ofertas ya frescas)

Nota: /buscar existe pero bloquea >45s. No lo usamos.
"""
from __future__ import annotations

import json
import re
from collections.abc import Iterator
from dataclasses import dataclass
from urllib.parse import urlparse

from bs4 import BeautifulSoup

from ..http import PoliteSession
from ..models import Offer
from ..ports import Progress
from ..text import normalize_name

BASE = "https://scry.cl"
_UUID = re.compile(r"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}")


def is_marketplace(url: str) -> bool:
    """True si la oferta la publica un particular en el marketplace de scry.

    El discriminador es el dominio, no el nombre. Las tiendas establecidas
    despachan desde su propio sitio (catlotus.cl, gameofmagicsingles.cl,
    www.paytowin.cl...); los vendedores particulares cuelgan todos de
    marketplace.scry.cl. Medido sobre 2 cartas: 62 ofertas de 31 vendedores
    distintos bajo ese unico dominio.
    """
    host = urlparse(url).netloc.lower()
    return host == "scry.cl" or host.endswith(".scry.cl")



def suggest_names(sess: PoliteSession, q: str, limit: int = 10) -> list[str]:
    """Nombres de carta sugeridos. Sirve para corregir tipeos antes de buscar."""
    if len(q.strip()) < 2:
        return []
    html = sess.get(f"{BASE}/autocomplete", params={"q": q}).text
    seen: list[str] = []
    for m in re.finditer(r"scryGoToCardPage\('((?:[^'\\]|\\.)*)'", html):
        name = m.group(1).replace("\\'", "'").replace("\\\\", "\\")
        if name not in seen:
            seen.append(name)
        if len(seen) >= limit:
            break
    return seen


def parse_offers_html(html: str, fallback_name: str = "") -> list[Offer]:
    soup = BeautifulSoup(html, "lxml")
    offers: list[Offer] = []
    seen: set[str] = set()

    for a in soup.select('a[data-track-type="store_offer_click"]'):
        d = a.attrs
        price = d.get("data-price-clp")
        url = d.get("data-product-url") or d.get("href") or ""
        if not price or not str(price).strip().isdigit():
            continue

        key = d.get("data-variant-key") or url
        if key in seen:
            continue
        seen.add(key)

        title = (d.get("data-offer-title") or a.get_text(" ", strip=True)).strip()
        lowered = title.lower()
        cond = next((c for c in ("near mint", "lightly played", "moderately played",
                                 "heavily played", "damaged") if c in lowered), "")

        offers.append(Offer(
            store=d.get("data-store-name") or "?",
            card_name=d.get("data-card-name") or fallback_name,
            title=title,
            price_clp=int(price),
            url=url,
            finish="Foil" if "foil" in lowered else "Normal",
            condition=cond.title(),
            key=key or "",
            source="scry",
            marketplace=is_marketplace(url),
        ))

    return sorted(offers, key=lambda o: o.price_clp)


def find_card(sess: PoliteSession, name: str) -> tuple[str | None, list[Offer]]:
    """Abre /card/{slug}. Devuelve (card_id, ofertas cacheadas).

    Es 1 solo request y suele alcanzar. El refresco en vivo es opcional.
    """
    r = sess.get(f"{BASE}/card/{normalize_name(name)}")
    html = r.text
    offers = parse_offers_html(html, name)

    card_id = None
    m = re.search(r'data-card-id="(' + _UUID.pattern + r')"', html) or _UUID.search(html)
    if m:
        card_id = m.group(1) if m.lastindex else m.group(0)

    return card_id, offers


def listen_refresh(sess: PoliteSession, card_id: str, finish: str = "",
                   timeout: float = 120.0) -> Iterator[dict]:
    """Consume el SSE que dispara la busqueda en vivo en las ~30 tiendas.

    Genera dicts de progreso: {'store':..., 'done':n, 'total':30}.
    OJO: cada llamada hace que scry golpee 30 tiendas. Usalo con criterio,
    no en loop sobre una decklist de 100 cartas.
    """
    r = sess.get(
        f"{BASE}/search_stream",
        params={"card_id": card_id, "finish": finish},
        headers={"Accept": "text/event-stream"},
        stream=True, timeout=timeout,
    )
    with r:
        for line in r.iter_lines(decode_unicode=True):
            if not line or not line.startswith("data:"):
                continue
            body = line[5:].strip()
            if not body:
                continue
            try:
                yield json.loads(body)
            except json.JSONDecodeError:
                continue


def read_cached_offers(sess: PoliteSession, card_id: str, finish: str = "",
                       name: str = "") -> list[Offer]:
    r = sess.get(f"{BASE}/buscar_cache", params={"card_id": card_id, "finish": finish})
    return parse_offers_html(r.text, name)


@dataclass
class ScrySource:
    """Cumple FuentePrincipal. Unico lugar que sabe como habla scry.cl.

    Contrato: https://scry.cl/openapi.json
    """
    sess: PoliteSession
    name: str = "scry.cl"

    def identify_card(self, name: str) -> tuple[str | None, list[Offer]]:
        return find_card(self.sess, name)

    def find_offers(self, card_name: str) -> list[Offer]:
        return find_card(self.sess, card_name)[1]

    def suggest_names(self, text: str) -> list[str]:
        """Best-effort: una sugerencia que falla no merece romper la busqueda."""
        try:
            return suggest_names(self.sess, text)
        except Exception:
            return []

    def refresh_offers(self, card_id: str) -> Iterator[Progress]:
        """Traduce el SSE del vendor a Avance antes de cruzar el puerto."""
        for ev in listen_refresh(self.sess, card_id):
            yield Progress(
                store=ev.get("store") or "...",
                done=ev.get("done") or 0,
                total=ev.get("total") or 30,
            )
