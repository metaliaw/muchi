"""Traduce el Dominio a JSON para el Front.

Las Reglas que antes vivían en `app.py` viven acá: el Front dibuja, no decide.
Así el Tratamiento, el Muchi Dólar y el Carrito se calculan una sola vez y del
mismo modo, sea cual sea la Interfaz que los consuma.
"""
from __future__ import annotations

from decimal import Decimal

from muchi.mtg import optimizer, treatment
from muchi.mtg.models import Offer, Order
from muchi.mtg.search import SearchItem, SearchOffer, SearchState

# La API marca "unknown" cuando la Fuente no publica Stock: Agregadores como
# scry.cl indexan Precios, no Inventario. No es Ausencia de Carta.
STOCK_LABELS = {"available": "En Stock", "unavailable": "Agotado",
                "unknown": "No confirmado"}
# Los Acabados van en Dorado; el resto del Tratamiento, en Gris.
FOIL_TAGS = ("Foil", "Etched")
# El Muchi Dólar solo convierte Dólares. Una Oferta en otra Moneda se muestra
# con su Valor original y queda fuera del Carrito, porque nadie sabe cuánto es.
MUCHI_DOLAR_CURRENCY = "USD"


def convert_to_clp(offer: SearchOffer, muchi_dolar: int) -> Decimal | None:
    """Lleva una Oferta a Pesos. Devuelve None si su Moneda no tiene Cambio."""
    if offer.currency == "CLP":
        return offer.amount
    if offer.currency == MUCHI_DOLAR_CURRENCY:
        return offer.amount * muchi_dolar
    return None


def build_pills(offer: SearchOffer, verified: bool = True) -> list[dict]:
    """Las Pastillas de una Oferta: quién la vende, cómo viene y su Stock."""
    pills = [{"kind": "tienda", "text": offer.store}]
    for tag in treatment.build_treatment(offer).split(treatment.SEPARATOR):
        if tag:
            pills.append({"kind": "foil" if tag in FOIL_TAGS else "cond", "text": tag})
    # Sin Re-verificación, el Stock es Ruido: toda Oferta diría lo mismo.
    if verified:
        label = STOCK_LABELS.get(offer.stock_status, offer.stock_status)
        pills.append({"kind": "tienda" if offer.stock_status == "available" else "cond",
                      "text": label})
    return pills


def build_offer(offer: SearchOffer, muchi_dolar: int,
                verified: bool = True) -> dict:
    price = convert_to_clp(offer, muchi_dolar)
    return {
        "card_name": offer.card_name,
        "store": offer.store,
        "amount": str(offer.amount),
        "currency": offer.currency,
        "price_clp": None if price is None else float(price),
        "url": offer.url,
        "stock_status": offer.stock_status,
        "stock_label": (STOCK_LABELS.get(offer.stock_status, offer.stock_status)
                        if verified else ""),
        # El Contrato conserva el Campo mientras la Medición está desactivada.
        "suspicious": False,
        "note": "",
        "action": "Ver",
        "treatment": treatment.build_treatment(offer),
        "pills": build_pills(offer, verified),
        # Lo que hace falta para pedir la Imagen de esta Impresion y no otra.
        "language": offer.language,
        "edition": offer.edition,
        "finish": offer.finish,
        "locations": list(offer.locations),
        "metadata": offer.metadata,
    }


# Buscar "Kuriboh" con MATCH_INCLUDES trae Winged Kuriboh y Linkuriboh: otras
# Cartas. Comparar sus Precios entre sí no dice nada, así que cada Tipo de
# Carta se ordena, se cuenta y se premia por separado.
MATCH_EXACT = "exact"
MATCH_INCLUDES = "includes"


# El Gemelo de `offer.MatchesCard` de muchi-api. La API ya filtró con ella; el
# Carrito la repite para volver a la Carta pedida cuando la Búsqueda fue ancha.
EDITION_BRACKETS = (" | ", " (", " [")
EDITION_DASHES = (" - ", " \u2013 ", " \u2014 ")
QUOTE_PAIRS = (("\u201c", "\u201d"), ('"', '"'), ("\u00ab", "\u00bb"))


def names_same_card(title: str, asked: str) -> bool:
    """El Título Nombra la Carta pedida, con o sin su Impresión detrás."""
    title = " ".join(title.lower().split())
    asked = " ".join(asked.lower().split())
    if not asked:
        return False
    if title == asked:
        return True
    if any(title.startswith(asked + bracket) for bracket in EDITION_BRACKETS):
        return True
    if any(opening + asked + closing in title for opening, closing in QUOTE_PAIRS):
        return True
    return any(follows_code(title, asked + dash) for dash in EDITION_DASHES)


def follows_code(title: str, opening: str) -> bool:
    """Tras un Guion solo Sigue la misma Carta si lo que viene es un Código."""
    if not title.startswith(opening):
        return False
    tail = title[len(opening):].split()
    return bool(tail) and any(character.isdigit() for character in tail[0])


def read_card_type(offer: SearchOffer, asked: str, match: str) -> str:
    """El Tipo de Carta al que Pertenece una Oferta.

    La API puede Nombrar la Carta funcional: sus Ediciones e Impresiones
    comparten esa Identidad. Sin ella, `exact` usa la Carta pedida e `includes`
    conserva cada Título distinto.
    """
    if offer.card_key:
        return offer.card_key
    if match == MATCH_INCLUDES:
        return offer.card_name.strip().lower()
    return asked.strip().lower()


def order_offers(offers: list[SearchOffer]) -> list[SearchOffer]:
    """Abre por Precio, barata primero, dentro de un mismo Tipo de Carta.

    Cada Moneda va en su propio Bloque: comparar 4 Dólares contra 4000 Pesos
    por su Número sería mentir, y el Muchi Dólar es un Cambio de Muchi, no el
    Precio que la Tienda publica.
    """
    return sorted(offers, key=lambda offer: (offer.currency, offer.amount))


def order_by_card_type(offers: list[SearchOffer], types: dict[int, str]) -> list[SearchOffer]:
    """Los Tipos en el Orden en que Aparecieron, y adentro de la barata a la cara."""
    seen: list[str] = []
    for offer in offers:
        if types[id(offer)] not in seen:
            seen.append(types[id(offer)])
    return sorted(offers, key=lambda offer: (
        seen.index(types[id(offer)]), offer.currency, offer.amount))


def pick_cheapest(offers: list[SearchOffer], muchi_dolar: int) -> SearchOffer | None:
    """La Oferta más barata que Muchi recomendaría: sin Agotados."""
    eligible = [offer for offer in offers
                if offer.stock_status != "unavailable"
                and convert_to_clp(offer, muchi_dolar) is not None]
    return min(eligible, key=lambda offer: convert_to_clp(offer, muchi_dolar),
               default=None)


def pick_cheapest_by_type(offers: list[SearchOffer], types: dict[int, str],
                          muchi_dolar: int) -> set[int]:
    """Una Oferta premiada por Tipo de Carta, no una sola para toda la Página."""
    grouped: dict[str, list[SearchOffer]] = {}
    for offer in offers:
        grouped.setdefault(types[id(offer)], []).append(offer)
    best = set()
    for group in grouped.values():
        if winner := pick_cheapest(group, muchi_dolar):
            best.add(id(winner))
    return best


def name_fallen_sources(card: str, faults: tuple[str, ...]) -> str:
    """Una Tienda caida se Dice por su Nombre y sin su Excusa.

    La Razon Trae el Cuerpo de la Respuesta — mil bytes de HTML cuando la
    Tienda sirve una Pagina de Mantencion. Eso es para el Log; quien Busca solo
    Necesita saber que este Precio se Comparo con una Tienda menos.
    """
    tiendas = ", ".join(faults)
    if len(faults) == 1:
        return f"{card}: no se pudo consultar {tiendas}; faltan sus Ofertas."
    return f"{card}: no se pudieron consultar {tiendas}; faltan sus Ofertas."


def build_state(state: SearchState) -> dict:
    return {
        "id": state.id, "status": state.status, "total": state.total,
        "processed": state.processed, "found": state.found,
        "errors": state.errors, "current_card": state.current_card,
        "done": state.done, "game": state.game,
    }


def build_results(items: tuple[SearchItem, ...], muchi_dolar: int,
                  verified: bool = True, match: str = MATCH_EXACT) -> dict:
    """Las Ofertas ya ordenadas, con la más barata de cada Tipo de Carta marcada."""
    offers: list[SearchOffer] = []
    positions: dict[int, int] = {}
    types: dict[int, str] = {}
    notices = []
    for item in items:
        if item.status == "source_error":
            notices.append({"level": "warning", "card": item.name,
                            "item_position": item.position,
                            "text": f"{item.name}: no se pudo completar la Consulta."})
        elif item.status == "not_found":
            notices.append({"level": "caption", "card": item.name,
                            "item_position": item.position,
                            "text": f"{item.name}: sin Ofertas."})
        if item.faults:
            notices.append({"level": "warning", "card": item.name,
                            "item_position": item.position,
                            "text": name_fallen_sources(item.name, item.faults)})
        positions.update((id(offer), item.position) for offer in item.offers)
        types.update((id(offer), read_card_type(offer, item.name, match))
                     for offer in item.offers)
        offers.extend(item.offers)

    offers = order_by_card_type(offers, types)
    best = pick_cheapest_by_type(offers, types, muchi_dolar)
    rows = []
    for offer in offers:
        row = build_offer(offer, muchi_dolar, verified)
        row["item_position"] = positions[id(offer)]
        row["card_type"] = types[id(offer)]
        row["card_label"] = offer.card_name
        # El Front conserva esta Decisión; no vuelve a comparar Precios.
        row["best"] = id(offer) in best
        rows.append(row)
    prices = [price for price in
              (convert_to_clp(offer, muchi_dolar) for offer in offers)
              if price is not None]
    return {
        "items": [{
            "id": item.id,
            "position": item.position,
            "sequence": item.sequence,
            "name": item.name,
            "quantity": item.quantity,
            "status": item.status,
            "offers": len(item.offers),
            "game": item.game,
        } for item in items],
        "offers": rows,
        "notices": notices,
        "summary": {
            "lowest_clp": None if not prices else float(round(min(prices))),
            "offers": len(offers),
            "stores": len({offer.store for offer in offers}),
        },
    }


def build_cart(items: tuple[SearchItem, ...], shipping: int, muchi_dolar: int,
               match: str = MATCH_EXACT) -> dict:
    """El Carrito en CLP: solo Ofertas sin Alerta de Precio ni Stock agotado.

    En `includes` la Búsqueda trae Derivados para Mirar, no para Comprar: pedir
    3 Kuriboh y recibir un Linkuriboh porque salía más barato no es un Carrito,
    es otra Carta. Así que el Carrito vuelve a la Carta pedida.
    """
    orders = [Order(item.quantity, item.name) for item in items]
    found: dict[str, list[Offer]] = {}
    converted = 0
    for item in items:
        for offer in item.offers:
            if offer.stock_status == "unavailable":
                continue
            if match == MATCH_INCLUDES and not names_same_card(offer.card_name, item.name):
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
