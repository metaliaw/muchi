"""Muchi - buscador kawaii de cartas Magic en tiendas chilenas."""
from __future__ import annotations

import pandas as pd
import streamlit as st

from mtgcl import db, decklist, estilo, optimizer
from mtgcl.http import PoliteSession
from mtgcl.models import Offer
from mtgcl.sources import scry

GATO = "\U0001F431"    # cara de gato
HUELLA = "\U0001F43E"  # huellitas
PEZ = "\U0001F41F"

st.set_page_config(page_title="Muchi", page_icon=GATO, layout="wide")
st.markdown(estilo.CSS, unsafe_allow_html=True)


def clp(n) -> str:
    return "$" + f"{int(n):,}".replace(",", ".")


@st.cache_resource
def sesion() -> PoliteSession:
    # 1.5s entre requests al mismo host: scry ya nos tiro un 429 durante el diseno.
    return PoliteSession(min_interval=1.5)


@st.cache_resource
def base():
    return db.conectar()


@st.cache_data(ttl=1800, show_spinner=False)
def buscar(nombre: str):
    return scry.buscar_carta(sesion(), nombre)


@st.cache_data(ttl=600, show_spinner=False)
def sugerencias(q: str):
    try:
        return scry.autocomplete(sesion(), q)
    except Exception:
        return []


def filtrar(ofertas, acabado: str, tiendas):
    out = ofertas
    if acabado == "Solo normal":
        out = [o for o in out if not o.es_foil]
    elif acabado == "Solo foil":
        out = [o for o in out if o.es_foil]
    if tiendas:
        out = [o for o in out if o.store in tiendas]
    return sorted(out, key=lambda o: o.price_clp)


def tarjeta_oferta(o: Offer, mejor: bool = False) -> str:
    pills = f'<span class="mu-pill tienda">{o.store}</span>'
    if o.es_foil:
        pills += '<span class="mu-pill foil">Foil</span>'
    if o.condition:
        pills += f'<span class="mu-pill cond">{o.condition}</span>'
    if mejor:
        pills += f'<span class="mu-pill mejor">{HUELLA} el mas barato</span>'
    clase = "mu-card mejor" if mejor else "mu-card"
    cp = "mu-precio mejor" if mejor else "mu-precio"
    return (
        f'<div class="{clase}"><div class="mu-fila">'
        f'<div class="mu-izq"><div class="mu-nombre">{o.title}</div>'
        f'<div style="margin-top:6px">{pills}</div></div>'
        f'<div style="text-align:right"><div class="{cp}">{clp(o.price_clp)}</div></div>'
        f'<a class="mu-btn" href="{o.url}" target="_blank" rel="noopener">Ver</a>'
        f"</div></div>"
    )


st.markdown(
    estilo.hero("Muchi", "Tu gatito buscador de cartas Magic en tiendas chilenas", GATO),
    unsafe_allow_html=True,
)
st.write("")

with st.sidebar:
    st.markdown(f"### {HUELLA} Preferencias")
    acabado = st.radio("Acabado", ["Todos", "Solo normal", "Solo foil"], index=0)
    envio = st.number_input(
        "Costo de envio por tienda (CLP)", 0, 20000, 4000, step=500,
        help="Muchi lo usa para decidir si conviene concentrar la compra en menos tiendas.",
    )
    st.divider()
    st.caption(
        "Precios via **scry.cl**, que indexa ~30 tiendas chilenas. "
        "Verifica edicion, estado y stock en la tienda antes de pagar."
    )

tab_buscar, tab_lista, tab_carrito = st.tabs(
    [f"{PEZ} Buscar", "Mi lista", "Carrito"]
)

# ------------------------------------------------------------------ buscar
with tab_buscar:
    consulta = st.text_input(
        "Que carta buscas?", placeholder="Ej: Sol Ring, Ragavan...", key="q_simple"
    )

    elegida = None
    if consulta and len(consulta) >= 3:
        opciones = sugerencias(consulta)
        if opciones and consulta not in opciones:
            st.caption("Quisiste decir?")
            cols = st.columns(min(len(opciones), 5))
            for col, nombre in zip(cols, opciones[:5]):
                if col.button(nombre, key=f"sug_{nombre}", use_container_width=True):
                    st.session_state["carta_elegida"] = nombre
                    st.session_state["carta_elegida_q"] = consulta
        # La sugerencia elegida solo vale para la consulta que la genero:
        # si el usuario escribe otra cosa, volvemos a su texto.
        if st.session_state.get("carta_elegida_q") == consulta:
            elegida = st.session_state.get("carta_elegida") or consulta
        else:
            elegida = consulta

    if elegida:
        with st.spinner(f"Muchi esta olfateando {elegida}..."):
            try:
                card_id, ofertas = buscar(elegida)
            except Exception as e:
                st.error(f"No pude consultar scry.cl: {e}")
                card_id, ofertas = None, []

        if ofertas:
            db.guardar(base(), elegida, ofertas)

        tiendas_disp = sorted({o.store for o in ofertas})
        sel = st.multiselect("Filtrar tiendas", tiendas_disp, default=[], key="f_tiendas")
        visibles = filtrar(ofertas, acabado, sel)

        if not visibles:
            st.info("Sin stock con esos filtros. Prueba el refresco en vivo mas abajo.")
        else:
            c1, c2, c3 = st.columns(3)
            c1.markdown(
                estilo.tile("Mas barato", clp(visibles[0].price_clp), ok=True),
                unsafe_allow_html=True,
            )
            c2.markdown(estilo.tile("Ofertas", str(len(visibles))), unsafe_allow_html=True)
            c3.markdown(
                estilo.tile("Tiendas", str(len({o.store for o in visibles}))),
                unsafe_allow_html=True,
            )
            st.write("")
            for i, o in enumerate(visibles):
                st.markdown(tarjeta_oferta(o, mejor=(i == 0)), unsafe_allow_html=True)

        if card_id and st.button("Refrescar en vivo (consulta las 30 tiendas)", key="refresh"):
            barra = st.progress(0.0, text="Conectando...")
            try:
                for ev in scry.refrescar(sesion(), card_id):
                    total, hechas = ev.get("total") or 30, ev.get("done") or 0
                    barra.progress(
                        min(hechas / total, 1.0),
                        text=f"Consultando {ev.get('store', '...')} ({hechas}/{total})",
                    )
                barra.progress(1.0, text="Listo")
                buscar.clear()
                st.rerun()
            except Exception as e:
                st.warning(f"El refresco fallo: {e}")

        hist = db.historico(base(), elegida)
        if len(hist) > 1:
            with st.expander("Historial de precios"):
                df = pd.DataFrame([dict(r) for r in hist])
                df["ts"] = pd.to_datetime(df["ts"])
                st.line_chart(df.set_index("ts")[["minimo", "promedio"]])

# ------------------------------------------------------------------ mi lista
with tab_lista:
    st.markdown("#### Pega tu mazo y Muchi busca todo")
    texto = st.text_area(
        "Una carta por linea",
        height=220,
        placeholder="4 Lightning Bolt\n1 Ragavan, Nimble Pilferer\n2x Sol Ring\nCounterspell",
        key="decklist",
    )

    pedidos, ignoradas = decklist.parse(texto or "")
    if pedidos:
        total_copias = sum(p.cantidad for p in pedidos)
        st.caption(f"**{len(pedidos)}** cartas distintas - **{total_copias}** copias")
    if ignoradas:
        st.warning("No entendi estas lineas: " + " / ".join(ignoradas[:5]))

    if pedidos and st.button("Buscar precios de la lista", type="primary"):
        barra = st.progress(0.0, text="Empezando...")
        encontrado = {}
        fallidas = []

        for i, p in enumerate(pedidos):
            barra.progress(i / len(pedidos), text=f"Buscando {p.nombre}...")
            try:
                _, ofertas = buscar(p.nombre)
                if ofertas:
                    encontrado[p.nombre.lower()] = ofertas
                    db.guardar(base(), p.nombre, ofertas)
            except Exception:
                fallidas.append(p.nombre)

        barra.progress(1.0, text="Listo")
        st.session_state["ofertas_lista"] = encontrado
        st.session_state["pedidos"] = pedidos
        if fallidas:
            st.warning("No pude consultar: " + ", ".join(fallidas))
        st.success(
            f"Encontre precios para {len(encontrado)} de {len(pedidos)} cartas. "
            "Anda a la pestana Carrito."
        )

# ------------------------------------------------------------------ carrito
with tab_carrito:
    pedidos = st.session_state.get("pedidos")
    crudas = st.session_state.get("ofertas_lista")

    if not pedidos or not crudas:
        st.info("Primero carga una lista en la pestana Mi lista.")
    else:
        ofertas_por_carta = {k: filtrar(v, acabado, None) for k, v in crudas.items()}
        ofertas_por_carta = {k: v for k, v in ofertas_por_carta.items() if v}

        estrategia = st.radio(
            "Como armamos el carrito?",
            ["Minimizar total (cartas + envios)", "Cada carta en su tienda mas barata"],
            horizontal=True,
        )

        if estrategia.startswith("Minimizar"):
            plan = optimizer.plan_optimo(pedidos, ofertas_por_carta, envio)
        else:
            plan = optimizer.plan_mas_barato(pedidos, ofertas_por_carta, envio)

        ingenuo = optimizer.plan_mas_barato(pedidos, ofertas_por_carta, envio)
        ahorro = ingenuo.total - plan.total

        c1, c2, c3, c4 = st.columns(4)
        c1.markdown(estilo.tile("Total", clp(plan.total), ok=True), unsafe_allow_html=True)
        c2.markdown(estilo.tile("Cartas", clp(plan.costo_cartas)), unsafe_allow_html=True)
        c3.markdown(
            estilo.tile(f"Envios ({len(plan.tiendas)})", clp(plan.costo_envios)),
            unsafe_allow_html=True,
        )
        c4.markdown(
            estilo.tile("Ahorro vs ingenuo", clp(max(ahorro, 0)), ok=ahorro > 0),
            unsafe_allow_html=True,
        )
        st.write("")

        if plan.faltantes:
            st.warning("Sin stock en ninguna tienda: " + ", ".join(plan.faltantes))

        por_tienda: dict[str, list] = {}
        for linea in plan.lineas:
            por_tienda.setdefault(linea.tienda, []).append(linea)

        orden = sorted(por_tienda, key=lambda t: -sum(x.subtotal for x in por_tienda[t]))
        for tienda in orden:
            lineas = sorted(por_tienda[tienda], key=lambda x: -x.subtotal)
            sub = sum(x.subtotal for x in lineas)
            st.markdown(
                f'<div class="mu-tienda-hd">{HUELLA} {tienda} &middot; {len(lineas)} cartas '
                f"&middot; {clp(sub)} + {clp(envio)} envio</div>",
                unsafe_allow_html=True,
            )
            for linea in lineas:
                st.markdown(
                    f'<div class="mu-card"><div class="mu-fila">'
                    f'<div class="mu-izq">'
                    f'<div class="mu-nombre">{linea.cantidad}x {linea.carta}</div>'
                    f'<div class="mu-sub">{linea.titulo}</div></div>'
                    f'<div style="text-align:right">'
                    f'<div class="mu-precio">{clp(linea.subtotal)}</div>'
                    f'<div class="mu-sub">{clp(linea.precio_unitario)} c/u</div></div>'
                    f'<a class="mu-btn" href="{linea.url}" target="_blank" '
                    f'rel="noopener">Comprar</a>'
                    f"</div></div>",
                    unsafe_allow_html=True,
                )

        df = pd.DataFrame(
            [
                {
                    "carta": x.carta,
                    "cantidad": x.cantidad,
                    "tienda": x.tienda,
                    "precio_unitario": x.precio_unitario,
                    "subtotal": x.subtotal,
                    "url": x.url,
                }
                for x in plan.lineas
            ]
        )
        st.download_button(
            "Descargar carrito (CSV)",
            df.to_csv(index=False).encode("utf-8"),
            "carrito-muchi.csv",
            "text/csv",
        )
