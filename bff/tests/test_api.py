"""Tests for the HTTP API, against a Cast built with fakes (no network).

Run with pytest. They assert the contract of each endpoint and the error codes,
so a change in the wire shape breaks here and not in the app.
"""
from __future__ import annotations

import json

import pytest
from fastapi.testclient import TestClient

from muchi.api import build_app
from muchi.mtg import catalog, cast as cast_mod, db
from muchi.mtg.models import Offer
from muchi.mtg.ports import CommanderNotFound, Progress, QueryFailed, Recommendation


def _offer(store, card_name="Sol Ring", price=1000, **kw):
    return Offer(store=store, card_name=card_name, title=card_name,
                 price_clp=price, url="http://x", key=f"{store}-{card_name}", **kw)


class _FakePrimary:
    name = "scry"

    def __init__(self, offers=(), raises=None):
        self._offers = list(offers)
        self._raises = raises

    def identify_card(self, name):
        if self._raises:
            raise self._raises
        return "card-1", list(self._offers)

    def find_offers(self, name):
        return list(self._offers)

    def suggest_names(self, text):
        return ["Sol Ring", "Sol Talisman"]

    def refresh_offers(self, card_id):
        yield Progress("T1", 1, 2)
        yield Progress("T2", 2, 2)


class _FailingRefresh(_FakePrimary):
    def refresh_offers(self, card_id):
        yield Progress("T1", 1, 2)
        raise QueryFailed("scry se cayo")


class _FakeAdvisor:
    def recommend_cards(self, commander):
        if commander == "Nadie":
            raise CommanderNotFound(commander)
        return [Recommendation("Sol Ring", "Top Cards", 0.9, 0.1)]


class _FakeCatalog:
    def list_indexable_stores(self):
        return {"PDA Chile": "https://pdachile.cl"}

    def list_blocked_stores(self):
        return {"Magic Chile": ("https://magic-chile.cl", "robots bloquea a todos")}

    def download_offers(self, store, url, progress=None):
        if progress:
            progress(1, 1)
        return [_offer(store)], 1


def _cx():
    import sqlite3
    cx = sqlite3.connect(":memory:", check_same_thread=False)
    cx.row_factory = sqlite3.Row
    cx.executescript(db._SCHEMA)
    catalog.ensure_tables(cx)
    return cx


def _make_cast(primary=None):
    return cast_mod.Cast(
        cx=_cx(),
        primary=primary or _FakePrimary(),
        extras=[],
        advisor=_FakeAdvisor(),
        stores=_FakeCatalog(),
        inventory=None,
    )


@pytest.fixture
def client():
    with TestClient(build_app(cast=_make_cast())) as c:
        yield c


def _events(text):
    return [json.loads(line[5:].strip()) for line in text.splitlines()
            if line.startswith("data:")]


# ------------------------------------------------------------------- search
def test_suggest_names(client):
    assert client.get("/suggest", params={"q": "sol"}).json() == \
        ["Sol Ring", "Sol Talisman"]


def test_find_offers(client):
    primary = _FakePrimary([_offer("T1", price=500), _offer("T2", price=300)])
    with TestClient(build_app(cast=_make_cast(primary))) as c:
        data = c.get("/offers", params={"q": "Sol Ring"}).json()
    assert data["card_id"] == "card-1"
    assert [o["price_clp"] for o in data["offers"]] == [300, 500]


def test_offers_fail_map_to_502():
    cast = _make_cast(_FakePrimary(raises=QueryFailed("caido")))
    with TestClient(build_app(cast=cast)) as c:
        r = c.get("/offers", params={"q": "Sol Ring"})
    assert r.status_code == 502
    assert "caido" in r.json()["detail"]


# -------------------------------------------------------------- recommendations
def test_recommend(client):
    data = client.get("/recommend", params={"commander": "Atraxa"}).json()
    assert data[0]["name"] == "Sol Ring"
    assert data[0]["inclusion"] == 0.9


def test_recommend_missing_is_404(client):
    r = client.get("/recommend", params={"commander": "Nadie"})
    assert r.status_code == 404


# ------------------------------------------------------------------- history
def test_history_roundtrip(client):
    client.post("/history", json={"card_name": "Sol Ring",
                                  "offers": [_offer("T1", price=1000).__dict__]})
    rows = client.get("/history", params={"card": "Sol Ring"}).json()
    assert len(rows) == 1
    assert rows[0]["minimo"] == 1000 and rows[0]["n"] == 1


# ---------------------------------------------------------- decklist quote
def test_quote_decklist_streams(client):
    primary = _FakePrimary([_offer("T1", price=500)])
    with TestClient(build_app(cast=_make_cast(primary))) as c:
        r = c.post("/decklist/quote", json={"orders": [{"quantity": 2, "name": "Sol Ring"}]})
    events = _events(r.text)
    card = next(e for e in events if "offers" in e)
    done = next(e for e in events if e.get("done"))
    assert card["card"] == "Sol Ring" and len(card["offers"]) == 1
    assert done["failed"] == [] and done["total"] == 1


# ------------------------------------------------------------------ cart plan
def test_cart_plan_optimal(client):
    orders = [{"quantity": 1, "name": "A"}, {"quantity": 1, "name": "B"}]
    by_card = {
        "a": [_offer("T1", "A", 1000), _offer("T2", "A", 900)],
        "b": [_offer("T1", "B", 1000), _offer("T3", "B", 900)],
    }
    r = client.post("/cart/plan", json={
        "orders": orders,
        "offers_by_card": {k: [o.__dict__ for o in v] for k, v in by_card.items()},
        "shipping_per_store": 5000,
        "strategy": "optimal",
    })
    plan = r.json()
    assert r.status_code == 200
    assert plan["missing"] == []
    assert {l["store"] for l in plan["lines"]} == {"T1"}, plan


def test_cart_plan_naive_reports_missing(client):
    r = client.post("/cart/plan", json={
        "orders": [{"quantity": 1, "name": "A"}, {"quantity": 2, "name": "Nada"}],
        "offers_by_card": {"a": [_offer("T1", "A", 1000).__dict__]},
        "shipping_per_store": 0,
        "strategy": "naive",
    })
    assert r.json()["missing"] == ["Nada"]


# -------------------------------------------------------------------- stores
def test_stores_status_and_blocked(client):
    stores = client.get("/stores").json()
    assert stores[0]["store"] == "PDA Chile" and stores[0]["indexed"] is False

    blocked = client.get("/stores/blocked").json()
    assert blocked["Magic Chile"]["reason"].startswith("robots")


def test_index_store_streams_and_persists(client):
    r = client.post("/stores/Wombat/index", params={"url": "https://x"})
    events = _events(r.text)
    assert events[-1]["done"] is True and events[-1]["saved"] == 1
    assert client.get("/stores").json()[0]["indexed"] is False  # PDA sigue sin indexar


# ----------------------------------------------------------------- inventory
def test_inventory_without_config_is_null(client):
    r = client.get("/inventory")
    assert r.status_code == 200 and r.json() is None


def test_import_inventory_without_config_is_404(client):
    r = client.post("/inventory/import")
    assert r.status_code == 404


# ------------------------------------------------------------------- refresh
def test_refresh_streams_every_store(client):
    """El avance viaja como "index": si se llamara "done", el cliente cortaria
    en el primer evento y el refresco quedaria trunco."""
    events = _events(client.get("/offers/card-1/refresh").text)
    assert [(e["store"], e["index"], e["total"]) for e in events if "store" in e] == \
        [("T1", 1, 2), ("T2", 2, 2)]
    assert events[-1] == {"done": True}


def test_refresh_reports_midstream_failure():
    """Con el stream ya empezado el error no puede ser un status: va como evento."""
    cast = _make_cast(_FailingRefresh())
    with TestClient(build_app(cast=cast)) as c:
        events = _events(c.get("/offers/card-1/refresh").text)
    assert "scry se cayo" in events[-1]["error"]
    assert not any(e.get("done") for e in events)


def test_client_reads_refresh_progress():
    """El cliente traduce el wire a Progress, sin cortar antes de tiempo."""
    from muchi.api.client import ApiError, MuchiClient

    def replay(events):
        c = MuchiClient(base_url="http://x")
        c._iter_sse = lambda *a, **k: iter(events)
        return c.refresh_offers("card-1")

    got = list(replay([{"store": "T1", "index": 1, "total": 2},
                       {"store": "T2", "index": 2, "total": 2},
                       {"done": True}]))
    assert [(p.store, p.done, p.total) for p in got] == [("T1", 1, 2), ("T2", 2, 2)]

    with pytest.raises(ApiError):
        list(replay([{"store": "T1", "index": 1, "total": 2}, {"error": "scry se cayo"}]))


# ------------------------------------------------------------------ contracts
def test_cart_plan_strategy_must_be_known(client):
    """Un typo en la estrategia se rechaza; antes caia callado en la naive."""
    r = client.post("/cart/plan", json={
        "orders": [{"quantity": 1, "name": "A"}],
        "offers_by_card": {"a": [_offer("T1", "A", 1000).__dict__]},
        "shipping_per_store": 0,
        "strategy": "optimla",
    })
    assert r.status_code == 422


def test_documented_contracts_are_typed(client):
    """El plan y las bloqueadas salen con contrato, no como dict suelto."""
    schema = build_app(cast=_make_cast()).openapi()
    for path, method in [("/cart/plan", "post"), ("/stores/blocked", "get")]:
        body = schema["paths"][path][method]["responses"]["200"]["content"]
        assert body["application/json"]["schema"], f"{path} sigue sin contrato"
