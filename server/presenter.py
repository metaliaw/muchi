"""Traduce el Dominio a JSON para el Front.

Las Reglas que antes vivían en `app.py` viven acá: el Front dibuja, no decide.
Así el Tratamiento, el Muchi Dólar y el Carrito se calculan una sola vez y del
mismo modo, sea cual sea la Interfaz que los consuma.
"""
from __future__ import annotations

import re
from decimal import Decimal

from muchi.mtg import optimizer, treatment
from muchi.mtg.models import Offer, Order
from muchi.mtg.search import SearchItem, SearchOffer, SearchState

# La API marca "unknown" cuando la Fuente no publica Stock: Agregadores como
# scry.cl indexan Precios, no Inventario. No es Ausencia de Carta.
STOCK_LABELS = {"available": "En Stock", "unavailable": "Agotado",
                "unknown": "No confirmado"}
SUSPICIOUS_REASON = re.compile(r"price_below_(\d+)_percent_median")
SUSPICIOUS_NOTE = ("Muy por debajo de las otras Ofertas; verifica la Variante "
                   "y el Precio final.")
# Los Acabados van en Dorado; el resto del Tratamiento, en Gris.
FOIL_TAGS = ("Foil", "Etched")
# El Muchi Dólar solo convierte Dólares. Una Oferta en otra Moneda se muestra
# con su Valor original y queda fuera del Carrito, porque nadie sabe cuánto es.
MUCHI_DOLAR_CURRENCY = "USD"


def read_suspicious_note(reason: str) -> str:
    """Traduce el Código de la API. Hoy solo emite uno; el resto pasa crudo."""
    if match := SUSPICIOUS_REASON.fullmatch(reason):
        return f"Precio bajo el {match[1]}% de la Mediana de su Moneda"
    return reason or "Precio fuera de Rango"


def convert_to_clp(offer: SearchOffer, muchi_dolar: int) -> Decimal | None:
    """Lleva una Oferta a Pesos. Devuelve None si su Moneda no tiene Cambio."""
    if offer.currency == "CLP":
        return offer.amount
    if offer.currency == MUCHI_DOLAR_CURRENCY:
        return offer.amount * muchi_dolar
    return None


def build_pills(offer: SearchOffer) -> list[dict]:
    """Las Pastillas de una Oferta: quién la vende, cómo viene y su Stock."""
    pills = [{"kind": "tienda", "text": offer.store}]
    for tag in treatment.build_treatment(offer).split(treatment.SEPARATOR):
        if tag:
            pills.append({"kind": "foil" if tag in FOIL_TAGS else "cond", "text": tag})
    label = STOCK_LABELS.get(offer.stock_status, offer.stock_status)
    pills.append({"kind": "tienda" if offer.stock_status == "available" else "cond",
                  "text": label})
    if offer.suspicious:
        pills.append({"kind": "cond",
                      "text": f"⚠ {read_suspicious_note(offer.suspicious_reason)}"})
    return pills


def build_offer(offer: SearchOffer, muchi_dolar: int) -> dict:
    price = convert_to_clp(offer, muchi_dolar)
    return {
        "card_name": offer.card_name,
        "store": offer.store,
        "amount": str(offer.amount),
        "currency": offer.currency,
        "price_clp": None if price is None else float(price),
        "url": offer.url,
        "stock_status": offer.stock_status,
        "stock_label": STOCK_LABELS.get(offer.stock_status, offer.stock_status),
        "suspicious": offer.suspicious,
        "note": SUSPICIOUS_NOTE if offer.suspicious else "",
        "action": "Verificar" if offer.suspicious else "Ver",
        "treatment": treatment.build_treatment(offer),
        "pills": build_pills(offer),
    }


def order_offers(offers: list[SearchOffer]) -> list[SearchOffer]:
    """Abre por Precio, barata primero, mezclando todas las Cartas.

    Cada Moneda va en su propio Bloque: comparar 4 Dólares contra 4000 Pesos
    por su Número sería mentir, y el Muchi Dólar es un Cambio de Muchi, no el
    Precio que la Tienda publica.
    """
    return sorted(offers, key=lambda offer: (offer.currency, offer.amount))


def pick_cheapest(offers: list[SearchOffer], muchi_dolar: int) -> SearchOffer | None:
    """La Oferta más barata que Muchi recomendaría: sin Alertas ni Agotados."""
    eligible = [offer for offer in offers
                if not offer.suspicious and offer.stock_status != "unavailable"
                and convert_to_clp(offer, muchi_dolar) is not None]
    return min(eligible, key=lambda offer: convert_to_clp(offer, muchi_dolar),
               default=None)


def build_state(state: SearchState) -> dict:
    return {
        "id": state.id, "status": state.status, "total": state.total,
        "processed": state.processed, "found": state.found,
        "errors": state.errors, "current_card": state.current_card,
        "done": state.done,
    }


def build_results(items: tuple[SearchItem, ...], muchi_dolar: int) -> dict:
    """Las Ofertas ya ordenadas, con la más barata marcada y el Resumen listo."""
    offers: list[SearchOffer] = []
    notices = []
    for item in items:
        if item.status == "source_error":
            notices.append({"level": "warning", "card": item.name,
                            "text": f"{item.name}: no se pudo completar la Consulta."})
        elif item.status == "not_found":
            notices.append({"level": "caption", "card": item.name,
                            "text": f"{item.name}: sin Ofertas."})
        offers.extend(item.offers)

    offers = order_offers(offers)
    cheapest = pick_cheapest(offers, muchi_dolar)
    rows = []
    for offer in offers:
        row = build_offer(offer, muchi_dolar)
        row["best"] = offer is cheapest
        rows.append(row)
    prices = [price for price in
              (convert_to_clp(offer, muchi_dolar) for offer in offers)
              if price is not None]
    return {
        "offers": rows,
        "notices": notices,
        "summary": {
            "lowest_clp": None if not prices else float(round(min(prices))),
            "offers": len(offers),
            "stores": len({offer.store for offer in offers}),
        },
    }


def build_cart(items: tuple[SearchItem, ...], shipping: int, muchi_dolar: int) -> dict:
    """El Carrito en CLP: solo Ofertas sin Alerta de Precio ni Stock agotado."""
    orders = [Order(item.quantity, item.name) for item in items]
    found: dict[str, list[Offer]] = {}
    converted = 0
    for item in items:
        for offer in item.offers:
            if offer.stock_status == "unavailable" or offer.suspicious:
                continue
            price = convert_to_clp(offer, muchi_dolar)
            if price is None:
                continue
            converted += offer.currency != "CLP"
            found.setdefault(item.name.lower(), []).append(Offer(
                store=offer.store, card_name=item.name, title=offer.card_name,
                price_clp=price, url=offer.url,
            ))
    plan = optimizer.build_optimal_plan(orders, found, shipping)
    stores = []
    for store in plan.stores:
        lines = [line for line in plan.lines if line.store == store]
        stores.append({
            "store": store,
            "cards": sum(line.quantity for line in lines),
            "subtotal": float(round(sum(line.subtotal for line in lines))),
            "lines": [{
                "card_name": line.card_name, "quantity": line.quantity,
                "unit_price": float(line.unit_price), "subtotal": float(line.subtotal),
                "url": line.url, "title": line.title,
            } for line in lines],
        })
    return {
        "total": float(plan.total),
        "cards_cost": float(plan.cards_cost),
        "shipping_cost": float(plan.shipping_cost),
        "shipping_per_store": plan.shipping_per_store,
        "converted_offers": converted,
        "muchi_dolar": muchi_dolar,
        "stores": stores,
        "missing": list(plan.missing),
    }
