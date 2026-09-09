"""El BFF traduce el Dominio a JSON y nunca deja escapar el Token."""
from decimal import Decimal

import pytest
from fastapi.testclient import TestClient

from muchi.mtg.ports import QueryFailed, SearchRejected
from muchi.mtg.search import SearchItem, SearchOffer, SearchState
from server import main


def build_offer(**changes) -> SearchOffer:
    values = dict(
        card_name="Sol Ring", store="Tienda", amount=Decimal("1000"),
        currency="CLP", url="https://tienda.cl/sol-ring", stock_status="available",
        suspicious=False, source="scry", variant="Foil NM Inglés",
    )
    values.update(changes)
    return SearchOffer(**values)


class FakeSearches:
    def __init__(self, state=None, items=()):
        self.state = state or SearchState("abc", "running", 2, 1, 1, 0, "Sol Ring")
        self.items = items
        self.created = []

    def create_search(self, orders, verify_stock, stores_only, key):
        self.created.append((orders, key))
        return self.state

    def read_search(self, search_id):
        return self.state

    def read_results(self, search_id):
        return self.items

    def cancel_search(self, search_id, key):
        return SearchState("abc", "cancelled", 2, 1, 1, 0)

    def read_sources(self):
        return [{"source": "scry", "status": "ok"}]


@pytest.fixture
def client(monkeypatch):
    searches = FakeSearches(items=(
        SearchItem("Sol Ring", 2, "found", (
            build_offer(amount=Decimal("4000")),
            build_offer(store="Otra", amount=Decimal("1500")),
            build_offer(store="Dudosa", amount=Decimal("10"), suspicious=True,
                        suspicious_reason="price_below_40_percent_median"),
            build_offer(store="Gringa", amount=Decimal("3"), currency="USD"),
        )),
        SearchItem("Black Lotus", 1, "not_found", ()),
    ))
    monkeypatch.setattr(main, "build_muchi", lambda: type("Cast", (), {"searches": searches})())
    return TestClient(main.app), searches


def test_read_search_orders_offers_and_marks_cheapest(client):
    http, _ = client
    reply = http.get("/api/searches/abc").json()
    assert reply["state"]["done"] is False
    amounts = [(offer["currency"], offer["amount"]) for offer in reply["offers"]]
    assert amounts == [("CLP", "10"), ("CLP", "1500"), ("CLP", "4000"), ("USD", "3")]
    best = [offer["store"] for offer in reply["offers"] if offer["best"]]
    assert best == ["Otra"]
    assert reply["summary"] == {"lowest_clp": 10.0, "offers": 4, "stores": 4}
    assert any("Black Lotus" in notice["text"] for notice in reply["notices"])


def test_suspicious_offer_explains_itself_in_words(client):
    http, _ = client
    reply = http.get("/api/searches/abc").json()
    dudosa = next(offer for offer in reply["offers"] if offer["store"] == "Dudosa")
    assert "⚠ Precio bajo el 40% de la Mediana de su Moneda" in [
        pill["text"] for pill in dudosa["pills"]
    ]
    assert dudosa["note"] and dudosa["action"] == "Verificar"


def test_cart_skips_suspicious_and_converts_dollars(client):
    http, _ = client
    plan = http.get("/api/searches/abc/cart?shipping=4000").json()
    assert plan["muchi_dolar"] == 1000
    assert plan["converted_offers"] == 1
    # La Oferta sospechosa de 10 pesos no entra: el Carrito toma la de 1500.
    assert plan["cards_cost"] == 3000
    assert plan["total"] == 7000
    assert plan["missing"] == ["Black Lotus"]


def test_create_search_parses_the_decklist(client):
    http, searches = client
    reply = http.post("/api/searches", json={"text": "4 Lightning Bolt", "key": "k" * 10})
    assert reply.status_code == 200
    orders, key = searches.created[0]
    assert (orders[0].quantity, orders[0].name, key) == (4, "Lightning Bolt", "k" * 10)


def test_create_search_rejects_lines_it_cannot_read(client):
    http, _ = client
    reply = http.post("/api/searches", json={"text": "https://tienda.cl/x", "key": "k" * 10})
    assert reply.status_code == 422


def test_rejected_search_stops_the_polling(client, monkeypatch):
    http, searches = client
    def reject(search_id):
        raise SearchRejected("La API rechazó el Código de Seguridad.")
    monkeypatch.setattr(searches, "read_search", reject)
    reply = http.get("/api/searches/abc")
    assert reply.status_code == 409 and reply.json()["retriable"] is False


def test_failed_query_can_be_retried(client, monkeypatch):
    http, searches = client
    def fail(search_id):
        raise QueryFailed("La API agotó el Tiempo de Espera.")
    monkeypatch.setattr(searches, "read_search", fail)
    reply = http.get("/api/searches/abc")
    assert reply.status_code == 502 and reply.json()["retriable"] is True


def test_config_never_publishes_the_token(client):
    http, _ = client
    body = http.get("/api/config").text
    assert "token" not in body.lower()
