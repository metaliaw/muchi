"""Muchi - buscador kawaii de cartas Magic en tiendas chilenas.

Este archivo es el Script: Casta el elenco una vez, Reparte las pestanas y
Cuenta la historia. No Nombra una sola Fuente concreta -- eso Vive en
muchi/mtg/cast.py, y detras de el, en muchi/mtg/sources/.
"""
from __future__ import annotations

import html

import pandas as pd
import streamlit as st

from muchi.mtg import (decklist, style, mascot, sprites, phrases, history, deck,
                       offers, optimizer, oracle, constants)
from muchi.mtg import cast as muchi_cast
from muchi.mtg import stores as store_index
from muchi.mtg.style import format_clp as clp
from muchi.mtg.ports import CommanderNotFound, InventoryUnavailable, QueryFailed

CAT = "\U0001F431"    # cara de gato
PAW = "\U0001F43E"  # huellitas
FISH = "\U0001F41F"

# Desde cuantos clics aparece el contador de caricias. Antes no molesta: es un
# detalle que se gana acariciando, no una pantalla de estadisticas.
CLICKS_BEFORE_COUNTER = 3

st.set_page_config(page_title="Muchi", page_icon=CAT, layout="wide")
st.markdown(style.CSS, unsafe_allow_html=True)
# La hoja de sprites viaja incrustada en este CSS, asi que va una sola vez.
st.markdown(sprites.build_sprite_css(), unsafe_allow_html=True)


# ------------------------------------------------------- los avisos de Muchi
def muchi_says(text: str, state: str = "alert") -> None:
    """Un aviso, pensado por Muchi en su globo.

    Reemplaza a st.info/warning/error/success. Las cajas de Streamlit no son de
    Muchi y traian cuatro colores que peleaban con la paleta; aca el estado del
    sprite y el color del borde dicen lo mismo, asi que el aviso se entiende
    antes de leerlo.
    """
    st.markdown(sprites.build_notice_html(text, state), unsafe_allow_html=True)


def remember_muchi(text: str, state: str = "happy") -> None:
    """Deja un aviso para DESPUES del st.rerun().

    Un aviso escrito justo antes de un rerun no alcanza a verse: la pagina se
    vuelve a dibujar desde cero y se lo lleva. Se guarda en session_state y lo
    saca show_pending_muchi() en la corrida siguiente.
    """
    st.session_state["muchi_aviso"] = (text, state)


def show_pending_muchi() -> None:
    pending = st.session_state.pop("muchi_aviso", None)
    if pending:
        muchi_says(*pending)


def click_muchi() -> None:
    """Suma una caricia, salta y, cada tantas, deja una frase para decir."""
    book = phrases.read_phrases()
    clicks = st.session_state.get("muchi_clicks", 0) + 1
    st.session_state["muchi_clicks"] = clicks
    st.session_state["muchi_salta"] = True
    if phrases.speaks_now(clicks, book.every):
        st.session_state["muchi_click_phrase"] = phrases.pick_phrase(book.phrases)


# ---------------------------------------------------------------- el elenco
@st.cache_resource
def build_muchi():
    """Main casta a los players una sola vez por sesion."""
    return muchi_cast.build_cast()


@st.cache_data(ttl=1800, show_spinner=False)
def find_offers(card_name: str):
    m = build_muchi()
    return offers.find_offers(m.primary, m.extras, card_name)


@st.cache_data(ttl=constants.STOCK_VERIFY_CACHE_SECONDS, show_spinner=False)
def verify_cheapest_stock(found: tuple):
    return offers.verify_cheapest_stock(build_muchi().stock_verifier, list(found))


@st.cache_data(ttl=600, show_spinner=False)
def suggest_names(text: str):
    return build_muchi().primary.suggest_names(text)


@st.cache_data(ttl=86400, show_spinner=False)
def recommend_cards(commander: str):
    return build_muchi().advisor.recommend_cards(commander)


@st.cache_data(ttl=3600, show_spinner=False)
def search_catalog(text: str, intents: tuple, colors: tuple, card_type: str,
                   format_name: str, max_mana: int | None, language: str):
    """Baja la escalera y se queda con el primer escalon que trae cartas.

    Sin try/except a proposito: un pedido que el catalogo no entiende ya vuelve
    vacio y la escalera sigue sola. Lo que puede explotar aca es la red, y eso
    no se arregla en el escalon siguiente --- que suba y se muestre de una vez.
    """
    catalog = build_muchi().cards
    for request in oracle.build_requests(text, intents=intents, colors=colors,
                                         card_type=card_type,
                                         format_name=format_name,
                                         max_mana=max_mana):
        page = catalog.search_cards(request)
        if page.cards:
            cards = catalog.translate_cards(list(page.cards), language)
            return cards, page.total, page.explain, request.note
    return [], 0, "", ""


@st.cache_data(ttl=3600, show_spinner=False)
def resolve_card_name(text: str, language: str):
    """Por si lo que escribio no era una habilidad sino un nombre mal tipeado."""
    catalog = build_muchi().cards
    try:
        card = catalog.resolve_name(text)
    except QueryFailed:
        return None
    return catalog.translate_cards([card], language)[0] if card else None


# ------------------------------------------------------------ barra lateral
def show_muchi_help() -> None:
    """Muchi saluda, tira corazones y explica de que se trata."""
    times = st.session_state.get("muchi_veces", 1)
    # La semilla cambia con cada clic: si no, los corazones caerian siempre
    # en el mismo lugar y se notaria que es la misma animacion.
    st.markdown(mascot.build_hearts_html(seed=times), unsafe_allow_html=True)
    st.markdown(
        mascot.build_bubble_html(mascot.GREETINGS[times % len(mascot.GREETINGS)]),
        unsafe_allow_html=True,
    )

    for title, detail in mascot.HELP_TOPICS:
        with st.expander(title):
            st.write(detail)

    if st.button("Gracias Muchi \U0001F49D", key="muchi_chau", use_container_width=True):
        st.session_state["muchi_habla"] = False
        st.rerun()


def show_preferences() -> tuple[str, int, str]:
    """Devuelve (acabado, envio, idioma): las decisiones que afectan a todo."""
    # El clicker va primero: el on_click deja el estado listo y asi el sprite
    # y el globo de abajo ya lo ven en esta misma pasada. El boton y el sprite
    # comparten contenedor para que el hover del contenedor agrande al sprite.
    with st.container(key="muchi_mascot"):
        st.button("Acariciar a Muchi", key="muchi_clicker", on_click=click_muchi,
                  help="Apreta a Muchi")

        # Muchi mueve la boca mientras esta explicando, y respira el resto del rato.
        talking = bool(st.session_state.get("muchi_habla"))
        phrase = st.session_state.get("muchi_click_phrase")
        state = "talk" if talking else (phrase.state if phrase else "idle")
        # El salto se gasta en la corrida que lo provoco: la siguiente pasada ya
        # no salta, asi solo rebota con el clic y no con cualquier rerun.
        salta = " mu-salta" if st.session_state.pop("muchi_salta", False) else ""
        st.markdown(
            f'<div class="mu-clicker-sprite{salta}">'
            + sprites.build_sprite_html(state, scale=sprites.CLICKER_SCALE)
            + "</div>",
            unsafe_allow_html=True,
        )

    if phrase:
        st.markdown(mascot.build_bubble_html(html.escape(phrase.text)),
                    unsafe_allow_html=True)

    clicks = st.session_state.get("muchi_clicks", 0)
    if clicks >= CLICKS_BEFORE_COUNTER:
        st.caption(f"{PAW} Has acariciado a Muchi {clicks} veces")

    if st.button("Muchi, ayudame!", key="muchi", use_container_width=True):
        st.session_state["muchi_habla"] = True
        st.session_state["muchi_veces"] = st.session_state.get("muchi_veces", 0) + 1
    if st.session_state.get("muchi_habla"):
        show_muchi_help()

    st.divider()
    st.markdown(f"### {PAW} Preferencias")
    finish = st.radio("Acabado", ["Todos", "Solo normal", "Solo foil"], index=0)
    language = "es" if st.radio(
        "Idioma de las cartas", ["Espanol", "English"], index=0, horizontal=True,
        help="Solo cambia como te las muestro. Los precios y el carrito siempre "
             "usan el nombre en ingles, que es el que indexan las tiendas.",
    ) == "Espanol" else "en"
    shipping = st.number_input(
        "Costo de envio por tienda (CLP)", 0, 20000, 4000, step=500,
        help="Muchi lo usa para decidir si conviene concentrar la compra en menos tiendas.",
    )

    st.divider()
    st.caption(
        "Precios via **scry.cl**, que indexa ~30 tiendas chilenas. "
        "Verifica edicion, estado y stock en la tienda antes de pagar."
    )
    return finish, int(shipping), language


def ask_seller_filter() -> bool:
    """True si hay que esconder a los particulares del marketplace.

    Va arriba de las pestanas a proposito: afecta a las tres (busqueda, lista
    y el total del carrito), asi que escondido en la barra lateral cambiaba
    los resultados sin que se viera desde donde.
    """
    col_filter, col_note = st.columns([2, 3])
    with col_filter:
        include = st.checkbox(
            "Incluir vendedores particulares",
            value=False,
            help="Las tiendas establecidas despachan desde su propio sitio. Los "
                 "particulares venden por el marketplace de scry.cl: suelen ser mas "
                 "baratos, pero son personas, no locales.",
        )
    with col_note:
        st.caption("Mostrando tiendas y particulares de scry.cl" if include
                   else "Mostrando solo tiendas establecidas")
    return not include


# ---------------------------------------------------------------- pestanas
def pick_card(query: str) -> str | None:
    """Ofrece correcciones y devuelve la carta que el usuario quiso decir."""
    if not query or len(query) < 3:
        return None

    options = suggest_names(query)
    if options and query not in options:
        st.caption("Quisiste decir?")
        cols = st.columns(min(len(options), 5))
        for col, name in zip(cols, options[:5]):
            if col.button(name, key=f"sug_{name}", use_container_width=True):
                st.session_state["carta_elegida"] = name
                st.session_state["carta_elegida_q"] = query

    # La sugerencia elegida solo vale para la consulta que la genero: si el
    # usuario escribe otra cosa, volvemos a su texto.
    if st.session_state.get("carta_elegida_q") == query:
        return st.session_state.get("carta_elegida") or query
    return query


def show_summary(visible: list, cheapest_suspicious: bool = False) -> None:
    c1, c2, c3 = st.columns(3)
    label = "Menor observado" if cheapest_suspicious else "Mas barato"
    c1.markdown(style.paint_tile(label, clp(visible[0].price_clp),
                                 ok=not cheapest_suspicious),
                unsafe_allow_html=True)
    c2.markdown(style.paint_tile("Ofertas", str(len(visible))), unsafe_allow_html=True)
    c3.markdown(style.paint_tile("Tiendas", str(len({o.store for o in visible}))),
                unsafe_allow_html=True)
    st.write("")


def remove_confirmed_out_of_stock(visible: list) -> tuple[list, dict, int]:
    """Verifica las cinco primeras y quita sólo agotados confirmados."""
    checks = verify_cheapest_stock(tuple(visible))
    available = [offer for offer in visible if checks.get(offer) is not False]
    removed = len(visible) - len(available)
    return available, checks, removed


def refresh_live(card_id: str) -> None:
    """Vuelve a preguntarle a las ~30 tiendas, con barra de avance."""
    bar = st.progress(0.0, text="Conectando...")
    try:
        for progress in build_muchi().primary.refresh_offers(card_id):
            bar.progress(
                min(progress.done / progress.total, 1.0),
                text=f"Consultando {progress.store} ({progress.done}/{progress.total})",
            )
        bar.progress(1.0, text="Listo")
        find_offers.clear()
        st.rerun()
    except Exception as e:
        muchi_says(f"El refresco fallo: {e}", "angry")


def show_price_history(card_name: str) -> None:
    rows = history.read_history(build_muchi().cx, card_name)
    if len(rows) <= 1:
        return
    with st.expander("Historial de precios"):
        df = pd.DataFrame(rows)
        df["ts"] = pd.to_datetime(df["ts"])
        st.line_chart(df.set_index("ts")[["minimo", "promedio"]])


def show_search(finish: str, stores_only: bool) -> None:
    """Una carta, sus ofertas ordenadas y el refresco en vivo."""
    query = st.text_input(
        "Que carta buscas?", placeholder="Ej: Sol Ring, Ragavan...", key="q_simple"
    )
    chosen = pick_card(query)
    if not chosen:
        return

    with st.spinner(f"Muchi esta olfateando {chosen}..."):
        try:
            card_id, found = find_offers(chosen)
        except Exception as e:
            muchi_says(f"No pude consultar las tiendas: {e}", "angry")
            card_id, found = None, []

    if found:
        history.save_prices(build_muchi().cx, chosen, found)

    # El filtro de particulares va primero: el multiselect no debe ofrecer
    # tiendas que despues quedarian excluidas igual.
    eligible = offers.filter_offers(found, stores_only=stores_only)
    sel = st.multiselect("Filtrar tiendas", sorted({o.store for o in eligible}),
                         default=[], key="f_tiendas")
    visible = offers.filter_offers(found, finish, sel, stores_only)

    with st.spinner("Comprobando stock de las ofertas mas baratas..."):
        visible, stock_checks, unavailable = remove_confirmed_out_of_stock(visible)

    hidden = len(found) - len(eligible)
    if hidden:
        st.caption(f"Hay {hidden} ofertas de particulares ocultas. "
                   "Marca la casilla de arriba para verlas.")
    if unavailable:
        st.caption(f"Se ocultaron {unavailable} ofertas que la tienda confirmo agotadas.")

    if not visible:
        muchi_says("Sin stock con esos filtros. Prueba el refresco en vivo mas abajo.",
                   "alert")
    else:
        suspicious = offers.suspicious_prices(visible)
        show_summary(visible, visible[0] in suspicious)
        for i, o in enumerate(visible):
            verified_stock = stock_checks.get(o) is True
            unverified_stock = (not verified_stock
                                and offers.stock_needs_verification(o))
            rendered = (style.paint_suspicious_offer(
                            o, unverified_stock, verified_stock)
                        if o in suspicious else
                        style.paint_offer(o, best=(i == 0),
                                          unverified_stock=unverified_stock,
                                          verified_stock=verified_stock))
            st.markdown(rendered, unsafe_allow_html=True)

    if card_id and st.button("Refrescar en vivo (consulta las 30 tiendas)", key="refresh"):
        refresh_live(card_id)

    show_price_history(chosen)


def ask_what_it_does() -> None:
    """El formulario del buscador por texto. Deja el pedido en session_state.

    Es un form y no widgets sueltos a proposito: sin el, cada tecla apretada
    dispararia un rerun y con el una consulta al catalogo. Aca se pregunta una
    vez, cuando la persona termino de escribir.
    """
    with st.form("oraculo"):
        what = st.text_area(
            "Que quieres que haga?", height=90, key="ora_texto",
            placeholder="Ej: gana vida cada vez que una criatura muere",
        )
        chips = st.multiselect(
            "O elige para que la quieres", [i.label for i in oracle.INTENTS],
            help="Describen que hace la carta, aunque su texto no diga esas palabras.",
        )
        c1, c2, c3 = st.columns(3)
        type_label = c1.selectbox("Tipo", list(oracle.CARD_TYPES))
        format_label = c2.selectbox("Formato", list(oracle.FORMATS))
        mana = c3.slider("Mana maximo", 0, 12, 12, help="12 = sin limite.")
        color_labels = st.multiselect(
            "Identidad de color", list(oracle.COLORS),
            help="Lo que cabe en un mazo de esos colores, no solo las cartas de ese color.",
        )
        if not st.form_submit_button("Buscar cartas", type="primary"):
            return

        st.session_state["ora_cfg"] = {
            "text": what or "",
            "intents": tuple(i.key for i in oracle.INTENTS if i.label in chips),
            "colors": tuple(oracle.COLORS[c] for c in color_labels),
            "card_type": oracle.CARD_TYPES[type_label],
            "format_name": oracle.FORMATS[format_label],
            "max_mana": None if mana >= 12 else mana,
        }
        st.session_state.pop("ora_cotizar", None)


def offer_card_actions(card, suffix: str = "") -> None:
    """Los dos botones bajo cada carta. Ambos usan el nombre en ingles."""
    key = f"{card.oracle_id or card.name}{suffix}"
    b1, b2, _ = st.columns([1, 1, 3])
    if b1.button("Ver precios", key=f"ora_p_{key}", use_container_width=True):
        st.session_state["ora_cotizar"] = card.name
        st.rerun()
    if b2.button("Sumar a Mi lista", key=f"ora_l_{key}", use_container_width=True):
        st.session_state["_sumar_al_mazo"] = f"1 {card.name}"
        remember_muchi(f"<b>{card.name}</b> quedo en Mi lista.", "happy")
        st.rerun()


def show_quick_prices(card_name: str, finish: str, stores_only: bool) -> None:
    """Cotiza una carta sin salir de la pestana, con las 8 mejores ofertas."""
    st.divider()
    st.markdown(f"##### Precios de {card_name}")
    with st.spinner(f"Muchi esta olfateando {card_name}..."):
        try:
            _, found = find_offers(card_name)
        except Exception as e:
            muchi_says(f"No pude consultar las tiendas: {e}", "angry")
            found = []

    if found:
        history.save_prices(build_muchi().cx, card_name, found)

    visible = offers.filter_offers(found, finish, None, stores_only)
    with st.spinner("Comprobando stock de las ofertas mas baratas..."):
        visible, stock_checks, unavailable = remove_confirmed_out_of_stock(visible)
    if unavailable:
        st.caption(f"Se ocultaron {unavailable} ofertas que la tienda confirmo agotadas.")
    if not visible:
        muchi_says("Sin stock con esos filtros. En la pestana <b>Buscar</b> ademas "
                   "tienes el refresco en vivo.", "alert")
    else:
        suspicious = offers.suspicious_prices(visible)
        for i, o in enumerate(visible[:8]):
            verified_stock = stock_checks.get(o) is True
            unverified_stock = (not verified_stock
                                and offers.stock_needs_verification(o))
            rendered = (style.paint_suspicious_offer(
                            o, unverified_stock, verified_stock)
                        if o in suspicious else
                        style.paint_offer(o, best=(i == 0),
                                          unverified_stock=unverified_stock,
                                          verified_stock=verified_stock))
            st.markdown(rendered, unsafe_allow_html=True)
        if len(visible) > 8:
            st.caption(f"Hay {len(visible) - 8} ofertas mas en la pestana Buscar.")

    if st.button("Cerrar precios", key="ora_cerrar"):
        st.session_state.pop("ora_cotizar", None)
        st.rerun()
    st.divider()


def show_card_finder(finish: str, stores_only: bool, language: str) -> None:
    """Para cuando no sabes el nombre: describes la habilidad y Muchi busca."""
    st.markdown("#### Cuentame que hace la carta")
    st.caption(
        "Muchi le pregunta al catalogo de Magic. Escribe en espanol, en ingles o "
        "mezclando: *destruye la criatura objetivo*, *roba una carta cuando muere "
        "una criatura*, *counter target spell*."
    )

    ask_what_it_does()

    quoting = st.session_state.get("ora_cotizar")
    if quoting:
        show_quick_prices(quoting, finish, stores_only)

    cfg = st.session_state.get("ora_cfg")
    if not cfg:
        return

    with st.spinner("Muchi esta hojeando el catalogo..."):
        try:
            cards, total, explain, note = search_catalog(language=language, **cfg)
        except QueryFailed as e:
            muchi_says(f"No pude consultar el catalogo: {e}", "angry")
            return

    if not cards:
        muchi_says("No encontre nada con eso. Prueba con menos palabras, o con los "
                   "chips de arriba: describir la habilidad en pocas palabras "
                   "(<i>destruye criatura</i>) funciona mejor que una frase larga.",
                   "alert")
        maybe = resolve_card_name(cfg["text"], language) if cfg["text"] else None
        if maybe:
            st.caption("Oye, no seria esta carta?")
            st.markdown(style.paint_card(maybe, language), unsafe_allow_html=True)
            offer_card_actions(maybe, "_aprox")
        return

    summary = f"**{style.format_thousands(total)}** cartas calzan."
    if total > len(cards):
        summary += f" Muchi te muestra las {len(cards)} mas jugadas."
    st.caption(summary)

    if note:
        muchi_says(f"Asi tal cual no encontre nada, asi que busque {note}.", "alert")

    with st.expander("Que le pregunte al catalogo"):
        st.code(explain, language="text")
        st.caption("Eso mismo se pega en scryfall.com/search si quieres afinarlo.")

    st.write("")
    for card in cards:
        st.markdown(style.paint_card(card, language), unsafe_allow_html=True)
        offer_card_actions(card)


def quote_deck_list(orders: list) -> None:
    """Busca cada carta de la lista y guarda lo que encontro para el carrito."""
    bar = st.progress(0.0, text="Empezando...")
    found_by_card: dict[str, list] = {}
    failed: list[str] = []

    for i, p in enumerate(orders):
        bar.progress(i / len(orders), text=f"Buscando {p.name}...")
        try:
            _, its_offers = find_offers(p.name)
            if its_offers:
                found_by_card[p.name.lower()] = its_offers
                history.save_prices(build_muchi().cx, p.name, its_offers)
        except Exception:
            failed.append(p.name)

    bar.progress(1.0, text="Listo")
    st.session_state["ofertas_lista"] = found_by_card
    st.session_state["pedidos"] = orders
    if failed:
        muchi_says("No pude consultar: " + ", ".join(failed), "angry")

    # El caso feliz es que esten todas: ahi Muchi celebra en vez de informar.
    total, got = len(orders), len(found_by_card)
    if got == total:
        muchi_says(f"Las encontre <b>todas</b>! {total} de {total} con precio. "
                   "Anda a la pestana <b>Carrito</b> y te las reparto entre tiendas.",
                   "happy")
    else:
        muchi_says(f"Encontre precios para <b>{got}</b> de {total} cartas. "
                   f"Las {total - got} que faltan no aparecieron en ninguna tienda; "
                   "igual puedes ir al <b>Carrito</b> con el resto.", "alert")


def show_my_list() -> None:
    """Pegas el mazo, Muchi lo entiende y lo cotiza entero."""
    st.markdown("#### Pega tu mazo y Muchi busca todo")

    # La pestana Comandante deja aca lo que quieres sumar. Se mezcla ANTES de
    # crear el textarea: Streamlit no deja tocar la session_state de un widget
    # una vez instanciado, y asi el texto sigue siendo la unica fuente de verdad.
    if st.session_state.get("_sumar_al_mazo"):
        extra = st.session_state.pop("_sumar_al_mazo")
        previous = st.session_state.get("decklist", "") or ""
        st.session_state["decklist"] = (previous.rstrip() + "\n" + extra).strip()

    text = st.text_area(
        "Una carta por linea",
        height=220,
        placeholder="4 Lightning Bolt\n1 Ragavan, Nimble Pilferer\n2x Sol Ring\nCounterspell",
        key="decklist",
    )

    orders, ignored = decklist.parse_decklist(text or "")
    if orders:
        copies_total = sum(p.quantity for p in orders)
        st.caption(f"**{len(orders)}** cartas distintas - **{copies_total}** copias")
    if ignored:
        sample = " / ".join(ignored[:5])
        rest = f" y {len(ignored) - 5} mas" if len(ignored) > 5 else ""
        muchi_says(f"No entendi {len(ignored)} lineas: {sample}{rest}. "
                   "Revisalas: no quedaron en el pedido.", "alert")

    if orders and st.button("Buscar precios de la lista", type="primary"):
        quote_deck_list(orders)


def show_commander() -> None:
    """Que juega la gente con ese comandante, menos lo que ya tienes."""
    st.markdown("#### Que le falta a tu mazo")
    st.caption("Muchi le pregunta a EDHREC que juega la gente con ese comandante "
               "y descuenta lo que ya tienes en Mi lista.")

    commander = st.text_input(
        "Tu comandante", placeholder="Ej: Atraxa, Praetors' Voice", key="comandante"
    )
    if not commander or len(commander) < 3:
        return

    try:
        recs = recommend_cards(commander)
    except CommanderNotFound:
        recs = []
        muchi_says(f"No hay recomendaciones para <b>{commander}</b>. Revisa que el "
                   "nombre este completo y en ingles "
                   "(ej: <i>Atraxa, Praetors' Voice</i>).", "alert")
    except Exception as e:
        recs = []
        muchi_says(f"No pude traer las recomendaciones: {e}", "angry")
    if not recs:
        return

    owned = {p.name for p in (st.session_state.get("pedidos") or [])}
    pending = deck.subtract_owned_cards(recs, owned)

    c1, c2, c3 = st.columns(3)
    c1.markdown(style.paint_tile("Recomendadas", str(len(recs))), unsafe_allow_html=True)
    c2.markdown(style.paint_tile("Ya las tienes", str(len(recs) - len(pending))),
                unsafe_allow_html=True)
    c3.markdown(style.paint_tile("Te faltan", str(len(pending)), ok=True),
                unsafe_allow_html=True)
    st.write("")

    if not owned:
        muchi_says("Carga tu mazo en <b>Mi lista</b> y descuento lo que ya tienes.",
                   "idle")

    col_category, col_n = st.columns([3, 1])
    category = col_category.selectbox("Categoria", ["Todas"] + deck.list_categories(pending))
    how_many = col_n.number_input("Cuantas", 5, 50, 15, step=5)

    visible = [r for r in pending if category == "Todas" or r.category == category]
    visible = visible[: int(how_many)]
    for r in visible:
        st.markdown(style.paint_recommendation(r), unsafe_allow_html=True)

    if visible and st.button(f"Sumar estas {len(visible)} a Mi lista",
                             type="primary", key="add_recs"):
        st.session_state["_sumar_al_mazo"] = "\n".join(f"1 {r.name}" for r in visible)
        st.rerun()


def show_totals(plan, naive, shipping: int) -> None:
    savings = naive.total - plan.total
    c1, c2, c3, c4 = st.columns(4)
    c1.markdown(style.paint_tile("Total", clp(plan.total), ok=True),
                unsafe_allow_html=True)
    c2.markdown(style.paint_tile("Cartas", clp(plan.cards_cost)),
                unsafe_allow_html=True)
    c3.markdown(style.paint_tile(f"Envios ({len(plan.stores)})", clp(plan.shipping_cost)),
                unsafe_allow_html=True)
    c4.markdown(style.paint_tile("Ahorro vs ingenuo", clp(max(savings, 0)), ok=savings > 0),
                unsafe_allow_html=True)
    st.write("")


def show_plan_by_store(plan, shipping: int) -> None:
    """Agrupa el carrito por tienda, la mas cara primero."""
    by_store: dict[str, list] = {}
    for line in plan.lines:
        by_store.setdefault(line.store, []).append(line)

    order = sorted(by_store, key=lambda t: -sum(x.subtotal for x in by_store[t]))
    for store in order:
        lines = sorted(by_store[store], key=lambda x: -x.subtotal)
        sub = sum(x.subtotal for x in lines)
        st.markdown(style.paint_store_header(store, len(lines), sub, shipping),
                    unsafe_allow_html=True)
        for line in lines:
            st.markdown(style.paint_line(line), unsafe_allow_html=True)


def offer_csv(plan) -> None:
    df = pd.DataFrame([
        {"carta": x.card_name, "cantidad": x.quantity, "tienda": x.store,
         "precio_unitario": x.unit_price, "subtotal": x.subtotal, "url": x.url}
        for x in plan.lines
    ])
    st.download_button("Descargar carrito (CSV)",
                       df.to_csv(index=False).encode("utf-8"),
                       "carrito-muchi.csv", "text/csv")


def show_cart(finish: str, shipping: int, stores_only: bool) -> None:
    """Reparte la lista entre tiendas pesando cartas contra envios."""
    orders = st.session_state.get("pedidos")
    raw = st.session_state.get("ofertas_lista")
    if not orders or not raw:
        muchi_says("Primero carga una lista en la pestana <b>Mi lista</b>.", "idle")
        return

    by_card = {k: offers.filter_offers(v, finish, None, stores_only)
               for k, v in raw.items()}
    by_card = {k: v for k, v in by_card.items() if v}

    strategy = st.radio(
        "Como armamos el carrito?",
        ["Minimizar total (cartas + envios)", "Cada carta en su tienda mas barata"],
        horizontal=True,
    )
    naive = optimizer.build_naive_plan(orders, by_card, shipping)
    plan = (optimizer.build_optimal_plan(orders, by_card, shipping)
            if strategy.startswith("Minimizar") else naive)

    show_totals(plan, naive, shipping)
    if plan.missing:
        muchi_says("Sin stock en ninguna tienda: " + ", ".join(plan.missing), "alert")

    show_plan_by_store(plan, shipping)
    offer_csv(plan)


def show_published_inventory() -> None:
    """Las listas publicadas que Muchi usa como catalogo de una tienda."""
    inv = build_muchi().inventory
    if inv is None:
        return

    st.divider()
    st.markdown("##### Inventario en listas publicadas")
    st.caption(f"{len(inv.list_rates())} listas, con precio de CardKingdom por la tasa de "
               "cada una. Los foils se cotizan con la tasa foil, no con la normal.")

    status = store_index.read_inventory_status(build_muchi().cx, inv.store)
    if status:
        st.markdown(style.paint_store(status), unsafe_allow_html=True)
    st.caption(", ".join(f"{label} x{rate}" for label, rate in inv.list_rates()))

    if not st.button("Indexar las listas publicadas", key="idx_mox"):
        return

    bar = st.progress(0.0, text="Bajando listas...")
    try:
        def progress(done, total, label):
            bar.progress(done / total, text=f"Bajando {label}...")

        saved, without_price = store_index.import_inventory(build_muchi().cx, inv, progress)
        bar.progress(1.0, text="Listo")
        notice = f"{style.format_thousands(saved)} ofertas indexadas."
        if without_price:
            notice += f" {without_price} entradas quedaron fuera por no traer precio."
        remember_muchi(notice, "happy")
        st.rerun()
    except InventoryUnavailable as e:
        muchi_says(f"Esa lista no existe o no es publica: {e}", "alert")
    except Exception as e:
        muchi_says(f"No pude importar: {e}", "angry")


def index_one_store(status) -> None:
    bar = st.progress(0.0, text=f"Bajando el catalogo de {status.store}...")
    try:
        def progress(prods, ofs, _t=status.store):
            bar.progress(min(prods / 4000, 0.95),
                         text=f"{_t}: {style.format_thousands(prods)} productos, "
                         f"{style.format_thousands(ofs)} ofertas")

        n = store_index.index_store(build_muchi().cx, build_muchi().stores,
                                    status.store, status.url, progress)
        bar.progress(1.0, text="Listo")
        remember_muchi(f"{status.store}: {style.format_thousands(n)} ofertas "
                       "indexadas.", "happy")
        st.rerun()
    except Exception as e:
        muchi_says(f"No pude indexar {status.store}: {e}", "angry")


def show_stores() -> None:
    """De donde salen los precios, y que tienda queda fuera de alcance."""
    st.markdown("#### De donde salen los precios")
    st.caption("La mayoria viene de scry.cl, que indexa 30 tiendas. Las de abajo Muchi "
               "las consulta directo, porque scry no las cubre o para contrastar.")

    for status in store_index.read_stores_status(build_muchi().cx, build_muchi().stores):
        c1, c2 = st.columns([3, 1])
        c1.markdown(style.paint_store(status), unsafe_allow_html=True)
        if c2.button("Indexar", key=f"idx_{status.store}", use_container_width=True):
            index_one_store(status)

    show_published_inventory()

    st.divider()
    st.markdown("##### Tiendas chilenas que Muchi no puede consultar")
    st.caption("No estan en scry y no exponen sus precios de forma automatizable. "
               "Muchi te enlaza para que las mires a mano.")
    for store, (url, reason) in build_muchi().stores.list_blocked_stores().items():
        st.markdown(style.paint_blocked(store, url, reason),
                    unsafe_allow_html=True)


# -------------------------------------------------------------------- muchi
st.markdown(
    style.paint_hero("Muchi",
                     "Tu gatito buscador de cartas Magic en tiendas chilenas",
                     CAT),
    unsafe_allow_html=True,
)
st.write("")
show_pending_muchi()

with st.sidebar:
    finish, shipping, language = show_preferences()

stores_only = ask_seller_filter()

tab_search, tab_finder, tab_list, tab_commander, tab_cart, tab_stores = st.tabs(
    [f"{FISH} Buscar", "No se que busco", "Mi lista", "Comandante", "Carrito",
     "Tiendas"]
)

with tab_search:
    show_search(finish, stores_only)

with tab_finder:
    show_card_finder(finish, stores_only, language)

with tab_list:
    show_my_list()

with tab_commander:
    show_commander()

with tab_cart:
    show_cart(finish, shipping, stores_only)

with tab_stores:
    show_stores()
