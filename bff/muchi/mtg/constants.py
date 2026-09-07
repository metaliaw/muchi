"""Carga el Catálogo de Stock, separado de la Configuración."""
from __future__ import annotations

from dataclasses import dataclass

import yaml

from muchi.paths import ROOT


@dataclass(frozen=True)
class StockCatalog:
    unverified_sources: frozenset[str]
    out_of_stock_markers: tuple[str, ...]


def read_stock_catalog() -> StockCatalog:
    """Valida las Fuentes y Señales compartidas al Arrancar."""
    values = yaml.safe_load((ROOT / "constants/stock.yaml").read_text(encoding="utf-8"))
    expected = {"unverified_sources", "out_of_stock_markers"}
    if not isinstance(values, dict) or set(values) != expected:
        raise ValueError("Invalid stock catalog keys")
    for entries in values.values():
        if not isinstance(entries, list) or not entries:
            raise ValueError("Stock catalog entries must be nonempty lists")
        if any(not isinstance(item, str) or not item.strip() for item in entries):
            raise ValueError("Stock catalog entries must be nonempty strings")

    return StockCatalog(
        frozenset(values["unverified_sources"]),
        tuple(values["out_of_stock_markers"]),
    )


STOCK_CATALOG = read_stock_catalog()
