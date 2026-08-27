"""Muchi en la barra lateral: su globo y los corazoncitos.

Los corazones se dibujan con CSS puro. Streamlit borra las etiquetas <script>
de st.markdown, asi que una animacion por JS no correria; una animacion por
keyframes se reproduce sola cada vez que la pagina se vuelve a renderizar, que
es justo lo que pasa al apretar el boton.
"""
from __future__ import annotations

import random


# Cosas que Muchi puede decir al saludar. Todas apuntan a algo que la app hace.
GREETINGS = [
    "Miau, en que te ayudo?",
    "Nya~ que carta andas buscando?",
    "Aca estoy! Que necesitas?",
    "Miau miau, armamos un mazo?",
]

HELP_TOPICS = [
    ("Buscar una carta", "Escribe el nombre en la pestana Buscar y te muestro todas las tiendas ordenadas por precio."),
    ("Cotizar un mazo entero", "Pega la lista en Mi lista y despues anda al Carrito: reparto la compra entre tiendas mirando tambien los envios."),
    ("Que le falta a mi mazo", "En Comandante pongo lo que juega la gente con ese comandante y descuento lo que ya tienes."),
    ("De donde salen los precios", "De scry.cl (30 tiendas) mas las que indexo directo. Mira la pestana Tiendas."),
]


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


def build_bubble_html(text: str) -> str:
    return f'<div class="mu-globo">{text}</div>'
