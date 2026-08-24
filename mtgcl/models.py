"""Tipos compartidos. Una Offer es 'esta copia concreta, en esta tienda, a este precio'."""
from __future__ import annotations

import re
from dataclasses import dataclass


@dataclass(frozen=True)
class Offer:
    store: str
    card_name: str
    title: str          # descripcion completa: set, condicion, foil
    price_clp: int
    url: str
    finish: str = ""    # "Foil" | "Normal"
    condition: str = ""  # NM, SP, ...
    language: str = ""
    stock: int | None = None
    source: str = "scry"
    key: str = ""       # id de variante, para deduplicar
    # True = vendedor particular del marketplace de scry; False = tienda con sitio
    # propio. Lo determina la fuente al parsear (ver sources/scry.py).
    marketplace: bool = False

    @property
    def es_foil(self) -> bool:
        return "foil" in f"{self.finish} {self.title}".lower()

    @property
    def edicion(self) -> str:
        m = re.search(r"\[([^\]]+)\]", self.title)
        return m.group(1).strip() if m else ""


@dataclass(frozen=True)
class Pedido:
    """Una linea de la decklist: cuantas copias de que carta."""
    cantidad: int
    nombre: str
