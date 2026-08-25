# -*- coding: utf-8 -*-
"""Mini libreria de pixel-art: canvas de caracteres -> PNG."""
from PIL import Image

T = '.'


class Canvas:
    def __init__(self, w, h, palette, fills):
        self.w, self.h = w, h
        self.pal = palette
        self.fills = set(fills)
        self.g = [[T] * w for _ in range(h)]

    def clone(self):
        c = Canvas(self.w, self.h, self.pal, self.fills)
        c.g = [row[:] for row in self.g]
        return c

    def px(self, r, c, ch):
        if 0 <= r < self.h and 0 <= c < self.w:
            self.g[r][c] = ch

    def span(self, r, c0, c1, ch):
        for c in range(c0, c1 + 1):
            self.px(r, c, ch)

    def rows(self, spec, ch, dr=0, dc=0):
        for r, v in spec.items():
            segs = v if isinstance(v, list) else [v]
            for c0, c1 in segs:
                self.span(r + dr, c0 + dc, c1 + dc, ch)

    def outline(self, ch='K'):
        out = [row[:] for row in self.g]
        for r in range(self.h):
            for c in range(self.w):
                if self.g[r][c] in self.fills:
                    for dr, dc in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                        rr, cc = r + dr, c + dc
                        if 0 <= rr < self.h and 0 <= cc < self.w and self.g[rr][cc] == T:
                            out[rr][cc] = ch
        self.g = out

    def image(self, scale=1, alpha=None):
        alpha = alpha or {}
        im = Image.new('RGBA', (self.w, self.h), (0, 0, 0, 0))
        p = im.load()
        for r in range(self.h):
            for c in range(self.w):
                ch = self.g[r][c]
                if ch != T:
                    col = self.pal[ch]
                    a = alpha.get(ch, 255)
                    p[c, r] = (col[0], col[1], col[2], a)
        if scale > 1:
            im = im.resize((self.w * scale, self.h * scale), Image.NEAREST)
        return im

    def dump(self):
        head = '   ' + ''.join(str(c % 10) for c in range(self.w))
        return head + '\n' + '\n'.join('%2d ' % i + ''.join(r) for i, r in enumerate(self.g))


def mirror(spec, w):
    out = {}
    for r, v in spec.items():
        segs = v if isinstance(v, list) else [v]
        out[r] = [(w - 1 - c1, w - 1 - c0) for c0, c1 in segs]
    return out


def shift(spec, dr=0, dc=0):
    return {r + dr: ([(a + dc, b + dc) for a, b in v] if isinstance(v, list)
                     else (v[0] + dc, v[1] + dc)) for r, v in spec.items()}


def sheet(frames, cols, scale=1, pad=0):
    """frames: lista de PIL Images del mismo tamano."""
    fw, fh = frames[0].size
    rows_n = (len(frames) + cols - 1) // cols
    im = Image.new('RGBA', (cols * (fw + pad), rows_n * (fh + pad)), (0, 0, 0, 0))
    for i, f in enumerate(frames):
        im.paste(f, ((i % cols) * (fw + pad), (i // cols) * (fh + pad)))
    if scale > 1:
        im = im.resize((im.width * scale, im.height * scale), Image.NEAREST)
    return im


def gif(frames, path, ms=120, scale=6):
    fs = [f.resize((f.width * scale, f.height * scale), Image.NEAREST) for f in frames]
    bg = []
    for f in fs:
        b = Image.new('RGBA', f.size, (255, 255, 255, 255))
        b.alpha_composite(f)
        bg.append(b.convert('P', palette=Image.ADAPTIVE))
    bg[0].save(path, save_all=True, append_images=bg[1:], duration=ms, loop=0, disposal=2)
