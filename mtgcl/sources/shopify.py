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
from collections.abc import Iterator

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
TIENDAS = {
    "Dragon Durmiente": "https://dragondurmiente.cl",
    "PayToWin": "https://www.paytowin.cl",
    "PDA Chile": "https://www.pdachile.cl",
}

# Tiendas chilenas que Muchi NO puede consultar, y por que. Se muestran en la
# app como enlaces para que las revises a mano en vez de fingir que no existen.
FUERA_DE_ALCANCE = {
    "El Wombat Rabioso TCG": (
        "https://buscadorcartas-wombat.streamlit.app",
        "Su catalogo vive en la base de datos de su propia app; no hay feed publico.",
    ),
    "Magic Chile": (
        "https://www.magic-chile.cl",
        "Su robots.txt bloquea a todos los bots (User-agent: * -> Disallow: /).",
    ),
    "Gaming Place": (
        "https://www.gamingplace.cl",
        "El servidor rechaza las peticiones automatizadas con 403.",
    ),
}

_TITULO = re.compile(
    r"^(?P<nombre>.+?)\s*(?:\((?P<variante>[^)]*)\))?\s*"
    r"\[(?P<set>[^\]\-]+?)\s*-\s*(?P<cn>[^\]]+)\]\s*$"
)


def parse_titulo(titulo: str) -> dict:
    m = _TITULO.match(titulo.strip())
    if not m:
        return {"nombre": titulo.strip(), "variante": "", "set": "", "cn": ""}
    d = m.groupdict()
    return {k: (v or "").strip() for k, v in d.items()}


def _partes_variante(v: str) -> tuple[str, str, str]:
    """'Near Mint / English / Foil' -> (condicion, idioma, acabado)."""
    trozos = [p.strip() for p in (v or "").split("/")]
    trozos += [""] * (3 - len(trozos))
    return trozos[0], trozos[1], trozos[2] or "Normal"


def catalogo(sess: PoliteSession, base: str, max_paginas: int = 200) -> Iterator[dict]:
    """Pagina /products.json. Son muchos requests: guardalo en SQLite y reusalo."""
    for pagina in range(1, max_paginas + 1):
        datos = sess.get(f"{base}/products.json",
                         params={"limit": 250, "page": pagina}).json()
        productos = datos.get("products") or []
        if not productos:
            return
        yield from productos


def ofertas_de_producto(p: dict, tienda: str, base: str) -> list[Offer]:
    info = parse_titulo(p.get("title", ""))
    handle = p.get("handle", "")
    salida: list[Offer] = []

    for v in p.get("variants") or []:
        if not v.get("available"):
            continue
        try:
            precio = int(round(float(v.get("price") or 0)))
        except (TypeError, ValueError):
            continue
        if precio <= 0:
            continue

        cond, idioma, acabado = _partes_variante(v.get("title", ""))
        etiqueta = f"{info['nombre']}"
        if info["variante"]:
            etiqueta += f" ({info['variante']})"
        if info["set"]:
            etiqueta += f" [{info['set']} - {info['cn']}]"

        salida.append(Offer(
            store=tienda,
            card_name=info["nombre"],
            title=f"{etiqueta} - {cond} {acabado}".strip(),
            price_clp=precio,
            url=f"{base}/products/{handle}?variant={v.get('id')}",
            finish=acabado,
            condition=cond,
            language=idioma,
            source="shopify",
            key=str(v.get("id") or ""),
        ))

    return salida
