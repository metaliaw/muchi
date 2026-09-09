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
from fastapi.responses import FileResponse, JSONResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel, Field

load_dotenv()

from muchi.api.settings import load_api_settings
from muchi.mtg import cast as muchi_cast
from muchi.mtg import decklist, phrases
from muchi.mtg.ports import QueryFailed, SearchRejected
from muchi.mtg.settings import load_rate_settings
from muchi.paths import ROOT

from server import presenter

# El Contrato pide ambas Opciones en cada Pedido. Se envían fijas: la API
# comprueba el Stock de las Ofertas más baratas y hoy ignora "stores_only".
VERIFY_STOCK = True
STORES_ONLY = True
WEB_DIST = ROOT / "web" / "dist"

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


@app.get("/api/config")
def read_config() -> dict:
    """Lo que el Front necesita saber al arrancar. Nunca incluye el Token."""
    settings = load_api_settings()
    return {
        "poll_seconds": settings.poll_seconds,
        "muchi_dolar": load_rate_settings().muchi_dolar,
        "environment": os.getenv("MUCHI_ENV", ""),
        "limits": {"max_cards": 500, "max_quantity": 99},
    }


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


@app.post("/api/searches")
def create_search(request: SearchRequest) -> dict:
    orders, ignored = decklist.parse_decklist(request.text)
    if ignored:
        raise HTTPException(422, {"detail": "Revisa estas Líneas: " + ", ".join(ignored),
                                  "ignored": list(ignored)})
    if not 1 <= len(orders) <= 500 or any(not 1 <= order.quantity <= 99 for order in orders):
        raise HTTPException(422, {"detail": "Ingresa entre 1 y 500 Cartas, con Cantidades de 1 a 99."})
    state = build_muchi().searches.create_search(
        orders=orders, verify_stock=VERIFY_STOCK, stores_only=STORES_ONLY,
        key=request.key,
    )
    return {
        "state": presenter.build_state(state),
        "label": ", ".join(order.name for order in orders),
    }


@app.get("/api/searches/{search_id}")
def read_search(search_id: str) -> dict:
    """Estado y Ofertas en una sola Consulta: el Front pregunta una vez por Ciclo."""
    searches = build_muchi().searches
    state = searches.read_search(search_id)
    results = presenter.build_results(searches.read_results(search_id),
                                      load_rate_settings().muchi_dolar)
    return {"state": presenter.build_state(state), **results}


@app.post("/api/searches/{search_id}/cancel")
def cancel_search(search_id: str, request: CancelRequest) -> dict:
    state = build_muchi().searches.cancel_search(search_id, request.key)
    return {"state": presenter.build_state(state)}


@app.get("/api/searches/{search_id}/cart")
def read_cart(search_id: str, shipping: int = Query(4000, ge=0, le=1_000_000)) -> dict:
    items = build_muchi().searches.read_results(search_id)
    return presenter.build_cart(items, shipping, load_rate_settings().muchi_dolar)


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
