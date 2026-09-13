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
    # La Edicion va ultima: las Pruebas arman esta Forma por Posicion.
    edition: str = ""
    metadata: dict[str, str] = field(default_factory=dict)


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
                      stores_only: bool, key: str, game: str = "magic") -> SearchState: ...
    def read_search(self, search_id: str) -> SearchState: ...
    def read_results(self, search_id: str, after: int = 0) -> SearchResults: ...
    def cancel_search(self, search_id: str, key: str) -> SearchState: ...
    def read_sources(self) -> list[dict]: ...
    def read_supported_games(self) -> list[dict]: ...
    def read_card_metadata(self, game: str, name: str, language: str = "",
                           edition: str = "", foil: bool = False) -> dict: ...
    def autocomplete_cards(self, game: str, name: str, language: str = "") -> list[str]: ...
