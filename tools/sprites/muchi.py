# -*- coding: utf-8 -*-
"""Muchi - generador de sprites pixel-art 32x32 + animaciones."""
from PIL import Image

W = H = 32
T = '.'

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


def grid():
    return [[T] * W for _ in range(H)]


def px(g, r, c, ch):
    if 0 <= r < H and 0 <= c < W:
        g[r][c] = ch


def span(g, r, c0, c1, ch):
    for c in range(c0, c1 + 1):
        px(g, r, c, ch)


def rows(g, spec, ch, dr=0, dc=0):
    """spec: dict fila -> (c0,c1) o lista de (c0,c1)"""
    for r, v in spec.items():
        segs = v if isinstance(v, list) else [v]
        for c0, c1 in segs:
            span(g, r + dr, c0 + dc, c1 + dc, ch)


def outline(g):
    out = [row[:] for row in g]
    for r in range(H):
        for c in range(W):
            if g[r][c] in FILLS:
                for dr, dc in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                    rr, cc = r + dr, c + dc
                    if 0 <= rr < H and 0 <= cc < W and g[rr][cc] == T:
                        out[rr][cc] = 'K'
    return out


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
TAIL_TIP = {16: (28, 30), 17: (27, 30), 18: (27, 30), 19: (27, 29)}


def mirror(spec):
    out = {}
    for r, v in spec.items():
        segs = v if isinstance(v, list) else [v]
        out[r] = [(W - 1 - c1, W - 1 - c0) for c0, c1 in segs]
    return out


def build(head_dy=0, body_dy=0, ear='normal', eyes='open', mouth='closed',
          tail=0, fx=None, squash=0):
    g = grid()
    hdy = head_dy
    bdy = body_dy

    # --- cola (detras) ---
    tspec = TAIL
    if tail == 1:
        tspec = {r: (c0 + 1, c1 + 1) for r, (c0, c1) in TAIL.items()}
    elif tail == 2:
        tspec = {r - 1: (c0 + 1, c1 + 1) for r, (c0, c1) in TAIL.items()}
    elif tail == -1:
        tspec = {r + 1: (c0 - 1, c1 - 1) for r, (c0, c1) in TAIL.items()}
    rows(g, tspec, 'O', dr=bdy)
    tip = {}
    for r, (c0, c1) in tspec.items():
        if r <= min(tspec) + 3:
            tip[r] = (c0 + (1 if r > min(tspec) + 2 else 0), c1)
    rows(g, tip, 'W', dr=bdy)

    # --- cuerpo ---
    body = BODY
    if squash:
        body = {r + squash: v for r, v in BODY.items() if r + squash < H - 2}
    rows(g, body, 'O', dr=bdy)
    rows(g, BELLY, 'W', dr=bdy + squash)

    # patitas delanteras
    pd = bdy + squash
    span(g, 26 + pd, 9, 13, 'W')
    span(g, 26 + pd, 18, 22, 'W')
    span(g, 27 + pd, 10, 13, 'W')
    span(g, 27 + pd, 18, 21, 'W')
    span(g, 25 + pd, 10, 12, 'S')
    span(g, 25 + pd, 19, 21, 'S')

    # --- orejas ---
    el, er = EAR_L, mirror(EAR_L)
    eli, eri = EAR_L_IN, mirror(EAR_L_IN)
    if ear == 'up':
        el = {r - 1: v for r, v in el.items()}
        er = {r - 1: v for r, v in er.items()}
        eli = {r - 1: v for r, v in eli.items()}
        eri = {r - 1: v for r, v in eri.items()}
    elif ear == 'down':
        el = {2: (5, 7), 3: (4, 9), 4: (4, 10), 5: (4, 11), 6: (5, 12), 7: (5, 12)}
        er, eri = mirror(el), mirror({4: (6, 8), 5: (6, 9), 6: (7, 11)})
        eli = {4: (6, 8), 5: (6, 9), 6: (7, 11)}
    rows(g, el, 'O', dr=hdy)
    rows(g, er, 'O', dr=hdy)
    rows(g, eli, 'P', dr=hdy)
    rows(g, eri, 'P', dr=hdy)

    # --- cabeza ---
    rows(g, HEAD, 'O', dr=hdy)

    g = outline(g)

    # ---------------------------------------------------------- detalles ---
    # rayas frente
    for c in (13, 15, 17):
        span(g, 7 + hdy, c, c, 'D')
        span(g, 8 + hdy, c, c, 'D')
    span(g, 9 + hdy, 15, 15, 'D')
    # rayas mejillas
    for r in (11, 12):
        px(g, r + hdy, 6, 'D')
        px(g, r + hdy, 25, 'D')
    px(g, 13 + hdy, 6, 'D')
    px(g, 13 + hdy, 25, 'D')
    # brillo superior cabeza
    span(g, 7 + hdy, 11, 12, 'L')
    span(g, 8 + hdy, 9, 10, 'L')
    span(g, 7 + hdy, 19, 20, 'L')

    # hocico crema
    rows(g, {14: (11, 20), 15: (10, 21), 16: (11, 20), 17: (12, 19)}, 'W', dr=hdy)

    # separacion cola / cuerpo
    for r, c in ((23, 25), (24, 24), (25, 23), (26, 22), (27, 21)):
        px(g, r + bdy, c, 'K')
    for r in (26, 27):
        px(g, r + bdy + squash, 14, 'K')
        px(g, r + bdy + squash, 17, 'K')

    draw_eyes(g, eyes, hdy)
    draw_mouth(g, mouth, hdy)

    from pixlib import Canvas
    cv = Canvas(W, H, PAL, FILLS)
    cv.g = g
    for f in (fx or ()):
        f(cv, hdy, bdy)

    # sombra en el suelo
    shadow_r = 28 + (squash if squash > 0 else 0) + bdy
    cv.span(min(shadow_r + 1, H - 1), 10, 21, 'B')
    return cv


def draw_eyes(g, kind, dy):
    L, R = 8, 19  # col inicio ojo izq / der  (4 de ancho)
    if kind in ('open', 'wide'):
        top, bot = (9, 14) if kind == 'wide' else (10, 14)
        for base in (L, R):
            for r in range(top, bot + 1):
                span(g, r + dy, base, base + 4, 'I')
            # recorte esquinas
            px(g, top + dy, base, 'O'); px(g, top + dy, base + 4, 'O')
            px(g, bot + dy, base, 'O'); px(g, bot + dy, base + 4, 'O')
            # brillos
            span(g, top + 1 + dy, base + 1, base + 2, 'E')
            px(g, top + 2 + dy, base + 1, 'E')
            px(g, bot - 1 + dy, base + 3, 'E')
    elif kind == 'blink':
        for base in (L, R):
            span(g, 12 + dy, base, base + 4, 'I')
            px(g, 13 + dy, base + 1, 'I'); px(g, 13 + dy, base + 3, 'I')
    elif kind == 'happy':  # ^ ^
        for base in (L, R):
            span(g, 13 + dy, base, base + 1, 'K')
            span(g, 11 + dy, base + 2, base + 2, 'K')
            span(g, 12 + dy, base + 1, base + 1, 'K')
            span(g, 12 + dy, base + 3, base + 3, 'K')
            span(g, 13 + dy, base + 3, base + 4, 'K')
    elif kind == 'angry':
        for i, base in enumerate((L, R)):
            for r in range(12, 15):
                span(g, r + dy, base + 1, base + 3, 'I')
            px(g, 12 + dy, base + 2, 'E')
            if i == 0:   # ceja bajando hacia el centro
                span(g, 10 + dy, base + 2, base + 4, 'K')
                px(g, 11 + dy, base + 3, 'K'); px(g, 11 + dy, base + 4, 'K')
                px(g, 11 + dy, base, 'K')
            else:
                span(g, 10 + dy, base, base + 2, 'K')
                px(g, 11 + dy, base, 'K'); px(g, 11 + dy, base + 1, 'K')
                px(g, 11 + dy, base + 4, 'K')
    elif kind == 'shock':
        for base in (L, R):
            for r in range(9, 15):
                span(g, r + dy, base, base + 4, 'I')
            px(g, 9 + dy, base, 'O'); px(g, 9 + dy, base + 4, 'O')
            px(g, 14 + dy, base, 'O'); px(g, 14 + dy, base + 4, 'O')
            span(g, 11 + dy, base + 1, base + 3, 'E')
            span(g, 12 + dy, base + 1, base + 3, 'E')
            px(g, 11 + dy, base + 2, 'K'); px(g, 12 + dy, base + 2, 'K')


def draw_mouth(g, kind, dy):
    # nariz
    span(g, 14 + dy, 15, 16, 'K')
    px(g, 15 + dy, 15, 'K'); px(g, 15 + dy, 16, 'K')
    if kind == 'closed':
        px(g, 16 + dy, 14, 'K'); px(g, 16 + dy, 17, 'K')
        px(g, 17 + dy, 15, 'K'); px(g, 17 + dy, 16, 'K')
    elif kind == 'small':
        span(g, 16 + dy, 15, 16, 'K')
        px(g, 17 + dy, 15, 'K'); px(g, 17 + dy, 16, 'K')
    elif kind == 'open':
        for r in range(16, 19):
            span(g, r + dy, 14, 17, 'K')
        px(g, 18 + dy, 14, 'W'); px(g, 18 + dy, 17, 'W')
        span(g, 18 + dy, 15, 16, 'P')
    elif kind == 'wide':
        for r in range(16, 20):
            span(g, r + dy, 13, 18, 'K')
        px(g, 16 + dy, 13, 'W'); px(g, 16 + dy, 18, 'W')
        px(g, 19 + dy, 13, 'W'); px(g, 19 + dy, 18, 'W')
        span(g, 18 + dy, 14, 17, 'P')
    elif kind == 'smile':
        px(g, 16 + dy, 13, 'K'); px(g, 16 + dy, 18, 'K')
        span(g, 17 + dy, 14, 17, 'K')
    elif kind == 'frown':
        span(g, 17 + dy, 14, 17, 'K')
        px(g, 16 + dy, 14, 'K'); px(g, 16 + dy, 17, 'K')
        px(g, 18 + dy, 15, 'K'); px(g, 18 + dy, 16, 'K')
    elif kind == 'flat':
        span(g, 16 + dy, 14, 17, 'K')


# ------------------------------------------------------------------- fx ---
def fx_sparkle(cells):
    def f(g, hdy, bdy):
        for r, c, ch in cells:
            px(g, r, c, ch)
    return f


def render(g, scale=1):
    im = Image.new('RGBA', (W, H), (0, 0, 0, 0))
    p = im.load()
    for r in range(H):
        for c in range(W):
            ch = g[r][c]
            if ch != T:
                col = PAL[ch]
                a = 90 if ch == 'B' else 255
                p[c, r] = (col[0], col[1], col[2], a)
    if scale > 1:
        im = im.resize((W * scale, H * scale), Image.NEAREST)
    return im


if __name__ == '__main__':
    build().image(10).save('/root/muchi/preview_base.png')
    print('ok')
