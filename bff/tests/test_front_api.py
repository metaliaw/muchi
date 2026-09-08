"""Prueba la Interfaz contra un Servicio simulado, sin Red ni SQLite."""
from decimal import Decimal
from unittest.mock import Mock

import pytest
from streamlit.testing.v1 import AppTest

from muchi.api.settings import load_api_settings
from muchi.mtg import cast
from muchi.mtg.search import SearchState, SearchItem, SearchOffer
from muchi.mtg.ports import QueryFailed, SearchRejected


@pytest.fixture
def front(monkeypatch):
    monkeypatch.setenv("MUCHI_ENV", "development")
    monkeypatch.setenv("MUCHI_API_URL", "http://localhost:8081")
    monkeypatch.setenv("MUCHI_API_TOKEN", "test-code")
    load_api_settings.cache_clear()
    service = Mock()
    service.create_search.return_value = SearchState("search-1", "queued", 1, 0, 0, 0)
    service.read_search.return_value = SearchState("search-1", "completed", 1, 1, 1, 0)
    service.read_results.return_value = (SearchItem("Sol Ring", 2, "found", (
        SearchOffer("Sol Ring", "Store", Decimal("123.99"), "CLP", "https://example.com",
                    "available", False, "source", "nonfoil"),
    )),)
    monkeypatch.setattr(cast, "build_cast", lambda: cast.Cast(service))
    # The real Streamlit resource cache is global across AppTest instances.
    import streamlit as st
    st.cache_resource.clear()
    app = AppTest.from_file("../app.py", default_timeout=10).run()
    yield app, service
    st.cache_resource.clear()
    load_api_settings.cache_clear()


def click_button(app, label):
    next(button for button in app.button if button.label == label).click().run()


def test_front_search_and_prices(front):
    app, service = front
    assert not app.exception
    app.text_area[0].input("2 Sol Ring")
    click_button(app, "Buscar")
    assert not app.exception
    assert app.session_state["search_id"] == "search-1"
    assert service.create_search.call_count == 1
    assert app.session_state["terminal_results"]
    assert app.metric[0].value == "CLP 4,247.98"
    assert app.dataframe[0].value.iloc[0]["Precio"] == "123.99"
    calls = service.read_search.call_count
    app.run()
    assert service.read_search.call_count == calls


def test_front_retry_keeps_request(front):
    app, service = front
    service.create_search.side_effect = QueryFailed("Timeout")
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    pending = app.session_state["pending_search"]
    assert next(button for button in app.button if button.label == "Buscar").disabled
    service.create_search.side_effect = None
    click_button(app, "Reintentar Envío")
    assert service.create_search.call_args.kwargs["key"] == pending["key"]
    assert not app.exception


def test_front_rejection_recovers(front):
    app, service = front
    service.create_search.side_effect = SearchRejected("Invalid request")
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    assert not app.exception
    assert "pending_search" not in app.session_state
    assert not next(button for button in app.button if button.label == "Buscar").disabled


def test_front_cancel_reads_final(front):
    app, service = front
    service.read_search.return_value = SearchState("search-1", "running", 1, 0, 0, 0)
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    before = service.read_results.call_count
    service.cancel_search.return_value = SearchState("search-1", "cancelled", 1, 1, 1, 0)
    service.read_search.return_value = service.cancel_search.return_value
    click_button(app, "Cancelar Búsqueda")
    assert not app.exception
    assert service.read_results.call_count > before
    assert app.session_state["terminal_results"]


def test_front_resume(front):
    app, service = front
    app.text_input[0].input("search-1")
    click_button(app, "Retomar")
    assert not app.exception
    assert app.session_state["search_id"] == "search-1"
    assert service.create_search.call_count == 0


def test_cancel_keeps_partial_on_error(front):
    app, service = front
    service.read_search.return_value = SearchState("search-1", "running", 1, 0, 0, 0)
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    previous = app.session_state["search_items"]
    service.cancel_search.return_value = SearchState("search-1", "cancelled", 1, 1, 1, 0)
    service.read_search.return_value = service.cancel_search.return_value
    service.read_results.side_effect = QueryFailed("Network error")
    click_button(app, "Cancelar Búsqueda")
    assert not app.exception
    assert app.session_state["search_items"] == previous
    assert not app.session_state["terminal_results"]
    service.read_results.side_effect = None
    app.run()
    assert app.session_state["terminal_results"]


def test_previous_search_can_be_opened_without_submission(front):
    app, service = front
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    service.create_search.return_value = SearchState("search-2", "queued", 1, 0, 0, 0)
    service.read_search.return_value = SearchState("search-2", "completed", 1, 1, 1, 0)
    app.text_area[0].input("Lightning Bolt")
    click_button(app, "Buscar")
    assert set(app.session_state["search_history"]) == {"search-1", "search-2"}
    service.read_search.return_value = SearchState("search-1", "completed", 1, 1, 1, 0)
    app.button(key="resume_search-1").click().run()
    assert not app.exception
    assert app.session_state["search_id"] == "search-1"
    assert service.create_search.call_count == 2
    assert app.session_state["search_history"]["search-1"]["label"] == "Sol Ring"


def test_status_survives_results_failure_and_recovers(front):
    app, service = front
    service.read_results.side_effect = QueryFailed("Network error")
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    assert not app.exception
    assert app.session_state["search_state"].done
    assert not app.session_state["terminal_results"]
    service.read_results.side_effect = None
    app.run()
    assert not app.exception
    assert app.session_state["terminal_results"]
    assert app.dataframe[0].value.iloc[0]["Carta"] == "Sol Ring"


def test_default_poll_interval_is_five_seconds(front):
    assert load_api_settings().poll_seconds == 5


def test_unavailable_search_stops_retrying_and_allows_new_search(front):
    app, service = front
    service.read_search.side_effect = SearchRejected("La API rechazó el Pedido: HTTP 404.")
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    assert not app.exception
    assert app.session_state["search_unavailable"]
    assert not next(button for button in app.button if button.label == "Buscar").disabled
    calls = service.read_search.call_count
    app.run()
    assert service.read_search.call_count == calls
    service.read_search.side_effect = None
    click_button(app, "Buscar")
    assert not app.exception
    assert "search_unavailable" not in app.session_state
    assert app.session_state["terminal_results"]
