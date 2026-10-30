"""Dónde Queda cada Tienda: un Archivo público, leído una sola vez.

Ninguna Fuente Publica Coordenadas, así que se Mantienen a mano en
`config/stores.yaml`, igual que las Redes en `config/socials.yaml`. El BFF
solo las Adjunta a cada Oferta; Ordenar por Cercanía lo Decide el Front, que
es el único que Sabe dónde Está quien Compra.
"""
from __future__ import annotations

from functools import lru_cache

import yaml

from muchi.paths import ROOT

STORES_FILE = ROOT / "config" / "stores.yaml"
FIELDS = {"name", "lat", "lng", "online", "source"}
# Cómo se Obtuvo cada Coordenada, para Afinar después las aproximadas: el
# Centro de la Comuna, una Dirección geocodificada o el Mapa de la Tienda.
SOURCES = {"comuna", "address", "maps"}


def fold_store(name: str) -> str:
    """La Fuente Escribe `Bazar del  León`; el Archivo, `Bazar del León`."""
    return " ".join(str(name).split()).casefold()


def read_point(row: dict) -> dict | None:
    """Una Tienda solo online o sin Coordenada no tiene Lugar."""
    if row.get("online") is True or row.get("lat") is None or row.get("lng") is None:
        return None
    lat, lng = float(row["lat"]), float(row["lng"])
    if not (-90 <= lat <= 90 and -180 <= lng <= 180):
        raise ValueError(f"{row['name']} has coordinates out of range")
    return {"lat": lat, "lng": lng}


def validate_stores(value) -> dict[str, dict | None]:
    """Un Archivo malo Detiene el Arranque; no Sirve medio Mapa."""
    if not isinstance(value, dict) or not isinstance(value.get("stores"), list):
        raise ValueError("stores.yaml must declare a list of stores")
    places = {}
    for row in value["stores"]:
        if not isinstance(row, dict) or set(row) - FIELDS:
            raise ValueError("Unknown store fields")
        name = str(row.get("name") or "").strip()
        if not name:
            raise ValueError("A store needs a name")
        if not isinstance(row.get("online", False), bool):
            raise ValueError(f"{name} online must be true or false")
        point = read_point(row)
        source = row.get("source")
        placed = row.get("lat") is not None and row.get("lng") is not None
        if placed and source not in SOURCES:
            raise ValueError(f"{name} must say where its coordinates came from")
        if not placed and source is not None:
            raise ValueError(f"{name} has a source but no coordinates")
        places[fold_store(name)] = point
    return places


@lru_cache(maxsize=1)
def load_places() -> dict[str, dict | None]:
    text = STORES_FILE.read_text(encoding="utf-8")
    return validate_stores(yaml.safe_load(text) or {})


def locate_store(name: str) -> dict | None:
    """El Lugar de la Tienda, o nulo cuando no se Sabe o no lo Tiene."""
    point = load_places().get(fold_store(name))
    return None if point is None else dict(point)
