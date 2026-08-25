"""Genera el sprite animado de Muchi en pixel art.

Muchi es una gata real: pelo largo blanco crema, nariz rosada, ojos claros y
una cola enorme y esponjosa. El estilo es chibi -- cabeza grande, cuerpo chico,
ojos enormes con destellos -- y contorno violeta oscuro en vez de negro.

Se dibuja a 34x34 con primitivas y se escala con NEAREST: los pixeles quedan
duros y parejos sin tener que tipear un grid a mano.

La animacion es sutil a proposito: respira, mueve la cola y parpadea una vez
por ciclo. Nada de saltos.

    python tools/generate_muchi.py
"""
from __future__ import annotations

import math
from pathlib import Path

from PIL import Image, ImageDraw

OUT_DIR = Path(__file__).resolve().parent.parent / "assets"
SIDE = 34
SCALE = 12

TRANSPARENT = (0, 0, 0, 0)
OUTLINE = (74, 53, 80, 255)      # violeta oscuro, como la referencia
CREAM = (253, 244, 237, 255)      # pelaje de Muchi
SHADOW = (233, 220, 226, 255)     # pelo en sombra
WHITE = (255, 255, 255, 255)     # pechera y patitas
PINK = (253, 191, 211, 255)       # oreja interna
NOSE = (224, 114, 155, 255)
EYE = (123, 143, 199, 255)        # periwinkle: ojos claros, como los de ella
EYE_DARK = (74, 68, 110, 255)
FLOOR = (233, 237, 246, 255)


def _ellipse(d, cx, cy, rx, ry, fill_color, outline_color=None):
    d.ellipse([cx - rx, cy - ry, cx + rx, cy + ry], fill=fill_color, outline=outline_color)


def build_frame(t: float, blink: bool) -> Image.Image:
    img = Image.new("RGBA", (SIDE, SIDE), TRANSPARENT)
    d = ImageDraw.Draw(img)

    rise = round(math.sin(t * 2 * math.pi) * 0.5)          # respiracion, 1 px
    tail = math.sin(t * 2 * math.pi + 0.8) * 1.6           # la cola va desfasada

    # ---- sombra en el piso ----
    _ellipse(d, 16, 32, 11, 2, FLOOR)

    # ---- cola: penacho de circulos subiendo por la derecha ----
    # Va bien pegada al borde para que se lea separada del cuerpo.
    points = [(26, 28, 4), (30, 25, 4), (31, 20, 4), (30, 15, 3)]
    for i, (tx, ty, r) in enumerate(points):
        offset = round(tail * (i + 1) / len(points))
        _ellipse(d, tx, ty + offset, r, r, CREAM, OUTLINE)
    for i, (tx, ty, r) in enumerate(points):
        offset = round(tail * (i + 1) / len(points))
        _ellipse(d, tx, ty + offset, r - 1, r - 1, WHITE if i >= 2 else CREAM)

    # ---- cuerpo ----
    _ellipse(d, 16, 24 + rise, 9, 8, CREAM, OUTLINE)
    # pechera blanca: el triangulo invertido que tiene casi todo gato asi
    d.polygon([(16, 18 + rise), (11, 26 + rise), (16, 30 + rise), (21, 26 + rise)],
              fill=WHITE)

    # patitas
    for px in (12, 20):
        _ellipse(d, px, 30 + rise, 3, 2, WHITE, OUTLINE)

    # ---- orejas: las puntas deben asomar por encima de la cabeza ----
    # La cabeza va centrada en y=13 con ry=9, o sea borde superior en y=4.
    # Si la punta no sube mas que eso, la cabeza se la come.
    # Base ancha y punta baja: con la punta muy alta y la base angosta parecen
    # antenas, no orejas.
    # Lo que se ve de la oreja es solo lo que asoma sobre la linea de la cabeza
    # (y=5). Si a esa altura el triangulo mide 2 px, parece una antena; por eso
    # la base va bien ancha y la punta arranca en el borde del lienzo.
    for side, cx in ((-1, 8), (1, 24)):
        d.polygon([(cx, 0 + rise), (cx + side * 5, 10 + rise),
                   (cx - side * 6, 9 + rise)], fill=CREAM, outline=OUTLINE)
        d.polygon([(cx, 3 + rise), (cx + side * 3, 9 + rise),
                   (cx - side * 3, 8 + rise)], fill=PINK)

    # ---- cabeza ----
    _ellipse(d, 16, 14 + rise, 11, 9, CREAM, OUTLINE)
    _ellipse(d, 16, 13 + rise, 9, 7, WHITE)          # frente mas clara

    # ---- ojos: enormes, con dos destellos. Es lo que los hace tiernos ----
    for ox in (11, 21):
        if blink:
            d.line([(ox - 3, 14 + rise), (ox + 3, 14 + rise)], fill=OUTLINE, width=1)
        else:
            _ellipse(d, ox, 14 + rise, 3, 4, OUTLINE)
            _ellipse(d, ox, 14 + rise, 2, 3, EYE_DARK)
            _ellipse(d, ox, 15 + rise, 2, 2, EYE)
            d.rectangle([ox - 2, 11 + rise, ox - 1, 12 + rise], fill=WHITE)
            d.point((ox + 1, 16 + rise), fill=WHITE)

    # ---- nariz y boca ----
    # Boca chiquita pegada a la nariz: si se estira parece bigotes cruzando la cara.
    d.polygon([(16, 20 + rise), (14, 18 + rise), (18, 18 + rise)], fill=NOSE)
    d.line([(15, 21 + rise), (14, 22 + rise)], fill=OUTLINE)
    d.line([(17, 21 + rise), (18, 22 + rise)], fill=OUTLINE)

    # motitas en el piso, como la referencia
    d.point((5, 31), fill=SHADOW)
    d.point((28, 32), fill=SHADOW)

    return img


def generate_gif(name="muchi.gif", frames=12, scale=SCALE) -> Path:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    imgs = [build_frame(i / frames, blink=(i == frames - 3)) for i in range(frames)]
    scaled = [im.resize((SIDE * scale, SIDE * scale), Image.NEAREST) for im in imgs]

    path = OUT_DIR / name
    scaled[0].save(path, save_all=True, append_images=scaled[1:],
                   duration=110, loop=0, disposal=2, transparency=0)
    scaled[0].save(OUT_DIR / "muchi-cuadro0.png")
    return path


if __name__ == "__main__":
    r = generate_gif()
    print("generado:", r, f"({r.stat().st_size/1024:.0f} KB)")
