"""Indice local de las tiendas que Muchi consulta directo, sin pasar por scry.

Por que hace falta: hay tiendas que no aceptan una busqueda por carta, solo
publican el catalogo entero. Asi que lo bajamos una vez, lo guardamos aca y
buscamos localmente. PDA Chile son ~3.250 productos: unos 20 segundos.

Quien lo baja es la Fuente, no este modulo: aca solo se Guarda y se Busca. La
descarga vive en sources/, y el reemplazo lo Coordina tiendas.py.
"""
from __future__ import annotations

import sqlite3
from dataclasses import dataclass
from datetime import datetime, timezone

from .models import Offer
from .text import normalize_name

_SCHEMA = """
CREATE TABLE IF NOT EXISTS catalogo (
    clave       TEXT PRIMARY KEY,
    tienda      TEXT NOT NULL,
    carta_slug  TEXT NOT NULL,
    carta       TEXT NOT NULL,
    titulo      TEXT NOT NULL,
    precio      INTEGER NOT NULL,
    url         TEXT NOT NULL,
    acabado     TEXT,
    condicion   TEXT,
    idioma      TEXT
);
CREATE INDEX IF NOT EXISTS ix_catalogo_carta ON catalogo (carta_slug);

CREATE TABLE IF NOT EXISTS catalogo_meta (
    tienda      TEXT PRIMARY KEY,
    actualizado TEXT NOT NULL,
    productos   INTEGER NOT NULL
);
"""


def ensure_tables(cx: sqlite3.Connection) -> None:
    cx.executescript(_SCHEMA)


def save_offers(cx: sqlite3.Connection, store: str, offers: list[Offer],
                products: int | None = None) -> int:
    """Reemplaza el catalogo de una tienda con las ofertas dadas.

    Lo usa el importador de Moxfield, que ya trae Oferta armadas y no necesita
    pasar por el parseo de titulos de Shopify.
    """
    ensure_tables(cx)
    rows = [
        (o.key or o.url, store, normalize_name(o.card_name), o.card_name, o.title,
         o.price_clp, o.url, o.finish, o.condition, o.language)
        for o in offers
    ]
    cx.execute("DELETE FROM catalogo WHERE tienda = ?", (store,))
    cx.executemany(
        "INSERT OR REPLACE INTO catalogo (clave, tienda, carta_slug, carta, titulo,"
        " precio, url, acabado, condicion, idioma) VALUES (?,?,?,?,?,?,?,?,?,?)",
        rows,
    )
    cx.execute(
        "INSERT OR REPLACE INTO catalogo_meta (tienda, actualizado, productos)"
        " VALUES (?,?,?)",
        (store, datetime.now(timezone.utc).isoformat(timespec="seconds"),
         products if products is not None else len(offers)),
    )
    cx.commit()
    return len(rows)


def find_local_offers(cx: sqlite3.Connection, name: str) -> list[Offer]:
    """Ofertas locales para una carta. Compara por slug, no por texto literal."""
    ensure_tables(cx)
    rows = cx.execute(
        "SELECT tienda, carta, titulo, precio, url, acabado, condicion, idioma"
        " FROM catalogo WHERE carta_slug = ? ORDER BY precio",
        (normalize_name(name),),
    ).fetchall()

    return [
        Offer(
            store=f["tienda"], card_name=f["carta"], title=f["titulo"],
            price_clp=f["precio"], url=f["url"], finish=f["acabado"] or "",
            condition=f["condicion"] or "", language=f["idioma"] or "",
            source="directo", marketplace=False, key=f["url"],
        )
        for f in rows
    ]


def read_catalog_status(cx: sqlite3.Connection) -> list[sqlite3.Row]:
    """Que tiendas hay indexadas y cuando."""
    ensure_tables(cx)
    return cx.execute(
        "SELECT m.tienda, m.actualizado, m.productos,"
        " (SELECT COUNT(*) FROM catalogo c WHERE c.tienda = m.tienda) AS ofertas"
        " FROM catalogo_meta m ORDER BY m.tienda"
    ).fetchall()


@dataclass
class IndexedOffers:
    """Cumple FuenteOfertas con lo que ya bajamos a SQLite.

    Su nombre no es el de una tienda: adentro viven varias. El descarte por
    tienda lo hace quien la consulta, oferta por oferta.
    """
    cx: sqlite3.Connection
    name: str = "indice local"

    def find_offers(self, card_name: str) -> list[Offer]:
        return find_local_offers(self.cx, card_name)
