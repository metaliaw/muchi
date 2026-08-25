"""El historico de precios de una carta, en vocabulario de negocio.

Envuelve el almacenamiento para que la app pida "guardar precios" y no
"insertar filas". Lo que sale son dicts planos: una Row de sqlite es un tipo
del vendor, y un tipo del vendor no cruza este archivo.
"""
from __future__ import annotations

import sqlite3

from . import db
from .models import Offer


def save_prices(cx: sqlite3.Connection, card_name: str,
                offers: list[Offer]) -> None:
    """Deja una foto de lo que valia esta carta hoy."""
    db.save_offers(cx, card_name, offers)


def read_history(cx: sqlite3.Connection, card_name: str) -> list[dict]:
    """Minimo y promedio por fecha. Vacio si nunca la buscamos."""
    return [dict(row) for row in db.read_history(cx, card_name)]
