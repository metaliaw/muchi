# -*- coding: utf-8 -*-
"""Muchi C - durmiendo, enroscada (32x24)."""
from pixlib import Canvas

W, H = 32, 24
PAL = {
    'K': (37, 35, 38),
    'O': (224, 139, 52),
    'D': (198, 106, 30),
    'L': (240, 175, 96),
    'W': (238, 214, 182),
    'S': (214, 186, 152),
    'P': (226, 150, 148),
    'E': (255, 255, 255),
    'B': (70, 62, 66),
    'R': (231, 76, 76),
    'Y': (255, 226, 120),
    'C': (170, 205, 225),
}
FILLS = 'ODLWSP'

BODY = {
    6: (16, 22), 7: (14, 25), 8: (12, 27), 9: (11, 28), 10: (10, 29),
    11: (10, 29), 12: (10, 29), 13: (10, 29), 14: (10, 29), 15: (10, 29),
    16: (11, 28), 17: (12, 27),
}
HEAD = {
    9: (4, 11), 10: (3, 12), 11: (2, 12), 12: (2, 12), 13: (2, 12),
    14: (2, 12), 15: (2, 12), 16: (3, 11), 17: (5, 10),
}
HEAD_EDGE = ((9, 12), (10, 13), (11, 13), (12, 14), (13, 14))
EAR_L = {6: (4, 6), 7: (4, 6), 8: (4, 6)}
EAR_R = {6: (9, 11), 7: (9, 11), 8: (9, 11)}
FACE = {11: (3, 7), 12: (2, 8), 13: (2, 9), 14: (2, 10), 15: (2, 10),
        16: (3, 10), 17: (5, 10)}
BACK_D = {
    7: (17, 21), 8: [(15, 16), (18, 23)], 9: [(14, 15), (19, 24)],
    10: [(16, 17), (21, 26)], 11: [(20, 22), (25, 27)], 12: (24, 26),
}
BACK_L = {6: (17, 21), 7: (22, 24)}
TAIL = {15: (12, 13), 16: (12, 27), 17: (12, 26)}
TAIL_EDGE = ((15, 14, 28), (14, 12, 13))
TAIL_BANDS = ((16, 16, 17), (16, 21, 22), (16, 26, 27),
              (17, 15, 16), (17, 20, 21), (17, 25, 26))


def build(breathe=0, eyes='closed', ears='normal', tail=0, mouth='none', fx=()):
    cv = Canvas(W, H, PAL, FILLS)
    b = -1 if breathe else 0     # el lomo sube 1px al inspirar

    body = {r + (b if r <= 12 else 0): v for r, v in BODY.items()}
    cv.rows(body, 'O')
    cv.rows(HEAD, 'O')
    ed = -1 if ears == 'up' else (1 if ears == 'twitch' else 0)
    cv.rows(EAR_L, 'O', dr=ed)
    cv.rows(EAR_R, 'O')
    cv.outline('K')

    cv.rows(BACK_D, 'D', dr=b)
    cv.rows(BACK_L, 'L', dr=b)

    # linea cabeza / lomo
    for r, c in HEAD_EDGE:
        cv.px(r, c, 'K')

    # cara + pecho crema
    cv.rows(FACE, 'W')
    cv.rows({16: (5, 10), 17: (6, 10)}, 'S')
    cv.px(16, 8, 'K'); cv.px(17, 8, 'K')      # separacion patitas

    # cola enroscada por delante
    tdy = -1 if tail else 0
    cv.rows(TAIL, 'D', dr=tdy)
    for r, c0, c1 in TAIL_BANDS:
        cv.span(r + tdy, c0, c1, 'L')
    for r, c0, c1 in TAIL_EDGE:
        cv.span(r + tdy, c0, c1, 'K')
    cv.px(15 + tdy, 11, 'K'); cv.px(16 + tdy, 11, 'K'); cv.px(17 + tdy, 11, 'K')

    # orejitas por dentro
    cv.px(7 + ed, 5, 'P'); cv.px(8 + ed, 5, 'P'); cv.px(8 + ed, 6, 'P')
    cv.px(7, 10, 'P'); cv.px(8, 9, 'P'); cv.px(8, 10, 'P')

    _eyes(cv, eyes)
    _mouth(cv, mouth)
    for f in fx:
        f(cv)

    cv.span(19, 5, 28, 'B')
    return cv


def _eyes(cv, kind):
    if kind == 'closed':
        cv.span(13, 5, 6, 'K')
        cv.px(12, 4, 'K'); cv.px(12, 7, 'K')
    elif kind == 'half':
        cv.span(12, 4, 7, 'K')
        cv.px(13, 5, 'K'); cv.px(13, 6, 'K')
    elif kind == 'open':
        for r in (11, 12, 13):
            cv.span(r, 4, 7, 'K')
        cv.px(11, 4, 'W'); cv.px(11, 7, 'W')
        cv.px(13, 4, 'W'); cv.px(13, 7, 'W')
        cv.px(11, 5, 'E'); cv.px(12, 5, 'E')


def _mouth(cv, kind):
    cv.px(15, 3, 'K')
    if kind == 'smile':
        cv.px(16, 4, 'K'); cv.px(15, 5, 'K')
    elif kind == 'open':
        cv.span(16, 3, 4, 'K')


if __name__ == '__main__':
    build().image(10).save('/root/muchi/preview_c.png')
