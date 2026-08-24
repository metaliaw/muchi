"""Parseo de decklists pegadas a mano.

Acepta los formatos que la gente realmente usa:
    4 Lightning Bolt
    4x Lightning Bolt
    1 Ragavan, Nimble Pilferer (MH2) 138
    Sol Ring
Ignora comentarios, lineas vacias y encabezados tipo "Sideboard".
"""
from __future__ import annotations

import re

from .models import Pedido

_LINEA = re.compile(
    r"""^\s*
        (?:(?P<cant>\d{1,3})\s*[xX]?\s+)?   # cantidad opcional
        (?P<nombre>.+?)                      # nombre
        (?:\s*\([A-Za-z0-9]{2,6}\))?         # (SET) opcional
        # Numero de coleccion opcional. El (?=\d) es imprescindible: sin el,
        # "Sol Ring" se parsea como "Sol" porque "Ring" entra en la clase.
        (?:\s+(?=\d)[0-9A-Za-z\-★]{1,6})?
        \s*$""",
    re.VERBOSE,
)

_ENCABEZADOS = {"deck", "sideboard", "maybeboard", "commander", "companion", "mazo"}


def parse(texto: str) -> tuple[list[Pedido], list[str]]:
    """Devuelve (pedidos, lineas_ignoradas)."""
    pedidos: dict[str, int] = {}
    orden: list[str] = []
    ignoradas: list[str] = []

    for cruda in texto.splitlines():
        linea = cruda.strip()
        if not linea or linea.startswith(("#", "//")):
            continue
        if linea.rstrip(":").lower() in _ENCABEZADOS:
            continue

        m = _LINEA.match(linea)
        if not m:
            ignoradas.append(cruda)
            continue

        nombre = m.group("nombre").strip(" -\t")
        # "Nombre // Otro" (cartas de doble cara): scry indexa por la cara frontal
        nombre = nombre.split("//")[0].strip()
        if not nombre or nombre.isdigit():
            ignoradas.append(cruda)
            continue

        cant = int(m.group("cant") or 1)
        clave = nombre.lower()
        if clave not in pedidos:
            orden.append(nombre)
        pedidos[clave] = pedidos.get(clave, 0) + cant

    return [Pedido(pedidos[n.lower()], n) for n in orden], ignoradas
