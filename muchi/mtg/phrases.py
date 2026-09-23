"""Carga y Valida el Catálogo de Textos de Muchi.

Elegir Frase es Trabajo del Front: el Navegador Sabe qué Clic ocurrió y qué
Tema se Encendió. Acá el Catálogo solo se Lee entero y se Comprueba.
"""
from __future__ import annotations

from dataclasses import dataclass
from functools import lru_cache
from pathlib import Path

import yaml

from muchi.paths import ROOT

# El Contenido de Muchi: Caricias, Saludos, Ayuda y Comentarios de la Luz.
PHRASES_PATH = ROOT / "constants" / "phrases.yaml"
# Estados del Protocolo visual, compartidos por Frases y Burbujas.
STATES = frozenset({"idle", "talk", "happy", "alert", "angry"})


@dataclass(frozen=True)
class Phrase:
    text: str
    state: str


@dataclass(frozen=True)
class PhraseBook:
    every: int
    phrases: tuple[Phrase, ...]
    greetings: tuple[Phrase, ...] = ()
    help_topics: tuple[tuple[str, str], ...] = ()
    dark: tuple[Phrase, ...] = ()
    light: tuple[Phrase, ...] = ()
    nerd: tuple[Phrase, ...] = ()
    libre: tuple[Phrase, ...] = ()
    bargain: tuple[Phrase, ...] = ()


def require_text(value) -> str:
    if not isinstance(value, str) or not value.strip():
        raise ValueError("Los Textos de Muchi deben ser cadenas no vacías.")
    return value


def build_phrase(text, state) -> Phrase:
    if not isinstance(state, str) or state not in STATES:
        raise ValueError("Estado de Frase desconocido.")
    return Phrase(require_text(text), state)


def build_line_phrases(rows) -> tuple[Phrase, ...]:
    return tuple(build_phrase(row["text"], row["state"]) for row in rows)


def build_phrase_book(doc: dict) -> PhraseBook:
    """Valida el Catálogo completo; no introduce Defaults alternativos."""
    expected = {"every", "phrases", "greetings", "help", "dark", "light",
                "nerd", "libre", "bargain"}
    if not isinstance(doc, dict) or set(doc) != expected:
        raise ValueError(
            "El Catálogo requiere every, phrases, greetings, help, dark, "
            "light, nerd, libre y bargain.")
    if type(doc["every"]) is not int or doc["every"] <= 0:
        raise ValueError("every debe ser un entero positivo.")
    for key in ("phrases", "greetings", "help", "dark", "light", "nerd",
                "libre", "bargain"):
        if not isinstance(doc[key], list) or not doc[key]:
            raise ValueError(f"{key} debe ser una Lista no vacía.")

    phrases = tuple(build_phrase(text, group["state"])
                    for group in doc["phrases"]
                    for text in read_group_texts(group))
    greetings = build_line_phrases(doc["greetings"])
    help_topics = tuple((require_text(row["title"]), require_text(row["detail"]))
                        for row in doc["help"])
    dark = build_line_phrases(doc["dark"])
    light = build_line_phrases(doc["light"])
    nerd = build_line_phrases(doc["nerd"])
    libre = build_line_phrases(doc["libre"])
    bargain = build_line_phrases(doc["bargain"])

    return PhraseBook(doc["every"], phrases, greetings, help_topics, dark,
                      light, nerd, libre, bargain)


def read_group_texts(group: dict) -> list[str]:
    texts = group["phrases"]
    if not isinstance(texts, list) or not texts:
        raise ValueError("Cada Grupo requiere una Lista de Frases no vacía.")
    return texts


@lru_cache(maxsize=1)
def read_phrases(path: Path = PHRASES_PATH) -> PhraseBook:
    """Lee una vez el Catálogo. Un Archivo ausente desactiva los Mensajes."""
    if not path.exists():
        return PhraseBook(every=0, phrases=())
    try:
        doc = yaml.safe_load(path.read_text(encoding="utf-8"))
        return build_phrase_book(doc)
    except (KeyError, TypeError, yaml.YAMLError) as error:
        raise ValueError("Estructura de Frases inválida.") from error
