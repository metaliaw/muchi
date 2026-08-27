"""El cliente HTTP que la app usa para hablar con la API.

Es el unico lugar del frontend que sabe de URLs y del formato del wire. La app
recibe de vuelta los tipos del dominio (Oferta, Plan, Recomendacion, ...), no
dicts crudos, asi el resto del codigo no nota que hay una red de por medio.

La base se lee de MUCHI_API_URL; sin ella, apunta a localhost.
"""
from __future__ import annotations

import json
import os
from collections.abc import Iterator
from dataclasses import asdict

import requests

from muchi.mtg.models import Offer, Order
from muchi.mtg.optimizer import Line, Plan
from muchi.mtg.ports import Progress, Recommendation, StoreStatus

DEFAULT_URL = "http://localhost:8000"


class ApiError(Exception):
    """La API respondio con un error. .status lleva el codigo HTTP."""

    def __init__(self, status: int, detail: str):
        self.status = status
        self.detail = detail
        super().__init__(detail)


class NotFound(ApiError):
    """La API respondio 404: no hay comandante, inventario o recurso."""


class MuchiClient:
    def __init__(self, base_url: str | None = None):
        self.base_url = (base_url or os.getenv("MUCHI_API_URL") or DEFAULT_URL).rstrip("/")
        self._s = requests.Session()

    # ------------------------------------------------------------- lo de adentro
    def _raise_for(self, r: requests.Response) -> requests.Response:
        if r.status_code < 400:
            return r
        try:
            detail = r.json().get("detail", r.text)
        except ValueError:
            detail = r.text
        if r.status_code == 404:
            raise NotFound(r.status_code, detail)
        raise ApiError(r.status_code, detail)

    def _get(self, path: str, params=None):
        return self._raise_for(self._s.get(self.base_url + path, params=params, timeout=120))

    def _post(self, path: str, payload: dict):
        return self._raise_for(self._s.post(self.base_url + path, json=payload, timeout=120))

    def _iter_sse(self, path: str, payload: dict | None = None, params=None):
        if payload is None:
            r = self._s.get(self.base_url + path, params=params, stream=True, timeout=120)
        else:
            r = self._s.post(self.base_url + path, json=payload, stream=True, timeout=120)
        self._raise_for(r)
        with r:
            for line in r.iter_lines(decode_unicode=True):
                if not line or not line.startswith("data:"):
                    continue
                body = line[5:].strip()
                if body:
                    yield json.loads(body)

    # ------------------------------------------------------------- el contrato
    def suggest_names(self, text: str) -> list[str]:
        return self._get("/suggest", {"q": text}).json()

    def find_offers(self, card_name: str) -> tuple[str | None, list[Offer]]:
        data = self._get("/offers", {"q": card_name}).json()
        return data["card_id"], [Offer(**o) for o in data["offers"]]

    def refresh_offers(self, card_id: str) -> Iterator[Progress]:
        # El avance viene como "index": "done" marca solo el fin del stream.
        # Un error a mitad de camino llega como evento, no como status HTTP.
        for ev in self._iter_sse(f"/offers/{card_id}/refresh"):
            if "error" in ev:
                raise ApiError(502, ev["error"])
            if ev.get("done"):
                return
            yield Progress(ev["store"], ev["index"], ev["total"])

    def recommend_cards(self, commander: str) -> list[Recommendation]:
        data = self._get("/recommend", {"commander": commander}).json()
        return [Recommendation(**r) for r in data]

    def read_history(self, card_name: str) -> list[dict]:
        return self._get("/history", {"card": card_name}).json()

    def save_prices(self, card_name: str, offers: list[Offer]) -> None:
        self._post("/history", {"card_name": card_name,
                                "offers": [asdict(o) for o in offers]})

    def quote_decklist(self, orders: list[Order]) -> Iterator[dict]:
        for ev in self._iter_sse("/decklist/quote",
                                  {"orders": [asdict(o) for o in orders]}):
            if "offers" in ev:
                ev["offers"] = [Offer(**o) for o in ev["offers"]]
            yield ev

    def build_cart_plan(self, orders: list[Order], offers_by_card: dict,
                        shipping_per_store: int, strategy: str) -> Plan:
        body = {
            "orders": [asdict(o) for o in orders],
            "offers_by_card": {k: [asdict(o) for o in v] for k, v in offers_by_card.items()},
            "shipping_per_store": shipping_per_store,
            "strategy": strategy,
        }
        data = self._post("/cart/plan", body).json()
        return Plan(lines=[Line(**l) for l in data["lines"]],
                    missing=data["missing"],
                    shipping_per_store=data["shipping_per_store"])

    def read_stores(self) -> list[StoreStatus]:
        return [StoreStatus(**s) for s in self._get("/stores").json()]

    def list_blocked_stores(self) -> dict:
        return self._get("/stores/blocked").json()

    def index_store(self, store: str, url: str) -> Iterator[dict]:
        for ev in self._iter_sse(f"/stores/{store}/index", params={"url": url}):
            if "error" in ev:
                raise ApiError(502, ev["error"])
            yield ev

    def import_inventory(self) -> Iterator[dict]:
        for ev in self._iter_sse("/inventory/import", {}):
            if "error" in ev:
                raise ApiError(502, ev["error"])
            yield ev

    def read_inventory(self) -> dict | None:
        data = self._get("/inventory").json()
        if data is None:
            return None
        data["rates"] = [(r["label"], r["rate"]) for r in data["rates"]]
        if data.get("status"):
            data["status"] = StoreStatus(**data["status"])
        return data
