"""Arbitraje de la única burbuja de Muchi durante un rerun."""
from __future__ import annotations

from dataclasses import dataclass
from enum import IntEnum
from collections.abc import Callable


class Priority(IntEnum):
    GREETING = 10
    PASSIVE = 20
    CLICK = 30
    HELP = 40
    PROGRESS = 50
    RESULT = 60
    ERROR = 70


@dataclass(frozen=True)
class Message:
    text: str
    state: str
    priority: Priority
    source: str


class MuchiMessenger:
    """Sólo deja pasar un mensaje si no desplaza otro más importante."""

    def __init__(self, render: Callable[[Message], None],
                 clear_render: Callable[[], None] | None = None):
        self._render = render
        self._clear_render = clear_render
        self.current: Message | None = None

    def publish(self, text: str, state: str, priority: Priority,
                source: str) -> bool:
        message = Message(text, state, priority, source)
        if self.current is not None and priority < self.current.priority:
            return False
        self.current = message
        self._render(message)
        return True

    def clear(self, source: str) -> bool:
        if self.current is None or self.current.source != source:
            return False
        self.current = None
        if self._clear_render:
            self._clear_render()
        return True

    def render_default(self) -> None:
        if self.current is None:
            self.publish("Miau, en que te ayudo?", "talk",
                         Priority.GREETING, "greeting")
