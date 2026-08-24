"""Normalizacion de nombres de carta compartida entre fuentes.

scry.cl y EDHREC arman sus URLs con la misma regla (minusculas, sin acentos,
todo lo que no sea alfanumerico se vuelve guion), asi que vive en un solo lugar.
"""
from __future__ import annotations

import re
import unicodedata


def slug(nombre: str) -> str:
    """'Atraxa, Praetors' Voice' -> 'atraxa-praetors-voice'."""
    s = unicodedata.normalize("NFD", nombre)
    s = "".join(c for c in s if unicodedata.category(c) != "Mn")
    s = re.sub(r"[^a-zA-Z0-9]+", "-", s).strip("-").lower()
    return re.sub(r"-{2,}", "-", s)
