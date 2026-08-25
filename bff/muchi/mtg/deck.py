"""Lo que se puede decir de un mazo sin preguntarle a nadie.

Son reglas puras sobre recomendaciones ya traidas: ningun puerto, ninguna red.
Viven aparte porque no dependen de quien las origino.
"""
from __future__ import annotations

from .ports import Recommendation
from .text import normalize_name


def subtract_owned_cards(recs: list[Recommendation],
                         owned: set[str]) -> list[Recommendation]:
    """Saca las que ya estan en tu mazo. Compara normalizado, nunca literal."""
    owned_keys = {normalize_name(n) for n in owned}
    return [r for r in recs if normalize_name(r.name) not in owned_keys]


def list_categories(recs: list[Recommendation]) -> list[str]:
    """Las categorias presentes, en el orden en que llegaron."""
    seen: list[str] = []
    for r in recs:
        if r.category not in seen:
            seen.append(r.category)
    return seen
