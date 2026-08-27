# -*- coding: utf-8 -*-
"""Arma la demo HTML (version artifact y version standalone)."""
import base64, json
import os
from pathlib import Path

HERE = Path(__file__).resolve().parent
# Por defecto escribe en assets/muchi/ del repo; MUCHI_OUT lo manda a otro lado.
OUT = str(Path(os.environ.get('MUCHI_OUT', HERE.parents[1] / 'assets' / 'muchi')))
os.makedirs(OUT, exist_ok=True)

META = json.load(open(f'{OUT}/muchi-sheets.json'))

LABELS = {
    'idle': 'Idle', 'talk': 'Hablando', 'happy': 'Evento feliz',
    'alert': 'Alerta', 'angry': 'Molesta', 'sleep': 'Durmiendo',
    'wake': 'Despertando', 'purr': 'Ronroneo',
}
STYLE_INFO = {
    'muchi-kawaii': ('Kawaii naranja', 'Chibi de cabeza grande, contorno ciruela y ojos brillantes. La de las referencias que me pasaste.'),
    'muchi-crema': ('Muchi de verdad', 'Misma geometria que la naranja, con la paleta de Muchi: pelo crema, oreja rosada y ojos periwinkle.'),
    'muchi-retro': ('Retro 8-bit', 'Bloques gruesos, contorno negro y cola a rayas. Menos pixeles, mas arcade.'),
    'muchi-dormida': ('Dormida', 'Enroscada, respirando y soltando Z. Para cuando la app esta esperando o no pasa nada hace rato.'),
}


def b64(path):
    return base64.b64encode(open(path, 'rb').read()).decode('ascii')


def sheet_css():
    css = []
    for name, m in META['styles'].items():
        uri = 'data:image/png;base64,' + b64(f'{OUT}/{name}-sheet.png')
        css.append(
            f'.mu[data-style="{name}"]{{--fw:{m["frameWidth"]};--fh:{m["frameHeight"]};'
            f'--cols:{m["columns"]};--rows:{len(m["animations"])};'
            f'background-image:url({uri})}}')
    return '\n'.join(css)


PAGE = r'''<title>Muchi Sprite Lab</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Silkscreen:wght@400;700&family=Karla:ital,wght@0,400;0,600;0,800;1,400&family=JetBrains+Mono:wght@400;600&display=swap">
<style>
:root{
  --paper:#FBF3E6; --card:#FFFAF1; --sunk:#F0E4D2;
  --ink:#2A1B33; --ink-soft:#6B5570; --line:#E0CFBA;
  --orange:#E85F22; --orange-soft:#F5733A; --gold:#D89A18;
  --plum:#7A4A86;
  --stage-a:#EFE2CF; --stage-b:#E6D6BE;
  --shadow:0 2px 0 var(--line);
  --fs-1:.72rem; --fs0:1rem; --fs1:1.22rem; --fs2:1.6rem; --fs3:2.3rem; --fs4:3.4rem;
}
@media (prefers-color-scheme:dark){
  :root:not([data-theme="light"]){
    --paper:#1A1020; --card:#241631; --sunk:#150C1B;
    --ink:#F6E9DC; --ink-soft:#B49BC0; --line:#3A2648;
    --orange:#FF8A4C; --orange-soft:#F5733A; --gold:#FFC85A;
    --plum:#C79BD6;
    --stage-a:#221430; --stage-b:#1B1027;
    --shadow:0 2px 0 #120A18;
  }
}
:root[data-theme="dark"]{
  --paper:#1A1020; --card:#241631; --sunk:#150C1B;
  --ink:#F6E9DC; --ink-soft:#B49BC0; --line:#3A2648;
  --orange:#FF8A4C; --orange-soft:#F5733A; --gold:#FFC85A;
  --plum:#C79BD6;
  --stage-a:#221430; --stage-b:#1B1027;
  --shadow:0 2px 0 #120A18;
}

*{box-sizing:border-box}
body{
  margin:0; background:var(--paper); color:var(--ink);
  font-family:Karla,"Trebuchet MS",system-ui,sans-serif; font-size:var(--fs0);
  line-height:1.6; -webkit-font-smoothing:antialiased;
}
.wrap{max-width:64rem;margin:0 auto;padding:2.5rem 1.25rem 5rem;display:flex;flex-direction:column;gap:3.25rem}
h1,h2,h3{font-family:Silkscreen,"Courier New",monospace;font-weight:700;line-height:1.15;text-wrap:balance;margin:0}
.eyebrow{font-family:Silkscreen,monospace;font-size:var(--fs-1);letter-spacing:.14em;text-transform:uppercase;color:var(--orange)}
p{margin:0}
.lede{max-width:34rem;color:var(--ink-soft)}
code,pre{font-family:"JetBrains Mono",ui-monospace,monospace}

/* ---------- header ---------- */
header{display:flex;flex-direction:column;gap:1.5rem}
.title{font-size:var(--fs4);letter-spacing:-.02em}
.title span{color:var(--orange)}
.parade{
  display:flex;align-items:flex-end;gap:1.5rem;flex-wrap:wrap;
  padding:1.25rem 1.5rem;border:2px solid var(--line);border-radius:4px;
  background:var(--sunk);
}

/* ---------- sprite ---------- */
.mu{
  --s:4; --n:1; --row:0;
  width:calc(var(--fw) * var(--s) * 1px);
  height:calc(var(--fh) * var(--s) * 1px);
  background-repeat:no-repeat;
  background-size:calc(var(--cols) * var(--fw) * var(--s) * 1px) calc(var(--rows) * var(--fh) * var(--s) * 1px);
  background-position-y:calc(var(--row) * var(--fh) * var(--s) * -1px);
  image-rendering:pixelated; image-rendering:crisp-edges;
  animation:mu-play var(--dur,1s) steps(var(--n)) infinite;
  flex:none;
}
@keyframes mu-play{
  from{background-position-x:0}
  to{background-position-x:calc(var(--n) * var(--fw) * var(--s) * -1px)}
}
@media (prefers-reduced-motion:reduce){ .mu{animation:none} }

/* ---------- tarjetas ---------- */
.grid{display:flex;flex-direction:column;gap:1.5rem}
.card{
  border:2px solid var(--line);border-radius:4px;background:var(--card);
  display:grid;grid-template-columns:minmax(0,15rem) minmax(0,1fr);
}
@media (max-width:46rem){ .card{grid-template-columns:1fr} }
.stage{
  display:flex;align-items:center;justify-content:center;padding:1.5rem;min-height:13rem;
  border-right:2px solid var(--line);
  background-color:var(--stage-a);
  background-image:linear-gradient(45deg,var(--stage-b) 25%,transparent 25%,transparent 75%,var(--stage-b) 75%),
                   linear-gradient(45deg,var(--stage-b) 25%,transparent 25%,transparent 75%,var(--stage-b) 75%);
  background-size:16px 16px; background-position:0 0,8px 8px;
}
@media (max-width:46rem){ .stage{border-right:0;border-bottom:2px solid var(--line)} }
.body{padding:1.4rem 1.5rem;display:flex;flex-direction:column;gap:.9rem;min-width:0}
.card h2{font-size:var(--fs2)}
.desc{color:var(--ink-soft);font-size:.95rem}
.chips{display:flex;flex-wrap:wrap;gap:.4rem}
.chip{
  font-family:Silkscreen,monospace;font-size:var(--fs-1);letter-spacing:.06em;
  padding:.45rem .7rem;border:2px solid var(--line);border-radius:3px;
  background:var(--card);color:var(--ink-soft);cursor:pointer;
  box-shadow:var(--shadow);transition:transform .08s,background .12s,color .12s;
}
.chip:hover{color:var(--ink);border-color:var(--orange)}
.chip:active{transform:translateY(2px);box-shadow:none}
.chip[aria-pressed="true"]{background:var(--orange);color:#fff;border-color:var(--orange)}
.chip:focus-visible{outline:3px solid var(--plum);outline-offset:2px}
.meta{
  display:flex;gap:1.25rem;flex-wrap:wrap;font-size:var(--fs-1);
  font-family:"JetBrains Mono",monospace;color:var(--ink-soft);
  border-top:2px dashed var(--line);padding-top:.8rem;font-variant-numeric:tabular-nums;
}
.meta b{color:var(--ink);font-weight:600}

/* ---------- globo ---------- */
.talkbox{display:flex;flex-direction:column;gap:.75rem}
.bubble{
  position:relative;background:var(--sunk);border:2px solid var(--line);border-radius:4px;
  padding:.7rem .9rem;min-height:2.9rem;font-size:.95rem;
}
.bubble::after{
  content:"";position:absolute;left:1.4rem;bottom:-9px;width:0;height:0;
  border:8px solid transparent;border-top-color:var(--line);border-bottom:0;
}
.caret{display:inline-block;width:.5ch;background:var(--orange);animation:blink 1s steps(2) infinite}
@keyframes blink{50%{opacity:0}}
input[type=text]{
  font:inherit;padding:.55rem .7rem;border:2px solid var(--line);border-radius:3px;
  background:var(--card);color:var(--ink);width:100%;
}
input[type=text]:focus-visible{outline:3px solid var(--plum);outline-offset:1px;border-color:var(--orange)}

/* ---------- sheets ---------- */
.sheetbox{border:2px solid var(--line);border-radius:4px;background:var(--sunk);padding:1rem;overflow-x:auto}
.sheetbox img{display:block;image-rendering:pixelated;min-width:100%;width:auto}
.sheetcap{font-family:"JetBrains Mono",monospace;font-size:var(--fs-1);color:var(--ink-soft);margin-bottom:.6rem}

/* ---------- codigo ---------- */
pre{
  margin:0;background:var(--sunk);border:2px solid var(--line);border-radius:4px;
  padding:1rem 1.1rem;overflow-x:auto;font-size:.82rem;line-height:1.65;color:var(--ink);
}
.tok-c{color:var(--ink-soft);font-style:italic}
.tok-k{color:var(--plum);font-weight:600}
.tok-s{color:var(--orange)}
table{border-collapse:collapse;width:100%;font-size:.9rem}
th,td{text-align:left;padding:.5rem .7rem;border-bottom:1px solid var(--line)}
th{font-family:Silkscreen,monospace;font-size:var(--fs-1);letter-spacing:.06em;color:var(--ink-soft);text-transform:uppercase}
td:first-child{font-family:"JetBrains Mono",monospace}
.files{display:grid;grid-template-columns:repeat(auto-fit,minmax(15rem,1fr));gap:.6rem}
.file{border:2px solid var(--line);border-radius:3px;padding:.7rem .9rem;background:var(--card)}
.file b{font-family:"JetBrains Mono",monospace;font-size:.85rem;display:block}
.file span{font-size:var(--fs-1);color:var(--ink-soft)}
section{display:flex;flex-direction:column;gap:1rem}
__SHEETCSS__
</style>

<div class="wrap">

<header>
  <p class="eyebrow">Sprite sheet + keyframes CSS</p>
  <h1 class="title">Muchi <span>Sprite Lab</span></h1>
  <p class="lede">Cuatro cuerpos para la mascota del carrito, cada uno con sus estados listos para animar. Todo sale de un PNG por estilo y una sola regla de <code>@keyframes</code> con <code>steps()</code> &mdash; sin JavaScript, asi que corre igual dentro de Streamlit.</p>
  <div class="parade">__PARADE__</div>
</header>

<section>
  <p class="eyebrow">Los cuatro cuerpos</p>
  <div class="grid">__CARDS__</div>
</section>

<section>
  <p class="eyebrow">Probala hablando</p>
  <h2>Escribe y Muchi mueve la boca</h2>
  <div class="card">
    <div class="stage"><div class="mu" id="talker" data-style="muchi-kawaii" style="--s:5"></div></div>
    <div class="body talkbox">
      <div class="bubble" id="bubble">Miau, en que te ayudo?</div>
      <label for="say" class="desc">Lo que Muchi va a decir</label>
      <input type="text" id="say" value="Nya~ encontre esa carta en tres tiendas" autocomplete="off">
      <p class="desc">Mientras el texto aparece letra por letra corre <b>talk</b>; al terminar vuelve a <b>idle</b>. Es exactamente lo que necesita el globo de la barra lateral.</p>
    </div>
  </div>
</section>

<section>
  <p class="eyebrow">Como esta armado el PNG</p>
  <h2>Una fila por animacion</h2>
  <p class="lede">Cada hoja se lee de izquierda a derecha: la fila es el estado, la columna es el frame. Las filas cortas quedan con celdas vacias a la derecha &mdash; <code>steps()</code> nunca llega ahi porque el contador de frames es por fila.</p>
  __SHEETS__
</section>

<section>
  <p class="eyebrow">Pegalo en tu app</p>
  <h2>El CSS minimo</h2>
  <pre>__CSSSNIP__</pre>
  <h2>En Streamlit, ya cableado</h2>
  <p class="lede">Streamlit borra las etiquetas <code>&lt;script&gt;</code> de <code>st.markdown</code>, pero no toca los keyframes. En el repo el retro ya es la mascota de la barra lateral, y cada <code>st.warning</code> / <code>st.error</code> / <code>st.info</code> / <code>st.success</code> paso a ser un pensamiento de Muchi: el estado del sprite y el color del borde dicen lo mismo que el texto.</p>
  <pre>__PYSNIP__</pre>
</section>

<section>
  <p class="eyebrow">Que hay en la carpeta</p>
  <h2>Archivos</h2>
  <div class="files">__FILES__</div>
</section>

</div>

<script>
const META = __META__;
function apply(el, style, anim){
  const m = META.styles[style].animations[anim];
  el.dataset.style = style;
  el.style.setProperty('--n', m.frames);
  el.style.setProperty('--row', m.row);
  el.style.setProperty('--dur', (m.frames * m.ms / 1000).toFixed(2) + 's');
}
document.querySelectorAll('.mu').forEach(el => apply(el, el.dataset.style, el.dataset.anim || Object.keys(META.styles[el.dataset.style].animations)[0]));
document.querySelectorAll('.chips').forEach(group => {
  const target = document.getElementById(group.dataset.target);
  group.addEventListener('click', e => {
    const b = e.target.closest('.chip'); if(!b) return;
    group.querySelectorAll('.chip').forEach(x => x.setAttribute('aria-pressed', x === b));
    apply(target, target.dataset.style, b.dataset.anim);
  });
});
const talker = document.getElementById('talker'), bubble = document.getElementById('bubble'), say = document.getElementById('say');
let timer = null;
function speak(){
  clearInterval(timer);
  const text = say.value || ' ';
  let i = 0;
  apply(talker, 'muchi-kawaii', 'talk');
  bubble.textContent = '';
  timer = setInterval(() => {
    bubble.textContent = text.slice(0, ++i);
    if(i >= text.length){ clearInterval(timer); apply(talker, 'muchi-kawaii', 'idle'); }
  }, 55);
}
say.addEventListener('input', speak);
speak();
</script>'''

CSS_SNIP = '''<span class="tok-c">/* una hoja = un PNG. la fila es el estado, la columna el frame */</span>
<span class="tok-k">.muchi</span> {
  <span class="tok-k">--s</span>: 4;                <span class="tok-c">/* zoom entero: 4 = 128px */</span>
  <span class="tok-k">--fw</span>: 32; <span class="tok-k">--fh</span>: 32;   <span class="tok-c">/* tamano del frame */</span>
  <span class="tok-k">--cols</span>: 8; <span class="tok-k">--rows</span>: 5;  <span class="tok-c">/* grilla de la hoja */</span>
  width:  <span class="tok-s">calc(var(--fw) * var(--s) * 1px)</span>;
  height: <span class="tok-s">calc(var(--fh) * var(--s) * 1px)</span>;
  background-image: <span class="tok-s">url(muchi-kawaii-sheet.png)</span>;
  background-repeat: no-repeat;
  background-size: <span class="tok-s">calc(var(--cols) * var(--fw) * var(--s) * 1px)
                        calc(var(--rows) * var(--fh) * var(--s) * 1px)</span>;
  background-position-y: <span class="tok-s">calc(var(--row) * var(--fh) * var(--s) * -1px)</span>;
  image-rendering: pixelated;
  animation: mu-play <span class="tok-s">var(--dur)</span> steps(<span class="tok-s">var(--n)</span>) infinite;
}
<span class="tok-k">@keyframes</span> mu-play {
  from { background-position-x: 0 }
  to   { background-position-x: <span class="tok-s">calc(var(--n) * var(--fw) * var(--s) * -1px)</span> }
}

<span class="tok-c">/* un estado = tres numeros */</span>
<span class="tok-k">.muchi.idle</span>  { <span class="tok-k">--row</span>:0; <span class="tok-k">--n</span>:8; <span class="tok-k">--dur</span>:1.12s }
<span class="tok-k">.muchi.talk</span>  { <span class="tok-k">--row</span>:1; <span class="tok-k">--n</span>:6; <span class="tok-k">--dur</span>:0.54s }
<span class="tok-k">.muchi.happy</span> { <span class="tok-k">--row</span>:2; <span class="tok-k">--n</span>:8; <span class="tok-k">--dur</span>:0.64s }
<span class="tok-k">.muchi.alert</span> { <span class="tok-k">--row</span>:3; <span class="tok-k">--n</span>:6; <span class="tok-k">--dur</span>:0.60s }
<span class="tok-k">.muchi.angry</span> { <span class="tok-k">--row</span>:4; <span class="tok-k">--n</span>:6; <span class="tok-k">--dur</span>:0.54s }'''

PY_SNIP = '''<span class="tok-c"># muchi/mtg/sprites.py -- ya cableado en app.py</span>
<span class="tok-k">from</span> muchi.mtg <span class="tok-k">import</span> sprites

<span class="tok-c"># una vez por pagina: la hoja viaja incrustada en este CSS</span>
st.markdown(sprites.build_sprite_css(), unsafe_allow_html=<span class="tok-k">True</span>)

<span class="tok-c"># la mascota de la barra lateral, hablando mientras explica</span>
st.markdown(sprites.build_sprite_html(<span class="tok-s">"talk"</span>, scale=5), unsafe_allow_html=<span class="tok-k">True</span>)

<span class="tok-c"># y los avisos: Muchi los piensa, en vez de st.warning y compania</span>
<span class="tok-k">def</span> muchi_says(text, state=<span class="tok-s">"alert"</span>):
    st.markdown(sprites.build_notice_html(text, state), unsafe_allow_html=<span class="tok-k">True</span>)

total, got = len(orders), len(found_by_card)
<span class="tok-k">if</span> got == total:
    muchi_says(<span class="tok-s">f"Las encontre todas! {total} de {total} con precio."</span>, <span class="tok-s">"happy"</span>)
<span class="tok-k">else</span>:
    muchi_says(<span class="tok-s">f"Encontre {got} de {total}. Las {total - got} que faltan no estan."</span>, <span class="tok-s">"alert"</span>)'''

FILES = [
    ('muchi-kawaii-sheet.png', '32x32 &middot; 5 animaciones &middot; 8 columnas'),
    ('muchi-crema-sheet.png', 'la misma, en la paleta real de Muchi'),
    ('muchi-retro-sheet.png', '24x24 &middot; 5 animaciones &middot; 8 columnas'),
    ('muchi-dormida-sheet.png', '32x24 &middot; 3 animaciones &middot; 6 columnas'),
    ('muchi-sheets.json', 'fila, cantidad de frames y ms de cada estado'),
    ('gif/*.gif', 'un GIF por animacion, para previsualizar o reemplazar muchi.gif'),
    ('frames/*.png', 'cada frame suelto a 8x, por si quieres retocar a mano'),
    ('muchi.py / muchi_b.py / muchi_c.py', 'el sprite como grilla de caracteres: se edita y se regenera'),
    ('anim.py', 'genera hojas, GIFs y JSON: python anim.py'),
]


def build():
    parade = ''.join(
        f'<div class="mu" data-style="{s}" data-anim="{a}" style="--s:3"></div>'
        for s, a in (('muchi-kawaii', 'idle'), ('muchi-crema', 'idle'),
                     ('muchi-retro', 'idle'), ('muchi-dormida', 'sleep')))

    cards = []
    for i, (name, m) in enumerate(META['styles'].items()):
        title, desc = STYLE_INFO[name]
        anims = list(m['animations'].items())
        first = anims[0][0]
        chips = ''.join(
            f'<button class="chip" data-anim="{a}" aria-pressed="{"true" if a == first else "false"}">{LABELS[a]}</button>'
            for a, _ in anims)
        total = sum(v['frames'] for _, v in anims)
        cards.append(f'''<article class="card">
  <div class="stage"><div class="mu" id="stage{i}" data-style="{name}" data-anim="{first}" style="--s:5"></div></div>
  <div class="body">
    <h2>{title}</h2>
    <p class="desc">{desc}</p>
    <div class="chips" data-target="stage{i}">{chips}</div>
    <div class="meta"><span>frame <b>{m["frameWidth"]}&times;{m["frameHeight"]}</b></span>
      <span>animaciones <b>{len(anims)}</b></span>
      <span>frames <b>{total}</b></span>
      <span>hoja <b>{m["columns"] * m["frameWidth"]}&times;{len(anims) * m["frameHeight"]}</b></span></div>
  </div>
</article>''')

    sheets = []
    for name, m in META['styles'].items():
        uri = 'data:image/png;base64,' + b64(f'{OUT}/{name}-sheet@4x.png')
        order = ' &rarr; '.join(LABELS[a] for a in m['animations'])
        sheets.append(f'<div class="sheetbox"><p class="sheetcap">{name}-sheet.png &nbsp;&mdash;&nbsp; filas: {order}</p>'
                      f'<img src="{uri}" alt="Hoja de sprites de {name}"></div>')

    files = ''.join(f'<div class="file"><b>{n}</b><span>{d}</span></div>' for n, d in FILES)

    html = (PAGE.replace('__SHEETCSS__', sheet_css())
                .replace('__PARADE__', parade)
                .replace('__CARDS__', '\n'.join(cards))
                .replace('__SHEETS__', '\n'.join(sheets))
                .replace('__CSSSNIP__', CSS_SNIP)
                .replace('__PYSNIP__', PY_SNIP)
                .replace('__FILES__', files)
                .replace('__META__', json.dumps(META)))
    open(f'{OUT}/artifact.html', 'w').write(html)
    standalone = ('<!doctype html>\n<html lang="es">\n<head>\n<meta charset="utf-8">\n'
                  '<meta name="viewport" content="width=device-width,initial-scale=1">\n'
                  + html.split('<div class="wrap">')[0] +
                  '</head>\n<body>\n<div class="wrap">'
                  + html.split('<div class="wrap">', 1)[1] + '\n</body>\n</html>\n')
    open(f'{OUT}/demo.html', 'w').write(standalone)
    print('artifact.html', os.path.getsize(f'{OUT}/artifact.html'))
    print('demo.html', os.path.getsize(f'{OUT}/demo.html'))


if __name__ == '__main__':
    build()
