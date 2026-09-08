"""Deduce el Tratamiento de una Oferta: Acabado, Idioma y Estado.

La API entrega esos Campos sueltos, pero casi siempre vienen nulos: cada
Tienda escribe el Tratamiento en el Texto de su Variante o de su Título.
Aquí se lee primero el Campo, después la Variante y al final el Título,
que es el más ruidoso porque arrastra el Nombre de la Carta y su Edición.
"""
from __future__ import annotations

import re
import unicodedata

SEPARATOR = " · "

# Cada Grupo entrega una sola Etiqueta. Las Frases se buscan dentro del Texto.
# Los Códigos solo calzan como Palabra entera y en Mayúsculas: "de" y "en" son
# Palabras corrientes en Español, y "Anillo de sol" no viene en Alemán.
# El primer Calce gana, así "no foil" se resuelve antes que "foil".
FINISHES = (
    ("Etched", ("etched",), ()),
    ("No Foil", ("no foil", "nonfoil", "non foil", "sin foil", "normal"), ()),
    ("Foil", ("foil",), ()),
)
LANGUAGES = (
    ("Inglés", ("ingles", "english"), ("en", "eng", "ing")),
    ("Español", ("espanol", "spanish", "castellano"), ("es", "esp")),
    ("Japonés", ("japones", "japanese"), ("jp", "ja")),
    ("Portugués", ("portugues", "portuguese"), ("pt",)),
    ("Alemán", ("aleman", "german"), ("de",)),
    ("Francés", ("frances", "french"), ("fr",)),
    ("Italiano", ("italiano", "italian"), ("it",)),
)
CONDITIONS = (
    ("NM", ("near mint",), ("nm",)),
    ("SP", ("slightly played",), ("sp",)),
    ("LP", ("lightly played",), ("lp",)),
    ("MP", ("moderately played",), ("mp",)),
    ("HP", ("heavily played",), ("hp",)),
    ("DMG", ("damaged",), ("dmg",)),
)
GROUPS = (FINISHES, LANGUAGES, CONDITIONS)


def flatten_text(text: str) -> str:
    """Baja a minúsculas y saca Tildes para que 'Inglés' calce con 'ingles'."""
    plain = unicodedata.normalize("NFKD", text.lower())
    return "".join(letter for letter in plain if not unicodedata.combining(letter))


def read_codes(text: str) -> set[str]:
    """Junta las Palabras escritas enteramente en Mayúsculas, ya sin Tildes."""
    words = re.findall(r"[^\W\d_]+", text, flags=re.UNICODE)
    return {flatten_text(word) for word in words if word.isupper()}


def read_tag(text: str, group) -> str:
    flat = flatten_text(text)
    codes = read_codes(text)
    for tag, phrases, group_codes in group:
        if any(phrase in flat for phrase in phrases) or codes & set(group_codes):
            return tag
    return ""


def build_treatment(offer) -> str:
    """Arma las Etiquetas del Tratamiento, de la Fuente más fiel a la menos."""
    # Los Campos del Contrato son canónicos: se suben a Mayúsculas para que
    # el "en" de la API cuente como Código y no como Palabra suelta.
    fields = (offer.finish, offer.language, offer.condition)
    sources = (" ".join(part for part in fields if part).upper(),
               offer.variant, offer.title)
    tags = []
    for group in GROUPS:
        for text in sources:
            if text and (tag := read_tag(text, group)):
                tags.append(tag)
                break
    return SEPARATOR.join(tags)
