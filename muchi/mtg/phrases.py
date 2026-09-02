"""The phrases Muchi says when you pet her or while she is working.

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
DEFAULT_WAIT_SECONDS = 30


@dataclass(frozen=True)
class Phrase:
    text: str
    state: str


@dataclass(frozen=True)
class PhraseBook:
    every: int
    phrases: tuple[Phrase, ...]


@dataclass(frozen=True)
class WaitingBook:
    every_seconds: int
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


@lru_cache(maxsize=1)
def read_waiting_phrases(path: Path = PHRASES_PATH) -> WaitingBook:
    """Mensajes ordenados para una tarea larga. No se mezclan con caricias."""
    if not path.exists():
        return WaitingBook(every_seconds=0, phrases=())
    doc = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    waiting = doc.get("waiting") or {}
    every = int(waiting.get("every_seconds", DEFAULT_WAIT_SECONDS))
    pool = tuple(
        Phrase(text=str(row.get("text", "")),
               state=str(row.get("state") or DEFAULT_STATE))
        for row in waiting.get("phrases", [])
        if str(row.get("text", "")).strip()
    )
    return WaitingBook(every_seconds=every, phrases=pool)


def waiting_phrase(book: WaitingBook, elapsed_seconds: float) -> Phrase | None:
    """Elige sin azar por bloque de tiempo, para no repetir consecutivamente."""
    if not book.phrases or book.every_seconds <= 0:
        return None
    slot = max(int(elapsed_seconds), 0) // book.every_seconds
    return book.phrases[slot % len(book.phrases)]


def format_elapsed(elapsed_seconds: float) -> str:
    """172.2 -> '2:52'."""
    seconds = max(int(elapsed_seconds), 0)
    minutes, seconds = divmod(seconds, 60)
    return f"{minutes}:{seconds:02d}"


def format_search_progress(processed: int, total: int, current: str,
                           found: int, failed: int, elapsed_seconds: float,
                           done: bool = False) -> str:
    """Texto estable para que una lista larga siempre explique su avance."""
    stats = (f"{found} encontradas · {failed} errores · "
             f"{format_elapsed(elapsed_seconds)}")
    if done:
        return f"Listo: {processed} de {total} procesadas · {stats}"
    return f"Buscando {processed + 1} de {total}: {current} · {stats}"


def speaks_now(clicks: int, every: int) -> bool:
    """True when click N lands exactly on the `every` boundary."""
    return every > 0 and clicks > 0 and clicks % every == 0


def pick_phrase(phrases) -> Phrase | None:
    """A random phrase, or None when there is nothing to draw from."""
    if not phrases:
        return None
    return random.choice(phrases)
