"""Carga el Entorno y las Credenciales del Servicio de Búsqueda."""
from dataclasses import dataclass, field
from functools import lru_cache
import math
import os
from urllib.parse import urlsplit

import yaml

from muchi.paths import ROOT


@dataclass(frozen=True)
class SearchSettings:
    base_url: str
    token: str = field(repr=False)
    timeout_seconds: float
    poll_seconds: float


def read_api_file(name: str) -> dict:
    text = (ROOT / "config" / f"api.{name}.yaml").read_text(encoding="utf-8")
    values = yaml.safe_load(text) if text.strip() else {}
    if not isinstance(values, dict) or set(values) - {"timeout_seconds", "poll_seconds"}:
        raise ValueError("Configuración de API desconocida.")
    for value in values.values():
        if type(value) not in (float, int) or not math.isfinite(value) or value <= 0:
            raise ValueError("Los Tiempos de API deben ser positivos y finitos.")
    return values


@lru_cache(maxsize=1)
def load_api_settings() -> SearchSettings:
    environment = os.getenv("MUCHI_ENV")
    if environment not in {"development", "production"}:
        raise ValueError("Define MUCHI_ENV como development o production.")
    values = read_api_file("defaults")
    values.update(read_api_file(environment))
    for key in ("timeout_seconds", "poll_seconds"):
        value = float(os.getenv(f"MUCHI_API_{key.upper()}", values[key]))
        if not math.isfinite(value) or value <= 0:
            raise ValueError("Los Tiempos de API deben ser positivos y finitos.")
        values[key] = value

    url = os.getenv("MUCHI_API_URL", "").rstrip("/")
    token = os.getenv("MUCHI_API_TOKEN", "").strip()
    parsed = urlsplit(url)
    if parsed.scheme not in {"http", "https"} or not parsed.hostname or parsed.username:
        raise ValueError("Define MUCHI_API_URL como URL HTTP(S) sin Credenciales.")
    if parsed.query or parsed.fragment or not token:
        raise ValueError("La API requiere URL sin Query y MUCHI_API_TOKEN.")
    if not url.endswith("/v1"):
        url += "/v1"
    return SearchSettings(url, token, **values)
