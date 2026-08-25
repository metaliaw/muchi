"""Muchi animada por sprite sheet, y sus avisos en burbuja de pensamiento.

El GIF de `mascot.py` sirve para una sola animacion: la que quedo horneada
adentro. Aca hay varias -- idle, hablando, feliz, alerta, molesta -- y todas
viven en un mismo PNG: una fila por estado, una columna por frame.

Se anima moviendo `background-position`, no con JS. Eso importa porque
Streamlit borra las etiquetas <script> de `st.markdown`, asi que cualquier
animacion por JS simplemente no correria; los `@keyframes`, en cambio, se
reproducen solos en cada rerun.

La hoja viaja como data URI por la misma razon que el GIF: Streamlit Cloud no
publica rutas locales del repo.

Los avisos van en burbuja de pensamiento en vez de `st.warning` y compania:
las cajas de Streamlit no son de Muchi, y con cuatro colores distintos peleaban
con la paleta. Aca el estado del sprite y el color del borde dicen lo mismo,
asi que el aviso se entiende antes de leerlo.

    from muchi.mtg import sprites

    st.markdown(sprites.build_sprite_css(), unsafe_allow_html=True)   # una vez
    st.markdown(sprites.build_sprite_html("idle"), unsafe_allow_html=True)
    st.markdown(sprites.build_notice_html("Las encontre todas!", "happy"),
                unsafe_allow_html=True)
"""
from __future__ import annotations

import base64
import html
import json
from functools import lru_cache

from muchi.paths import ASSETS

SPRITES = ASSETS / "muchi"
META = SPRITES / "muchi-sheets.json"

# El retro 8-bit es la mascota de la pagina. Las otras hojas siguen ahi:
# "muchi-crema" es la paleta real de la gata, "muchi-kawaii" la naranja chibi,
# "muchi-dormida" la enroscada. Se cambian con el argumento `style`.
DEFAULT_STYLE = "muchi-retro"
DEFAULT_STATE = "idle"
DEFAULT_SCALE = 4
NOTICE_SCALE = 3
# El tamano del sprite del clicker en la barra lateral. Vive aca para que el
# CSS del boton invisible y el sprite que lo tapa usen la misma cuenta.
CLICKER_SCALE = 5

# Que estado le toca a cada cosa que hace la app.
STATE_FOR = {
    "saludo": "talk",
    "buscando": "idle",
    "encontrado": "happy",
    "sin_stock": "alert",
    "error": "angry",
    "esperando": "idle",
}

# Y de que color va el borde de la burbuja segun el estado. Reusa los tokens de
# style.py: periwinkle es el color que la app ya usa para "esto esta bien".
TONE_FOR = {
    "idle": "calma",
    "talk": "calma",
    "happy": "bien",
    "alert": "ojo",
    "angry": "uy",
}


@lru_cache(maxsize=1)
def read_meta() -> dict:
    """Fila, cantidad de frames y milisegundos de cada estado."""
    if not META.exists():
        return {"styles": {}}
    return json.loads(META.read_text(encoding="utf-8"))


@lru_cache(maxsize=8)
def read_sheet_datauri(style: str = DEFAULT_STYLE) -> str | None:
    """La hoja como data URI. None si todavia no se genero."""
    path = SPRITES / f"{style}-sheet.png"
    if not path.exists():
        return None
    b64 = base64.b64encode(path.read_bytes()).decode("ascii")
    return f"data:image/png;base64,{b64}"


def available_states(style: str = DEFAULT_STYLE) -> list[str]:
    return list(read_meta().get("styles", {}).get(style, {}).get("animations", {}))


def build_sprite_css(style: str = DEFAULT_STYLE, scale: int = DEFAULT_SCALE) -> str:
    """El <style> completo: la hoja incrustada, una clase por estado y la burbuja.

    Devuelve cadena vacia si falta la hoja, asi la pagina no queda con un bloque
    de CSS apuntando a nada.
    """
    meta = read_meta().get("styles", {}).get(style)
    uri = read_sheet_datauri(style)
    if not meta or not uri:
        return ""

    fw, fh = meta["frameWidth"], meta["frameHeight"]
    cols, rows = meta["columns"], len(meta["animations"])
    states = "\n".join(
        f'.mu-sprite.mu-{name} {{ --row:{a["row"]}; --n:{a["frames"]}; '
        f'--dur:{a["frames"] * a["ms"] / 1000:.2f}s }}'
        for name, a in meta["animations"].items()
    )
    return f"""<style>
/* ---------- el sprite ---------- */
.mu-sprite {{
  --s:{scale}; --fw:{fw}; --fh:{fh}; --cols:{cols}; --rows:{rows};
  --row:0; --n:1; --dur:1s;
  width:calc(var(--fw) * var(--s) * 1px);
  height:calc(var(--fh) * var(--s) * 1px);
  background-image:url({uri});
  background-repeat:no-repeat;
  background-size:calc(var(--cols) * var(--fw) * var(--s) * 1px)
                  calc(var(--rows) * var(--fh) * var(--s) * 1px);
  background-position-y:calc(var(--row) * var(--fh) * var(--s) * -1px);
  image-rendering:pixelated;              /* nada de suavizado */
  animation:mu-play var(--dur) steps(var(--n)) infinite;
}}
@keyframes mu-play {{
  from {{ background-position-x:0 }}
  to   {{ background-position-x:calc(var(--n) * var(--fw) * var(--s) * -1px) }}
}}
{states}

.mu-gato .mu-sprite {{
  margin:0 auto; cursor:pointer; transition:transform .18s ease;
  filter:drop-shadow(0 4px 8px rgba(224,114,155,.28));
}}
.mu-gato:hover .mu-sprite {{ transform:translateY(-3px) scale(1.05); }}

/* ---------- el aviso: Muchi y su burbuja de pensamiento ---------- */
.mu-dice {{
  --mu-borde:var(--mu-rosa-cl);
  display:flex; align-items:flex-end; gap:22px; flex-wrap:wrap;
  margin:10px 0 18px;
}}
.mu-dice .mu-sprite {{
  flex:none; filter:drop-shadow(0 4px 8px rgba(224,114,155,.28));
}}
.mu-dice--bien {{ --mu-borde:var(--mu-peri); }}
.mu-dice--ojo  {{ --mu-borde:var(--mu-acento); }}
/* La paleta base no trae un rojo; este es el acento bajado en luz, lo justo
   para leerse como "algo salio mal" sin salirse de los pasteles. */
.mu-dice--uy   {{ --mu-borde:#B4506B; }}

.mu-piensa {{ position:relative; flex:1 1 260px; min-width:0; }}
.mu-nube {{
  position:relative; z-index:1;
  background:var(--mu-blanco); border:2px solid var(--mu-borde);
  border-radius:28px 32px 26px 30px;
  padding:12px 18px;
  font-family:'Quicksand',sans-serif; font-weight:600; font-size:.92rem;
  line-height:1.5; color:var(--mu-tinta);
  box-shadow:var(--mu-sombra-sw);
  animation:mu-piensa-entra .32s ease-out both;
}}
/* Las dos pelusas son lo que la vuelve un globo de pensamiento y no uno de
   dialogo: suben desde la cabeza de Muchi hasta la nube. Por eso la fila se
   alinea abajo -- asi la cabeza queda por encima del borde de la nube y las
   pelusas tienen por donde subir. */
.mu-pelusa {{
  position:absolute; z-index:0;
  background:var(--mu-blanco); border:2px solid var(--mu-borde);
  border-radius:50%; box-shadow:var(--mu-sombra-sw);
  animation:mu-piensa-entra .32s ease-out both;
}}
.mu-pelusa--g  {{ width:15px; height:15px; left:-16px; top:-9px;  animation-delay:.06s; }}
.mu-pelusa--ch {{ width:9px;  height:9px;  left:-29px; top:-21px; animation-delay:.12s; }}

@keyframes mu-piensa-entra {{
  from {{ opacity:0; transform:translateY(6px) scale(.94); }}
  to   {{ opacity:1; transform:none; }}
}}
.mu-dice--uy .mu-nube {{ animation:mu-piensa-entra .32s ease-out both, mu-tiembla .34s .3s; }}
@keyframes mu-tiembla {{
  0%,100% {{ transform:translateX(0); }}
  25%     {{ transform:translateX(-3px); }}
  75%     {{ transform:translateX(3px); }}
}}

/* ---------- el clicker: un boton invisible encima del sprite ---------- */
/* El boton (st.key="muchi_clicker") va primero en el DOM y el sprite se posa
   encima con pointer-events:none: asi el clic pasa de largo y le llega al
   boton, que es quien avisa a Streamlit. */
.st-key-muchi_clicker button, button.st-key-muchi_clicker {{
  display:block; width:calc({fw}px * {CLICKER_SCALE}); height:calc({fh}px * {CLICKER_SCALE});
  margin:0 auto; padding:0; border:none !important; border-radius:0 !important;
  background:transparent !important; box-shadow:none !important; opacity:0; cursor:pointer;
  overflow:hidden;
}}
.mu-clicker-sprite {{ margin-top:calc({fh}px * {CLICKER_SCALE} * -1); pointer-events:none; }}
.mu-clicker-sprite .mu-gato {{ padding:0; }}

@media (prefers-reduced-motion:reduce) {{
  .mu-sprite, .mu-nube, .mu-pelusa {{ animation:none; }}
}}
</style>"""


def build_sprite_html(state: str = DEFAULT_STATE, style: str = DEFAULT_STYLE,
                      scale: int | None = None, title: str = "Muchi") -> str:
    """El div del sprite. Sin hoja cae al emoji, igual que `mascot.build_cat_html`."""
    if read_sheet_datauri(style) is None:
        return '<div class="mu-gato" style="font-size:4rem">\U0001F431</div>'
    if state not in available_states(style):
        state = DEFAULT_STATE
    zoom = f' style="--s:{int(scale)}"' if scale else ""
    return (f'<div class="mu-gato"><div class="mu-sprite mu-{state}"{zoom} '
            f'role="img" aria-label="{html.escape(title, quote=True)}"></div></div>')


def build_notice_html(text: str, state: str = "alert",
                      style: str = DEFAULT_STYLE, scale: int = NOTICE_SCALE) -> str:
    """Muchi con el estado que corresponda, pensando el aviso.

    `text` puede traer markup simple (<b>, <br>): lo arma la app, no el usuario.
    """
    tone = TONE_FOR.get(state, "calma")
    if read_sheet_datauri(style) is None:
        # Sin hoja el aviso igual tiene que verse; cae al emoji.
        cat = '<div style="font-size:2.6rem;line-height:1">\U0001F431</div>'
    else:
        if state not in available_states(style):
            state = DEFAULT_STATE
        cat = f'<div class="mu-sprite mu-{state}" style="--s:{int(scale)}" aria-hidden="true"></div>'
    return (
        f'<div class="mu-dice mu-dice--{tone}" role="status">{cat}'
        f'<div class="mu-piensa">'
        f'<span class="mu-pelusa mu-pelusa--g"></span>'
        f'<span class="mu-pelusa mu-pelusa--ch"></span>'
        f'<div class="mu-nube">{text}</div>'
        f"</div></div>"
    )
