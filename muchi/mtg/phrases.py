"""Las frases que Muchi dice cuando la acarician.

Viven en un YAML, agrupadas por caracterizacion: cada grupo trae un `estado`
del sprite (idle, talk, happy, alert, angry) -- la cara con la que Muchi la
dice -- y la lista de frases de esa caracterizacion. El `cada` de arriba es
cada cuantos clics habla.

El YAML se lee una vez por proceso; para ver cambios hay que reiniciar la app.
"""
from __future__ import annotations

import random
from dataclasses import dataclass
from functools import lru_cache
from pathlib import Path

import yaml

from muchi.paths import ROOT

PHRASES_PATH = ROOT / "muchi-frases.yaml"
DEFAULT_EVERY = 10
DEFAULT_STATE = "talk"


@dataclass(frozen=True)
class Phrase:
    text: str
    state: str


@dataclass(frozen=True)
class PhraseBook:
    every: int
    phrases: tuple[Phrase, ...]


@lru_cache(maxsize=1)
def read_phrases(path: Path = PHRASES_PATH) -> PhraseBook:
    """Aplana el YAML en (cada, frases). Sin archivo no hay nada que decir."""
    if not path.exists():
        return PhraseBook(every=0, phrases=())
    doc = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    every = int(doc.get("cada", DEFAULT_EVERY))
    phrases = []
    for group in doc.get("frases", []):
        state = group.get("estado") or group.get("caracterizacion") or DEFAULT_STATE
        for text in group.get("frases", []):
            phrases.append(Phrase(text=str(text), state=str(state)))
    return PhraseBook(every=every, phrases=tuple(phrases))


def speaks_now(clicks: int, every: int) -> bool:
    """True cuando el clic N cae justo en la raya de `cada`."""
    return every > 0 and clicks > 0 and clicks % every == 0


def pick_phrase(phrases) -> Phrase | None:
    """Una frase al azar, o None si no hay de donde sacar."""
    if not phrases:
        return None
    return random.choice(phrases)
