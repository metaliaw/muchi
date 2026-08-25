"""Reune las ofertas de todas las fuentes y las deja listas para mostrar.

Este modulo es nucleo: recibe puertos y nunca importa una fuente concreta.
Cambiar de agregador no toca una linea de aqui.
"""
from __future__ import annotations

from .models import Offer
from .ports import OfferSource, PrimarySource


def find_offers(primary: PrimarySource, extras: list[OfferSource],
                card_name: str) -> tuple[str | None, list[Offer]]:
    """Pregunta a la fuente principal, suma las demas y ordena por precio.

    Si una tienda ya vino por la fuente principal, se descarta en las demas:
    esa version trae el precio ya normalizado, y duplicarla solo inflaria el
    conteo. Se descarta por tienda, nunca por URL.
    """
    card_id, offers = primary.identify_card(card_name)

    covered = {o.store for o in offers}
    offers += gather_extra_offers(extras, card_name, covered)

    return card_id, sorted(offers, key=lambda o: o.price_clp)


def gather_extra_offers(sources: list[OfferSource], card_name: str,
                        covered: set[str]) -> list[Offer]:
    """Consulta las fuentes secundarias. Una que falla no tumba la busqueda."""
    out: list[Offer] = []
    for source in sources:
        if source.name in covered:
            continue
        try:
            out += [o for o in source.find_offers(card_name)
                    if o.store not in covered]
        except Exception:
            continue
    return out


def filter_offers(offers: list[Offer], finish: str = "Todos",
                  stores: list[str] | None = None,
                  stores_only: bool = True) -> list[Offer]:
    """Aplica las preferencias de la barra lateral y reordena por precio."""
    out = [o for o in offers if not (stores_only and o.marketplace)]

    if finish == "Solo normal":
        out = [o for o in out if not o.is_foil]
    elif finish == "Solo foil":
        out = [o for o in out if o.is_foil]

    if stores:
        out = [o for o in out if o.store in stores]
    return sorted(out, key=lambda o: o.price_clp)
