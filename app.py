"""Muchi - buscador kawaii de cartas Magic en tiendas chilenas."""
from __future__ import annotations

import json
from pathlib import Path

import pandas as pd
import streamlit as st

from mtgcl import catalogo, db, decklist, estilo, optimizer
from mtgcl.http import PoliteSession
from mtgcl.models import Offer
from mtgcl.sources import api_tienda, edhrec, moxfield, scry, shopify

INVENTARIOS = Path(__file__).resolve().parent / "inventarios-moxfield.json"


def cargar_inventarios():
    """Listas de Moxfield usadas como catalogo. Opcional: sin archivo, nada."""
    if not INVENTARIOS.exists():
        return None, []
    cfg = json.loads(INVENTARIOS.read_text(encoding="utf-8"))
    tienda = cfg.get("tienda") or "Inventario Moxfield"
    listas = [
        moxfield.Inventario(tienda, l["etiqueta"], moxfield.id_de_url(l["url"]),
                            int(l.get("tasa", 700)))
        for l in cfg.get("listas", [])
    ]
    return tienda, listas

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
def buscar_en_scry(nombre: str):
    return scry.buscar_carta(sesion(), nombre)


def buscar(nombre: str):
    """scry + las tiendas indexadas localmente, deduplicado por URL.

    Si una tienda esta en scry y ademas indexada directo, la version de scry
    manda: es la que trae el precio ya normalizado por su pipeline.
    """
    card_id, ofertas = buscar_en_scry(nombre)
    # Se descarta por tienda, no por URL: si scry ya cubre esa tienda para esta
    # carta, duplicar sus ofertas desde el indice local solo inflaria el conteo.
    cubiertas = {o.store for o in ofertas}
    locales = [o for o in catalogo.buscar(base(), nombre) if o.store not in cubiertas]

    # Tiendas que exponen una API de solo lectura (ver INTEGRAR-TIENDA.md).
    # Si una falla no puede tumbar la busqueda entera.
    por_api: list[Offer] = []
    for tienda in api_tienda.cargar():
        if tienda.nombre in cubiertas:
            continue
        try:
            por_api += api_tienda.buscar(sesion(), tienda, nombre)
        except Exception:
            continue

    return card_id, sorted(ofertas + locales + por_api, key=lambda o: o.price_clp)


@st.cache_data(ttl=600, show_spinner=False)
def sugerencias(q: str):
    try:
        return scry.autocomplete(sesion(), q)
    except Exception:
        return []


def filtrar(ofertas, acabado: str, tiendas, solo_tiendas: bool = True):
    out = ofertas
    if solo_tiendas:
        # Deja fuera a los particulares de marketplace.scry.cl
        out = [o for o in out if not o.marketplace]
    if acabado == "Solo normal":
        out = [o for o in out if not o.es_foil]
    elif acabado == "Solo foil":
        out = [o for o in out if o.es_foil]
    if tiendas:
        out = [o for o in out if o.store in tiendas]
    return sorted(out, key=lambda o: o.price_clp)


def tarjeta_oferta(o: Offer, mejor: bool = False) -> str:
    clase_vendedor = "particular" if o.marketplace else "tienda"
    pills = f'<span class="mu-pill {clase_vendedor}">{o.store}</span>'
    if o.marketplace:
        pills += '<span class="mu-pill particular">particular</span>'
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

# Va arriba de las pestanas a proposito: afecta a las tres (busqueda, lista y
# el total del carrito), asi que escondido en la barra lateral cambiaba los
# resultados sin que se viera desde donde.
col_filtro, col_nota = st.columns([2, 3])
with col_filtro:
    incluir_particulares = st.checkbox(
        "Incluir vendedores particulares",
        value=False,
        help="Las tiendas establecidas despachan desde su propio sitio. Los "
             "particulares venden por el marketplace de scry.cl: suelen ser mas "
             "baratos, pero son personas, no locales.",
    )
solo_tiendas = not incluir_particulares
with col_nota:
    st.caption(
        "Mostrando tiendas y particulares de scry.cl"
        if incluir_particulares else
        "Mostrando solo tiendas establecidas"
    )

tab_buscar, tab_lista, tab_comandante, tab_carrito, tab_tiendas = st.tabs(
    [f"{PEZ} Buscar", "Mi lista", "Comandante", "Carrito", "Tiendas"]
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

        # El filtro de marketplace va primero: el multiselect no debe ofrecer
        # tiendas que despues quedarian excluidas igual.
        elegibles = [o for o in ofertas if not (solo_tiendas and o.marketplace)]
        tiendas_disp = sorted({o.store for o in elegibles})
        sel = st.multiselect("Filtrar tiendas", tiendas_disp, default=[], key="f_tiendas")
        visibles = filtrar(ofertas, acabado, sel, solo_tiendas)

        ocultas = len(ofertas) - len(elegibles)
        if ocultas:
            st.caption(f"Hay {ocultas} ofertas de particulares ocultas. "
                       "Marca la casilla de arriba para verlas.")

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
                buscar_en_scry.clear()
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

    # La pestana Comandante deja aca lo que querés sumar. Se mezcla ANTES de
    # crear el textarea: Streamlit no deja tocar la session_state de un widget
    # una vez instanciado, y asi el texto sigue siendo la unica fuente de verdad.
    if st.session_state.get("_sumar_al_mazo"):
        extra = st.session_state.pop("_sumar_al_mazo")
        previo = st.session_state.get("decklist", "") or ""
        st.session_state["decklist"] = (previo.rstrip() + "\n" + extra).strip()

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

# ------------------------------------------------------------------ comandante
@st.cache_data(ttl=86400, show_spinner=False)
def recomendaciones(comandante: str):
    return edhrec.recomendaciones(sesion(), comandante)


with tab_comandante:
    st.markdown("#### Que le falta a tu mazo")
    st.caption("Muchi le pregunta a EDHREC que juega la gente con ese comandante "
               "y descuenta lo que ya tenes en Mi lista.")

    comandante = st.text_input(
        "Tu comandante", placeholder="Ej: Atraxa, Praetors' Voice", key="comandante"
    )

    if comandante and len(comandante) >= 3:
        try:
            recs = recomendaciones(comandante)
        except edhrec.ComandanteNoEncontrado:
            recs = []
            st.error(
                f"EDHREC no tiene pagina para **{comandante}**. Revisa que el nombre "
                "este completo y en ingles (ej: *Atraxa, Praetors' Voice*)."
            )
        except Exception as e:
            recs = []
            st.error(f"No pude consultar EDHREC: {e}")

        if recs:
            mis_pedidos = st.session_state.get("pedidos") or []
            ya_tengo = {p.nombre for p in mis_pedidos}
            pendientes = edhrec.faltantes(recs, ya_tengo)

            c1, c2, c3 = st.columns(3)
            c1.markdown(estilo.tile("Recomendadas", str(len(recs))), unsafe_allow_html=True)
            c2.markdown(estilo.tile("Ya las tenes", str(len(recs) - len(pendientes))),
                        unsafe_allow_html=True)
            c3.markdown(estilo.tile("Te faltan", str(len(pendientes)), ok=True),
                        unsafe_allow_html=True)
            st.write("")

            if not ya_tengo:
                st.info("Carga tu mazo en **Mi lista** y Muchi descuenta lo que ya tenes.")

            cats = edhrec.categorias(pendientes)
            col_cat, col_n = st.columns([3, 1])
            cat = col_cat.selectbox("Categoria", ["Todas"] + cats)
            cuantas = col_n.number_input("Cuantas", 5, 50, 15, step=5)

            visibles = [r for r in pendientes if cat == "Todas" or r.categoria == cat]
            visibles = visibles[: int(cuantas)]

            for r in visibles:
                st.markdown(
                    f'<div class="mu-card"><div class="mu-fila">'
                    f'<div class="mu-izq"><div class="mu-nombre">{r.nombre}</div>'
                    f'<div style="margin-top:6px">'
                    f'<span class="mu-pill tienda">{r.categoria}</span>'
                    f'<span class="mu-pill cond">sinergia {r.sinergia:+.2f}</span>'
                    f'</div></div>'
                    f'<div style="text-align:right">'
                    f'<div class="mu-precio">{r.inclusion_pct:.0f}%</div>'
                    f'<div class="mu-sub">de los mazos</div></div>'
                    f"</div></div>",
                    unsafe_allow_html=True,
                )

            if visibles and st.button(
                f"Sumar estas {len(visibles)} a Mi lista", type="primary",
                key="add_recs",
            ):
                st.session_state["_sumar_al_mazo"] = "\n".join(
                    f"1 {r.nombre}" for r in visibles
                )
                st.rerun()


# ------------------------------------------------------------------ carrito
with tab_carrito:
    pedidos = st.session_state.get("pedidos")
    crudas = st.session_state.get("ofertas_lista")

    if not pedidos or not crudas:
        st.info("Primero carga una lista en la pestana Mi lista.")
    else:
        ofertas_por_carta = {k: filtrar(v, acabado, None, solo_tiendas)
                             for k, v in crudas.items()}
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

# ------------------------------------------------------------------ tiendas
with tab_tiendas:
    st.markdown("#### De donde salen los precios")
    st.caption(
        "La mayoria viene de scry.cl, que indexa 30 tiendas. Las de abajo Muchi "
        "las consulta directo, porque scry no las cubre o para contrastar."
    )

    filas = {f["tienda"]: f for f in catalogo.estado(base())}

    for tienda, url in shopify.TIENDAS.items():
        f = filas.get(tienda)
        c1, c2 = st.columns([3, 1])
        with c1:
            if f:
                st.markdown(
                    f'<div class="mu-card"><div class="mu-fila"><div class="mu-izq">'
                    f'<div class="mu-nombre">{tienda}</div>'
                    f'<div class="mu-sub">{f["ofertas"]:,} ofertas de '
                    f'{f["productos"]:,} productos &middot; {f["actualizado"][:16]}</div>'
                    f'</div><a class="mu-btn" href="{url}" target="_blank" '
                    f'rel="noopener">Ir</a></div></div>'.replace(",", "."),
                    unsafe_allow_html=True,
                )
            else:
                st.markdown(
                    f'<div class="mu-card"><div class="mu-fila"><div class="mu-izq">'
                    f'<div class="mu-nombre">{tienda}</div>'
                    f'<div class="mu-sub">sin indexar</div></div>'
                    f'<a class="mu-btn" href="{url}" target="_blank" '
                    f'rel="noopener">Ir</a></div></div>',
                    unsafe_allow_html=True,
                )
        if c2.button("Indexar", key=f"idx_{tienda}", use_container_width=True):
            barra = st.progress(0.0, text=f"Bajando el catalogo de {tienda}...")
            try:
                def avance(prods, ofs, _t=tienda):
                    barra.progress(min(prods / 4000, 0.95),
                                   text=f"{_t}: {prods:,} productos, {ofs:,} ofertas"
                                        .replace(",", "."))

                n = catalogo.indexar(sesion(), base(), tienda, url, avance)
                barra.progress(1.0, text="Listo")
                st.success(f"{tienda}: {n:,} ofertas indexadas.".replace(",", "."))
                st.rerun()
            except Exception as e:
                st.error(f"No pude indexar {tienda}: {e}")

    tienda_mox, listas_mox = cargar_inventarios()
    if listas_mox:
        st.divider()
        st.markdown("##### Inventario en listas de Moxfield")
        st.caption(
            f"{len(listas_mox)} listas, con precio de CardKingdom por la tasa de cada una. "
            "Los foils se cotizan con ck_foil, no con ck."
        )

        f = filas.get(tienda_mox)
        if f:
            st.markdown(
                f'<div class="mu-card"><div class="mu-fila"><div class="mu-izq">'
                f'<div class="mu-nombre">{tienda_mox}</div>'
                f'<div class="mu-sub">{f["ofertas"]:,} ofertas &middot; '
                f'{f["actualizado"][:16]}</div></div></div></div>'.replace(",", "."),
                unsafe_allow_html=True,
            )

        tasas = ", ".join(f'{i.etiqueta} x{i.tasa}' for i in listas_mox)
        st.caption(tasas)

        if st.button("Indexar las listas de Moxfield", key="idx_mox"):
            barra = st.progress(0.0, text="Bajando listas...")
            todas, sin_precio = [], 0
            try:
                for i, inv in enumerate(listas_mox):
                    barra.progress(i / len(listas_mox), text=f"Bajando {inv.etiqueta}...")
                    r = moxfield.inventario(sesion(), inv)
                    todas += r.ofertas
                    sin_precio += r.sin_precio
                catalogo.guardar_ofertas(base(), tienda_mox, todas)
                barra.progress(1.0, text="Listo")
                aviso = f"{len(todas):,} ofertas indexadas.".replace(",", ".")
                if sin_precio:
                    aviso += f" {sin_precio} entradas quedaron fuera por no traer precio de CardKingdom."
                st.success(aviso)
                st.rerun()
            except moxfield.ListaNoEncontrada as e:
                st.error(f"Esa lista no existe o no es publica: {e}")
            except Exception as e:
                st.error(f"No pude importar: {e}")

    st.divider()
    st.markdown("##### Tiendas chilenas que Muchi no puede consultar")
    st.caption("No estan en scry y no exponen sus precios de forma automatizable. "
               "Muchi te enlaza para que las mires a mano.")
    for tienda, (url, motivo) in shopify.FUERA_DE_ALCANCE.items():
        st.markdown(
            f'<div class="mu-card"><div class="mu-fila"><div class="mu-izq">'
            f'<div class="mu-nombre">{tienda}</div>'
            f'<div class="mu-sub">{motivo}</div></div>'
            f'<a class="mu-btn" href="{url}" target="_blank" rel="noopener">Buscar ahi</a>'
            f"</div></div>",
            unsafe_allow_html=True,
        )
