"""The phrases Muchi says when you pet her.

They live in a YAML, grouped by mood: each group carries a sprite `state`
(idle, talk, happy, alert, angry) -- the face Muchi wears when she says it --
and the list of phrases for that mood. The top-level `every` is how many
clicks between phrases.

The YAML is read once per process; to see changes, restart the app.
"""
from __future__ import annotations

import random
from dataclasses import dataclass
from functools import lru_cache
from pathlib import Path

import yaml

from muchi.paths import ROOT

PHRASES_PATH = ROOT / "muchi-phrases.yaml"
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
    """Flatten the YAML into (every, phrases). No file means nothing to say."""
    if not path.exists():
        return PhraseBook(every=0, phrases=())
    doc = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    every = int(doc.get("every", DEFAULT_EVERY))
    phrases = []
    for group in doc.get("phrases", []):
        state = group.get("state") or group.get("mood") or DEFAULT_STATE
        for text in group.get("phrases", []):
            phrases.append(Phrase(text=str(text), state=str(state)))
    return PhraseBook(every=every, phrases=tuple(phrases))


def speaks_now(clicks: int, every: int) -> bool:
    """True when click N lands exactly on the `every` boundary."""
    return every > 0 and clicks > 0 and clicks % every == 0


def pick_phrase(phrases) -> Phrase | None:
    """A random phrase, or None when there is nothing to draw from."""
    if not phrases:
        return None
    return random.choice(phrases)
