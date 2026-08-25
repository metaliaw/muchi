"""SQLite: cache de busquedas + historico de precios.

El historico es la parte que en realidad importa a largo plazo: te deja ver
si una carta esta cara hoy o si vale la pena esperar.
"""
from __future__ import annotations

import sqlite3
from datetime import datetime, timezone
from pathlib import Path

from muchi.paths import DATA
from .models import Offer

PATH = DATA / "precios.db"

_SCHEMA = """
CREATE TABLE IF NOT EXISTS ofertas (
    ts        TEXT NOT NULL,
    carta     TEXT NOT NULL,
    tienda    TEXT NOT NULL,
    titulo    TEXT NOT NULL,
    precio    INTEGER NOT NULL,
    url       TEXT,
    acabado   TEXT,
    condicion TEXT,
    idioma    TEXT,
    fuente    TEXT,
    clave     TEXT
);
CREATE INDEX IF NOT EXISTS ix_ofertas_carta ON ofertas (carta, ts);
CREATE INDEX IF NOT EXISTS ix_ofertas_clave ON ofertas (clave, ts);
"""


def connect_database(path: Path | str = PATH) -> sqlite3.Connection:
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    cx = sqlite3.connect(path, check_same_thread=False)
    cx.row_factory = sqlite3.Row
    cx.executescript(_SCHEMA)
    return cx


def save_offers(cx: sqlite3.Connection, card_name: str, offers: list[Offer]) -> None:
    if not offers:
        return
    ts = datetime.now(timezone.utc).isoformat(timespec="seconds")
    cx.executemany(
        "INSERT INTO ofertas (ts, carta, tienda, titulo, precio, url,"
        " acabado, condicion, idioma, fuente, clave)"
        " VALUES (?,?,?,?,?,?,?,?,?,?,?)",
        [(ts, card_name, o.store, o.title, o.price_clp, o.url,
          o.finish, o.condition, o.language, o.source, o.key) for o in offers],
    )
    cx.commit()


def read_history(cx: sqlite3.Connection, card_name: str) -> list[sqlite3.Row]:
    return cx.execute(
        "SELECT ts, MIN(precio) AS minimo, AVG(precio) AS promedio, COUNT(*) AS n"
        " FROM ofertas WHERE carta = ? GROUP BY ts ORDER BY ts",
        (card_name,),
    ).fetchall()
