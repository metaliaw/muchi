"""Indice local de las tiendas que Muchi consulta directo, sin pasar por scry.

Por que hace falta: /products.json de Shopify no acepta busqueda, solo pagina el
catalogo entero. Asi que en vez de pedirle a la tienda una consulta por carta
-- imposible -- bajamos su catalogo una vez, lo guardamos en SQLite y buscamos
localmente. PDA Chile son ~3.250 productos en 13 requests: unos 20 segundos.

Esto es descarga masiva, no una consulta puntual. Reindexa con criterio: una vez
al dia sobra, y el catalogo local sirve mientras tanto.
"""
from __future__ import annotations

import sqlite3
from collections.abc import Callable
from datetime import datetime, timezone

from .http import PoliteSession
from .models import Offer
from .sources import shopify
from .texto import slug

_ESQUEMA = """
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


def asegurar_tablas(cx: sqlite3.Connection) -> None:
    cx.executescript(_ESQUEMA)


def indexar(sess: PoliteSession, cx: sqlite3.Connection, tienda: str, base: str,
            progreso: Callable[[int, int], None] | None = None) -> int:
    """Baja el catalogo completo de una tienda Shopify y lo guarda.

    Reemplaza lo que hubiera de esa tienda: los precios y el stock cambian, y
    quedarnos con filas viejas seria peor que no tenerlas.
    """
    asegurar_tablas(cx)

    filas: list[tuple] = []
    productos = 0
    for p in shopify.catalogo(sess, base):
        productos += 1
        info = shopify.parse_titulo(p.get("title", ""))
        for o in shopify.ofertas_de_producto(p, tienda, base):
            filas.append((
                f"{tienda}:{o.key}", tienda, slug(info["nombre"]), info["nombre"],
                o.title, o.price_clp, o.url, o.finish, o.condition, o.language,
            ))
        if progreso and productos % 250 == 0:
            progreso(productos, len(filas))

    cx.execute("DELETE FROM catalogo WHERE tienda = ?", (tienda,))
    cx.executemany(
        "INSERT OR REPLACE INTO catalogo (clave, tienda, carta_slug, carta, titulo,"
        " precio, url, acabado, condicion, idioma) VALUES (?,?,?,?,?,?,?,?,?,?)",
        filas,
    )
    cx.execute(
        "INSERT OR REPLACE INTO catalogo_meta (tienda, actualizado, productos)"
        " VALUES (?,?,?)",
        (tienda, datetime.now(timezone.utc).isoformat(timespec="seconds"), productos),
    )
    cx.commit()
    return len(filas)


def guardar_ofertas(cx: sqlite3.Connection, tienda: str, ofertas: list[Offer],
                    productos: int | None = None) -> int:
    """Reemplaza el catalogo de una tienda con las ofertas dadas.

    Lo usa el importador de Moxfield, que ya trae Offer armadas y no necesita
    pasar por el parseo de titulos de Shopify.
    """
    asegurar_tablas(cx)
    filas = [
        (o.key or o.url, tienda, slug(o.card_name), o.card_name, o.title,
         o.price_clp, o.url, o.finish, o.condition, o.language)
        for o in ofertas
    ]
    cx.execute("DELETE FROM catalogo WHERE tienda = ?", (tienda,))
    cx.executemany(
        "INSERT OR REPLACE INTO catalogo (clave, tienda, carta_slug, carta, titulo,"
        " precio, url, acabado, condicion, idioma) VALUES (?,?,?,?,?,?,?,?,?,?)",
        filas,
    )
    cx.execute(
        "INSERT OR REPLACE INTO catalogo_meta (tienda, actualizado, productos)"
        " VALUES (?,?,?)",
        (tienda, datetime.now(timezone.utc).isoformat(timespec="seconds"),
         productos if productos is not None else len(ofertas)),
    )
    cx.commit()
    return len(filas)


def buscar(cx: sqlite3.Connection, nombre: str) -> list[Offer]:
    """Ofertas locales para una carta. Compara por slug, no por texto literal."""
    asegurar_tablas(cx)
    filas = cx.execute(
        "SELECT tienda, carta, titulo, precio, url, acabado, condicion, idioma"
        " FROM catalogo WHERE carta_slug = ? ORDER BY precio",
        (slug(nombre),),
    ).fetchall()

    return [
        Offer(
            store=f["tienda"], card_name=f["carta"], title=f["titulo"],
            price_clp=f["precio"], url=f["url"], finish=f["acabado"] or "",
            condition=f["condicion"] or "", language=f["idioma"] or "",
            source="directo", marketplace=False, key=f["url"],
        )
        for f in filas
    ]


def estado(cx: sqlite3.Connection) -> list[sqlite3.Row]:
    """Que tiendas hay indexadas y cuando."""
    asegurar_tablas(cx)
    return cx.execute(
        "SELECT m.tienda, m.actualizado, m.productos,"
        " (SELECT COUNT(*) FROM catalogo c WHERE c.tienda = m.tienda) AS ofertas"
        " FROM catalogo_meta m ORDER BY m.tienda"
    ).fetchall()
