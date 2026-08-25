"""Parseo de decklists pegadas a mano.

Acepta los formatos que la gente realmente usa:
    4 Lightning Bolt
    4x Lightning Bolt
    1 Ragavan, Nimble Pilferer (MH2) 138
    1 Sol Ring (LTC) 344 *F*
    1 Sol Ring #!Commander
    Sol Ring

Ignora comentarios, lineas vacias y encabezados. Lo que no reconoce lo
devuelve aparte, para que la app lo muestre en vez de tragarselo.

El limite es honesto y conviene tenerlo claro: sin consultar el catalogo nadie
puede saber si "Kaon Ambrosia" es una carta oscura o un dedazo. Aca se rechaza
lo que esta mal *formado* -- caracteres que ningun nombre de carta lleva, URLs,
totales -- y de lo que simplemente no existe se encarga la busqueda, que
informa para cuantas cartas no encontro precio.
"""
from __future__ import annotations

import re

from .models import Order
from .text import normalize_name

# Lo que el copy-paste ensucia: comillas curvas de la web, guiones largos.
_NOISE = {
    "‘": "'", "’": "'",   # comillas simples curvas
    "“": '"', "”": '"',   # comillas dobles curvas
    "–": "-", "—": "-",   # guion corto y raya
}
_SPACES = re.compile(r"[\s ​]+")

_QUANTITY = re.compile(r"^(?P<cant>\d{1,3})\s*[xX]?\s+(?P<resto>.+)$")

# Lo que los exportadores cuelgan al final del nombre.
_DECORATIONS = [
    re.compile(r"\s*\*[A-Za-z]{1,2}\*\s*$"),                        # *F*, *E*
    re.compile(r"\s*#!\S*\s*$"),                                    # #!Commander
    re.compile(r"\s*\[[^\]]*\]\s*$"),                               # [MH2], [Ramp]
    re.compile(r"\s*\([A-Za-z0-9]{2,6}\)\s*[0-9A-Za-z\-★]{0,6}\s*$"),
    re.compile(r"\s+[0-9]{1,6}[a-z]?\s*$"),                         # numero suelto
]

# Todo lo que un nombre de carta real puede llevar. Las comillas son de
# 'Kongming, "Sleeping Dragon"'; el signo mas, de '+2 Mace'; los parentesis,
# de 'B.F.M. (Big Furry Monster)'. Lo que quede fuera no es una carta.
_ALLOWED = re.compile(r"^[\w ,.'\"!?&+:/()-]+$", re.UNICODE)
_LETTER = re.compile(r"[^\W\d_]", re.UNICODE)
_NON_ALNUM = re.compile(r"[^a-z0-9]+")
_MAX_LENGTH = 120

_HEADERS = {
    "deck", "decklist", "mazo", "sideboard", "banquillo", "maybeboard",
    "commander", "comandante", "companion", "tokens", "token", "about",
    "name", "considering", "artifact", "artifacts", "creature", "creatures",
    "enchantment", "enchantments", "instant", "instants", "land", "lands",
    "planeswalker", "planeswalkers", "sorcery", "sorceries", "battle",
    "battles", "total", "totales", "cards", "cartas",
}
# "Creatures (12)", "Tierras (37)": encabezado con su cuenta al lado.
_CATEGORY = re.compile(r"^[^\W\d_][\w ]*\(\d+\)$", re.UNICODE)


def clean_line(raw_line: str) -> str:
    """Normaliza lo que el copy-paste ensucia antes de mirar el contenido."""
    line = raw_line
    for weird, sane in _NOISE.items():
        line = line.replace(weird, sane)

    line = _SPACES.sub(" ", line).strip()
    return line.lstrip("-•* ").strip()


def is_header(line: str) -> bool:
    """Los exportadores separan secciones, y ninguna seccion es una carta."""
    if line.endswith(":") or _CATEGORY.match(line):
        return True

    cleaned = line.rstrip(":").strip().lower()
    return cleaned in _HEADERS or line.split(":")[0].strip().lower() in _HEADERS


def split_quantity(line: str) -> tuple[int, str]:
    """'4x Sol Ring' -> (4, 'Sol Ring'). Sin numero adelante, una copia."""
    m = _QUANTITY.match(line)
    if not m:
        return 1, line
    return int(m.group("cant")), m.group("resto").strip()


def strip_decorations(name: str) -> str:
    """Saca set, numero de coleccion y marcas de foil, en cualquier orden."""
    previous = None
    while previous != name:
        previous = name
        for pattern in _DECORATIONS:
            name = pattern.sub("", name)

    return name.strip(" -")


def pick_front_face(name: str) -> str:
    """'Fire // Ice': scry indexa las de doble cara por la cara de adelante."""
    return name.split("//")[0].strip()


def flatten_name(name: str) -> str:
    """Clave para juntar copias. 'Yawgmoths Will' y "Yawgmoth's Will" son una."""
    return _NON_ALNUM.sub("", normalize_name(name))


def is_card_name(name: str) -> bool:
    """Rechaza lo que esta mal formado, nunca lo que simplemente no conoce."""
    if not name or len(name) > _MAX_LENGTH or "://" in name:
        return False
    if not _LETTER.search(name):
        return False
    return bool(_ALLOWED.match(name))


def parse_decklist(text: str) -> tuple[list[Order], list[str]]:
    """Devuelve (pedidos, lineas_ignoradas).

    Las copias se suman por nombre normalizado, no literal: pegar la lista dos
    veces con distinta puntuacion tiene que dar el mismo pedido.
    """
    orders: dict[str, int] = {}
    names: dict[str, str] = {}
    ignored: list[str] = []

    for raw_line in text.splitlines():
        line = clean_line(raw_line)
        if not line or line.startswith(("#", "//", ";")) or is_header(line):
            continue

        quantity, rest = split_quantity(line)
        name = strip_decorations(rest)
        if not is_card_name(name):
            ignored.append(raw_line.strip())
            continue

        name = pick_front_face(name)
        key = flatten_name(name)
        names.setdefault(key, name)
        orders[key] = orders.get(key, 0) + quantity

    return [Order(orders[c], names[c]) for c in names], ignored
