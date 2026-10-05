"""Herramientas MCP para buscar y leer Ofertas de MUCHI."""
from __future__ import annotations

from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
import os
from typing import Any, Literal
from uuid import uuid4

from mcp.server.mcpserver import MCPServer
from mcp.server.transport_security import TransportSecuritySettings
from mcp.types import ToolAnnotations

from muchi.mtg import cast as muchi_cast
from muchi.mtg import decklist
from muchi.mtg.models import Order
from muchi.mtg.search import SearchState
from muchi.mtg.settings import load_rate_settings
from server import presenter

MAX_CARDS = 100
MAX_QUANTITY = 99
MAX_SEARCH_LENGTH = 20_000
MATCH_EXACT = "exact"
KIND_SINGLE = "single"
PRICE_ASC = "price_asc"
PRICE_DESC = "price_desc"
VERIFY_STOCK = os.getenv("MUCHI_VERIFY_STOCK", "0").strip() in {
    "1", "true", "si", "yes",
}

MCP_HOSTS = [
    "muchitcg.cl", "muchitcg.cl:*",
    "www.muchitcg.cl", "www.muchitcg.cl:*",
    # Firebase Hosting reescribe la Petición y Cloud Run recibe su Host de Servicio.
    "muchi-web-c2ce6c7oga-rj.a.run.app",
    "muchi-web-c2ce6c7oga-rj.a.run.app:*",
    "localhost:*", "127.0.0.1:*",
]
MCP_ORIGINS = [
    "https://muchitcg.cl", "https://www.muchitcg.cl",
    "http://localhost:8000", "http://127.0.0.1:8000",
]

mcp = MCPServer("MUCHI")


def render_search_state(state: SearchState) -> dict[str, Any]:
    """Devuelve el avance en una Forma útil para un Agente."""
    return {
        "id": state.id,
        "status": state.status,
        "total": state.total,
        "processed": state.processed,
        "found": state.found,
        "errors": state.errors,
        "current_card": state.current_card,
        "done": state.done,
        "game": state.game,
    }


def validate_decklist(text: str, kind: str) -> list[Order]:
    """Interpreta la Lista y aplica los mismos Topes que el Sitio."""
    if not text.strip() or len(text) > MAX_SEARCH_LENGTH:
        raise ValueError(f"La Lista debe tener entre 1 y {MAX_SEARCH_LENGTH} caracteres.")
    orders, ignored = decklist.parse_decklist(text, sealed=kind == "sealed")
    if ignored:
        raise ValueError("Revisa las líneas ignoradas: " + ", ".join(ignored))
    if not 1 <= len(orders) <= MAX_CARDS:
        raise ValueError(f"La Lista debe tener entre 1 y {MAX_CARDS} entradas.")
    if any(not 1 <= order.quantity <= MAX_QUANTITY for order in orders):
        raise ValueError(f"Cada cantidad debe estar entre 1 y {MAX_QUANTITY}.")
    return orders


def create_search(text: str, game: str, match: str, kind: str) -> SearchState:
    """Crea una Búsqueda persistida en la API de MUCHI."""
    if not 1 <= len(game) <= 50:
        raise ValueError("El juego debe tener entre 1 y 50 caracteres.")
    orders = validate_decklist(text, kind)
    return muchi_cast.build_cast().searches.create_search(
        orders=orders,
        verify_stock=VERIFY_STOCK,
        key=uuid4().hex,
        game=game,
        match=match,
        kind=kind,
    )


def read_search_page(search_id: str, after: int, match: str) -> dict[str, Any]:
    """Lee el Estado y una Página de Ofertas de la Búsqueda."""
    searches = muchi_cast.build_cast().searches
    state = searches.read_search(search_id)
    page = searches.read_results(search_id, after)
    results = presenter.build_results(
        page.items,
        load_rate_settings().muchi_dolar,
        verified=VERIFY_STOCK,
        match=match,
    )
    return {
        "state": render_search_state(state),
        **results,
        "cursor": page.cursor,
        "has_more": page.has_more,
    }


def sort_offers_by_price(offers: list[dict], order: str) -> list[dict]:
    """Ordena por Precio convertido, conservando separados los Tipos."""
    groups: dict[str, list[dict]] = {}
    for offer in offers:
        groups.setdefault(offer["card_type"], []).append(offer)

    ordered = []
    for rows in groups.values():
        priced = [row for row in rows if row["price_clp"] is not None]
        unpriced = [row for row in rows if row["price_clp"] is None]
        priced.sort(key=lambda row: row["price_clp"], reverse=order == PRICE_DESC)
        ordered.extend(priced + unpriced)
    return ordered


@mcp.tool(annotations=ToolAnnotations(open_world_hint=True))
def search_cards(
    decklist: str,
    game: str = "magic",
    match: Literal["exact", "includes"] = MATCH_EXACT,
    kind: Literal["single", "sealed"] = KIND_SINGLE,
) -> dict[str, Any]:
    """Inicia una Búsqueda de Cartas o Productos Sellados en MUCHI.

    Args:
        decklist: Lista con Cantidades opcionales, una Entrada por Línea.
        game: Clave de un Juego que soporta MUCHI API.
        match: Modo exacto o inclusivo para hacer coincidir los Nombres.
        kind: Si se buscan Cartas sueltas o Productos Sellados.
    """
    state = create_search(decklist, game, match, kind)
    return {"search_id": state.id, "state": render_search_state(state)}


@mcp.tool(annotations=ToolAnnotations(read_only_hint=True, open_world_hint=True))
def get_search_results(
    search_id: str,
    after: int = 0,
    match: Literal["exact", "includes"] = MATCH_EXACT,
    sort_by: Literal["price_asc", "price_desc"] = PRICE_ASC,
) -> dict[str, Any]:
    """Lee el Estado y una Página de Entradas; repite con el Cursor si hay más.

    Args:
        search_id: Identificador devuelto por `search_cards`.
        after: Cursor de la Página anterior, o cero para empezar.
        match: El mismo Modo enviado al crear la Búsqueda.
        sort_by: Orden de Precio CLP para las Ofertas de cada Tipo de Carta.
    """
    if after < 0:
        raise ValueError("El cursor debe ser cero o mayor.")
    page = read_search_page(search_id, after, match)
    page["offers"] = sort_offers_by_price(page["offers"], sort_by)
    return page


mcp_app = mcp.streamable_http_app(
    streamable_http_path="/",
    stateless_http=True,
    json_response=True,
    transport_security=TransportSecuritySettings(
        allowed_hosts=MCP_HOSTS,
        allowed_origins=MCP_ORIGINS,
    ),
)


@asynccontextmanager
async def mcp_lifespan(_app) -> AsyncIterator[None]:
    """Mantiene disponible el Administrador de Sesiones MCP."""
    async with mcp.session_manager.run():
        yield
