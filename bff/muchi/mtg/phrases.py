"""Carga los Textos de Muchi y Presenta su Burbuja y sus Corazones."""
from __future__ import annotations

import random
from dataclasses import dataclass
from functools import lru_cache
from pathlib import Path

import yaml

from muchi.paths import ROOT

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


def require_text(value) -> str:
    if not isinstance(value, str) or not value.strip():
        raise ValueError("Los Textos de Muchi deben ser cadenas no vacías.")
    return value


def build_phrase(text, state) -> Phrase:
    if not isinstance(state, str) or state not in STATES:
        raise ValueError("Estado de Frase desconocido.")
    return Phrase(require_text(text), state)


def build_phrase_book(doc: dict) -> PhraseBook:
    """Valida el Catálogo completo; no introduce Defaults alternativos."""
    expected = {"every", "phrases", "greetings", "help"}
    if not isinstance(doc, dict) or set(doc) != expected:
        raise ValueError("El Catálogo requiere every, phrases, greetings y help.")
    if type(doc["every"]) is not int or doc["every"] <= 0:
        raise ValueError("every debe ser un entero positivo.")
    for key in ("phrases", "greetings", "help"):
        if not isinstance(doc[key], list) or not doc[key]:
            raise ValueError(f"{key} debe ser una Lista no vacía.")

    phrases = tuple(build_phrase(text, group["state"])
                    for group in doc["phrases"]
                    for text in read_group_texts(group))
    greetings = tuple(build_phrase(row["text"], row["state"])
                      for row in doc["greetings"])
    help_topics = tuple((require_text(row["title"]), require_text(row["detail"]))
                        for row in doc["help"])

    return PhraseBook(doc["every"], phrases, greetings, help_topics)


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
    doc = yaml.safe_load(path.read_text(encoding="utf-8"))
    try:
        return build_phrase_book(doc)
    except (KeyError, TypeError) as error:
        raise ValueError("Estructura de Frases inválida.") from error


def select_greeting(book: PhraseBook, index: int = 0) -> Phrase | None:
    return book.greetings[index % len(book.greetings)] if book.greetings else None


def speaks_now(clicks: int, every: int) -> bool:
    return every > 0 and clicks > 0 and clicks % every == 0


def pick_phrase(phrases) -> Phrase | None:
    return random.choice(phrases) if phrases else None


def build_hearts_html(quantity: int = 9, seed: int | None = None) -> str:
    """Corazoncitos subiendo, cada uno con su desfase para que no vayan en fila."""
    rnd = random.Random(seed)
    pieces = []
    for _ in range(quantity):
        left = rnd.randint(4, 88)
        delay = rnd.uniform(0, 0.7)
        scale = rnd.uniform(0.75, 1.35)
        emoji = rnd.choice(["\U0001F49D", "\U0001F495", "\U0001F49E", "\U0001F338"])
        pieces.append(
            f'<span class="mu-corazon" style="left:{left}%;'
            f'animation-delay:{delay:.2f}s;font-size:{scale:.2f}rem">{emoji}</span>'
        )
    return '<div class="mu-corazones">' + "".join(pieces) + "</div>"

def build_bubble_html(text: str, state: str = "talk") -> str:
    """La única burbuja del Muchi lateral, con tono según el mensaje."""
    return f'<div class="mu-globo mu-globo--{state}">{text}</div>'
