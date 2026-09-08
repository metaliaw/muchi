"""Carga la Configuración de Stock para un Entorno explícito."""
from __future__ import annotations

import math
import os
from dataclasses import dataclass
from functools import lru_cache

import yaml

from muchi.paths import ROOT


@dataclass(frozen=True)
class StockSettings:
    timeout_seconds: float
    cache_seconds: float


def read_stock_file(name: str) -> dict:
    path = ROOT / "config" / f"stock.{name}.yaml"
    text = path.read_text(encoding="utf-8")
    values = yaml.safe_load(text) if text.strip() else {}
    if not isinstance(values, dict):
        raise ValueError("Stock settings must be an object")
    if set(values) - {"timeout_seconds", "cache_seconds"}:
        raise ValueError("Unknown stock settings; global constants cannot be overridden")
    for value in values.values():
        if type(value) not in (int, float) or not math.isfinite(value) or value <= 0:
            raise ValueError("Stock settings must be positive finite numbers")
    return values


def read_stock_overrides() -> dict[str, float]:
    overrides = {}
    declared = {
        "MUCHI_STOCK_TIMEOUT_SECONDS": "timeout_seconds",
        "MUCHI_STOCK_CACHE_SECONDS": "cache_seconds",
    }
    protected = {"MUCHI_UNVERIFIED_STOCK_SOURCES", "MUCHI_OUT_OF_STOCK_MARKERS"}
    if protected.intersection(os.environ):
        raise ValueError("Global stock constants cannot be overridden")
    for variable, key in declared.items():
        if variable in os.environ:
            value = float(os.environ[variable])
            if not math.isfinite(value) or value <= 0:
                raise ValueError(f"{variable} must be a positive finite number")
            overrides[key] = value
    return overrides


@lru_cache(maxsize=1)
def load_stock_settings() -> StockSettings:
    """Aplica Defaults, Archivo de Entorno y Variables declaradas una vez."""
    environment = os.environ.get("MUCHI_ENV")
    if environment not in {"development", "production"}:
        raise ValueError("Set MUCHI_ENV to development or production")

    values = read_stock_file("defaults")
    values.update(read_stock_file(environment))
    values.update(read_stock_overrides())
    if set(values) != {"timeout_seconds", "cache_seconds"}:
        raise ValueError("Missing required stock settings")

    return StockSettings(**values)


@dataclass(frozen=True)
class OfferSettings:
    price_gap_ratio: float
    minimum_offers: int
    maximum_low_offers: int
    maximum_low_share: float
    stock_check_limit: int


def validate_offer_values(values: dict) -> dict:
    """Valida los Umbrales parciales antes de combinarlos."""
    fields = OfferSettings.__dataclass_fields__
    if not isinstance(values, dict) or set(values) - fields.keys():
        raise ValueError("Unknown offer settings")
    integers = {"minimum_offers", "maximum_low_offers", "stock_check_limit"}
    for key, value in values.items():
        if type(value) not in (int, float) or not math.isfinite(value) or value <= 0:
            raise ValueError(f"{key} must be a positive finite number")
        if key in integers and type(value) is not int:
            raise ValueError(f"{key} must be an integer")
        if key == "maximum_low_share" and value >= 1:
            raise ValueError("maximum_low_share must be less than one")
        if key == "price_gap_ratio" and value <= 1:
            raise ValueError("price_gap_ratio must exceed one")
    return values


def read_offer_file(name: str) -> dict:
    path = ROOT / "config" / f"offers.{name}.yaml"
    text = path.read_text(encoding="utf-8")
    values = yaml.safe_load(text) if text.strip() else {}
    return validate_offer_values(values)


def read_offer_overrides() -> dict:
    """Lee sólo las Variables declaradas por los Campos de Ofertas."""
    overrides = {}
    integers = {"minimum_offers", "maximum_low_offers", "stock_check_limit"}
    for key in OfferSettings.__dataclass_fields__:
        variable = f"MUCHI_OFFERS_{key.upper()}"
        if variable in os.environ:
            parse = int if key in integers else float
            overrides[key] = parse(os.environ[variable])
    return validate_offer_values(overrides)


@lru_cache(maxsize=1)
def load_offer_settings() -> OfferSettings:
    """Carga Defaults, Entorno y Variables una vez al Arrancar."""
    environment = os.environ.get("MUCHI_ENV")
    if environment not in {"development", "production"}:
        raise ValueError("Set MUCHI_ENV to development or production")

    values = read_offer_file("defaults")
    values.update(read_offer_file(environment))
    values.update(read_offer_overrides())
    if set(values) != OfferSettings.__dataclass_fields__.keys():
        raise ValueError("Missing required offer settings")

    return OfferSettings(**values)


@dataclass(frozen=True)
class RateSettings:
    """El Muchi Dólar: cuántos Pesos cobra Muchi por un Dólar.

    Algunas Tiendas publican su propia Tasa y la API ya convierte con ella.
    Cuando una Oferta llega en Dólares sin Referencia, Muchi usa este Valor.

    No sigue al Dólar del Mercado: cubre el Costo de traer la Carta y el
    Margen de los Intermediarios que harán la Compra. Por eso vive en
    `config/rates.defaults.yaml`, a la vista de cualquiera, y se muestra en la
    Interfaz junto a los Precios que convirtió.
    """
    muchi_dolar: int


def validate_rate_values(values: dict) -> dict:
    if not isinstance(values, dict):
        raise ValueError("Rate settings must be an object")
    if set(values) - RateSettings.__dataclass_fields__.keys():
        raise ValueError("Unknown rate settings")
    for value in values.values():
        if type(value) is not int or value <= 0:
            raise ValueError("The Muchi Dólar must be a positive whole number of pesos")
    return values


def read_rate_file(name: str) -> dict:
    path = ROOT / "config" / f"rates.{name}.yaml"
    text = path.read_text(encoding="utf-8")
    return validate_rate_values(yaml.safe_load(text) if text.strip() else {})


@lru_cache(maxsize=1)
def load_rate_settings() -> RateSettings:
    """Carga el Muchi Dólar una vez al Arrancar."""
    environment = os.environ.get("MUCHI_ENV")
    if environment not in {"development", "production"}:
        raise ValueError("Set MUCHI_ENV to development or production")

    values = read_rate_file("defaults")
    values.update(read_rate_file(environment))
    if variable := os.environ.get("MUCHI_RATES_MUCHI_DOLAR"):
        values.update(validate_rate_values({"muchi_dolar": int(variable)}))
    if set(values) != RateSettings.__dataclass_fields__.keys():
        raise ValueError("Missing required rate settings")
    return RateSettings(**values)
