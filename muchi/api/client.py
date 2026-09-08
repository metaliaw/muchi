"""Adapta el Contrato HTTP de Muchi a Búsquedas del Dominio."""
from __future__ import annotations

from dataclasses import dataclass, field
from decimal import Decimal, InvalidOperation
from urllib.parse import quote

import requests

from muchi.mtg.models import Order
from muchi.mtg.ports import QueryFailed, SearchRejected
from muchi.mtg.search import SearchItem, SearchOffer, SearchState


def build_search(reply: dict) -> SearchState:
    return SearchState(
        id=reply["id"], status=reply["status"], total=reply["total"],
        processed=reply["processed"], found=reply["found"],
        errors=reply["errors"], current_card=reply.get("current_card", ""),
    )


def build_offer(row: dict) -> SearchOffer:
    amount = Decimal(row["price_amount"])
    if not amount.is_finite() or amount < 0:
        raise ValueError("Invalid price")
    return SearchOffer(
        card_name=row["card_name"], store=row["store"], amount=amount,
        currency=row["price_currency"], url=row["url"],
        stock_status=row["stock_status"], suspicious=row["suspicious"],
        source=row["source"], finish=row.get("finish", ""),
        suspicious_reason=row.get("suspicious_reason", ""),
    )


def build_items(reply: dict) -> tuple[SearchItem, ...]:
    ordered = sorted(reply["items"], key=lambda row: row["position"])
    return tuple(SearchItem(
        name=row["original_name"], quantity=row["quantity"], status=row["status"],
        offers=tuple(build_offer(offer) for offer in row["offers"]),
        error_message=row.get("error_message", ""),
    ) for row in ordered)


@dataclass
class SearchProvider:
    base_url: str
    token: str = field(repr=False)
    timeout_seconds: float = 10
    session: requests.Session = field(default_factory=requests.Session, repr=False)

    def request_reply(self, method: str, path: str, *, payload=None,
                      key: str | None = None, params=None) -> dict:
        headers = {"Authorization": f"Bearer {self.token}", "Accept": "application/json"}
        if key:
            headers["Idempotency-Key"] = key
        try:
            response = self.session.request(
                method, f"{self.base_url}{path}", headers=headers,
                json=payload, params=params, timeout=self.timeout_seconds,
                allow_redirects=False,
            )
            with response:
                if response.status_code in (401, 403):
                    raise SearchRejected("La API rechazó el Código de Seguridad.")
                if response.status_code == 409:
                    raise SearchRejected("La API informó un Conflicto de Estado o Idempotencia.")
                if 400 <= response.status_code < 500 and response.status_code not in (408, 429):
                    raise SearchRejected(f"La API rechazó el Pedido: HTTP {response.status_code}.")
                if not 200 <= response.status_code < 300:
                    raise QueryFailed(f"La API respondió HTTP {response.status_code}.")
                reply = response.json()
                if not isinstance(reply, dict):
                    raise ValueError("Expected object")
                return reply
        except (requests.RequestException, ValueError):
            raise QueryFailed("No se pudo obtener una Respuesta válida de la API.") from None

    def parse_reply(self, parser, reply):
        try:
            return parser(reply)
        except (KeyError, TypeError, ValueError, InvalidOperation):
            raise QueryFailed("La Respuesta no cumple el Contrato de la API.") from None

    def create_search(self, orders: list[Order], verify_stock: bool,
                      stores_only: bool, key: str) -> SearchState:
        if not 1 <= len(orders) <= 500:
            raise QueryFailed("La Búsqueda admite entre 1 y 500 Cartas.")
        if any(not order.name.strip() or not 1 <= order.quantity <= 99 for order in orders):
            raise QueryFailed("Cada Carta requiere Nombre y Cantidad entre 1 y 99.")
        payload = {
            "cards": [{"name": order.name, "quantity": order.quantity} for order in orders],
            "options": {"verify_stock": verify_stock, "stores_only": stores_only},
        }
        reply = self.request_reply("POST", "/searches", payload=payload, key=key)
        return self.parse_reply(build_search, reply)

    def read_search(self, search_id: str) -> SearchState:
        reply = self.request_reply("GET", f"/searches/{quote(search_id, safe='')}")
        return self.parse_reply(build_search, reply)

    def read_results(self, search_id: str) -> tuple[SearchItem, ...]:
        reply = self.request_reply("GET", f"/searches/{quote(search_id, safe='')}/results")
        return self.parse_reply(build_items, reply)

    def cancel_search(self, search_id: str, key: str) -> SearchState:
        reply = self.request_reply(
            "POST", f"/searches/{quote(search_id, safe='')}/cancel", key=key,
        )
        return self.parse_reply(build_search, reply)

    def read_sources(self) -> list[dict]:
        reply = self.request_reply("GET", "/health/sources")
        return self.parse_reply(lambda value: value["sources"], reply)

    def find_offers(self, name: str) -> tuple[SearchOffer, ...]:
        reply = self.request_reply("GET", "/cards/offers", params={"name": name})
        return self.parse_reply(
            lambda value: tuple(build_offer(row) for row in value["offers"]), reply,
        )
