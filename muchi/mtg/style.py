"""Identidad visual de Muchi.

Paleta base elegida (los 5 pasteles) + dos derivados que la paleta no trae:
una tinta oscura y un acento saturado. Sin ellos no hay contraste legible:
cinco pasteles claros no pueden sostener texto ni jerarquia por si solos.

  #FDC9DA  rosa lavanda    #E9EDF6  niebla azulada    #FDBFD3  rosa
  #FFCFE2  rosa claro      #FDEEF5  rosa papel
  derivados -> #6E5A68 tinta   #E0729B acento   #7B8FC7 periwinkle
"""

CSS = """
<style>
@import url('https://fonts.googleapis.com/css2?family=Baloo+2:wght@500;700;800&family=Quicksand:wght@400;500;600;700&display=swap');

:root {
  --mu-rosa-lav: #FDC9DA;
  --mu-niebla:   #E9EDF6;
  --mu-rosa:     #FDBFD3;
  --mu-rosa-cl:  #FFCFE2;
  --mu-papel:    #FDEEF5;
  --mu-tinta:    #6E5A68;
  --mu-tinta-sw: #A08F9C;
  --mu-acento:   #E0729B;
  --mu-peri:     #7B8FC7;
  --mu-blanco:   #FFFFFF;
  --mu-sombra:   0 6px 20px rgba(224, 114, 155, .16);
  --mu-sombra-sw:0 3px 12px rgba(123, 143, 199, .12);
}

html, body, [class*="css"], .stMarkdown, .stTextInput, .stTextArea, .stSelectbox {
  font-family: 'Quicksand', ui-rounded, system-ui, sans-serif;
}
h1, h2, h3, h4 { font-family: 'Baloo 2', 'Quicksand', sans-serif !important; color: var(--mu-tinta); }

.stApp {
  background:
    radial-gradient(circle at 10% 6%,  var(--mu-niebla)   0%, transparent 40%),
    radial-gradient(circle at 90% 3%,  var(--mu-rosa-lav) 0%, transparent 38%),
    radial-gradient(circle at 50% 97%, var(--mu-rosa-cl)  0%, transparent 42%),
    var(--mu-papel);
}

/* ---------- encabezado ---------- */
.mu-hero {
  background: linear-gradient(120deg, var(--mu-rosa-lav) 0%, var(--mu-niebla) 48%, var(--mu-rosa) 100%);
  border: 3px solid var(--mu-blanco);
  border-radius: 30px;
  padding: 24px 30px;
  box-shadow: var(--mu-sombra);
  display: flex; align-items: center; gap: 18px;
}
.mu-hero .emoji { font-size: 3rem; line-height: 1; filter: drop-shadow(0 3px 5px rgba(110,90,104,.18)); }
.mu-hero h1 { margin: 0; font-size: 2.6rem; font-weight: 800; letter-spacing: -1px; color: #fff;
              text-shadow: 0 2px 8px rgba(224,114,155,.45); }
.mu-hero p  { margin: 4px 0 0; color: var(--mu-tinta); font-weight: 700; font-size: .95rem; opacity: .85; }

/* ---------- tarjeta de oferta ---------- */
.mu-card {
  background: var(--mu-blanco);
  border: 2px solid var(--mu-rosa-cl);
  border-radius: 22px;
  padding: 14px 18px;
  box-shadow: var(--mu-sombra-sw);
  transition: transform .15s ease, box-shadow .15s ease;
}
.mu-card:hover { transform: translateY(-2px); box-shadow: var(--mu-sombra); }
.mu-card.mejor { border-color: var(--mu-peri); background: linear-gradient(180deg, var(--mu-niebla) 0%, #fff 65%); }

.mu-nombre { font-family:'Baloo 2',sans-serif; font-size:1.06rem; font-weight:700; color:var(--mu-tinta); }
.mu-sub    { font-size:.8rem; color:var(--mu-tinta-sw); font-weight:600; }
.mu-precio { font-family:'Baloo 2',sans-serif; font-size:1.55rem; font-weight:800; color:var(--mu-acento); line-height:1; }
.mu-precio.mejor { color: var(--mu-peri); }

/* ---------- pastillas ---------- */
.mu-pill { display:inline-block; padding:3px 11px; border-radius:999px;
           font-size:.71rem; font-weight:700; margin:0 5px 3px 0; white-space:nowrap; }
.mu-pill.tienda { background: var(--mu-rosa-lav); color:#8A4C66; }
.mu-pill.foil   { background: #FFF0C2;            color:#8A6A11; }
.mu-pill.mejor  { background: var(--mu-niebla);   color:#4A5F96; }
.mu-pill.cond   { background: #F4F1F5;            color:#7C6C78; }
/* Particular del marketplace: en niebla, no en rosa, para que se lea distinto
   de una tienda establecida sin gritar "peligro". */
.mu-pill.particular { background: var(--mu-niebla); color:#5B6B96;
                      border:1px dashed #A8B6D8; }

/* ---------- tiles ---------- */
.mu-tile { background:var(--mu-blanco); border:2px solid var(--mu-rosa-cl); border-radius:22px;
           padding:14px 16px; text-align:center; box-shadow:var(--mu-sombra-sw); height:100%; }
.mu-tile .et  { font-size:.72rem; font-weight:700; color:var(--mu-tinta-sw);
                text-transform:uppercase; letter-spacing:.7px; }
.mu-tile .val { font-family:'Baloo 2',sans-serif; font-size:1.65rem; font-weight:800;
                color:var(--mu-tinta); line-height:1.2; }
.mu-tile.ok { border-color: var(--mu-peri); }
.mu-tile.ok .val { color: var(--mu-peri); }

/* ---------- encabezado de tienda en el carrito ---------- */
.mu-tienda-hd {
  background: linear-gradient(120deg, var(--mu-rosa-cl), var(--mu-niebla));
  border-radius: 999px; padding: 8px 18px; margin: 4px 0 10px;
  font-family:'Baloo 2',sans-serif; font-weight:800; color:var(--mu-tinta); font-size:1.05rem;
}

/* ---------- widgets ---------- */
.stButton > button, .stLinkButton > a, .stDownloadButton > button {
  border-radius:999px !important; font-weight:700 !important;
  border:2px solid var(--mu-rosa-cl) !important; color:var(--mu-tinta) !important;
  transition:all .15s ease;
}
.stButton > button[kind="primary"], .stDownloadButton > button {
  background:linear-gradient(120deg, var(--mu-acento), var(--mu-rosa-lav)) !important;
  color:#fff !important; border:none !important;
}
.stButton > button:hover, .stLinkButton > a:hover, .stDownloadButton > button:hover {
  transform:translateY(-1px); box-shadow:var(--mu-sombra);
}
.stTextInput input, .stTextArea textarea, .stNumberInput input {
  border-radius:16px !important; border:2px solid var(--mu-rosa-cl) !important; background:#fff !important;
}
.stTextInput input:focus, .stTextArea textarea:focus { border-color:var(--mu-acento) !important; }

/* Streamlit >=1.60 dejo de exponer data-baseweb="tab"; se apunta a ambos. */
.stTabs [data-baseweb="tab-list"], .stTabs [role="tablist"] { gap:8px; background:transparent; border-bottom:none; }
.stTabs [data-baseweb="tab"],
.stTabs [data-testid="stTab"] { border-radius:999px !important; padding:8px 22px !important; background:#fff;
  border:2px solid var(--mu-rosa-cl); font-weight:700; color:var(--mu-tinta-sw); }
.stTabs [data-testid="stTab"] [data-testid="stMarkdownContainer"] p { font-weight:700; margin:0; }
.stTabs [data-testid="stTab"]::after { display:none !important; }  /* subrayado nativo */
.stTabs [aria-selected="true"] {
  background:linear-gradient(120deg, var(--mu-acento), var(--mu-rosa-lav)) !important;
  color:#fff !important; border-color:transparent !important;
}
/* ---------- Muchi, el gatito ---------- */
.mu-gato { position:relative; text-align:center; padding:4px 0 0; }
.mu-gato img {
  width:132px; height:132px; image-rendering:pixelated;   /* nada de suavizado */
  cursor:pointer; transition:transform .18s ease;
  filter: drop-shadow(0 4px 8px rgba(224,114,155,.28));
}
.mu-gato:hover img { transform:translateY(-3px) scale(1.05); }

/* globo de dialogo */
.mu-globo {
  position:relative; background:var(--mu-blanco);
  border:2px solid var(--mu-rosa-cl); border-radius:18px;
  padding:10px 14px; margin:6px 4px 10px;
  font-family:'Quicksand',sans-serif; font-weight:600; font-size:.9rem;
  color:var(--mu-tinta); box-shadow:var(--mu-sombra-sw);
}
.mu-globo::after {
  content:''; position:absolute; top:-9px; left:50%; margin-left:-8px;
  border-left:8px solid transparent; border-right:8px solid transparent;
  border-bottom:9px solid var(--mu-rosa-cl);
}

/* corazoncitos que suben al apretarla */
.mu-corazones { position:relative; height:0; pointer-events:none; }
.mu-corazon {
  position:absolute; bottom:0; font-size:1.1rem; opacity:0;
  animation: mu-flota 2.1s ease-out forwards;
}
@keyframes mu-flota {
  0%   { transform:translateY(0) scale(.5) rotate(0deg);      opacity:0; }
  12%  { opacity:1; }
  70%  { opacity:1; }
  100% { transform:translateY(-120px) scale(1.15) rotate(18deg); opacity:0; }
}

/* ---------- fila interna de las tarjetas ---------- */
.mu-fila { display:flex; align-items:center; gap:14px; flex-wrap:wrap; }
.mu-izq  { flex:1; min-width:240px; }

.mu-btn {
  display:inline-block; padding:8px 20px; border-radius:999px;
  font-family:'Quicksand',sans-serif; font-weight:700; font-size:.86rem;
  text-decoration:none !important; white-space:nowrap;
  background:linear-gradient(120deg, var(--mu-acento), var(--mu-rosa-lav));
  color:#fff !important; box-shadow:var(--mu-sombra-sw); transition:all .15s ease;
}
.mu-btn:hover { transform:translateY(-1px); box-shadow:var(--mu-sombra); }

.stProgress > div > div > div > div { background:linear-gradient(90deg, var(--mu-acento), var(--mu-peri)); }
[data-testid="stExpander"] { border-radius:20px; border:2px solid var(--mu-rosa-cl); background:#fff; }
[data-testid="stSidebar"] { background: linear-gradient(180deg, var(--mu-rosa-cl) 0%, var(--mu-niebla) 100%); }
#MainMenu, footer { visibility:hidden; }
</style>
"""


def paint_hero(title: str, subtitle: str, emoji: str = "\U0001F431") -> str:
    return (f'<div class="mu-hero"><div class="emoji">{emoji}</div>'
            f'<div><h1>{title}</h1><p>{subtitle}</p></div></div>')


def paint_tile(label: str, value: str, ok: bool = False) -> str:
    css_class = "mu-tile ok" if ok else "mu-tile"
    return (f'<div class="{css_class}"><div class="et">{label}</div>'
            f'<div class="val">{value}</div></div>')


def format_clp(n) -> str:
    """1234567 -> '$1.234.567'. El separador chileno es el punto."""
    return "$" + format_thousands(n)


def format_thousands(n) -> str:
    return f"{int(n):,}".replace(",", ".")


def paint_offer(o, best: bool = False) -> str:
    """Una oferta: quien la vende, en que estado y a cuanto."""
    seller = "particular" if o.marketplace else "tienda"
    card_class = "mu-card mejor" if best else "mu-card"
    price = "mu-precio mejor" if best else "mu-precio"

    pills = f'<span class="mu-pill {seller}">{o.store}</span>'
    if o.marketplace:
        pills += '<span class="mu-pill particular">particular</span>'
    if o.is_foil:
        pills += '<span class="mu-pill foil">Foil</span>'
    if o.condition:
        pills += f'<span class="mu-pill cond">{o.condition}</span>'
    if best:
        pills += '<span class="mu-pill mejor">\U0001F43E el mas barato</span>'

    return (
        f'<div class="{card_class}"><div class="mu-fila">'
        f'<div class="mu-izq"><div class="mu-nombre">{o.title}</div>'
        f'<div style="margin-top:6px">{pills}</div></div>'
        f'<div style="text-align:right">'
        f'<div class="{price}">{format_clp(o.price_clp)}</div>'
        f'</div>'
        f'<a class="mu-btn" href="{o.url}" target="_blank" rel="noopener">Ver</a>'
        f"</div></div>"
    )


def paint_recommendation(r) -> str:
    """Una carta sugerida: su categoria, su sinergia y que tan comun es."""
    return (
        f'<div class="mu-card"><div class="mu-fila">'
        f'<div class="mu-izq"><div class="mu-nombre">{r.name}</div>'
        f'<div style="margin-top:6px">'
        f'<span class="mu-pill tienda">{r.category}</span>'
        f'<span class="mu-pill cond">sinergia {r.synergy:+.2f}</span>'
        f'</div></div>'
        f'<div style="text-align:right">'
        f'<div class="mu-precio">{r.inclusion_pct:.0f}%</div>'
        f'<div class="mu-sub">de los mazos</div></div>'
        f"</div></div>"
    )


def paint_store_header(store: str, cards: int, subtotal: int, shipping: int) -> str:
    return (f'<div class="mu-tienda-hd">\U0001F43E {store} &middot; {cards} cartas '
            f"&middot; {format_clp(subtotal)} + {format_clp(shipping)} envio</div>")


def paint_line(line) -> str:
    """Una linea del carrito: cuantas copias, de donde y a cuanto la unidad."""
    return (
        f'<div class="mu-card"><div class="mu-fila">'
        f'<div class="mu-izq">'
        f'<div class="mu-nombre">{line.quantity}x {line.card_name}</div>'
        f'<div class="mu-sub">{line.title}</div></div>'
        f'<div style="text-align:right">'
        f'<div class="mu-precio">{format_clp(line.subtotal)}</div>'
        f'<div class="mu-sub">{format_clp(line.unit_price)} c/u</div></div>'
        f'<a class="mu-btn" href="{line.url}" target="_blank" rel="noopener">Comprar</a>'
        f"</div></div>"
    )


def paint_store(status) -> str:
    """Una tienda indexable, con lo que el indice local sabe de ella."""
    if status.indexed:
        detail = (f'{format_thousands(status.offers)} ofertas de '
                  f'{format_thousands(status.products)} productos '
                  f'&middot; {status.updated}')
    else:
        detail = "sin indexar"

    return (
        f'<div class="mu-card"><div class="mu-fila"><div class="mu-izq">'
        f'<div class="mu-nombre">{status.store}</div>'
        f'<div class="mu-sub">{detail}</div></div>'
        f'<a class="mu-btn" href="{status.url}" target="_blank" rel="noopener">Ir</a>'
        f"</div></div>"
    )


def paint_blocked(store: str, url: str, reason: str) -> str:
    """Una tienda que Muchi no puede consultar, y por que."""
    return (
        f'<div class="mu-card"><div class="mu-fila"><div class="mu-izq">'
        f'<div class="mu-nombre">{store}</div>'
        f'<div class="mu-sub">{reason}</div></div>'
        f'<a class="mu-btn" href="{url}" target="_blank" rel="noopener">Buscar ahi</a>'
        f"</div></div>"
    )
