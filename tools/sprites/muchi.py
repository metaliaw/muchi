"""Muchi — generador de sprites pixel-art 32×32 + animaciones."""
from pixlib import Canvas, mirror

W = H = 32

PAL = {
    'K': (58, 26, 54),      # outline oscuro
    'O': (245, 115, 58),    # naranjo principal
    'L': (253, 158, 58),    # naranjo claro / highlight
    'D': (217, 79, 43),     # naranjo oscuro (rayas)
    'W': (253, 248, 238),   # crema
    'S': (232, 222, 208),   # sombra crema
    'P': (235, 169, 166),   # rosa oreja
    'I': (58, 26, 54),      # masa del ojo (se recolorea en la paleta crema)
    'E': (255, 255, 255),   # brillo ojo
    'B': (95, 46, 92),      # sombra suelo
    'R': (231, 76, 76),
    'Y': (255, 226, 120),   # chispas
    'C': (140, 210, 235),   # gotita / azul
}
FILLS = set('OLDWSP')  # cuentan como silueta para el auto-outline


# ---------------------------------------------------------------- silueta ---
HEAD = {
    6:  (13, 18),
    7:  (10, 21),
    8:  (5, 26),
    9:  (5, 26),
    10: (5, 26),
    11: (5, 26),
    12: (5, 26),
    13: (5, 26),
    14: (5, 26),
    15: (6, 25),
    16: (6, 25),
    17: (7, 24),
    18: (9, 22),
}
EAR_L = {2: (7, 8), 3: (6, 9), 4: (6, 10), 5: (5, 11), 6: (5, 12), 7: (5, 12)}
EAR_L_IN = {4: (8, 9), 5: (7, 10), 6: (7, 11)}
BODY = {
    17: (12, 19),
    18: (11, 20),
    19: (10, 21),
    20: (9, 22),
    21: (9, 22),
    22: (8, 23),
    23: (8, 23),
    24: (8, 23),
    25: (8, 23),
    26: (8, 23),
    27: (9, 22),
}
BELLY = {
    19: (13, 18),
    20: (12, 19),
    21: (11, 20),
    22: (11, 20),
    23: (11, 20),
    24: (12, 19),
}
TAIL = {
    16: (28, 30),
    17: (27, 30),
    18: (27, 30),
    19: (27, 30),
    20: (26, 30),
    21: (26, 29),
    22: (25, 29),
    23: (24, 28),
    24: (23, 27),
    25: (22, 26),
    26: (21, 25),
}


def build(head_dy=0, body_dy=0, ear='normal', eyes='open', mouth='closed',
          tail=0, fx=None, squash=0):
    """Arma un frame de Muchi en un Canvas de 32×32.

    Los deltas desplazan la cabeza y el cuerpo (respiracion, salto).
    `tail` acepta -1, 0, 1, 2 para cuatro posiciones de la cola.
    `ear` puede ser 'normal', 'up' o 'down'.
    `fx` es una lista de funciones (cv, hDy, bDy) para efectos extra.
    """
    cv = Canvas(W, H, PAL, FILLS)

    tailSpec = TAIL
    if tail == 1:
        tailSpec = {r: (c0 + 1, c1 + 1) for r, (c0, c1) in TAIL.items()}
    elif tail == 2:
        tailSpec = {r - 1: (c0 + 1, c1 + 1) for r, (c0, c1) in TAIL.items()}
    elif tail == -1:
        tailSpec = {r + 1: (c0 - 1, c1 - 1) for r, (c0, c1) in TAIL.items()}
    cv.rows(tailSpec, 'O', dr=body_dy)

    tip = {}
    for r, (c0, c1) in tailSpec.items():
        if r <= min(tailSpec) + 3:
            tip[r] = (c0 + (1 if r > min(tailSpec) + 2 else 0), c1)
    cv.rows(tip, 'W', dr=body_dy)

    body = BODY
    if squash:
        body = {r + squash: v for r, v in BODY.items() if r + squash < H - 2}
    cv.rows(body, 'O', dr=body_dy)
    cv.rows(BELLY, 'W', dr=body_dy + squash)

    # patitas delanteras
    pd = body_dy + squash
    cv.span(26 + pd, 9, 13, 'W')
    cv.span(26 + pd, 18, 22, 'W')
    cv.span(27 + pd, 10, 13, 'W')
    cv.span(27 + pd, 18, 21, 'W')
    cv.span(25 + pd, 10, 12, 'S')
    cv.span(25 + pd, 19, 21, 'S')

    el, er = EAR_L, mirror(EAR_L, W)
    eli, eri = EAR_L_IN, mirror(EAR_L_IN, W)
    if ear == 'up':
        el = {r - 1: v for r, v in el.items()}
        er = {r - 1: v for r, v in er.items()}
        eli = {r - 1: v for r, v in eli.items()}
        eri = {r - 1: v for r, v in eri.items()}
    elif ear == 'down':
        el = {2: (5, 7), 3: (4, 9), 4: (4, 10), 5: (4, 11), 6: (5, 12), 7: (5, 12)}
        er = mirror(el, W)
        eli = {4: (6, 8), 5: (6, 9), 6: (7, 11)}
        eri = mirror(eli, W)
    cv.rows(el, 'O', dr=head_dy)
    cv.rows(er, 'O', dr=head_dy)
    cv.rows(eli, 'P', dr=head_dy)
    cv.rows(eri, 'P', dr=head_dy)

    cv.rows(HEAD, 'O', dr=head_dy)

    cv.outline()

    # rayas frente
    for c in (13, 15, 17):
        cv.span(7 + head_dy, c, c, 'D')
        cv.span(8 + head_dy, c, c, 'D')
    cv.span(9 + head_dy, 15, 15, 'D')
    # rayas mejillas
    for row in (11, 12):
        cv.px(row + head_dy, 6, 'D')
        cv.px(row + head_dy, 25, 'D')
    cv.px(13 + head_dy, 6, 'D')
    cv.px(13 + head_dy, 25, 'D')
    # brillo superior cabeza
    cv.span(7 + head_dy, 11, 12, 'L')
    cv.span(8 + head_dy, 9, 10, 'L')
    cv.span(7 + head_dy, 19, 20, 'L')

    # hocico crema
    cv.rows({14: (11, 20), 15: (10, 21), 16: (11, 20), 17: (12, 19)}, 'W', dr=head_dy)

    # separacion cola / cuerpo
    for r, c in ((23, 25), (24, 24), (25, 23), (26, 22), (27, 21)):
        cv.px(r + body_dy, c, 'K')
    for r in (26, 27):
        cv.px(r + body_dy + squash, 14, 'K')
        cv.px(r + body_dy + squash, 17, 'K')

    drawEyes(cv, eyes, head_dy)
    drawMouth(cv, mouth, head_dy)

    for f in (fx or ()):
        f(cv, head_dy, body_dy)

    shadowRow = 28 + (squash if squash > 0 else 0) + body_dy
    cv.span(min(shadowRow + 1, H - 1), 10, 21, 'B')
    return cv


def drawEyes(cv, kind, dy):
    """Dibuja los ojos en el Canvas con el estilo indicado."""
    L, R = 8, 19  # col inicio ojo izq / der  (4 de ancho)
    if kind in ('open', 'wide'):
        top, bot = (9, 14) if kind == 'wide' else (10, 14)
        for base in (L, R):
            for r in range(top, bot + 1):
                cv.span(r + dy, base, base + 4, 'I')
            cv.px(top + dy, base, 'O'); cv.px(top + dy, base + 4, 'O')
            cv.px(bot + dy, base, 'O'); cv.px(bot + dy, base + 4, 'O')
            cv.span(top + 1 + dy, base + 1, base + 2, 'E')
            cv.px(top + 2 + dy, base + 1, 'E')
            cv.px(bot - 1 + dy, base + 3, 'E')
    elif kind == 'blink':
        for base in (L, R):
            cv.span(12 + dy, base, base + 4, 'I')
            cv.px(13 + dy, base + 1, 'I'); cv.px(13 + dy, base + 3, 'I')
    elif kind == 'happy':
        for base in (L, R):
            cv.span(13 + dy, base, base + 1, 'K')
            cv.span(11 + dy, base + 2, base + 2, 'K')
            cv.px(12 + dy, base + 1, 'K')
            cv.px(12 + dy, base + 3, 'K')
            cv.span(13 + dy, base + 3, base + 4, 'K')
    elif kind == 'angry':
        for i, base in enumerate((L, R)):
            for r in range(12, 15):
                cv.span(r + dy, base + 1, base + 3, 'I')
            cv.px(12 + dy, base + 2, 'E')
            if i == 0:
                cv.span(10 + dy, base + 2, base + 4, 'K')
                cv.px(11 + dy, base + 3, 'K'); cv.px(11 + dy, base + 4, 'K')
                cv.px(11 + dy, base, 'K')
            else:
                cv.span(10 + dy, base, base + 2, 'K')
                cv.px(11 + dy, base, 'K'); cv.px(11 + dy, base + 1, 'K')
                cv.px(11 + dy, base + 4, 'K')
    elif kind == 'shock':
        for base in (L, R):
            for r in range(9, 15):
                cv.span(r + dy, base, base + 4, 'I')
            cv.px(9 + dy, base, 'O'); cv.px(9 + dy, base + 4, 'O')
            cv.px(14 + dy, base, 'O'); cv.px(14 + dy, base + 4, 'O')
            cv.span(11 + dy, base + 1, base + 3, 'E')
            cv.span(12 + dy, base + 1, base + 3, 'E')
            cv.px(11 + dy, base + 2, 'K'); cv.px(12 + dy, base + 2, 'K')


def drawMouth(cv, kind, dy):
    """Dibuja la nariz y la boca en el Canvas."""
    cv.span(14 + dy, 15, 16, 'K')
    cv.px(15 + dy, 15, 'K'); cv.px(15 + dy, 16, 'K')
    if kind == 'closed':
        cv.px(16 + dy, 14, 'K'); cv.px(16 + dy, 17, 'K')
        cv.px(17 + dy, 15, 'K'); cv.px(17 + dy, 16, 'K')
    elif kind == 'small':
        cv.span(16 + dy, 15, 16, 'K')
        cv.px(17 + dy, 15, 'K'); cv.px(17 + dy, 16, 'K')
    elif kind == 'open':
        for r in range(16, 19):
            cv.span(r + dy, 14, 17, 'K')
        cv.px(18 + dy, 14, 'W'); cv.px(18 + dy, 17, 'W')
        cv.span(18 + dy, 15, 16, 'P')
    elif kind == 'wide':
        for r in range(16, 20):
            cv.span(r + dy, 13, 18, 'K')
        cv.px(16 + dy, 13, 'W'); cv.px(16 + dy, 18, 'W')
        cv.px(19 + dy, 13, 'W'); cv.px(19 + dy, 18, 'W')
        cv.span(18 + dy, 14, 17, 'P')
    elif kind == 'smile':
        cv.px(16 + dy, 13, 'K'); cv.px(16 + dy, 18, 'K')
        cv.span(17 + dy, 14, 17, 'K')
    elif kind == 'frown':
        cv.span(17 + dy, 14, 17, 'K')
        cv.px(16 + dy, 14, 'K'); cv.px(16 + dy, 17, 'K')
        cv.px(18 + dy, 15, 'K'); cv.px(18 + dy, 16, 'K')
    elif kind == 'flat':
        cv.span(16 + dy, 14, 17, 'K')


def fxSparkle(cells):
    """Efecto de chispas que se dibuja sobre el Canvas."""
    def f(cv, _headDy, _bodyDy):
        for r, c, ch in cells:
            cv.px(r, c, ch)
    return f


if __name__ == '__main__':
    cv = build()
    cv.image(10).save('preview_kawaii.png')
    print('ok')
