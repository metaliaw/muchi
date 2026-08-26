# -*- coding: utf-8 -*-
"""Genera todos los frames, el sprite sheet y los GIFs de Muchi."""
import json
from PIL import Image
import pixlib
from pixlib import Canvas
import muchi, muchi_b, muchi_c

import os
from pathlib import Path

HERE = Path(__file__).resolve().parent
# Por defecto escribe en assets/muchi/ del repo; MUCHI_OUT lo manda a otro lado.
OUT = str(Path(os.environ.get('MUCHI_OUT', HERE.parents[1] / 'assets' / 'muchi')))
os.makedirs(OUT, exist_ok=True)



# ------------------------------------------------------------------ fx ----
def fx_cells(cells):
    """cells: [(r, c, char)] en coordenadas absolutas."""
    def f(cv, *a):
        for r, c, ch in cells:
            cv.px(r, c, ch)
    return f


def sparkles(pts, ch='Y'):
    def f(cv, *a):
        for r, c in pts:
            cv.px(r, c, ch)
            cv.px(r - 1, c, ch); cv.px(r + 1, c, ch)
            cv.px(r, c - 1, ch); cv.px(r, c + 1, ch)
    return f


def bang(r, c, ch='Y'):
    """signo de exclamacion"""
    def f(cv, *a):
        cv.px(r, c, ch); cv.px(r + 1, c, ch); cv.px(r + 2, c, ch)
        cv.px(r + 4, c, ch)
    return f


def anger(r, c, ch='R'):
    """venita de enojo estilo anime"""
    def f(cv, *a):
        cv.px(r, c, ch); cv.px(r, c + 2, ch)
        cv.px(r + 1, c + 1, ch)
        cv.px(r + 2, c, ch); cv.px(r + 2, c + 2, ch)
    return f


def zzz(items, ch='C'):
    """items: [(r,c,size)] -> Z de 3x3 o 4x4"""
    def f(cv, *a):
        for r, c, s in items:
            cv.span(r, c, c + s - 1, ch)
            cv.span(r + s - 1, c, c + s - 1, ch)
            for i in range(1, s - 1):
                cv.px(r + i, c + s - 1 - i, ch)
    return f


def shift_canvas(cv, dc=0, dr=0):
    if dc == 0 and dr == 0:
        return cv
    n = Canvas(cv.w, cv.h, cv.pal, cv.fills)
    for r in range(cv.h):
        for c in range(cv.w):
            ch = cv.g[r][c]
            if ch != pixlib.T:
                n.px(r + dr, c + dc, ch)
    return n


# -------------------------------------------------------------- estilo A --
def frames_a():
    A = {}
    # 16 frames y no 8: con 8 el parpadeo caia una vez por segundo y se leia
    # como un tic. Aca respira una vez por ciclo y parpadea una sola vez.
    br = [0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0]
    tl = [0, 0, 1, 1, 2, 2, 1, 1, 0, 0, -1, -1, 0, 0, 1, 1]
    ey = ['open'] * 16
    ey[12] = 'blink'
    A['idle'] = [muchi.build(head_dy=br[i], body_dy=br[i], tail=tl[i], eyes=ey[i])
                 for i in range(16)]

    # Sin parpadeo: en un loop de medio segundo se veia como un parpadeo epileptico.
    mo = ['closed', 'small', 'open', 'wide', 'open', 'small']
    hb = [0, 0, -1, -1, 0, 0]
    tl_t = [0, 1, 2, 2, 1, 0]
    A['talk'] = [muchi.build(head_dy=hb[i], tail=tl_t[i] - 1, mouth=mo[i], eyes='open')
                 for i in range(6)]

    hp = [
        dict(squash=1, head_dy=1, body_dy=0, eyes='happy', mouth='smile', ear='up'),
        dict(squash=2, head_dy=1, eyes='happy', mouth='smile', ear='up'),
        dict(head_dy=-3, body_dy=-3, eyes='happy', mouth='open', ear='up', tail=2),
        dict(head_dy=-4, body_dy=-4, eyes='happy', mouth='open', ear='up', tail=2,
             fx=[sparkles([(6, 2), (7, 29), (12, 1)])]),
        dict(head_dy=-3, body_dy=-3, eyes='happy', mouth='open', ear='up', tail=1,
             fx=[sparkles([(4, 3), (9, 28)])]),
        dict(head_dy=-1, body_dy=-1, eyes='happy', mouth='smile', ear='up', tail=1),
        dict(squash=1, head_dy=1, eyes='happy', mouth='smile', ear='up'),
        dict(eyes='happy', mouth='smile', ear='up', tail=1),
    ]
    A['happy'] = [muchi.build(**k) for k in hp]

    al = [
        dict(eyes='shock', mouth='wide', ear='up', head_dy=-2, body_dy=-1, tail=2),
        dict(eyes='shock', mouth='wide', ear='up', head_dy=-2, body_dy=-1, tail=2,
             fx=[bang(2, 29)]),
        dict(eyes='shock', mouth='open', ear='up', head_dy=-1, tail=2,
             fx=[bang(2, 29)]),
        dict(eyes='shock', mouth='open', ear='up', tail=1, fx=[bang(3, 29)]),
        dict(eyes='wide', mouth='open', ear='up', tail=1),
        dict(eyes='wide', mouth='small', ear='normal', tail=0),
    ]
    A['alert'] = [muchi.build(**k) for k in al]

    an = []
    for i in range(6):
        cv = muchi.build(eyes='angry', mouth='frown' if i % 2 == 0 else 'flat',
                         ear='down', tail=-1 if i % 2 else 1,
                         fx=[anger(1, 26)] if i < 4 else [])
        an.append(shift_canvas(cv, dc=(1, -1, 1, -1, 0, 0)[i]))
    A['angry'] = an
    return A


# -------------------------------------------------------------- estilo B --
def frames_b():
    B = {}
    br = [0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0]
    tl = [0, 0, 1, 1, 2, 2, 1, 1, 0, 0, 0, 1, 1, 2, 2, 1]
    ey = ['open'] * 16
    ey[12] = 'blink'
    B['idle'] = [muchi_b.build(body_dy=0, squash=br[i], head_dy=br[i],
                               tail=tl[i], eyes=ey[i]) for i in range(16)]

    mo = ['closed', 'small', 'open', 'wide', 'open', 'small']
    tl_t = [0, 1, 2, 2, 1, 0]
    B['talk'] = [muchi_b.build(head_dy=-1 if i in (2, 3) else 0, mouth=mo[i],
                               tail=tl_t[i], eyes='open')
                 for i in range(6)]

    hp = [
        dict(squash=1, eyes='happy', mouth='smile', ears='up'),
        dict(squash=2, eyes='happy', mouth='smile', ears='up'),
        dict(head_dy=-3, body_dy=-3, eyes='happy', mouth='open', ears='up', tail=1),
        dict(head_dy=-4, body_dy=-4, eyes='happy', mouth='open', ears='up', tail=1,
             fx=[sparkles([(3, 1), (5, 22), (10, 1)])]),
        dict(head_dy=-3, body_dy=-3, eyes='happy', mouth='open', ears='up', tail=1,
             fx=[sparkles([(2, 2), (7, 21)])]),
        dict(head_dy=-1, body_dy=-1, eyes='happy', mouth='smile', ears='up'),
        dict(squash=1, eyes='happy', mouth='smile', ears='up'),
        dict(eyes='happy', mouth='smile', ears='up', tail=1),
    ]
    B['happy'] = [muchi_b.build(**k) for k in hp]

    al = [
        dict(eyes='shock', mouth='wide', ears='up', head_dy=-2, body_dy=-1, tail=1),
        dict(eyes='shock', mouth='wide', ears='up', head_dy=-2, body_dy=-1, tail=1,
             fx=[bang(0, 22)]),
        dict(eyes='shock', mouth='open', ears='up', head_dy=-1, tail=1, fx=[bang(0, 22)]),
        dict(eyes='shock', mouth='open', ears='up', fx=[bang(1, 22)]),
        dict(eyes='open', mouth='open', ears='up'),
        dict(eyes='open', mouth='small', ears='normal'),
    ]
    B['alert'] = [muchi_b.build(**k) for k in al]

    an = []
    for i in range(6):
        cv = muchi_b.build(eyes='angry', mouth='frown' if i % 2 == 0 else 'flat',
                           ears='down', tail=1 if i % 2 else 0,
                           fx=[anger(0, 20)] if i < 4 else [])
        an.append(shift_canvas(cv, dc=(1, -1, 1, -1, 0, 0)[i]))
    B['angry'] = an
    return B


# -------------------------------------------------------------- estilo C --
def frames_c():
    C = {}
    z = [[], [(4, 22, 3)], [(4, 22, 3), (1, 26, 3)],
         [(3, 22, 3), (0, 26, 3)], [(2, 22, 3)], []]
    br = [0, 0, 1, 1, 1, 0]
    C['sleep'] = [muchi_c.build(breathe=br[i],
                                ears='twitch' if i == 4 else 'normal',
                                fx=[zzz(z[i])] if z[i] else [])
                  for i in range(6)]

    wk = [
        dict(eyes='closed', ears='twitch'),
        dict(eyes='closed', ears='up'),
        dict(eyes='half', ears='up'),
        dict(eyes='half', ears='normal', mouth='open'),
        dict(eyes='open', ears='up', mouth='open'),
        dict(eyes='open', ears='normal', mouth='smile'),
    ]
    C['wake'] = [muchi_c.build(**k) for k in wk]

    C['purr'] = [muchi_c.build(breathe=i % 2, mouth='smile',
                               fx=[sparkles([(9, 2)], 'P')] if i in (1, 3) else [])
                 for i in range(4)]
    return C


PAL_CREMA = dict(muchi.PAL)
PAL_CREMA.update({
    'K': (74, 53, 80),      # violeta oscuro, el contorno que ya usa el repo
    'O': (253, 244, 237),   # pelaje crema
    'L': (255, 255, 255),   # luz
    'D': (233, 220, 226),   # pelo en sombra
    'W': (255, 255, 255),   # pechera
    'S': (233, 220, 226),
    'P': (253, 191, 211),   # oreja interna
    'I': (109, 126, 180),   # iris periwinkle
    'B': (214, 206, 220),
})

STYLES = [
    ('muchi-kawaii', 32, 32, frames_a, {'B': 90}, None),
    ('muchi-crema', 32, 32, frames_a, {'B': 120}, PAL_CREMA),
    ('muchi-retro', 24, 24, frames_b, {'B': 110}, None),
    ('muchi-dormida', 32, 24, frames_c, {'B': 90}, None),
]
SPEED = {'idle': 130, 'talk': 90, 'happy': 80, 'alert': 100, 'angry': 90,
         'sleep': 260, 'wake': 140, 'purr': 180}

# Las que tienen principio y final NO se repiten: en loop, el salto del ultimo
# frame al primero se ve como un corte. Se reproducen una vez y quedan quietas
# en su ultimo frame, que por eso es una pose de reposo valida.
LOOP = {'idle': True, 'talk': True, 'sleep': True, 'purr': True,
        'happy': False, 'alert': False, 'angry': False, 'wake': False}


def main():
    os.makedirs(f'{OUT}/frames', exist_ok=True)
    os.makedirs(f'{OUT}/gif', exist_ok=True)
    meta = {'styles': {}}
    for name, w, h, fn, alpha, pal in STYLES:
        anims = fn()
        if pal:
            for cvs in anims.values():
                for cv in cvs:
                    cv.pal = pal
        maxlen = max(len(v) for v in anims.values())
        sheet = Image.new('RGBA', (maxlen * w, len(anims) * h), (0, 0, 0, 0))
        rows_meta = {}
        for ri, (an, cvs) in enumerate(anims.items()):
            for ci, cv in enumerate(cvs):
                im = cv.image(1, alpha)
                sheet.paste(im, (ci * w, ri * h))
                im.resize((w * 8, h * 8), Image.NEAREST).save(
                    f'{OUT}/frames/{name}_{an}_{ci:02d}.png')
            pixlib.gif([c.image(1, alpha) for c in cvs],
                       f'{OUT}/gif/{name}_{an}.gif', ms=SPEED[an], scale=6)
            rows_meta[an] = {'row': ri, 'frames': len(cvs), 'ms': SPEED[an],
                             'loop': LOOP[an]}
        sheet.save(f'{OUT}/{name}-sheet.png')
        sheet.resize((sheet.width * 4, sheet.height * 4), Image.NEAREST).save(
            f'{OUT}/{name}-sheet@4x.png')
        meta['styles'][name] = {'frameWidth': w, 'frameHeight': h,
                                'columns': maxlen, 'animations': rows_meta}
    json.dump(meta, open(f'{OUT}/muchi-sheets.json', 'w'), indent=2)
    print(json.dumps(meta, indent=2))


if __name__ == '__main__':
    main()
