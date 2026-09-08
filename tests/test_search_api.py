"""Contrato HTTP, Autenticación y Estados de la API, sin Red."""
from decimal import Decimal
from unittest.mock import Mock

import pytest
import requests

from muchi.api.client import SearchProvider
from muchi.api.settings import load_api_settings
from muchi.mtg.models import Order
from muchi.mtg.ports import QueryFailed


def search_reply(status="running"):
    return dict(id="search-1", status=status, total=1, processed=0, found=0, errors=0)


def offer_reply():
    return dict(card_name="Sol Ring", store="Store", price_amount="1.69",
                price_currency="USD", url="https://example.com/card", source="scryfall",
                stock_status="unknown", suspicious=True, suspicious_reason="gap")


def result_reply():
    return {"items": [dict(position=0, original_name="Sol Ring", quantity=2,
                           status="found", offers=[offer_reply()])]}


def make_provider(reply, status=200):
    response = Mock(status_code=status)
    response.json.return_value = reply
    response.__enter__ = Mock(return_value=response)
    response.__exit__ = Mock(return_value=False)
    session = Mock()
    session.request.return_value = response
    return SearchProvider("https://example.com/v1", "private-code", session=session)


def test_creates_authenticated_search():
    provider = make_provider(search_reply(), 202)
    state = provider.create_search([Order(2, "Sol Ring")], True, False, "stable-key")
    assert state.id == "search-1"
    args, kwargs = provider.session.request.call_args
    assert args == ("POST", "https://example.com/v1/searches")
    assert kwargs["headers"]["Authorization"] == "Bearer private-code"
    assert kwargs["headers"]["Idempotency-Key"] == "stable-key"
    assert kwargs["json"] == {"cards": [{"name": "Sol Ring", "quantity": 2}],
                              "options": {"verify_stock": True, "stores_only": False}}
    assert kwargs["allow_redirects"] is False
    assert "private-code" not in repr(provider)


def test_retry_keeps_idempotency():
    provider = make_provider(search_reply(), 202)
    provider.session.request.side_effect = requests.Timeout("private-code")
    with pytest.raises(QueryFailed) as caught:
        provider.create_search([Order(1, "Card")], True, True, "same-key")
    assert "private-code" not in str(caught.value)
    provider.session.request.side_effect = None
    provider.create_search([Order(1, "Card")], True, True, "same-key")
    assert [call.kwargs["headers"]["Idempotency-Key"]
            for call in provider.session.request.call_args_list] == ["same-key", "same-key"]


@pytest.mark.parametrize("status", [401, 403, 404, 409, 429, 500, 302])
def test_maps_http_errors(status):
    provider = make_provider({}, status)
    with pytest.raises(QueryFailed):
        provider.read_search("search-1")


def test_reads_partial_results():
    provider = make_provider(result_reply())
    item, = provider.read_results("search-1")
    assert item.quantity == 2
    assert item.offers[0].amount == Decimal("1.69")
    assert item.offers[0].currency == "USD"
    assert item.offers[0].suspicious
    assert item.offers[0].stock_status == "unknown"


@pytest.mark.parametrize("status", ["completed", "completed_with_errors", "failed", "cancelled"])
def test_terminal_states(status):
    assert make_provider(search_reply(status)).read_search("search-1").done


def test_cancel_authentication():
    provider = make_provider(search_reply("cancelled"))
    assert provider.cancel_search("search/1", "cancel-key").done
    args, kwargs = provider.session.request.call_args
    assert args[1].endswith("/searches/search%2F1/cancel")
    assert kwargs["headers"]["Idempotency-Key"] == "cancel-key"


@pytest.mark.parametrize("orders", [[], [Order(0, "Card")], [Order(100, "Card")],
                                     [Order(1, " ")], [Order(1, "Card")] * 501])
def test_rejects_invalid_search(orders):
    provider = make_provider({})
    with pytest.raises(QueryFailed):
        provider.create_search(orders, True, True, "key")
    provider.session.request.assert_not_called()


def test_rejects_bad_results():
    with pytest.raises(QueryFailed):
        make_provider({"items": [{}]}).read_results("search-1")


def test_reads_offers_and_health():
    provider = make_provider({"offers": [offer_reply()]})
    assert provider.find_offers("Sol Ring")[0].amount == Decimal("1.69")
    provider = make_provider({"sources": []})
    assert provider.read_sources() == []


@pytest.mark.parametrize("suffix", ["", "/", "/v1", "/v1/"])
def test_normalizes_api_url(monkeypatch, suffix):
    monkeypatch.setenv("MUCHI_ENV", "development")
    monkeypatch.setenv("MUCHI_API_URL", "https://example.com" + suffix)
    monkeypatch.setenv("MUCHI_API_TOKEN", "private-code")
    load_api_settings.cache_clear()
    try:
        settings = load_api_settings()
        assert settings.base_url == "https://example.com/v1"
        assert "private-code" not in repr(settings)
    finally:
        load_api_settings.cache_clear()


@pytest.mark.parametrize("failure, message", [
    (requests.ConnectionError("private-code"), "No se pudo conectar"),
    (requests.Timeout("private-code"), "Tiempo de Espera"),
])
def test_connection_failure_explains_recovery_without_credentials(failure, message):
    provider = make_provider(search_reply())
    provider.session.request.side_effect = failure
    with pytest.raises(QueryFailed, match=message) as caught:
        provider.read_search("search-1")
    assert "private-code" not in str(caught.value)
