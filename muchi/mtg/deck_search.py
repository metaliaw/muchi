"""Estado reanudable de una cotizacion de lista.

Streamlit vuelve a ejecutar la aplicacion completa con cada interaccion. Este
objeto vive en ``session_state`` para que una caricia a Muchi no borre las
cartas que ya se alcanzaron a consultar.
"""
from __future__ import annotations

import time
from collections.abc import Callable
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


@dataclass(frozen=True)
class StepOutcome:
    """Errores separados: consultar afecta el resultado; archivar, no."""
    query_error: Exception | None = None
    archive_error: Exception | None = None


def run_next(job: DeckSearchJob,
             lookup: Callable[[str], list[Offer]],
             archive: Callable[[str, list[Offer]], None] | None = None,
             ) -> StepOutcome:
    """Consulta una carta y la confirma antes de intentar guardar historial.

    El historial es una comodidad secundaria. Un SQLite bloqueado o un disco
    de solo lectura no puede convertir ofertas ya recibidas en un falso error
    de consulta ni vaciar el carrito.
    """
    current = job.current
    if current is None:
        raise RuntimeError("the deck search is already complete")

    try:
        its_offers = lookup(current.name)
    except Exception as exc:
        job.record(failed=True)
        return StepOutcome(query_error=exc)

    job.record(its_offers)
    if not its_offers or archive is None:
        return StepOutcome()

    try:
        archive(current.name, its_offers)
    except Exception as exc:
        return StepOutcome(archive_error=exc)
    return StepOutcome()
