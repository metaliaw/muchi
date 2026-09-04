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
