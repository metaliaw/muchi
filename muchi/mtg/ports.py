"""Los puertos que el nucleo declara y que algo de afuera cumple.

Cada puerto se llama por la necesidad que resuelve, nunca por el proveedor
que la cumple: FuenteOfertas, no ClienteScry. El nucleo depende de la forma;
el vendor vive detras, en un solo archivo bajo sources/.

Los errores tambien son del dominio. La app atrapa ComandanteSinDatos, jamas
el requests.HTTPError que lo origino: un error del vendor que cruza el puerto
es una fuga, igual que un tipo del vendor.
"""
from __future__ import annotations

from collections.abc import Iterator
from dataclasses import dataclass
from typing import Protocol, runtime_checkable

from .models import Offer


class QueryFailed(RuntimeError):
    """La fuente no respondio, o respondio algo que no entendemos."""


class CommanderNotFound(LookupError):
    """Nadie publica recomendaciones para ese comandante."""


class InventoryUnavailable(LookupError):
    """Esa lista de inventario no existe o no es publica."""


@dataclass(frozen=True)
class Progress:
    """Un paso del refresco en vivo, ya traducido del formato de la fuente."""
    store: str
    done: int
    total: int


@dataclass(frozen=True)
class Recommendation:
    """Una carta que la gente juega con ese comandante, y cuanto."""
    name: str
    category: str
    inclusion: float    # 0..1
    synergy: float

    @property
    def inclusion_pct(self) -> float:
        return round(self.inclusion * 100, 1)


@dataclass(frozen=True)
class StoreStatus:
    """Que sabemos de una tienda indexada, para mostrarlo en la app."""
    store: str
    url: str
    indexed: bool = False
    offers: int = 0
    products: int = 0
    updated: str = ""


@runtime_checkable
class OfferSource(Protocol):
    """De aqui salen las ofertas de una carta. La app no sabe de quien."""

    name: str

    def find_offers(self, card_name: str) -> list[Offer]:
        ...


@runtime_checkable
class PrimarySource(OfferSource, Protocol):
    """La fuente que ademas corrige nombres y refresca en vivo."""

    def suggest_names(self, text: str) -> list[str]:
        ...

    def identify_card(self, name: str) -> tuple[str | None, list[Offer]]:
        ...

    def refresh_offers(self, card_id: str) -> Iterator[Progress]:
        ...


@runtime_checkable
class DeckAdvisor(Protocol):
    """Que le falta a un mazo, segun lo que juega el resto."""

    def recommend_cards(self, commander: str) -> list[Recommendation]:
        ...


@runtime_checkable
class StoreCatalog(Protocol):
    """Una tienda que publica su catalogo entero, para indexarlo de una vez."""

    def list_indexable_stores(self) -> dict[str, str]:
        ...

    def list_blocked_stores(self) -> dict[str, tuple[str, str]]:
        ...

    def download_offers(self, store: str, url: str,
                        progress=None) -> tuple[list[Offer], int]:
        ...


@runtime_checkable
class PublishedInventory(Protocol):
    """Un inventario llevado como lista publica, con su propia tasa de cambio."""

    store: str

    def list_rates(self) -> list[tuple[str, int]]:
        ...

    def import_offers(self, progress=None) -> tuple[list[Offer], int]:
        ...
