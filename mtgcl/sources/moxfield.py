"""Importa inventarios publicados como listas de Moxfield.

Sirve para las tiendas que no tienen e-commerce y llevan su stock en Moxfield,
con precios derivados de CardKingdom por una tasa fija (el clasico "CK x 700").

La API que usa el propio sitio devuelve el precio de CardKingdom ya calculado:

    GET https://api2.moxfield.com/v3/decks/all/<publicId>
    -> boards.mainboard.cards[*].card.prices.ck        (no foil)
                                        .prices.ck_foil (foil)

O sea que "CK x 700" es exacto, no una estimacion.

Dos cosas que hay que respetar o los precios salen mal:

  - Los foils NO usan `ck` sino `ck_foil`. En estas listas hay 361 foils de
    2.249 entradas; tomar `ck` para todos los subvaluaria muchisimo.
  - Un 8% de las entradas no trae precio de CK. Se omiten y se informa cuantas,
    en vez de inventarles un valor.

El robots.txt de Moxfield permite /decks y /api. Aun asi la respuesta pesa ~2 MB
por lista: esto se corre a mano, no en cada busqueda.
"""
from __future__ import annotations

import re
from dataclasses import dataclass

from ..http import PoliteSession
from ..models import Offer

API = "https://api2.moxfield.com/v3/decks/all/"
_URL = re.compile(r"moxfield\.com/decks/([A-Za-z0-9_-]+)")


class ListaNoEncontrada(LookupError):
    """La lista no existe o no es publica."""


@dataclass(frozen=True)
class Inventario:
    """Una lista de Moxfield usada como catalogo de tienda."""
    tienda: str
    etiqueta: str       # "Rojo", "Tierras", "Japo/Foil"...
    deck_id: str
    tasa: int = 700     # CLP por dolar de CardKingdom


@dataclass
class Resumen:
    ofertas: list[Offer]
    sin_precio: int = 0
    copias: int = 0


def id_de_url(url: str) -> str:
    """Acepta la URL completa o el id pelado."""
    m = _URL.search(url or "")
    if m:
        return m.group(1)
    limpio = (url or "").strip().strip("/")
    if limpio and "/" not in limpio:
        return limpio
    raise ValueError(f"No reconozco una lista de Moxfield en: {url!r}")


def _precio_ck(precios: dict, es_foil: bool) -> float | None:
    """El foil se cotiza por ck_foil. Si falta, no inventamos con el no-foil."""
    clave = "ck_foil" if es_foil else "ck"
    valor = (precios or {}).get(clave)
    try:
        return float(valor) if valor else None
    except (TypeError, ValueError):
        return None


def inventario(sess: PoliteSession, inv: Inventario) -> Resumen:
    import requests

    try:
        datos = sess.get(f"{API}{inv.deck_id}").json()
    except requests.HTTPError as e:
        if getattr(e.response, "status_code", None) in (403, 404):
            raise ListaNoEncontrada(inv.deck_id) from e
        raise

    url_publica = datos.get("publicUrl") or f"https://moxfield.com/decks/{inv.deck_id}"

    entradas: list[dict] = []
    for tablero in (datos.get("boards") or {}).values():
        entradas.extend((tablero.get("cards") or {}).values())

    resumen = Resumen(ofertas=[])
    for e in entradas:
        carta = e.get("card") or {}
        nombre = (carta.get("name") or "").strip()
        if not nombre:
            continue

        cantidad = int(e.get("quantity") or 0)
        es_foil = bool(e.get("isFoil")) or (e.get("finish") or "") == "foil"
        ck = _precio_ck(carta.get("prices") or {}, es_foil)
        if ck is None:
            resumen.sin_precio += 1
            continue

        resumen.copias += cantidad
        edicion = (carta.get("set") or "").upper()
        titulo = nombre + (f" [{edicion}]" if edicion else "")
        titulo += " - Foil" if es_foil else ""

        resumen.ofertas.append(Offer(
            store=inv.tienda,
            card_name=nombre,
            title=f"{titulo} ({inv.etiqueta})",
            price_clp=int(round(ck * inv.tasa)),
            url=url_publica,
            finish="Foil" if es_foil else "Normal",
            condition="",
            stock=cantidad or None,
            source="moxfield",
            marketplace=False,
            key=f"{inv.tienda}:{inv.deck_id}:{carta.get('scryfall_id') or nombre}"
                f":{'f' if es_foil else 'n'}",
        ))

    resumen.ofertas.sort(key=lambda o: o.price_clp)
    return resumen
