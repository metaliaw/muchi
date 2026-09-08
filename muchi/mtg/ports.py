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


class SearchRejected(QueryFailed):
    """El Servicio rechazó el Pedido; corregirlo permite un nuevo Envío."""


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
class Card:
    """Una carta del catalogo: que es y que dice. Sin precio ni tienda.

    `name` es SIEMPRE el ingles y es la clave canonica: es lo que indexan las
    tiendas chilenas, asi que es lo que viaja al carrito. Lo traducido va en
    los campos `local_*`, que existen solo para mostrar y pueden venir vacios
    --- media biblioteca no tiene impresion en espanol.
    """
    name: str
    oracle_id: str = ""
    text: str = ""
    type_line: str = ""
    mana_cost: str = ""
    image: str = ""
    url: str = ""
    rarity: str = ""
    local_name: str = ""
    local_text: str = ""
    local_type: str = ""
    local_image: str = ""

    @property
    def is_translated(self) -> bool:
        return bool(self.local_name)

    def show_as(self, language: str = "es") -> tuple[str, str, str, str]:
        """(nombre, tipo, texto, imagen) en ese idioma, cayendo al ingles."""
        if language == "es":
            return (self.local_name or self.name,
                    self.local_type or self.type_line,
                    self.local_text or self.text,
                    self.local_image or self.image)
        return self.name, self.type_line, self.text, self.image


@dataclass(frozen=True)
class CardRequest:
    """Un intento de busqueda, dicho en necesidad y no en sintaxis de nadie.

    El nucleo arma varios, del mas estricto al mas suelto; la fuente los
    traduce a lo suyo. `note` cuenta que se solto para llegar hasta aca, y la
    app la muestra tal cual: si Muchi aflojo la busqueda hay que decirlo.
    """
    phrases: tuple[str, ...] = ()      # frases que el texto de la carta debe traer
    words: tuple[str, ...] = ()        # palabras sueltas, mismo trato
    intents: tuple[str, ...] = ()      # claves de oracle.INTENTS
    literal_text: str = ""             # el texto del usuario crudo, ultimo recurso
    colors: tuple[str, ...] = ()       # letras wubrg
    card_type: str = ""
    format_name: str = ""
    max_mana: int | None = None
    note: str = ""

    @property
    def is_empty(self) -> bool:
        return not (self.phrases or self.words or self.intents
                    or self.literal_text or self.colors or self.card_type
                    or self.format_name) and self.max_mana is None


@dataclass(frozen=True)
class CardPage:
    """Lo que devolvio una busqueda de catalogo."""
    cards: tuple[Card, ...] = ()
    total: int = 0
    # Como entendio la fuente el pedido, en su propio idioma, para mostrarselo
    # a quien quiera repetir la busqueda a mano. La app no lo interpreta.
    explain: str = ""


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
class StockVerifier(Protocol):
    """Comprueba una variante en la tienda: True, False o desconocido."""

    def verify_stock(self, offer: Offer) -> bool | None:
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
class CardCatalog(Protocol):
    """Que cartas existen y que dicen. No sabe nada de precios ni de tiendas.

    Es el otro lado de FuenteOfertas: aquella responde "cuanto vale Sol Ring",
    esta responde "que cartas destruyen una criatura". La app las usa juntas
    --- se elige una carta aca y se cotiza alla --- pero son necesidades
    distintas y nada obliga a que las cumpla el mismo proveedor.
    """

    def search_cards(self, request: CardRequest) -> CardPage:
        ...

    def resolve_name(self, text: str) -> Card | None:
        ...

    def translate_cards(self, cards: list[Card], language: str) -> list[Card]:
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
