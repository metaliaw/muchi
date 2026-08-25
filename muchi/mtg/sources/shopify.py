"""Fuente directa para las tiendas Shopify, sin intermediarios.

Sirve para dos cosas: contrastar los precios que reporta scry, y seguir
funcionando si scry se cae. Usa /products.json, el feed publico de Shopify
(NO /search/suggest.json: el robots.txt de dragondurmiente.cl prohibe /search).

Formato de titulo observado en ambas tiendas:
    "Growth Spiral (7054) [SLD - 7054]"
    "Ragavan, Nimble Pilferer (Borderless) [MH2 - 138]"
y la variante trae "Near Mint / English / Foil".
"""
from __future__ import annotations

import re
from collections.abc import Callable, Iterator
from dataclasses import dataclass

from ..http import PoliteSession
from ..models import Offer

# Tiendas Shopify que Muchi puede consultar directo.
#
# Dragon Durmiente y PayToWin ya estan en scry: aca sirven para contrastar
# precios o si scry se cae. PDA Chile NO esta en scry, asi que esta es la unica
# via para verla. Su robots.txt permite a User-agent: * -- pero bloquea
# explicitamente a los crawlers de IA (ClaudeBot, GPTBot, CCBot...). Muchi no es
# ninguno de esos y no se hace pasar por nadie, pero bajarle el catalogo entero
# es descarga masiva: por eso la indexacion es manual y no automatica.
STORES = {
    "Dragon Durmiente": "https://dragondurmiente.cl",
    "PayToWin": "https://www.paytowin.cl",
    "PDA Chile": "https://www.pdachile.cl",
}

# Tiendas chilenas que Muchi NO puede consultar, y por que. Se muestran en la
# app como enlaces para que las revises a mano en vez de fingir que no existen.
BLOCKED_STORES = {
    "Magic Chile": (
        "https://www.magic-chile.cl",
        "Su robots.txt bloquea a todos los bots (User-agent: * -> Disallow: /).",
    ),
    "Gaming Place": (
        "https://www.gamingplace.cl",
        "El servidor rechaza las peticiones automatizadas con 403.",
    ),
}

_TITLE = re.compile(
    r"^(?P<nombre>.+?)\s*(?:\((?P<variante>[^)]*)\))?\s*"
    r"\[(?P<set>[^\]\-]+?)\s*-\s*(?P<cn>[^\]]+)\]\s*$"
)


def parse_title(title: str) -> dict:
    m = _TITLE.match(title.strip())
    if not m:
        return {"nombre": title.strip(), "variante": "", "set": "", "cn": ""}
    d = m.groupdict()
    return {k: (v or "").strip() for k, v in d.items()}


def split_variant(v: str) -> tuple[str, str, str]:
    """'Near Mint / English / Foil' -> (condicion, idioma, acabado)."""
    parts = [p.strip() for p in (v or "").split("/")]
    parts += [""] * (3 - len(parts))
    return parts[0], parts[1], parts[2] or "Normal"


def paginate_catalog(sess: PoliteSession, base: str,
                     max_pages: int = 200) -> Iterator[dict]:
    """Pagina /products.json. Son muchos requests: guardalo en SQLite y reusalo."""
    for page in range(1, max_pages + 1):
        data = sess.get(f"{base}/products.json",
                        params={"limit": 250, "page": page}).json()
        products = data.get("products") or []
        if not products:
            return
        yield from products


def build_product_offers(p: dict, store: str, base: str) -> list[Offer]:
    info = parse_title(p.get("title", ""))
    handle = p.get("handle", "")
    out: list[Offer] = []

    for v in p.get("variants") or []:
        if not v.get("available"):
            continue
        try:
            price = int(round(float(v.get("price") or 0)))
        except (TypeError, ValueError):
            continue
        if price <= 0:
            continue

        cond, language, finish = split_variant(v.get("title", ""))
        label = f"{info['nombre']}"
        if info["variante"]:
            label += f" ({info['variante']})"
        if info["set"]:
            label += f" [{info['set']} - {info['cn']}]"

        out.append(Offer(
            store=store,
            card_name=info["nombre"],
            title=f"{label} - {cond} {finish}".strip(),
            price_clp=price,
            url=f"{base}/products/{handle}?variant={v.get('id')}",
            finish=finish,
            condition=cond,
            language=language,
            source="shopify",
            key=f"{store}:{v.get('id') or ''}",
        ))

    return out


def download_offers(sess: PoliteSession, store: str, base: str,
                    progress: Callable[[int, int], None] | None = None
                      ) -> tuple[list[Offer], int]:
    """Baja el catalogo entero de una tienda y lo devuelve como ofertas.

    Son muchos requests seguidos: esto se corre a mano, y lo que vuelve se
    guarda una vez. Avisa el avance cada 250 productos.
    """
    offers: list[Offer] = []
    products = 0

    for p in paginate_catalog(sess, base):
        products += 1
        offers += build_product_offers(p, store, base)
        if progress and products % 250 == 0:
            progress(products, len(offers))

    return offers, products


@dataclass
class ShopifyCatalog:
    """Cumple CatalogoTienda para las tiendas que corren sobre Shopify."""
    sess: PoliteSession

    def list_indexable_stores(self) -> dict[str, str]:
        return dict(STORES)

    def list_blocked_stores(self) -> dict[str, tuple[str, str]]:
        return dict(BLOCKED_STORES)

    def download_offers(self, store: str, url: str,
                        progress: Callable[[int, int], None] | None = None
                          ) -> tuple[list[Offer], int]:
        return download_offers(self.sess, store, url, progress)
