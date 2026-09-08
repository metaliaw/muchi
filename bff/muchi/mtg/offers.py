"""Reune las ofertas de todas las fuentes y las deja listas para mostrar.

Este modulo es nucleo: recibe puertos y nunca importa una fuente concreta.
Cambiar de agregador no toca una linea de aqui.
"""
from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor

from .constants import STOCK_CATALOG
from .models import Offer
from .settings import OfferSettings
from .ports import OfferSource, PrimarySource, StockVerifier


def find_suspicious_prices(offers: list[Offer], settings: OfferSettings) -> set[Offer]:
    """Detecta precios bajos aislados sin inventar un precio corregido.

    No usamos un piso fijo: una carta de $100 puede ser perfectamente real. En
    cambio comparamos una cola inferior pequeña con el resto de la Muestra.
    Los Defaults de Ofertas definen el Salto, el Tamaño mínimo y la Cola máxima.

    Es una advertencia, no un filtro. La persona todavia puede abrir la tienda
    y comprobar la variante exacta.
    """
    if len(offers) < settings.minimum_offers:
        return set()

    ordered = sorted(offers, key=lambda o: o.price_clp)
    max_low_count = min(
        settings.maximum_low_offers,
        len(ordered) - 1,
        int(len(ordered) * settings.maximum_low_share),
    )
    for low_count in range(1, max_low_count + 1):
        low = ordered[low_count - 1].price_clp
        high = ordered[low_count].price_clp
        if low > 0 and high >= low * settings.price_gap_ratio:
            return set(ordered[:low_count])
    return set()


def find_cheapest_offer(found: list[Offer],
                            suspicious: set[Offer]) -> Offer | None:
    """La menor oferta apta para el resumen y la insignia de mejor precio."""
    return next((offer for offer in found if offer not in suspicious), None)


def requires_stock_check(offer: Offer) -> bool:
    """True cuando la disponibilidad viene de una fuente no contrastada."""
    return offer.source in STOCK_CATALOG.unverified_sources


def verify_cheapest_stock(verifier: StockVerifier,
                          found: list[Offer],
                          settings: OfferSettings) -> dict[Offer, bool | None]:
    """Comprueba las primeras ofertas no sospechosas que pueden recibir clic."""
    suspicious = find_suspicious_prices(found, settings)
    candidates = [offer for offer in found if offer not in suspicious]
    candidates = candidates[:settings.stock_check_limit]
    if not candidates:
        return {}
    with ThreadPoolExecutor(max_workers=len(candidates)) as pool:
        states = pool.map(verifier.verify_stock, candidates)
    return dict(zip(candidates, states))


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
