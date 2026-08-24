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
import unicodedata
from collections.abc import Iterator

from bs4 import BeautifulSoup

from ..http import PoliteSession
from ..models import Offer

BASE = "https://scry.cl"
_UUID = re.compile(r"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}")


def slug(nombre: str) -> str:
    """'Ragavan, Nimble Pilferer' -> 'ragavan-nimble-pilferer'."""
    s = unicodedata.normalize("NFD", nombre)
    s = "".join(c for c in s if unicodedata.category(c) != "Mn")
    s = re.sub(r"[^a-zA-Z0-9]+", "-", s).strip("-").lower()
    return re.sub(r"-{2,}", "-", s)


def autocomplete(sess: PoliteSession, q: str, limite: int = 10) -> list[str]:
    """Nombres de carta sugeridos. Sirve para corregir tipeos antes de buscar."""
    if len(q.strip()) < 2:
        return []
    html = sess.get(f"{BASE}/autocomplete", params={"q": q}).text
    vistos: list[str] = []
    for m in re.finditer(r"scryGoToCardPage\('((?:[^'\\]|\\.)*)'", html):
        nombre = m.group(1).replace("\\'", "'").replace("\\\\", "\\")
        if nombre not in vistos:
            vistos.append(nombre)
        if len(vistos) >= limite:
            break
    return vistos


def _ofertas_desde_html(html: str, nombre_fallback: str = "") -> list[Offer]:
    sopa = BeautifulSoup(html, "lxml")
    ofertas: list[Offer] = []
    vistos: set[str] = set()

    for a in sopa.select('a[data-track-type="store_offer_click"]'):
        d = a.attrs
        precio = d.get("data-price-clp")
        url = d.get("data-product-url") or d.get("href") or ""
        if not precio or not str(precio).strip().isdigit():
            continue

        clave = d.get("data-variant-key") or url
        if clave in vistos:
            continue
        vistos.add(clave)

        titulo = (d.get("data-offer-title") or a.get_text(" ", strip=True)).strip()
        bajo = titulo.lower()
        cond = next((c for c in ("near mint", "lightly played", "moderately played",
                                 "heavily played", "damaged") if c in bajo), "")

        ofertas.append(Offer(
            store=d.get("data-store-name") or "?",
            card_name=d.get("data-card-name") or nombre_fallback,
            title=titulo,
            price_clp=int(precio),
            url=url,
            finish="Foil" if "foil" in bajo else "Normal",
            condition=cond.title(),
            key=clave or "",
            source="scry",
        ))

    return sorted(ofertas, key=lambda o: o.price_clp)


def buscar_carta(sess: PoliteSession, nombre: str) -> tuple[str | None, list[Offer]]:
    """Abre /card/{slug}. Devuelve (card_id, ofertas cacheadas).

    Es 1 solo request y suele alcanzar. El refresco en vivo es opcional.
    """
    r = sess.get(f"{BASE}/card/{slug(nombre)}")
    html = r.text
    ofertas = _ofertas_desde_html(html, nombre)

    card_id = None
    m = re.search(r'data-card-id="(' + _UUID.pattern + r')"', html) or _UUID.search(html)
    if m:
        card_id = m.group(1) if m.lastindex else m.group(0)

    return card_id, ofertas


def refrescar(sess: PoliteSession, card_id: str, finish: str = "",
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
        for linea in r.iter_lines(decode_unicode=True):
            if not linea or not linea.startswith("data:"):
                continue
            cuerpo = linea[5:].strip()
            if not cuerpo:
                continue
            try:
                yield json.loads(cuerpo)
            except json.JSONDecodeError:
                continue


def ofertas_cacheadas(sess: PoliteSession, card_id: str, finish: str = "",
                      nombre: str = "") -> list[Offer]:
    r = sess.get(f"{BASE}/buscar_cache", params={"card_id": card_id, "finish": finish})
    return _ofertas_desde_html(r.text, nombre)
