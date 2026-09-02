"""Estado reanudable de una cotizacion de lista.

Streamlit vuelve a ejecutar la aplicacion completa con cada interaccion. Este
objeto vive en ``session_state`` para que una caricia a Muchi no borre las
cartas que ya se alcanzaron a consultar.
"""
from __future__ import annotations

import time
from dataclasses import dataclass, field

from .models import Offer, Order


@dataclass
class DeckSearchJob:
    orders: tuple[Order, ...]
    next_index: int = 0
    found_by_card: dict[str, list[Offer]] = field(default_factory=dict)
    failed: list[str] = field(default_factory=list)
    started_at: float = field(default_factory=time.monotonic)

    @classmethod
    def start(cls, orders: list[Order]) -> "DeckSearchJob":
        if not orders:
            raise ValueError("a deck search needs at least one order")
        return cls(tuple(orders))

    @property
    def total(self) -> int:
        return len(self.orders)

    @property
    def done(self) -> bool:
        return self.next_index >= self.total

    @property
    def current(self) -> Order | None:
        return None if self.done else self.orders[self.next_index]

    @property
    def elapsed(self) -> float:
        return max(0.0, time.monotonic() - self.started_at)

    def record(self, its_offers: list[Offer] | None = None,
               *, failed: bool = False) -> None:
        """Guarda el resultado de la carta actual y avanza exactamente una."""
        current = self.current
        if current is None:
            raise RuntimeError("the deck search is already complete")
        if failed:
            self.failed.append(current.name)
        elif its_offers:
            self.found_by_card[current.name.lower()] = its_offers
        self.next_index += 1
