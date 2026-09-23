"""Traduce el Dominio a JSON para el Front.

Las Reglas que antes vivían en `app.py` viven acá: el Front dibuja, no decide.
Así el Tratamiento, el Muchi Dólar y el Carrito se calculan una sola vez y del
mismo modo, sea cual sea la Interfaz que los consuma.
"""
from __future__ import annotations

import unicodedata
from dataclasses import replace
from decimal import Decimal

from muchi.mtg import optimizer, treatment
from muchi.mtg.models import Offer, Order
from muchi.mtg.search import SearchItem, SearchOffer, SearchState, StockCheck

# La API marca "unknown" cuando la Fuente no publica Stock: Agregadores como
# scry.cl indexan Precios, no Inventario. No es Ausencia de Carta.
STOCK_LABELS = {"available": "En stock", "unavailable": "Agotado",
                "unknown": "No confirmado"}
# Los Acabados van en Dorado; el resto del Tratamiento, en Gris.
FOIL_TAGS = ("Foil", "Etched")
# Una Tienda que llama "MTG Single" a su Edicion no Nombro una Edicion: Nombro
# su Catalogo. Esa Palabra no Distingue una Impresion de otra, asi que no Vale
# una Pastilla.
EDITION_NOISE = ("single", "singles")
# Una Edicion de hasta cuatro Letras es un Codigo: `c21`, `otc`, `mh3`.
EDITION_CODE_LENGTH = 4
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


def name_edition(offer: SearchOffer) -> str:
    """La Edicion que Distingue esta Impresion, o nada cuando solo Repite el Juego."""
    edition = offer.edition.strip()
    words = edition.lower().replace(":", " ").split()
    if not words or words[-1] in EDITION_NOISE:
        return ""
    # `cmm` es un Codigo de Edicion, no una Palabra: se Lee en Mayusculas.
    if len(words) == 1 and len(edition) <= EDITION_CODE_LENGTH:
        return edition.upper()
    return edition


def build_pills(offer: SearchOffer, verified: bool = True) -> list[dict]:
    """Las Pastillas de una Oferta: quién la vende, de qué Edición, cómo viene y su Stock."""
    pills = [{"kind": "tienda", "text": offer.store}]
    if edition := name_edition(offer):
        pills.append({"kind": "edicion", "text": edition})
    for tag in treatment.build_treatment(offer).split(treatment.SEPARATOR):
        if tag:
            pills.append({"kind": "foil" if tag in FOIL_TAGS else "cond", "text": tag})
    # Sin Re-verificación, el Stock es Ruido: toda Oferta diría lo mismo. Una
    # Oferta comprobada una por una sí lo declara, aunque la Flag esté apagada.
    if label := name_stock(offer.stock_status, offer.stock_quantity, verified):
        available = offer.stock_status == "available" and offer.stock_quantity != 0
        pills.append({"kind": "tienda" if available else "cond", "text": label})
    return pills


def name_stock(status: str, quantity: int | None = None,
               verified: bool = True) -> str:
    """Lo que se puede Decir del Stock: la Cantidad cuando se Sabe, si no la Etiqueta.

    Sin Cantidad y sin Re-verificación no se Dice nada: una Etiqueta que toda
    Oferta repetiría no Informa. Cero Unidades es Agotado, lo diga o no la
    Tienda en su Estado.
    """
    if not verified and quantity is None:
        return ""
    if quantity == 0 or status == "unavailable":
        return STOCK_LABELS["unavailable"]
    if quantity is not None:
        return f"{quantity} {'unidad' if quantity == 1 else 'unidades'}"
    return STOCK_LABELS.get(status, status)


def build_offer(offer: SearchOffer, muchi_dolar: int,
                verified: bool = True) -> dict:
    price = convert_to_clp(offer, muchi_dolar)
    return {
        "card_name": offer.card_name,
        "store": offer.store,
        # El Front Distingue los Catálogos que no tienen una Página propia.
        "source": offer.source,
        "amount": str(offer.amount),
        "currency": offer.currency,
        "price_clp": None if price is None else float(price),
        "url": offer.url,
        "stock_status": offer.stock_status,
        "stock_label": name_stock(offer.stock_status, offer.stock_quantity, verified),
        "stock_quantity": offer.stock_quantity,
        # El Nombre con el que se vuelve a Preguntar por esta Oferta.
        "offer_id": offer.offer_id,
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
        # La Foto de la Tienda. Es la unica que una Caja sellada va a tener.
        "image": offer.image,
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


def fold_name(value: str) -> str:
    """El Nombre sin Acentos ni Espacios de más.

    Mitos y Leyendas Nombra sus Cartas en Español y su Índice las Archiva sin
    Tildes: «Dragón de Magma» y «dragon de magma» son la misma Carta escrita
    por dos Manos. Nadie Imprime dos Cartas que se Diferencien en una Tilde.
    """
    plain = unicodedata.normalize("NFKD", value)
    return " ".join("".join(
        letter for letter in plain if not unicodedata.combining(letter)).lower().split())


def names_same_card(title: str, asked: str) -> bool:
    """El Título Nombra la Carta pedida, con o sin su Impresión detrás."""
    title = fold_name(title)
    asked = fold_name(asked)
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
        return fold_name(offer.card_name)
    return fold_name(asked)


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


def rank_stock_candidates(items: tuple[SearchItem, ...], muchi_dolar: int,
                          match: str = MATCH_EXACT) -> dict[str, list[str]]:
    """Por cada Tipo de Carta, sus Ofertas en pie, de la barata a la cara.

    El Orden es el mismo con el que se premia la más barata, porque la Corona
    Recorre esta Lista: si la barata no tiene, pasa a la siguiente. Una Oferta
    sin Cambio a Pesos no compite por la Corona, y una ya Agotada no necesita
    que se le pregunte de nuevo.
    """
    types: dict[int, str] = {}
    offers: list[SearchOffer] = []
    for item in items:
        types.update((id(offer), read_card_type(offer, item.name, match))
                     for offer in item.offers)
        offers.extend(item.offers)
    ranking: dict[str, list[str]] = {}
    for offer in order_by_card_type(offers, types):
        if not offer.offer_id or offer.stock_status == "unavailable":
            continue
        if convert_to_clp(offer, muchi_dolar) is None:
            continue
        ranking.setdefault(types[id(offer)], []).append(offer.offer_id)
    return ranking


def plan_stock_checks(items: tuple[SearchItem, ...], muchi_dolar: int,
                      match: str = MATCH_EXACT,
                      limit: int = 3) -> dict[str, list[str]]:
    """A quién le Preguntamos nosotros, y en qué Orden.

    El Tope es de la Pregunta, no de la Corona: cada Consulta que Sale de acá
    es una Visita nuestra a la Tienda, y `stock_check_limit` la Acota. Lo que
    el Navegador Averigua por su cuenta no Pasa por este Tope —no nos Cuesta
    una Visita— y Compite por la Corona igual, esté en el Puesto que esté.
    """
    return {card_type: candidates[:limit]
            for card_type, candidates
            in rank_stock_candidates(items, muchi_dolar, match=match).items()}


def crown_checked_offers(plan: dict[str, list[str]],
                         checks: dict[str, StockCheck]) -> dict[str, str]:
    """La Corona de cada Tipo de Carta: la primera que la Tienda Confirmo.

    Una Tienda que no Declara Stock no Niega, pero tampoco Confirma. Entre una
    Duda barata y un Si caro, la Corona es del Si: la Recomendacion existe para
    que alguien Compre, y una Carta que no Llega no es una Compra barata. La
    Duda solo Corona cuando nadie Confirmo, y una Carta sin ninguna Oferta en
    pie se queda sin Corona: mentir una Recomendacion es peor que no darla.
    """
    crowned = {}
    for card_type, candidates in plan.items():
        checked = [checks[offer_id] for offer_id in candidates if offer_id in checks]
        winner = next((check for check in checked if check.confirmed), None)
        winner = winner or next((check for check in checked if check.available), None)
        if winner:
            crowned[card_type] = winner.offer_id
    return crowned


def refresh_offer(offer: SearchOffer, check: StockCheck) -> SearchOffer:
    """La misma Oferta con lo que la Tienda acaba de Decir de su Stock."""
    return replace(offer, stock_status=check.stock_status,
                   stock_quantity=check.stock_quantity)


def build_stock_answer(offers: dict[str, SearchOffer], ranking: dict[str, list[str]],
                       checks: dict[str, StockCheck], muchi_dolar: int) -> dict:
    """Lo Comprobado, ya presentado, y a quién le toca la Corona ahora.

    Las Filas vuelven enteras —Pastillas incluidas— porque el Stock cambia la
    Etiqueta y el Color de la Pastilla, y esas son Decisiones de acá. El Front
    reemplaza la Fila que comparte `offer_id` y no vuelve a decidir nada.
    """
    crowned = crown_checked_offers(ranking, checks)
    return {
        "offers": [build_offer(refresh_offer(offers[offer_id], check), muchi_dolar)
                   for offer_id, check in checks.items() if offer_id in offers],
        "best": [{"card_type": card_type, "offer_id": offer_id}
                 for card_type, offer_id in crowned.items()],
        # Un Tipo sin Corona ya no Recomienda: el Front apaga la Marca vieja.
        "uncrowned": [card_type for card_type in ranking if card_type not in crowned],
    }


def name_fallen_sources(card: str, faults: tuple[str, ...]) -> str:
    """Una Tienda caida se Dice por su Nombre y sin su Excusa.

    La Razon Trae el Cuerpo de la Respuesta — mil bytes de HTML cuando la
    Tienda sirve una Pagina de Mantencion. Eso es para el Log; quien Busca solo
    Necesita saber que este Precio se Comparo con una Tienda menos.
    """
    tiendas = ", ".join(faults)
    if len(faults) == 1:
        return f"{card}: no se pudo consultar {tiendas}; faltan sus ofertas."
    return f"{card}: no se pudieron consultar {tiendas}; faltan sus ofertas."


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
                            "text": f"{item.name}: no se pudo completar la consulta."})
        elif item.status == "not_found":
            notices.append({"level": "caption", "card": item.name,
                            "item_position": item.position,
                            "text": f"{item.name}: sin ofertas."})
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
               match: str = MATCH_EXACT, picks: dict[str, int] | None = None) -> dict:
    """El Carrito en CLP: solo Ofertas sin Alerta de Precio ni Stock agotado.

    En `includes` la Búsqueda trae Derivados para Mirar, no para Comprar: pedir
    3 Kuriboh y recibir un Linkuriboh porque salía más barato no es un Carrito,
    es otra Carta. Así que el Carrito vuelve a la Carta pedida.
    """
    picks = picks or {}
    orders = [Order(item.quantity, item.name) for item in items]
    found: dict[str, list[Offer]] = {}
    catalog: dict[str, Offer] = {}
    # La Impresion de cada Oferta, para que el Carrito Diga cual se Eligio:
    # dos Lineas con el mismo Nombre y distinta Edicion no son la misma Compra.
    printings: dict[str, dict[str, str]] = {}
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
            row = Offer(
                store=offer.store, card_name=item.name, title=offer.card_name,
                price_clp=price, url=offer.url, key=offer.offer_id,
                # Lo que la Tienda Declara y nadie mas: la Cantidad que Escribe
                # quien Compra ya no es un Tope, es su Pedido.
                stock=offer.stock_quantity,
            )
            found.setdefault(item.name.lower(), []).append(row)
            if offer.offer_id:
                catalog[offer.offer_id] = row
                printings[offer.offer_id] = {
                    "edition": offer.edition, "finish": offer.finish,
                    "condition": offer.condition, "language": offer.language,
                }
    # Sin Elecciones, el Carrito es la Recomendacion: el Reparto que Muchi
    # Haria. Con Elecciones, es lo que la Persona Armo.
    if picks:
        # Una Carta que nadie Vende no se Elige mal: Falta, y eso ya se Dice
        # en `missing`. Repetirlo en `short` seria Contarlo dos veces.
        sellable = [order for order in orders if found.get(order.name.lower())]
        plan = optimizer.build_chosen_plan(sellable, catalog, picks, shipping)
        plan.missing = [order.name for order in orders if not found.get(order.name.lower())]
    else:
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
                "url": line.url, "title": line.title, "offer_id": line.offer_id,
                **printings.get(line.offer_id, {}),
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
        # Lo que se Consiguio a medias, Carta por Carta. Un Carrito que Calla
        # esto Entrega menos Copias de las que se Pidieron sin Avisar.
        "short": [{"card_name": name, "units": count}
                  for name, count in plan.short.items()],
    }
