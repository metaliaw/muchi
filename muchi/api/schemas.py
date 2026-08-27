"""Los contratos HTTP de la API, en Pydantic.

Los tipos del dominio (Oferta, Pedido, Plan, ...) no cruzan la API: viven en
muchi/mtg y aca se traducen. Los modelos de salida leen atributos del dominio
con from_attributes; los de entrada se convierten de vuelta en el borde de las
rutas.
"""
from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, ConfigDict

from muchi.mtg.models import Offer, Order


class OfferOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    store: str
    card_name: str
    title: str
    price_clp: int
    url: str
    finish: str = ""
    condition: str = ""
    language: str = ""
    stock: int | None = None
    source: str = "scry"
    key: str = ""
    marketplace: bool = False


class OrderIn(BaseModel):
    quantity: int
    name: str


class RecommendationOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    name: str
    category: str
    inclusion: float
    synergy: float


class HistoryRowOut(BaseModel):
    ts: str
    minimo: float
    promedio: float
    n: int


class SavePricesRequest(BaseModel):
    card_name: str
    offers: list[OfferOut]


class DecklistQuoteRequest(BaseModel):
    orders: list[OrderIn]


class CartPlanRequest(BaseModel):
    orders: list[OrderIn]
    offers_by_card: dict[str, list[OfferOut]]
    shipping_per_store: int = 0
    strategy: Literal["optimal", "naive"] = "optimal"


class CartLineOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    card_name: str
    quantity: int
    store: str
    unit_price: int
    url: str
    title: str


class CartPlanOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    lines: list[CartLineOut]
    missing: list[str]
    shipping_per_store: int


class StoreStatusOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    store: str
    url: str
    indexed: bool = False
    offers: int = 0
    products: int = 0
    updated: str = ""


class OffersOut(BaseModel):
    card_id: str | None
    offers: list[OfferOut]


class BlockedStoreOut(BaseModel):
    url: str
    reason: str


class InventoryListRateOut(BaseModel):
    label: str
    rate: int


class InventoryStatusOut(BaseModel):
    store: str
    rates: list[InventoryListRateOut]
    status: StoreStatusOut | None


def offer_from_in(x: OfferOut) -> Offer:
    """Traduce el contrato de vuelta al tipo del dominio, en el borde."""
    return Offer(
        store=x.store, card_name=x.card_name, title=x.title, price_clp=x.price_clp,
        url=x.url, finish=x.finish, condition=x.condition, language=x.language,
        stock=x.stock, source=x.source, key=x.key, marketplace=x.marketplace,
    )


def order_from_in(x: OrderIn) -> Order:
    return Order(quantity=x.quantity, name=x.name)
