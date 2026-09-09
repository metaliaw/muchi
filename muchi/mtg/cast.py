"""Conecta la Interfaz con el Servicio de Búsqueda."""
from dataclasses import dataclass

from muchi.api.client import SearchProvider
from muchi.api.settings import load_api_settings
from .search import SearchService
from .sources.scryfall import NameTranslator


@dataclass(frozen=True)
class Cast:
    searches: SearchService
    translator: NameTranslator


def build_cast() -> Cast:
    settings = load_api_settings()
    searches = SearchProvider(
        settings.base_url, settings.token, settings.timeout_seconds,
    )
    return Cast(searches, NameTranslator())
