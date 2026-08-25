"""Que tiendas tenemos indexadas, y como se indexa una.

Nucleo otra vez: recibe un puerto y guarda el resultado. No sabe si detras
hay Shopify, una lista publicada o cualquier otra cosa que publique catalogo.
"""
from __future__ import annotations

import sqlite3

from . import catalog
from .ports import StoreCatalog, StoreStatus, PublishedInventory


def read_stores_status(cx: sqlite3.Connection,
                       source: StoreCatalog) -> list[StoreStatus]:
    """Cada tienda indexable, con lo que el indice local sabe de ella."""
    rows = {f["tienda"]: f for f in catalog.read_catalog_status(cx)}

    return [
        _build_status(store, url, rows.get(store))
        for store, url in source.list_indexable_stores().items()
    ]


def read_inventory_status(cx: sqlite3.Connection,
                          store: str) -> StoreStatus | None:
    """El estado de una sola tienda, o None si nunca se indexo."""
    rows = catalog.read_catalog_status(cx)
    row = next((f for f in rows if f["tienda"] == store), None)
    return _build_status(store, "", row) if row else None


def _build_status(store: str, url: str, row) -> StoreStatus:
    if row is None:
        return StoreStatus(store=store, url=url)
    return StoreStatus(
        store=store, url=url, indexed=True,
        offers=row["ofertas"], products=row["productos"],
        updated=(row["actualizado"] or "")[:16],
    )


def index_store(cx: sqlite3.Connection, source: StoreCatalog,
                store: str, url: str, progress=None) -> int:
    """Baja el catalogo de una tienda y reemplaza lo que hubiera de ella.

    Reemplaza en vez de sumar: los precios y el stock cambian, y quedarnos
    con filas viejas seria peor que no tenerlas.
    """
    offers, products = source.download_offers(store, url, progress)
    return catalog.save_offers(cx, store, offers, products)


def import_inventory(cx: sqlite3.Connection, inv: PublishedInventory,
                     progress=None) -> tuple[int, int]:
    """Trae las listas publicadas y las guarda como una tienda mas."""
    offers, without_price = inv.import_offers(progress)
    saved = catalog.save_offers(cx, inv.store, offers)
    return saved, without_price
