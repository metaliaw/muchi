# -*- coding: utf-8 -*-
"""Muchi B - retro 8-bit (24x24)."""
from pixlib import Canvas, shift

W = H = 24
PAL = {
    'K': (31, 26, 22),
    'O': (249, 162, 39),
    'L': (253, 199, 90),
    'D': (224, 118, 24),
    'W': (255, 246, 227),
    'P': (232, 130, 130),
    'E': (255, 255, 255),
    'B': (60, 50, 40),
    'R': (231, 76, 76),
    'Y': (255, 226, 120),
    'C': (140, 210, 235),
}
FILLS = 'OLDWP'

EARS = {1: [(3, 4), (8, 9)], 2: [(3, 4), (8, 9)]}
EARS_UP = {0: [(3, 4), (8, 9)], 1: [(3, 4), (8, 9)], 2: [(3, 4), (8, 9)]}
EARS_DOWN = {2: [(1, 3), (9, 11)], 3: [(1, 2), (10, 11)]}

HEAD = {3: (2, 10), 4: (2, 10), 5: (2, 10), 6: (2, 10), 7: (2, 10), 8: (2, 11)}
BODY = {
    9: (2, 12), 10: (2, 13), 11: (2, 14), 12: (2, 14), 13: (2, 14),
    14: (2, 15), 15: (2, 15), 16: (2, 15), 17: (2, 15), 18: (2, 15), 19: (3, 15),
}
TAIL = {
    8: (18, 20), 9: (17, 20), 10: (17, 20), 11: (17, 20), 12: (17, 20),
    13: (17, 20), 14: (17, 20), 15: (17, 20), 16: (17, 20), 17: (16, 20),
    18: (15, 19),
}


def build(head_dy=0, body_dy=0, ears='normal', eyes='open', mouth='closed',
          tail=0, squash=0, fx=()):
    cv = Canvas(W, H, PAL, FILLS)
    hd, bd = head_dy, body_dy

    tdy = bd + squash
    tspec = dict(TAIL)
    if tail == 1:                      # puntita mas arriba
        tspec[7] = (18, 20)
    elif tail == 2:                    # puntita ladeada
        tspec[7] = (18, 20)
        tspec[8] = (18, 21)
        tspec[9] = (17, 21)
    cv.rows(tspec, 'O', dr=tdy)

    body = shift(BODY, dr=squash) if squash else BODY
    cv.rows(body, 'O', dr=bd)
    cv.rows(HEAD, 'O', dr=hd)
    e = {'normal': EARS, 'up': EARS_UP, 'down': EARS_DOWN}[ears]
    cv.rows(e, 'O', dr=hd)
    cv.outline('K')

    # bandas de la cola (siguen a la cola dondequiera que este)
    for i, r in enumerate(sorted(tspec)):
        if i == 0:
            continue
        c0, c1 = tspec[r]
        cv.span(r + tdy, c0, c1, 'L' if ((i - 1) // 2) % 2 == 0 else 'D')

    # pecho claro
    cv.rows({11: (4, 8), 12: (4, 9), 13: (4, 9), 14: (5, 9), 15: (5, 8)}, 'L', dr=bd + squash)
    # manchas del lomo
    cv.rows({10: (11, 13), 11: (12, 14), 14: (12, 15), 15: (13, 15)}, 'D', dr=bd + squash)
    # rayas de la frente
    cv.rows({3: [(5, 5), (7, 7)], 4: [(4, 4), (6, 6), (8, 8)]}, 'D', dr=hd)
    # patas delanteras
    for r in (17, 18, 19):
        cv.px(r + bd + squash, 6, 'K')
        cv.px(r + bd + squash, 10, 'K')

    _eyes(cv, eyes, hd)
    _mouth(cv, mouth, hd)
    for f in fx:
        f(cv, hd, bd)

    cv.span(min(20 + bd + max(squash, 0), H - 1), 3, 15, 'B')
    return cv


def _eyes(cv, kind, dy):
    for base in (4, 8):
        if kind == 'open':
            cv.span(6 + dy, base, base + 1, 'K')
            cv.span(7 + dy, base, base + 1, 'K')
            cv.px(6 + dy, base, 'E')
        elif kind == 'blink':
            cv.span(7 + dy, base, base + 1, 'K')
        elif kind == 'happy':
            cv.px(7 + dy, base, 'K')
            cv.px(6 + dy, base + 1, 'K')
            cv.px(7 + dy, base + 2, 'K')
        elif kind == 'angry':
            cv.span(7 + dy, base, base + 1, 'K')
            cv.px(6 + dy, base + (1 if base == 4 else 0), 'K')
        elif kind == 'shock':
            for r in range(5, 8):
                cv.span(r + dy, base - 1, base + 1, 'K')
            cv.px(6 + dy, base, 'E')


def _mouth(cv, kind, dy):
    if kind == 'closed':
        cv.px(9 + dy, 6, 'K')
        cv.px(9 + dy, 7, 'K')
    elif kind == 'small':
        cv.span(9 + dy, 6, 7, 'K')
        cv.px(10 + dy, 6, 'K')
    elif kind == 'open':
        cv.span(9 + dy, 6, 8, 'K')
        cv.span(10 + dy, 6, 8, 'P')
        cv.px(11 + dy, 7, 'K')
    elif kind == 'wide':
        for r in range(9, 12):
            cv.span(r + dy, 5, 9, 'K')
        cv.span(10 + dy, 6, 8, 'P')
    elif kind == 'smile':
        cv.px(9 + dy, 5, 'K')
        cv.span(10 + dy, 6, 8, 'K')
        cv.px(9 + dy, 9, 'K')
    elif kind == 'frown':
        cv.span(9 + dy, 6, 8, 'K')
        cv.px(8 + dy, 5, 'K')
        cv.px(8 + dy, 9, 'K')
    elif kind == 'flat':
        cv.span(9 + dy, 6, 8, 'K')


if __name__ == '__main__':
    build().image(12).save('/root/muchi/preview_b.png')
    print(build().dump())
