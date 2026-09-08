"""Prueba de integración opt-in contra muchi-api local.

Ejecutar con muchi-api levantada y:
MUCHI_API_INTEGRATION=1 pytest tests/test_muchi_api_integration.py
"""
from __future__ import annotations

import os

import pytest
import requests
from dotenv import load_dotenv


load_dotenv()

pytestmark = pytest.mark.skipif(
    os.getenv("MUCHI_API_INTEGRATION") != "1",
    reason="requiere muchi-api local y MUCHI_API_INTEGRATION=1",
)


@pytest.fixture
def api_config() -> tuple[str, dict[str, str]]:
    url = os.environ["MUCHI_API_URL"].rstrip("/")
    token = os.environ["MUCHI_API_TOKEN"]

    headers = {"Authorization": f"Bearer {token}"}
    return url, headers


def test_local_api_health(api_config: tuple[str, dict[str, str]]) -> None:
    url, _ = api_config
    response = requests.get(f"{url}/v1/health", timeout=3)

    response.raise_for_status()
    assert response.json() == {"status": "ok"}


def test_local_api_auth(api_config: tuple[str, dict[str, str]]) -> None:
    url, headers = api_config
    response = requests.get(
        f"{url}/v1/health/sources",
        headers=headers,
        timeout=3,
    )

    response.raise_for_status()
    assert isinstance(response.json()["sources"], list)


@pytest.mark.skipif(
    os.getenv("MUCHI_API_SEARCH_INTEGRATION") != "1",
    reason="requiere MUCHI_API_SEARCH_INTEGRATION=1; consulta Tiendas reales",
)
def test_front_receives_sol_ring_offers() -> None:
    import time
    from pathlib import Path
    from streamlit.testing.v1 import AppTest
    from muchi.api.settings import load_api_settings
    import streamlit as st

    load_api_settings.cache_clear()
    st.cache_resource.clear()
    app = AppTest.from_file(str(Path(__file__).resolve().parents[1] / "app.py"),
                            default_timeout=30)
    identifier = os.getenv("MUCHI_API_SEARCH_ID")
    if identifier:
        app.query_params["search"] = identifier
    app.run()
    assert not app.exception
    if not identifier:
        app.text_area[0].input("Sol Ring")
        next(button for button in app.button if button.label == "Buscar").click().run()
    deadline = time.monotonic() + 600
    while not app.session_state["terminal_results"] and time.monotonic() < deadline:
        assert not app.exception
        assert not app.error, [error.value for error in app.error]
        time.sleep(5)
        app.run()
    assert not app.exception
    assert app.session_state["terminal_results"], "La Búsqueda no terminó en 10 minutos"
    assert any(item.name.lower() == "sol ring" and item.offers
               for item in app.session_state["search_items"])
    assert any("Carta" in table.value.columns and
               table.value["Carta"].str.lower().eq("sol ring").any()
               for table in app.dataframe)
