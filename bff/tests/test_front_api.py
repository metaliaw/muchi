"""Prueba la Interfaz contra un Servicio simulado, sin Red ni SQLite."""
import re
from decimal import Decimal
from unittest.mock import Mock

import pytest
from streamlit.testing.v1 import AppTest

from muchi.api.settings import load_api_settings
from muchi.mtg import cast
from muchi.mtg.search import SearchState, SearchItem, SearchOffer
from muchi.mtg.ports import QueryFailed, SearchRejected


@pytest.fixture
def service_offer():
    def build(amount="123.99", currency="CLP", suspicious=False, reason=""):
        return SearchOffer("Sol Ring", "Store", Decimal(amount), currency,
                           "https://example.com", "available", suspicious,
                           "source", "nonfoil", "en", suspicious_reason=reason)
    return build


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
                    "available", False, "source", "nonfoil", "en"),
    )),)
    monkeypatch.setattr(cast, "build_cast", lambda: cast.Cast(service))
    # The real Streamlit resource cache is global across AppTest instances.
    import streamlit as st
    st.cache_resource.clear()
    app = AppTest.from_file("../app.py", default_timeout=10).run()
    yield app, service
    st.cache_resource.clear()
    load_api_settings.cache_clear()


def read_cards(app):
    """Las Tarjetas de Ofertas, tal como Muchi las pinta, en su Orden."""
    return "\n".join(element.value for element in app.markdown)


def read_prices(app):
    """Los Precios de las Tarjetas de Ofertas, de arriba hacia abajo.

    Las Líneas del Carrito también son Tarjetas con Precio, pero se compran,
    no se miran: se reconocen por su Botón y quedan fuera.
    """
    offers = [element.value for element in app.markdown
              if "mu-precio" in element.value and ">Comprar<" not in element.value]
    return re.findall(r'class="mu-precio[^"]*">([^<]+)<', "\n".join(offers))


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
    cards = read_cards(app)
    assert "$124" in cards
    assert "En Stock" in cards and "No Foil" in cards and "Inglés" in cards
    calls = service.read_search.call_count
    app.run()
    assert service.read_search.call_count == calls


def test_suspicious_column_explains_itself_in_spanish(front, service_offer):
    app, service = front
    service.read_results.return_value = (SearchItem("Sol Ring", 1, "found", (
        service_offer(suspicious=True, reason="price_below_30_percent_median"),
        service_offer(amount="5000"),
    )),)
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    assert not app.exception
    cards = read_cards(app)
    assert "Precio bajo el 30% de la Mediana de su Moneda" in cards
    assert "Verificar" in cards


def test_dollar_offers_join_the_cart_at_the_muchi_dolar(front, service_offer):
    app, service = front
    service.read_results.return_value = (SearchItem("Sol Ring", 1, "found", (
        service_offer(amount="4.00", currency="USD"),
    )),)
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    assert not app.exception
    assert app.metric[0].value == "CLP 8,000.00"
    assert any("1 USD = CLP 1,000" in caption.value for caption in app.caption)


def test_unknown_currency_stays_out_of_the_cart(front, service_offer):
    app, service = front
    service.read_results.return_value = (SearchItem("Sol Ring", 1, "found", (
        service_offer(amount="4.00", currency="EUR"),
    )),)
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    assert not app.exception
    assert app.metric[0].value == "CLP 0.00"


def test_table_opens_ordered_by_price_across_every_card(front, service_offer):
    app, service = front
    service.read_results.return_value = (
        SearchItem("Sol Ring", 1, "found", (service_offer(amount="9000"),
                                            service_offer(amount="1500"))),
        SearchItem("Bolt", 1, "found", (service_offer(amount="800"),)),
    )
    app.text_area[0].input("Sol Ring\nBolt")
    click_button(app, "Buscar")
    assert not app.exception
    assert read_prices(app) == ["$800", "$1.500", "$9.000"]


def test_dollars_stay_in_their_own_block_when_ordering(front, service_offer):
    app, service = front
    service.read_results.return_value = (SearchItem("Sol Ring", 1, "found", (
        service_offer(amount="4.00", currency="USD"),
        service_offer(amount="9000"), service_offer(amount="1500"),
    )),)
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    assert not app.exception
    assert read_prices(app) == ["$1.500", "$9.000", "USD 4,00"]


def test_prices_are_written_with_chilean_separators(front, service_offer):
    app, service = front
    service.read_results.return_value = (SearchItem("Sol Ring", 1, "found", (
        service_offer(amount="1234567"), service_offer(amount="1791"),
    )),)
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    assert not app.exception
    assert read_prices(app) == ["$1.791", "$1.234.567"]


def test_cents_bring_back_the_decimal_comma(front, service_offer):
    app, service = front
    service.read_results.return_value = (SearchItem("Sol Ring", 1, "found", (
        service_offer(amount="1500"), service_offer(amount="3.49", currency="USD"),
    )),)
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    assert not app.exception
    # Los Pesos no llevan Centavos; el Dólar sí, con su Coma.
    assert read_prices(app) == ["$1.500", "USD 3,49"]


def test_search_sends_fixed_options_without_asking(front):
    app, service = front
    app.text_area[0].input("Sol Ring")
    click_button(app, "Buscar")
    assert not app.checkbox
    options = service.create_search.call_args.kwargs
    assert options["verify_stock"] and options["stores_only"]


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
    assert "Sol Ring" in read_cards(app)


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


def test_dark_mode_toggle_makes_muchi_talk(front):
    app, _ = front
    assert not app.session_state["muchi_oscuro"]

    app.toggle[0].set_value(True).run()

    assert not app.exception
    assert app.session_state["muchi_oscuro"]
    spoken = "\n".join(block.value for block in app.markdown)
    assert "me pongo darkzz" in spoken or (
        "se apaga la luz , baila como pokemon en cOnVerS3" in spoken)


def test_light_mode_toggle_embarrasses_muchi(front):
    app, _ = front
    app.toggle[0].set_value(True).run()

    app.toggle[0].set_value(False).run()

    assert not app.exception
    assert not app.session_state["muchi_oscuro"]
    spoken = "\n".join(block.value for block in app.markdown)
    assert "oh no prendieron las luces, no me vean estoy gordo" in spoken
