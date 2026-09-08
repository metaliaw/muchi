"""Guarda la Elección de Tema y Repinta la Paleta cuando se apaga la Luz.

La Identidad vive en [style](style.py) sobre Variables CSS: el Modo Oscuro no
reescribe Reglas, solo les cambia el Valor. Lo único que se repite acá son los
Lugares donde style escribió un Blanco literal en vez de la Variable.
"""
from __future__ import annotations

THEME_COOKIE = "muchi_tema"
DARK_THEME = "oscuro"
LIGHT_THEME = "claro"
# La Elección dura un Año: es una Preferencia, no una Sesión.
COOKIE_SECONDS = 365 * 24 * 60 * 60
# El Iframe que escribe la Cookie no dibuja nada, pero Streamlit le exige un
# Alto: se le da el mínimo y se lo esconde por su Contenedor.
COOKIE_SLOT = "muchi_cookie"
COOKIE_HEIGHT = 1

HIDDEN_CSS = f"<style>.st-key-{COOKIE_SLOT} {{ display:none; }}</style>"

DARK_CSS = """
<style>
:root {
  --mu-rosa-lav: #3B2A3B;
  --mu-niebla:   #262B3A;
  --mu-rosa:     #4A2E3F;
  --mu-rosa-cl:  #4C3547;
  --mu-papel:    #1D181F;
  --mu-tinta:    #F3E6EF;
  --mu-tinta-sw: #B7A4B3;
  --mu-acento:   #E0729B;
  --mu-peri:     #93A6DC;
  --mu-blanco:   #2A2230;
  --mu-sombra:   0 6px 20px rgba(0, 0, 0, .45);
  --mu-sombra-sw:0 3px 12px rgba(0, 0, 0, .35);
}

/* Los Blancos que style escribió literales, uno por uno. */
.stTextInput input, .stTextArea textarea, .stNumberInput input,
.stTabs [data-baseweb="tab"], .stTabs [data-testid="stTab"],
[data-testid="stExpander"] { background: var(--mu-blanco) !important; }
.mu-card.mejor { background: linear-gradient(180deg, var(--mu-niebla) 0%, var(--mu-blanco) 65%); }
.mu-pill.foil { background:#4A3D14; color:#F0D98A; }
.mu-pill.cond { background:#3A323C; color:#D3C6D0; }
.mu-pill.tienda { background:var(--mu-rosa-lav); color:#F0BDD1; }
.mu-pill.mejor, .mu-pill.particular { color:#C3D0F2; }

/* La Tabla y los Avisos son de Streamlit: no pasan por las Variables. */
[data-testid="stDataFrame"], [data-testid="stAlert"] { color: var(--mu-tinta); }
input, textarea { color: var(--mu-tinta) !important; }
</style>
"""


def read_theme_choice(cookies) -> str:
    """Lee la Elección guardada. Sin Cookie, Muchi abre con la Luz encendida."""
    return DARK_THEME if cookies.get(THEME_COOKIE) == DARK_THEME else LIGHT_THEME


def name_theme(dark: bool) -> str:
    return DARK_THEME if dark else LIGHT_THEME


def paint_theme(dark: bool) -> str:
    return HIDDEN_CSS + (DARK_CSS if dark else "")


def build_cookie_script(theme: str) -> str:
    """Escribe la Cookie en la Página, no en el Iframe del Componente.

    Streamlit no expone Escritura de Cookies: el Componente corre en un iframe
    del mismo Origen y alcanza el `document` de arriba. Si el Navegador lo
    impide, la Elección sigue viva en la Sesión y solo se pierde al volver.
    """
    return (
        "<script>try{window.parent.document.cookie="
        f'"{THEME_COOKIE}={theme};path=/;max-age={COOKIE_SECONDS};SameSite=Lax";'
        "}catch(e){}</script>"
    )
