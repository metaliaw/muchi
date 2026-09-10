"""API interna del Front: el Navegador habla con esto, nunca con la API de Muchi.

El Código de Seguridad de la API vive solo acá. Un Front que lo llevara en el
Navegador lo estaría publicando, así que el BFF conserva el Token, valida el
Pedido con el mismo Dominio de siempre y devuelve JSON ya presentado.
"""
from __future__ import annotations

import os
from pathlib import Path

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
from muchi.mtg.settings import load_rate_settings
from muchi.paths import ROOT

from server import presenter

# El Contrato pide ambas Opciones en cada Pedido. Se envían fijas: la API
# comprueba el Stock de las Ofertas más baratas y hoy ignora "stores_only".
VERIFY_STOCK = True
STORES_ONLY = True
WEB_DIST = ROOT / "web" / "dist"
ADSENSE_AUTHORITY = "f08c47fec0942fa0"
ADSENSE_CLIENT = "ca-pub-6368656861543000"

app = FastAPI(title="Muchi Front", docs_url="/api/docs", openapi_url="/api/openapi.json")


class SearchRequest(BaseModel):
    text: str = Field(min_length=1, max_length=20000)
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


# Cada Red es un Nombre, un Icono y la Variable que la enciende. Una Red sin
# Direccion no existe: el Pie solo muestra las que alguien configuro.
# Un Mazo de Commander tiene cien Cartas, y de ahí salen los dos Topes: 99
# copias más el Comandante. Cien Entradas cubren el Mazo entero y sobran,
# porque las Tierras básicas se repiten en una sola Línea.
#
# No se configuran: son la Forma del Formato, no una Preferencia. El Front los
# lee de /api/config, así que el Número sigue viviendo en un solo lugar.
MAX_CARDS = 100
MAX_QUANTITY = 99

REPOSITORY_URL = "https://github.com/metaliaw/muchi"

SOCIALS = (
    ("Instagram", "📸", "MUCHI_INSTAGRAM_URL"),
    ("Discord", "💬", "MUCHI_DISCORD_URL"),
    ("X", "𝕏", "MUCHI_X_URL"),
    ("YouTube", "▶️", "MUCHI_YOUTUBE_URL"),
    ("TikTok", "🎵", "MUCHI_TIKTOK_URL"),
)


def read_socials() -> list[dict]:
    return [{"name": name, "icon": icon, "url": url}
            for name, icon, variable in SOCIALS
            if (url := os.getenv(variable, "").strip())]


@app.get("/api/config")
def read_config() -> dict:
    """Lo que el Front necesita saber al arrancar. Nunca incluye el Token."""
    settings = load_api_settings()
    return {
        "poll_seconds": settings.poll_seconds,
        "muchi_dolar": load_rate_settings().muchi_dolar,
        "environment": os.getenv("MUCHI_ENV", ""),
        "limits": {"max_cards": MAX_CARDS, "max_quantity": MAX_QUANTITY},
        "donation_url": os.getenv("MUCHI_DONATION_URL", ""),
        "sponsor_name": os.getenv("MUCHI_SPONSOR_NAME", ""),
        "sponsor_text": os.getenv("MUCHI_SPONSOR_TEXT", ""),
        "sponsor_url": os.getenv("MUCHI_SPONSOR_URL", ""),
        "adsense_client": os.getenv("MUCHI_ADSENSE_CLIENT", "") or ADSENSE_CLIENT,
        "adsense_slot": os.getenv("MUCHI_ADSENSE_SLOT", ""),
        "repository_url": REPOSITORY_URL,
        "socials": read_socials(),
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
        "help": [{"title": title, "detail": detail} for title, detail in book.help_topics],
    }


@app.post("/api/decklist")
def read_decklist(request: SearchRequest) -> dict:
    """Muestra cómo quedó leída la Lista antes de gastar una Búsqueda."""
    orders, ignored = decklist.parse_decklist(request.text)
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


@app.get("/api/card/art")
def read_card_art(name: str = Query(min_length=1, max_length=200),
                  language: str = Query("", max_length=5),
                  edition: str = Query("", max_length=10),
                  foil: bool = Query(False)) -> dict:
    """La Imagen de una Carta, servida por Scryfall directo al Navegador."""
    translator = build_muchi().translator
    if language and language not in translator.codes:
        raise HTTPException(400, f"El Idioma «{language}» no está en la Lista.")
    art = translator.find_art(name, language, edition, foil)
    if not art:
        raise HTTPException(404, f"No hay Imagen de «{name}».")
    return art


@app.get("/api/card/suggestions")
def read_suggestions(name: str = Query(min_length=1, max_length=200),
                     language: str = Query(min_length=2, max_length=5)) -> dict:
    """Nombres que empiezan como lo escrito, para quien dudó en el Campo."""
    translator = build_muchi().translator
    if language not in translator.codes:
        raise HTTPException(400, f"El Idioma «{language}» no está en la Lista.")
    return {"suggestions": list(translator.suggest_names(name, language))}


@app.get("/api/card")
def read_card(name: str = Query(min_length=1, max_length=200),
              language: str = Query("", max_length=5)) -> dict:
    """Busca una Carta: el Front escribe en su Idioma, Scryfall traduce."""
    translator = build_muchi().translator
    if language and language not in translator.codes:
        raise HTTPException(400, f"El Idioma «{language}» no está en la Lista.")
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
    orders, ignored = decklist.parse_decklist(request.text)
    if ignored:
        raise HTTPException(422, {"detail": "Revisa estas Líneas: " + ", ".join(ignored),
                                  "ignored": list(ignored)})
    if not 1 <= len(orders) <= MAX_CARDS or any(
            not 1 <= order.quantity <= MAX_QUANTITY for order in orders):
        raise HTTPException(422, {"detail": f"Ingresa entre 1 y {MAX_CARDS} Cartas, "
                                            f"con Cantidades de 1 a {MAX_QUANTITY}."})
    state = build_muchi().searches.create_search(
        orders=orders, verify_stock=VERIFY_STOCK, stores_only=STORES_ONLY,
        key=request.key,
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
def read_search(search_id: str, after: int = Query(0, ge=0)) -> dict:
    """Estado y Ofertas en una sola Consulta: el Front pregunta una vez por Ciclo."""
    searches = build_muchi().searches
    state = searches.read_search(search_id)
    page = searches.read_results(search_id, after)
    results = presenter.build_results(page.items, load_rate_settings().muchi_dolar)
    return {"state": presenter.build_state(state), **results,
            "cursor": page.cursor, "has_more": page.has_more}


@app.post("/api/searches/{search_id}/cancel")
def cancel_search(search_id: str, request: CancelRequest) -> dict:
    state = build_muchi().searches.cancel_search(search_id, request.key)
    return {"state": presenter.build_state(state)}


@app.get("/api/searches/{search_id}/cart")
def read_cart(search_id: str, shipping: int = Query(4000, ge=0, le=1_000_000)) -> dict:
    searches = build_muchi().searches
    items = read_all_results(searches, search_id)
    return presenter.build_cart(items, shipping, load_rate_settings().muchi_dolar)


def read_all_results(searches, search_id: str):
    items = []
    cursor = 0
    while True:
        page = searches.read_results(search_id, cursor)
        items.extend(page.items)
        if not page.has_more:
            return tuple(items)
        if page.cursor <= cursor:
            raise QueryFailed("La API no avanzó el Cursor de Resultados.")
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
