"""SQLite: cache de busquedas + historico de precios.

El historico es la parte que en realidad importa a largo plazo: te deja ver
si una carta esta cara hoy o si vale la pena esperar.
"""
from __future__ import annotations

import sqlite3
from datetime import datetime, timezone
from pathlib import Path

from .models import Offer

RUTA = Path(__file__).resolve().parent.parent / "data" / "precios.db"

_ESQUEMA = """
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


def conectar(ruta: Path | str = RUTA) -> sqlite3.Connection:
    ruta = Path(ruta)
    ruta.parent.mkdir(parents=True, exist_ok=True)
    cx = sqlite3.connect(ruta, check_same_thread=False)
    cx.row_factory = sqlite3.Row
    cx.executescript(_ESQUEMA)
    return cx


def guardar(cx: sqlite3.Connection, carta: str, ofertas: list[Offer]) -> None:
    if not ofertas:
        return
    ts = datetime.now(timezone.utc).isoformat(timespec="seconds")
    cx.executemany(
        "INSERT INTO ofertas (ts, carta, tienda, titulo, precio, url,"
        " acabado, condicion, idioma, fuente, clave)"
        " VALUES (?,?,?,?,?,?,?,?,?,?,?)",
        [(ts, carta, o.store, o.title, o.price_clp, o.url,
          o.finish, o.condition, o.language, o.source, o.key) for o in ofertas],
    )
    cx.commit()


def historico(cx: sqlite3.Connection, carta: str) -> list[sqlite3.Row]:
    return cx.execute(
        "SELECT ts, MIN(precio) AS minimo, AVG(precio) AS promedio, COUNT(*) AS n"
        " FROM ofertas WHERE carta = ? GROUP BY ts ORDER BY ts",
        (carta,),
    ).fetchall()
