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

import json
import re
from dataclasses import dataclass
from pathlib import Path

from muchi.paths import ROOT
from ..http import PoliteSession
from ..models import Offer
from ..ports import InventoryUnavailable

API = "https://api2.moxfield.com/v3/decks/all/"
_LIST_URL = re.compile(r"moxfield\.com/decks/([A-Za-z0-9_-]+)")


class ListNotFound(LookupError):
    """La lista no existe o no es publica."""


@dataclass(frozen=True)
class InventoryList:
    """Una lista de Moxfield usada como catalogo de tienda."""
    store: str
    label: str       # "Rojo", "Tierras", "Japo/Foil"...
    deck_id: str
    rate: int = 700     # CLP por dolar de CardKingdom


@dataclass
class ImportSummary:
    offers: list[Offer]
    without_price: int = 0
    copies_total: int = 0


def extract_list_id(url: str) -> str:
    """Acepta la URL completa o el id pelado."""
    m = _LIST_URL.search(url or "")
    if m:
        return m.group(1)
    clean = (url or "").strip().strip("/")
    if clean and "/" not in clean:
        return clean
    raise ValueError(f"No reconozco una lista de Moxfield en: {url!r}")


def read_ck_price(prices: dict, is_foil: bool) -> float | None:
    """El foil se cotiza por ck_foil. Si falta, no inventamos con el no-foil."""
    key = "ck_foil" if is_foil else "ck"
    value = (prices or {}).get(key)
    try:
        return float(value) if value else None
    except (TypeError, ValueError):
        return None


def fetch_inventory(sess: PoliteSession, inv: InventoryList) -> ImportSummary:
    import requests

    try:
        data = sess.get(f"{API}{inv.deck_id}").json()
    except requests.HTTPError as e:
        if getattr(e.response, "status_code", None) in (403, 404):
            raise ListNotFound(inv.deck_id) from e
        raise

    public_url = data.get("publicUrl") or f"https://moxfield.com/decks/{inv.deck_id}"

    entries: list[dict] = []
    for board in (data.get("boards") or {}).values():
        entries.extend((board.get("cards") or {}).values())

    summary = ImportSummary(offers=[])
    for e in entries:
        card_name = e.get("card") or {}
        name = (card_name.get("name") or "").strip()
        if not name:
            continue

        quantity = int(e.get("quantity") or 0)
        is_foil = bool(e.get("isFoil")) or (e.get("finish") or "") == "foil"
        ck = read_ck_price(card_name.get("prices") or {}, is_foil)
        if ck is None:
            summary.without_price += 1
            continue

        summary.copies_total += quantity
        edition = (card_name.get("set") or "").upper()
        title = name + (f" [{edition}]" if edition else "")
        title += " - Foil" if is_foil else ""

        summary.offers.append(Offer(
            store=inv.store,
            card_name=name,
            title=f"{title} ({inv.label})",
            price_clp=int(round(ck * inv.rate)),
            url=public_url,
            finish="Foil" if is_foil else "Normal",
            condition="",
            stock=quantity or None,
            source="moxfield",
            marketplace=False,
            key=f"{inv.store}:{inv.deck_id}:{card_name.get('scryfall_id') or name}"
                f":{'f' if is_foil else 'n'}",
        ))

    summary.offers.sort(key=lambda o: o.price_clp)
    return summary


CONFIG = ROOT / "moxfield-inventories.json"


@dataclass
class MoxfieldInventory:
    """Cumple InventarioPublicado. Unico lugar que sabe de Moxfield y de CK.

    Contrato: https://api2.moxfield.com/v3/decks/all/<publicId>
    """
    sess: PoliteSession
    store: str
    inventories: list[InventoryList]

    def list_rates(self) -> list[tuple[str, int]]:
        return [(i.label, i.rate) for i in self.inventories]

    def import_offers(self, progress=None) -> tuple[list[Offer], int]:
        """Baja cada lista y las junta. Informa cuantas quedaron sin precio."""
        offers: list[Offer] = []
        without_price = 0

        for i, inv in enumerate(self.inventories):
            if progress:
                progress(i, len(self.inventories), inv.label)
            try:
                r = fetch_inventory(self.sess, inv)
            except ListNotFound as e:
                raise InventoryUnavailable(inv.deck_id) from e
            offers += r.offers
            without_price += r.without_price

        return offers, without_price


def load_inventory(sess: PoliteSession,
                   path: Path | str = CONFIG) -> MoxfieldInventory | None:
    """Lee la config. Sin archivo no hay inventario: la feature es opcional."""
    path = Path(path)
    if not path.exists():
        return None

    cfg = json.loads(path.read_text(encoding="utf-8"))
    lists = [
        InventoryList(cfg.get("tienda") or "Inventario Moxfield", l["etiqueta"],
                      extract_list_id(l["url"]), int(l.get("tasa", 700)))
        for l in cfg.get("listas", [])
    ]

    if not lists:
        return None
    return MoxfieldInventory(sess, lists[0].store, lists)
