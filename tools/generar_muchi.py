"""Genera el sprite animado de Muchi en pixel art.

Muchi es una gata real: pelo largo blanco crema, nariz rosada, ojos claros y
una cola enorme y esponjosa. El estilo es chibi -- cabeza grande, cuerpo chico,
ojos enormes con destellos -- y contorno violeta oscuro en vez de negro.

Se dibuja a 34x34 con primitivas y se escala con NEAREST: los pixeles quedan
duros y parejos sin tener que tipear un grid a mano.

La animacion es sutil a proposito: respira, mueve la cola y parpadea una vez
por ciclo. Nada de saltos.

    python tools/generar_muchi.py
"""
from __future__ import annotations

import math
from pathlib import Path

from PIL import Image, ImageDraw

SALIDA = Path(__file__).resolve().parent.parent / "assets"
LADO = 34
ESCALA = 12

TRANSP = (0, 0, 0, 0)
CONTORNO = (74, 53, 80, 255)      # violeta oscuro, como la referencia
CREMA = (253, 244, 237, 255)      # pelaje de Muchi
SOMBRA = (233, 220, 226, 255)     # pelo en sombra
BLANCO = (255, 255, 255, 255)     # pechera y patitas
ROSA = (253, 191, 211, 255)       # oreja interna
NARIZ = (224, 114, 155, 255)
OJO = (123, 143, 199, 255)        # periwinkle: ojos claros, como los de ella
OJO_OSC = (74, 68, 110, 255)
PISO = (233, 237, 246, 255)


def _el(d, cx, cy, rx, ry, relleno, borde=None):
    d.ellipse([cx - rx, cy - ry, cx + rx, cy + ry], fill=relleno, outline=borde)


def cuadro(t: float, parpadeo: bool) -> Image.Image:
    img = Image.new("RGBA", (LADO, LADO), TRANSP)
    d = ImageDraw.Draw(img)

    sube = round(math.sin(t * 2 * math.pi) * 0.5)          # respiracion, 1 px
    cola = math.sin(t * 2 * math.pi + 0.8) * 1.6           # la cola va desfasada

    # ---- sombra en el piso ----
    _el(d, 16, 32, 11, 2, PISO)

    # ---- cola: penacho de circulos subiendo por la derecha ----
    # Va bien pegada al borde para que se lea separada del cuerpo.
    puntos = [(26, 28, 4), (30, 25, 4), (31, 20, 4), (30, 15, 3)]
    for i, (tx, ty, r) in enumerate(puntos):
        desp = round(cola * (i + 1) / len(puntos))
        _el(d, tx, ty + desp, r, r, CREMA, CONTORNO)
    for i, (tx, ty, r) in enumerate(puntos):
        desp = round(cola * (i + 1) / len(puntos))
        _el(d, tx, ty + desp, r - 1, r - 1, BLANCO if i >= 2 else CREMA)

    # ---- cuerpo ----
    _el(d, 16, 24 + sube, 9, 8, CREMA, CONTORNO)
    # pechera blanca: el triangulo invertido que tiene casi todo gato asi
    d.polygon([(16, 18 + sube), (11, 26 + sube), (16, 30 + sube), (21, 26 + sube)],
              fill=BLANCO)

    # patitas
    for px in (12, 20):
        _el(d, px, 30 + sube, 3, 2, BLANCO, CONTORNO)

    # ---- orejas: las puntas deben asomar por encima de la cabeza ----
    # La cabeza va centrada en y=13 con ry=9, o sea borde superior en y=4.
    # Si la punta no sube mas que eso, la cabeza se la come.
    # Base ancha y punta baja: con la punta muy alta y la base angosta parecen
    # antenas, no orejas.
    # Lo que se ve de la oreja es solo lo que asoma sobre la linea de la cabeza
    # (y=5). Si a esa altura el triangulo mide 2 px, parece una antena; por eso
    # la base va bien ancha y la punta arranca en el borde del lienzo.
    for lado, cx in ((-1, 8), (1, 24)):
        d.polygon([(cx, 0 + sube), (cx + lado * 5, 10 + sube),
                   (cx - lado * 6, 9 + sube)], fill=CREMA, outline=CONTORNO)
        d.polygon([(cx, 3 + sube), (cx + lado * 3, 9 + sube),
                   (cx - lado * 3, 8 + sube)], fill=ROSA)

    # ---- cabeza ----
    _el(d, 16, 14 + sube, 11, 9, CREMA, CONTORNO)
    _el(d, 16, 13 + sube, 9, 7, BLANCO)          # frente mas clara

    # ---- ojos: enormes, con dos destellos. Es lo que los hace tiernos ----
    for ox in (11, 21):
        if parpadeo:
            d.line([(ox - 3, 14 + sube), (ox + 3, 14 + sube)], fill=CONTORNO, width=1)
        else:
            _el(d, ox, 14 + sube, 3, 4, CONTORNO)
            _el(d, ox, 14 + sube, 2, 3, OJO_OSC)
            _el(d, ox, 15 + sube, 2, 2, OJO)
            d.rectangle([ox - 2, 11 + sube, ox - 1, 12 + sube], fill=BLANCO)
            d.point((ox + 1, 16 + sube), fill=BLANCO)

    # ---- nariz y boca ----
    # Boca chiquita pegada a la nariz: si se estira parece bigotes cruzando la cara.
    d.polygon([(16, 20 + sube), (14, 18 + sube), (18, 18 + sube)], fill=NARIZ)
    d.line([(15, 21 + sube), (14, 22 + sube)], fill=CONTORNO)
    d.line([(17, 21 + sube), (18, 22 + sube)], fill=CONTORNO)

    # motitas en el piso, como la referencia
    d.point((5, 31), fill=SOMBRA)
    d.point((28, 32), fill=SOMBRA)

    return img


def generar(nombre="muchi.gif", cuadros=12, escala=ESCALA) -> Path:
    SALIDA.mkdir(parents=True, exist_ok=True)
    imgs = [cuadro(i / cuadros, parpadeo=(i == cuadros - 3)) for i in range(cuadros)]
    grandes = [im.resize((LADO * escala, LADO * escala), Image.NEAREST) for im in imgs]

    ruta = SALIDA / nombre
    grandes[0].save(ruta, save_all=True, append_images=grandes[1:],
                    duration=110, loop=0, disposal=2, transparency=0)
    grandes[0].save(SALIDA / "muchi-cuadro0.png")
    return ruta


if __name__ == "__main__":
    r = generar()
    print("generado:", r, f"({r.stat().st_size/1024:.0f} KB)")
