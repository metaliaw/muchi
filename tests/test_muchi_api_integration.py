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
def test_the_front_receives_sol_ring_offers() -> None:
    """Sol Ring de punta a punta: el BFF crea la Búsqueda y espera sus Ofertas.

    El Front dibuja y no decide, así que se prueba por donde el Navegador
    pregunta: las Rutas del BFF, con la API y su Worker de verdad al otro lado.
    """
    import time

    from fastapi.testclient import TestClient

    from muchi.api.settings import load_api_settings
    from server import main

    load_api_settings.cache_clear()
    client = TestClient(main.app)

    identifier = os.getenv("MUCHI_API_SEARCH_ID")
    if not identifier:
        created = client.post("/api/searches",
                              json={"text": "Sol Ring", "key": "integration"})
        assert created.status_code == 200, created.text
        identifier = created.json()["state"]["id"]

    deadline = time.monotonic() + 600
    body = {}
    while time.monotonic() < deadline:
        reply = client.get(f"/api/searches/{identifier}")
        assert reply.status_code == 200, reply.text
        body = reply.json()
        if body["state"]["done"]:
            break
        time.sleep(5)

    assert body.get("state", {}).get("done"), "La Búsqueda no terminó en 10 minutos"
    assert any(offer["card_name"].lower() == "sol ring" for offer in body["offers"]), \
        body["offers"]
