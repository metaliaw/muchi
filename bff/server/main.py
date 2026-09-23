"""API interna del Front: el Navegador habla con esto, nunca con la API de Muchi.

El Código de Seguridad de la API vive solo acá. Un Front que lo llevara en el
Navegador lo estaría publicando, así que el BFF conserva el Token, valida el
Pedido con el mismo Dominio de siempre y devuelve JSON ya presentado.
"""
from __future__ import annotations

import os
from pathlib import Path
from typing import Literal

from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException, Query
from fastapi.responses import FileResponse, JSONResponse, PlainTextResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel, Field

load_dotenv()

from muchi.api.settings import load_api_settings
from muchi.mtg import cast as muchi_cast
from muchi.mtg import decklist, phrases
from muchi.mtg.ports import CardNotFound, QueryFailed, SearchRejected, TranslationFailed
from muchi.mtg.search import StockCheck
from muchi.mtg.settings import load_offer_settings, load_rate_settings
from muchi.paths import ROOT

from server import links, presenter

# La Re-verificación pega una segunda Vez por Oferta: nace Apagada, y
# MUCHI_VERIFY_STOCK=1 la Enciende el Día que la Certeza pese más que la Carga.
VERIFY_STOCK = os.getenv("MUCHI_VERIFY_STOCK", "0").strip() in {"1", "true", "si", "yes"}
# El Carrito Reparte una Lista entre Tiendas, pero todavía no Compra. Se Muestra
# igual —Comparar ya Sirve por sí solo—, con un Aviso encima que Dice en qué
# anda. MUCHI_CART_READY=1 Retira ese Aviso el Día que la Compra esté: Apagar
# una Frase no Debería Costar un Deploy de la Interfaz.
CART_READY = os.getenv("MUCHI_CART_READY", "0").strip() in {"1", "true", "si", "yes"}
WEB_DIST = ROOT / "web" / "dist"
ADSENSE_AUTHORITY = "f08c47fec0942fa0"
ADSENSE_CLIENT = "ca-pub-6368656861543000"

app = FastAPI(title="Muchi Front", docs_url="/api/docs", openapi_url="/api/openapi.json")


class SearchRequest(BaseModel):
    text: str = Field(min_length=1, max_length=20000)
    game: str = Field("magic", min_length=1, max_length=50)
    # `includes` Ensancha la Búsqueda a los Derivados del Nombre. Vale para una
    # Carta, no para una Lista: el Front manda solo la primera Línea.
    match: Literal["exact", "includes"] = "exact"
    # `sealed` Cambia el Catálogo, no el Modo: se Buscan Cajas sin Abrir en vez
    # de Cartas sueltas. La API lo Trata como una Opción de la Búsqueda entera.
    kind: Literal["single", "sealed"] = "single"
    # La Clave de Idempotencia la elige el Front: reintentar un Envío que no
    # supo su Suerte no debe crear una segunda Búsqueda.
    key: str = Field(min_length=8, max_length=100)


class CancelRequest(BaseModel):
    key: str = Field(min_length=8, max_length=100)


def build_muchi():
    return muchi_cast.build_cast()


@app.exception_handler(SearchRejected)
def handle_rejected(request, error: SearchRejected):
    return JSONResponse({"detail": str(error), "retriable": False}, status_code=409)


@app.exception_handler(QueryFailed)
def handle_failed(request, error: QueryFailed):
    return JSONResponse({"detail": str(error), "retriable": True}, status_code=502)


@app.get("/api/health")
def read_health() -> dict:
    return {"status": "ok"}


# Un Mazo de Commander tiene cien Cartas, y de ahí salen los dos Topes: 99
# copias más el Comandante. Cien Entradas cubren el Mazo entero y sobran,
# porque las Tierras básicas se repiten en una sola Línea.
#
# No se configuran: son la Forma del Formato, no una Preferencia. El Front los
# lee de /api/config, así que el Número sigue viviendo en un solo lugar.
MAX_CARDS = 100
MAX_QUANTITY = 99

# Muchi entero es Software Libre, y son dos Repositorios: la Interfaz con su BFF
# acá, la API en el suyo. El Front los enlaza, así que las dos Direcciones salen
# del mismo lugar.
REPOSITORY_URL = "https://github.com/metaliaw/muchi"
API_REPOSITORY_URL = "https://github.com/cangrejometralleta/muchi-api"


@app.get("/api/config")
def read_config() -> dict:
    """Lo que el Front necesita saber al arrancar. Nunca incluye el Token."""
    settings = load_api_settings()
    return {
        "poll_seconds": settings.poll_seconds,
        "muchi_dolar": load_rate_settings().muchi_dolar,
        "environment": os.getenv("MUCHI_ENV", ""),
        "limits": {"max_cards": MAX_CARDS, "max_quantity": MAX_QUANTITY},
        # Cuánto Vale un Stock ya confirmado. El Front lo Guarda con su Hora y
        # lo Descarta solo; el Tope lo Dice el Servidor, como todos los demás.
        "stock_fresh_seconds": load_offer_settings().stock_fresh_seconds,
        # Cuántas Tiendas Consulta el Navegador por Carta. El Tope lo Dice el
        # Servidor aunque las Visitas no Salgan de acá.
        "browser_check_limit": load_offer_settings().browser_check_limit,
        "donation_url": os.getenv("MUCHI_DONATION_URL", ""),
        "sponsor_name": os.getenv("MUCHI_SPONSOR_NAME", ""),
        "sponsor_text": os.getenv("MUCHI_SPONSOR_TEXT", ""),
        "sponsor_url": os.getenv("MUCHI_SPONSOR_URL", ""),
        "adsense_client": os.getenv("MUCHI_ADSENSE_CLIENT", "") or ADSENSE_CLIENT,
        "adsense_slot": os.getenv("MUCHI_ADSENSE_SLOT", ""),
        "cart_ready": CART_READY,
        "repository_url": REPOSITORY_URL,
        "api_repository_url": API_REPOSITORY_URL,
        "socials": links.read_socials(),
    }


@app.get("/ads.txt", response_class=PlainTextResponse)
def read_ads_txt() -> str:
    """Declara a Google como Vendedor autorizado cuando AdSense está activo."""
    client = os.getenv("MUCHI_ADSENSE_CLIENT", "") or ADSENSE_CLIENT
    if not client.startswith("ca-pub-"):
        raise HTTPException(404, "AdSense no está configurado.")
    return f"google.com, {client.removeprefix('ca-')}, DIRECT, {ADSENSE_AUTHORITY}\n"


@app.get("/api/muchi")
def read_muchi() -> dict:
    """Las Frases del Gato: el Front las dice, el Servidor las conserva."""
    book = phrases.read_phrases()
    def rows(items):
        return [{"text": phrase.text, "state": phrase.state} for phrase in items]
    return {
        "every": book.every,
        "phrases": rows(book.phrases),
        "greetings": rows(book.greetings),
        "dark": rows(book.dark),
        "light": rows(book.light),
        "nerd": rows(book.nerd),
        "libre": rows(book.libre),
        "bargain": rows(book.bargain),
        "help": [{"title": title, "detail": detail} for title, detail in book.help_topics],
    }


@app.post("/api/decklist")
def read_decklist(request: SearchRequest) -> dict:
    """Muestra cómo quedó leída la Lista antes de gastar una Búsqueda."""
    orders, ignored = decklist.parse_decklist(request.text,
                                              sealed=request.kind == "sealed")
    return {
        "orders": [{"name": order.name, "quantity": order.quantity} for order in orders],
        "ignored": list(ignored),
    }


@app.get("/api/languages")
def read_languages() -> dict:
    """Los Idiomas que el Selector ofrece los nombra la Fuente, no el Front."""
    return {"languages": [{"code": code, "label": label, "example": example}
                          for code, label, example
                          in build_muchi().translator.languages]}


@app.get("/api/supported-games")
def read_supported_games(kind: Literal["", "single", "sealed"] = "") -> dict:
    """Los Juegos disponibles los nombra muchi-api, no el Front.

    Cada uno Dice si se Puede pedir en Cartas sueltas, en Sellado o en ambas.
    Nadie lo Declara por Juego: lo Declara cada Tienda y el Juego lo Hereda.
    """
    return {"games": build_muchi().searches.read_supported_games(kind)}


@app.get("/api/card/art")
def read_card_art(name: str = Query(min_length=1, max_length=200),
                  language: str = Query("", max_length=5),
                  edition: str = Query("", max_length=10),
                  foil: bool = Query(False)) -> dict:
    """La Imagen de una Carta, servida por Scryfall directo al Navegador."""
    translator = build_muchi().translator
    if language and language not in translator.codes:
        raise HTTPException(400, f"El idioma «{language}» no está en la lista.")
    art = translator.find_art(name, language, edition, foil)
    if not art:
        raise HTTPException(404, f"No hay imagen de «{name}».")
    return art


@app.get("/api/card/metadata")
def read_card_metadata(game: str = Query(min_length=1, max_length=50),
                       name: str = Query(min_length=1, max_length=200),
                       language: str = Query("", max_length=20),
                       edition: str = Query("", max_length=50),
                       foil: bool = Query(False)) -> dict:
    """La Metadata visual viene del Catálogo que corresponde al Juego."""
    return build_muchi().searches.read_card_metadata(game, name, language, edition, foil)


@app.get("/api/card/autocomplete")
def autocomplete_cards(game: str = Query(min_length=1, max_length=50),
                       name: str = Query(min_length=1, max_length=200),
                       language: str = Query("", max_length=20)) -> dict:
    """Los Nombres sugeridos vienen del Catálogo del Juego seleccionado."""
    return {"suggestions": build_muchi().searches.autocomplete_cards(game, name, language)}


@app.get("/api/card/suggestions")
def read_suggestions(name: str = Query(min_length=1, max_length=200),
                     language: str = Query(min_length=2, max_length=5)) -> dict:
    """Nombres que empiezan como lo escrito, para quien dudó en el Campo."""
    translator = build_muchi().translator
    if language not in translator.codes:
        raise HTTPException(400, f"El idioma «{language}» no está en la lista.")
    return {"suggestions": list(translator.suggest_names(name, language))}


@app.get("/api/card")
def read_card(name: str = Query(min_length=1, max_length=200),
              language: str = Query("", max_length=5)) -> dict:
    """Busca una Carta: el Front escribe en su Idioma, Scryfall traduce."""
    translator = build_muchi().translator
    if language and language not in translator.codes:
        raise HTTPException(400, f"El idioma «{language}» no está en la lista.")
    try:
        canonical = translator.translate_name(name, language)
    except CardNotFound:
        book = phrases.read_phrases()
        # El Grupo «not_found» del Catálogo nombra el Fracaso con voz propia.
        phrase = next((row for row in book.phrases if "no la encontré" in row.text), None)
        detail = phrase.text if phrase else f"No encontramos «{name}»."
        raise HTTPException(404, detail)
    except TranslationFailed as error:
        raise HTTPException(502, str(error))
    return {"name": name, "canonical_name": canonical, "language": language}


@app.post("/api/searches")
def create_search(request: SearchRequest) -> dict:
    orders, ignored = decklist.parse_decklist(request.text,
                                              sealed=request.kind == "sealed")
    if ignored:
        raise HTTPException(422, {"detail": "Revisa estas líneas: " + ", ".join(ignored),
                                  "ignored": list(ignored)})
    if not 1 <= len(orders) <= MAX_CARDS or any(
            not 1 <= order.quantity <= MAX_QUANTITY for order in orders):
        raise HTTPException(422, {"detail": f"Ingresa entre 1 y {MAX_CARDS} cartas, "
                                            f"con cantidades de 1 a {MAX_QUANTITY}."})
    state = build_muchi().searches.create_search(
        orders=orders, verify_stock=VERIFY_STOCK,
        key=request.key, game=request.game, match=request.match,
        kind=request.kind,
    )
    return {
        "state": presenter.build_state(state),
        "label": ", ".join(order.name for order in orders),
        "items": [{"name": order.name, "quantity": order.quantity,
                   "position": position, "sequence": 0,
                   "status": "queued", "offers": 0}
                  for position, order in enumerate(orders)],
    }


@app.get("/api/searches/{search_id}")
def read_search(search_id: str, after: int = Query(0, ge=0),
                match: Literal["exact", "includes"] = "exact") -> dict:
    """Estado y Ofertas en una sola Consulta: el Front pregunta una vez por Ciclo.

    `match` no Cambia lo que la API Devuelve; Cambia cómo se Agrupa. La API no
    Recuerda el Modo en el Estado, así que el Front lo Repite en cada Ciclo.
    """
    searches = build_muchi().searches
    state = searches.read_search(search_id)
    page = searches.read_results(search_id, after)
    results = presenter.build_results(page.items, load_rate_settings().muchi_dolar,
                                      verified=VERIFY_STOCK, match=match)
    return {"state": presenter.build_state(state), **results,
            "cursor": page.cursor, "has_more": page.has_more}


class BrowserCheck(BaseModel):
    """Lo que el Navegador de quien Compra vio en la Tienda, Oferta por Oferta."""
    offer_id: str = Field(min_length=1, max_length=200)
    available: bool


class StockRequest(BaseModel):
    # Una Lista larga no Confirma más: lo que no está en el Plan se Descarta
    # igual, y el Tope Evita un Cuerpo que crezca sin Razón.
    checks: list[BrowserCheck] = Field(default_factory=list, max_length=500)
    # A quien Preguntamos nosotros, cuando la Pregunta es por unas pocas
    # Ofertas y no por la Lista entera. Vacio Significa el Plan de siempre.
    asking: list[str] = Field(default_factory=list, max_length=50)


def answer_stock(search_id: str, match: str, known: dict[str, StockCheck],
                 ask: bool = True, only: tuple[str, ...] = ()) -> dict:
    """Comprueba la más barata de cada Carta y Corona la primera que sí Tiene.

    Pregunta por Rondas: la primera Candidata de cada Tipo de Carta viaja en
    una sola Consulta, y solo los Tipos cuya Candidata no Tenía pasan a la
    siguiente. Así el Costo crece con la Mala Suerte, no con el Largo de la
    Lista, y `stock_check_limit` le pone Techo.

    Lo que el Navegador ya Averiguó entra como Sabido: esas Ofertas no se le
    Preguntan a nadie. Una Tienda que Contestó al Comprador no necesita
    Contestarnos también a nosotros.

    `only` Recorta el Plan a unas Ofertas nombradas. Es el Toque suelto sobre
    una Tienda que el Navegador no Alcanza —un Catálogo que no Sirve JSON, una
    Tienda leída de Listas—: se Pregunta por esa y por nadie más, para que un
    Toque no Desate la Ronda entera.
    """
    searches = build_muchi().searches
    items = read_all_results(searches, search_id)
    limit = load_offer_settings().stock_check_limit
    # Dos Listas, no una. `ranking` es quién Compite por la Corona: todas las
    # Ofertas en pie, de la barata a la cara. `plan` es a quién le Preguntamos
    # nosotros, y ahí sí Manda el Tope, porque cada una es una Visita nuestra.
    ranking = presenter.rank_stock_candidates(items, load_rate_settings().muchi_dolar,
                                              match=match)
    plan = {card_type: candidates[:limit] for card_type, candidates in ranking.items()}
    if only:
        plan = {card_type: [offer_id for offer_id in candidates if offer_id in only]
                for card_type, candidates in ranking.items()}
    competing = {offer_id for candidates in ranking.values() for offer_id in candidates}
    # Una Oferta que no Compite no Corona ni Descorona nada: el Navegador
    # Informa sobre esta Búsqueda, no sobre el Catálogo entero. Pero sí Vale
    # aunque esté en el Puesto nueve: Confirmarla no nos Costó una Visita.
    checks: dict[str, StockCheck] = {offer_id: check for offer_id, check in known.items()
                                     if offer_id in competing}

    def waiting() -> dict[str, list[str]]:
        """Los Tipos sin un Sí barato todavía, y a quién les Queda por preguntar.

        Un Sí puede Venir de cualquier Puesto —el Navegador Llega a Tiendas que
        el Plan no Alcanza—, pero no Cierra la Búsqueda por sí solo: si Quedan
        Dudas más baratas sin preguntar, Vale la Pena preguntarlas, porque una
        de ellas Confirmada Recomienda mejor. La Pregunta sale solo hacia el
        Plan: lo de más allá del Tope no nos Cuesta una Visita ni la Pide.
        """
        pending: dict[str, list[str]] = {}
        for card_type, candidates in ranking.items():
            said_yes = next((turn for turn, offer_id in enumerate(candidates)
                             if offer_id in checks and checks[offer_id].confirmed), None)
            cheaper = candidates if said_yes is None else candidates[:said_yes]
            asking = [offer_id for offer_id in cheaper
                      if offer_id in plan[card_type] and offer_id not in checks]
            if asking:
                pending[card_type] = asking
        return pending

    pending = waiting() if ask else {}
    for _ in range(limit):
        asking = {card_type: candidates[0] for card_type, candidates in pending.items()
                  if candidates}
        if not asking:
            break
        for check in searches.check_stock(search_id, tuple(asking.values())):
            checks[check.offer_id] = check
        # Una Duda no Cierra la Ronda: se sigue preguntando por si alguna
        # Tienda Confirma, y la Duda barata espera su turno como Reserva.
        pending = waiting()
    offers = {offer.offer_id: offer for item in items for offer in item.offers
              if offer.offer_id}
    return presenter.build_stock_answer(offers, ranking, checks,
                                        load_rate_settings().muchi_dolar)


@app.get("/api/searches/{search_id}/stock")
def check_stock(search_id: str,
                match: Literal["exact", "includes"] = "exact") -> dict:
    """Comprueba el Stock preguntando solo el Servicio."""
    return answer_stock(search_id, match, {})


@app.post("/api/searches/{search_id}/stock")
def check_stock_with_browser(search_id: str, request: StockRequest,
                             match: Literal["exact", "includes"] = "exact",
                             ask: bool = Query(True)) -> dict:
    """Comprueba el Stock con lo que el Navegador ya Confirmó por su cuenta.

    Una Tienda Shopify Sirve su Catálogo con CORS abierto: el Navegador lo Lee
    directo y Manda acá el Resultado. La Corona se Decide igual de este Lado —
    el Front Averigua, no Recomienda.
    """
    known = {row.offer_id: StockCheck(
        offer_id=row.offer_id,
        stock_status="available" if row.available else "unavailable",
    ) for row in request.checks}
    return answer_stock(search_id, match, known, ask=ask or bool(request.asking),
                        only=tuple(request.asking))


@app.post("/api/searches/{search_id}/cancel")
def cancel_search(search_id: str, request: CancelRequest) -> dict:
    state = build_muchi().searches.cancel_search(search_id, request.key)
    return {"state": presenter.build_state(state)}


class OfferPick(BaseModel):
    """Cuantas Copias se Compran en esta Oferta."""
    offer_id: str = Field(min_length=1, max_length=200)
    units: int = Field(ge=0, le=999)


class CartRequest(BaseModel):
    picks: list[OfferPick] = Field(default_factory=list, max_length=2000)


@app.get("/api/searches/{search_id}/cart")
def read_cart(search_id: str, shipping: int = Query(4000, ge=0, le=1_000_000),
              match: Literal["exact", "includes"] = "exact") -> dict:
    """El Carrito con lo que cada Tienda Declara, que casi siempre es nada."""
    searches = build_muchi().searches
    items = read_all_results(searches, search_id)
    return presenter.build_cart(items, shipping, load_rate_settings().muchi_dolar,
                                match=match)


@app.post("/api/searches/{search_id}/cart")
def read_chosen_cart(search_id: str, request: CartRequest,
                     shipping: int = Query(4000, ge=0, le=1_000_000),
                     match: Literal["exact", "includes"] = "exact") -> dict:
    """El Carrito que Armó quien Compra, Oferta por Oferta.

    El `GET` Devuelve la Recomendación —el Reparto que Muchi Haría— y con ella
    se Llenan los Selectores. Desde ahí Manda la Persona: esto Suma lo Elegido
    y Cuenta los Envíos, sin Volver a Optimizar por encima de su Decisión.
    """
    searches = build_muchi().searches
    items = read_all_results(searches, search_id)
    return presenter.build_cart(items, shipping, load_rate_settings().muchi_dolar,
                                match=match,
                                picks={row.offer_id: row.units for row in request.picks})


def read_all_results(searches, search_id: str):
    items = []
    cursor = 0
    while True:
        page = searches.read_results(search_id, cursor)
        items.extend(page.items)
        if not page.has_more:
            return tuple(items)
        if page.cursor <= cursor:
            raise QueryFailed("La API no avanzó el cursor de resultados.")
        cursor = page.cursor


@app.get("/api/sources")
def read_sources() -> dict:
    return {"sources": build_muchi().searches.read_sources()}


if WEB_DIST.is_dir():
    # El Front compilado se sirve desde el mismo Servicio: un solo Despliegue,
    # un solo Origen y ningún CORS que configurar.
    app.mount("/assets", StaticFiles(directory=WEB_DIST / "assets"), name="assets")

    @app.get("/{path:path}")
    def read_spa(path: str) -> FileResponse:
        candidate = (WEB_DIST / path).resolve()
        if path and candidate.is_file() and candidate.is_relative_to(WEB_DIST.resolve()):
            return FileResponse(candidate)
        return FileResponse(WEB_DIST / "index.html")
