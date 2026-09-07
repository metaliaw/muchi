"""Contrato de Búsquedas persistidas por el Servicio."""
from __future__ import annotations

from dataclasses import dataclass
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
    suspicious_reason: str = ""


@dataclass(frozen=True)
class SearchItem:
    name: str
    quantity: int
    status: str
    offers: tuple[SearchOffer, ...]
    error_message: str = ""


@dataclass(frozen=True)
class SearchState:
    id: str
    status: str
    total: int
    processed: int
    found: int
    errors: int
    current_card: str = ""

    @property
    def done(self) -> bool:
        return self.status in {
            "completed", "completed_with_errors", "failed", "cancelled",
        }


class SearchService(Protocol):
    def create_search(self, orders: list[Order], verify_stock: bool,
                      stores_only: bool, key: str) -> SearchState: ...
    def read_search(self, search_id: str) -> SearchState: ...
    def read_results(self, search_id: str) -> tuple[SearchItem, ...]: ...
    def cancel_search(self, search_id: str, key: str) -> SearchState: ...
    def read_sources(self) -> list[dict]: ...
