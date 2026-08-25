"""Donde esta cada cosa, calculado una sola vez.

Antes cada modulo contaba niveles por su cuenta con Path(__file__).parents[n],
y mover el paquete los desalineaba a todos en silencio: una ruta que sube un
nivel de mas no explota, simplemente apunta a otro lado.

Aca la cuenta se hace una vez. Si el paquete se mueve, se corrige aqui y nada
mas. Los tests no importan esto a proposito: calculan la raiz por su cuenta y
comparan, asi un error en esta cuenta no se vuelve invisible.

Quien lo use debe importarlo absoluto -- `from muchi.paths import ROOT`, nunca
`from ..paths` -- porque la cantidad de puntos tambien depende de donde este el
modulo, que es justo el problema que este archivo viene a sacar del medio.
"""
from __future__ import annotations

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ASSETS = ROOT / "assets"
DATA = ROOT / "data"
