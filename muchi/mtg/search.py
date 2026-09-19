"""Contrato de Búsquedas persistidas por el Servicio."""
from __future__ import annotations

from dataclasses import dataclass, field
from decimal import Decimal
from typing import Protocol

from .models import Order


@dataclass(frozen=True)
class SearchOffer:
    card_name: str
    store: str
    amount: Decimal
    currency: str
    url: str
    stock_status: str
    suspicious: bool
    source: str
    finish: str = ""
    language: str = ""
    condition: str = ""
    variant: str = ""
    title: str = ""
    suspicious_reason: str = ""
    # La Carta que Nombra la Oferta, sin su Impresion. La API la Calcula; vacia
    # cuando Responde una Version anterior, y ahi manda el Titulo.
    card_key: str = ""
    # La Edicion conserva su Posicion: las Pruebas arman esta Forma por Posicion.
    edition: str = ""
    locations: tuple[str, ...] = ()
    metadata: dict[str, str] = field(default_factory=dict)
    # El Nombre que la API le da a esta Oferta. Sin el, nadie puede volver a
    # preguntar por ella: una URL repetida no Distingue dos Impresiones.
    offer_id: str = ""
    # Cuantas Unidades declara la Tienda. None es "no lo sabemos"; 0 es
    # "lo preguntamos y no hay". Son Estados distintos y se muestran distinto.
    stock_quantity: int | None = None
    # La Foto que publica la Tienda. Una Carta puede pedirsela al Catalogo del
    # Juego; una Caja sellada no Existe en ese Catalogo, y esta es la unica que
    # va a tener. Ultima en la Forma: las Pruebas arman Ofertas por Posicion.
    image: str = ""


@dataclass(frozen=True)
class SearchItem:
    name: str
    quantity: int
    status: str
    offers: tuple[SearchOffer, ...]
    error_message: str = ""
    id: str = ""
    position: int = 0
    sequence: int = 0
    game: str = ""
    # Las Fuentes que se Cayeron mientras otras Respondian. Una Carta con
    # `found` y una Tienda caida no es lo mismo que una Carta completa.
    faults: tuple[str, ...] = ()


@dataclass(frozen=True)
class StockCheck:
    """Lo que la Tienda contesto cuando se le volvio a preguntar por una Oferta."""
    offer_id: str
    stock_status: str
    stock_quantity: int | None = None

    @property
    def available(self) -> bool:
        """Agotado es Ausencia; `unknown` no lo es: la Tienda no Nego, no Supo."""
        if self.stock_status == "unavailable":
            return False
        return self.stock_quantity is None or self.stock_quantity > 0

    @property
    def confirmed(self) -> bool:
        """La Tienda Dijo que si Tiene. `unknown` sigue siendo una Duda."""
        return self.stock_status == "available" and self.available


@dataclass(frozen=True)
class SearchResults:
    items: tuple[SearchItem, ...]
    cursor: int
    has_more: bool


@dataclass(frozen=True)
class SearchState:
    id: str
    status: str
    total: int
    processed: int
    found: int
    errors: int
    current_card: str = ""
    game: str = ""

    @property
    def done(self) -> bool:
        return self.status in {
            "completed", "completed_with_errors", "failed", "cancelled",
        }


class SearchService(Protocol):
    def create_search(self, orders: list[Order], verify_stock: bool,
                      key: str, game: str = "magic", match: str = "exact",
                      kind: str = "single") -> SearchState: ...
    def read_search(self, search_id: str) -> SearchState: ...
    def read_results(self, search_id: str, after: int = 0) -> SearchResults: ...
    def cancel_search(self, search_id: str, key: str) -> SearchState: ...
    def check_stock(self, search_id: str,
                    offer_ids: tuple[str, ...]) -> tuple[StockCheck, ...]: ...
    def read_sources(self) -> list[dict]: ...
    def read_supported_games(self) -> list[dict]: ...
    def read_card_metadata(self, game: str, name: str, language: str = "",
                           edition: str = "", foil: bool = False) -> dict: ...
    def autocomplete_cards(self, game: str, name: str, language: str = "") -> list[str]: ...
