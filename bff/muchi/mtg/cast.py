"""Main casta a los players y despues se baja del escenario.

Este es el unico modulo que importa una fuente concreta. De aqui para adentro
todo habla de puertos: FuentePrincipal, FuenteOfertas, RecomendadorMazo. Si
manana el agregador cierra, se cambia una linea aqui y nada mas.
"""
from __future__ import annotations

import sqlite3
from dataclasses import dataclass

from . import catalog, db
from .http import PoliteSession
from .ports import (
    StoreCatalog,
    OfferSource,
    PrimarySource,
    PublishedInventory,
    DeckAdvisor,
)
from .sources import store_api, edhrec, moxfield, scry, shopify


@dataclass(frozen=True)
class Cast:
    """El elenco ya armado. La app recibe esto y nunca nombra a un vendor."""
    cx: sqlite3.Connection
    primary: PrimarySource
    extras: list[OfferSource]
    advisor: DeckAdvisor
    stores: StoreCatalog
    inventory: PublishedInventory | None


def build_cast() -> Cast:
    """Abre la base, arma la sesion y reparte los papeles.

    El intervalo de 1.5s entre requests al mismo host no es decorativo: la
    fuente principal nos tiro un 429 durante el diseno.
    """
    sess = PoliteSession(min_interval=1.5)
    cx = db.connect_database()

    extras: list[OfferSource] = [catalog.IndexedOffers(cx)]
    extras += store_api.build_api_sources(sess)

    return Cast(
        cx=cx,
        primary=scry.ScrySource(sess),
        extras=extras,
        advisor=edhrec.EdhrecAdvisor(sess),
        stores=shopify.ShopifyCatalog(sess),
        inventory=moxfield.load_inventory(sess),
    )
