"""Muchi Presenta Búsquedas y Resultados persistidos por la API."""
from __future__ import annotations

import html
from uuid import uuid4
from decimal import Decimal

import streamlit as st
from dotenv import load_dotenv

load_dotenv()

from muchi.api.settings import load_api_settings
from muchi.mtg import decklist, style, sprites, phrases, messaging, optimizer
from muchi.mtg import cast as muchi_cast
from muchi.mtg.models import Offer, Order
from muchi.mtg.ports import QueryFailed, SearchRejected

CAT = "🐱"
PAW = "🐾"
CLICKS_BEFORE_COUNTER = 3
MUCHI_MESSENGER = None

st.set_page_config(page_title="Muchi.cl", page_icon=CAT, layout="wide")
st.markdown(style.CSS, unsafe_allow_html=True)
st.markdown(sprites.build_sprite_css(), unsafe_allow_html=True)


@st.cache_resource
def build_muchi():
    return muchi_cast.build_cast()


def muchi_says(text: str, state: str = "alert",
               priority: messaging.Priority | None = None,
               source: str = "app") -> None:
    """Reemplaza el contenido de la unica burbuja del Muchi lateral.

    Las cajas de Streamlit no son de Muchi y un segundo gato hacia que los
    errores parecieran venir de otro personaje.
    """
    if priority is None:
        priority = {
            "angry": messaging.Priority.ERROR,
            "happy": messaging.Priority.RESULT,
            "idle": messaging.Priority.PASSIVE,
        }.get(state, messaging.Priority.RESULT)
    if MUCHI_MESSENGER is not None:
        MUCHI_MESSENGER.publish(text, state, priority, source)
        return
    st.markdown(phrases.build_bubble_html(text, state), unsafe_allow_html=True)

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
        muchi_says(*pending, priority=messaging.Priority.RESULT,
                   source="pending")

def click_muchi() -> None:
    """Suma una caricia, salta y, cada tantas, deja una frase para decir."""
    book = phrases.read_phrases()
    clicks = st.session_state.get("muchi_clicks", 0) + 1
    st.session_state["muchi_clicks"] = clicks
    st.session_state["muchi_salta"] = True
    if phrases.speaks_now(clicks, book.every):
        st.session_state["muchi_click_phrase"] = phrases.pick_phrase(book.phrases)

def show_muchi_help() -> None:
    """Muchi saluda, tira corazones y explica de que se trata."""
    times = st.session_state.get("muchi_veces", 1)
    # La semilla cambia con cada clic: si no, los corazones caerian siempre
    # en el mismo lugar y se notaria que es la misma animacion.
    st.markdown(phrases.build_hearts_html(seed=times), unsafe_allow_html=True)
    greeting = phrases.select_greeting(phrases.read_phrases(), times)
    if greeting:
        muchi_says(html.escape(greeting.text), greeting.state,
                   priority=messaging.Priority.HELP, source="help")

    for title, detail in phrases.read_phrases().help_topics:
        with st.expander(title):
            st.write(detail)

    if st.button("Gracias Muchi \U0001F49D", key="muchi_chau", use_container_width=True):
        st.session_state["muchi_habla"] = False
        st.rerun()

def show_sidebar_muchi():
    """Dibuja al Muchi original y devuelve el slot de su unica burbuja."""
    global MUCHI_MESSENGER
    # El clicker va primero: el on_click deja el estado listo y asi el sprite
    # y el globo de abajo ya lo ven en esta misma pasada. El boton y el sprite
    # comparten contenedor para que el hover del contenedor agrande al sprite.
    with st.container(key="muchi_mascot"):
        st.button("Acariciar a Muchi", key="muchi_clicker", on_click=click_muchi,
                  help="Apreta a Muchi")

        # Muchi mueve la boca mientras esta explicando, y respira el resto del rato.
        talking = bool(st.session_state.get("muchi_habla"))
        phrase = st.session_state.pop("muchi_click_phrase", None)
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

    message_slot = st.empty()
    MUCHI_MESSENGER = messaging.MuchiMessenger(
        lambda message: message_slot.markdown(
            phrases.build_bubble_html(message.text, message.state),
            unsafe_allow_html=True,
        ),
        message_slot.empty,
    )
    greeting = phrases.select_greeting(phrases.read_phrases())
    if greeting:
        MUCHI_MESSENGER.render_default(html.escape(greeting.text), greeting.state)
    if phrase:
        MUCHI_MESSENGER.publish(
            html.escape(phrase.text), phrase.state,
            messaging.Priority.CLICK, "clicker",
        )

    clicks = st.session_state.get("muchi_clicks", 0)
    if clicks >= CLICKS_BEFORE_COUNTER:
        st.caption(f"{PAW} Has acariciado a Muchi {clicks} veces")

    if st.button("Muchi, ayudame!", key="muchi", use_container_width=True):
        st.session_state["muchi_habla"] = True
        st.session_state["muchi_veces"] = st.session_state.get("muchi_veces", 0) + 1
    if st.session_state.get("muchi_habla"):
        show_muchi_help()
    return MUCHI_MESSENGER

def submit_search() -> None:
    pending = st.session_state["pending_search"]
    try:
        state = build_muchi().searches.create_search(**pending)
    except SearchRejected as error:
        st.session_state.pop("pending_search", None)
        st.session_state["search_error"] = str(error)
        st.rerun()
    except QueryFailed as error:
        st.session_state["search_error"] = str(error)
        st.rerun()
    st.session_state["terminal_results"] = False
    st.session_state["search_id"] = state.id
    st.session_state["search_state"] = state
    st.session_state.pop("search_items", None)
    st.session_state.pop("pending_search", None)
    st.query_params["search"] = state.id
    st.rerun()


def show_search_form() -> None:
    error = st.session_state.pop("search_error", None)
    if error:
        st.error(error)
    state = st.session_state.get("search_state")
    active = state is not None and not state.done
    pending = "pending_search" in st.session_state
    with st.form("search_form"):
        text = st.text_area(
            "Una Carta o tu Lista", placeholder="Sol Ring\n4 Lightning Bolt",
        )
        verify_stock = st.checkbox("Comprobar Stock", value=True)
        stores_only = st.checkbox("Sólo Tiendas", value=True)
        submitted = st.form_submit_button("Buscar", disabled=active or pending)
    if submitted and not active and not pending:
        orders, ignored = decklist.parse_decklist(text)
        if ignored:
            st.warning("Revisa estas Líneas: " + ", ".join(ignored))
            return
        if not 1 <= len(orders) <= 500 or any(not 1 <= row.quantity <= 99 for row in orders):
            st.error("Ingresa entre 1 y 500 Cartas, con Cantidades de 1 a 99.")
            return
        st.session_state["pending_search"] = dict(
            orders=orders, verify_stock=verify_stock, stores_only=stores_only,
            key=str(uuid4()),
        )
        submit_search()
    if "pending_search" in st.session_state:
        st.info("El Envío está pendiente. Reintentar conserva la misma Búsqueda.")
        if st.button("Reintentar Envío"):
            submit_search()


def show_search_resume() -> None:
    with st.expander("Retomar una Búsqueda"):
        identifier = st.text_input("Identificador de Búsqueda")
        if st.button("Retomar", disabled="pending_search" in st.session_state) and identifier.strip():
            try:
                state = build_muchi().searches.read_search(identifier.strip())
            except QueryFailed as error:
                st.error(str(error))
                return
            st.session_state["terminal_results"] = False
            st.session_state["search_id"] = state.id
            st.session_state["search_state"] = state
            st.session_state.pop("search_items", None)
            st.query_params["search"] = state.id
            st.rerun()


def show_search_results(items) -> None:
    selected = st.selectbox("Acabado", ["Todos", "Normal", "Foil"])
    rows = []
    for item in items:
        if item.status == "source_error":
            st.warning(f"{item.name}: no se pudo completar la Consulta.")
        elif item.status == "not_found":
            st.caption(f"{item.name}: sin Ofertas.")
        for offer in item.offers:
            finish = offer.finish.lower().replace("-", "").replace("_", "")
            foil = finish in {"foil", "etched", "etched foil", "etchedfoil"}
            if selected == "Foil" and not foil or selected == "Normal" and foil:
                continue
            rows.append({
                "Carta": offer.card_name, "Tienda": offer.store,
                "Precio": str(offer.amount), "Moneda": offer.currency,
                "Stock": offer.stock_status, "Precio sospechoso": offer.suspicious,
                "Motivo": offer.suspicious_reason, "Oferta": offer.url,
            })
    if rows:
        st.dataframe(rows, hide_index=True, use_container_width=True,
                     column_config={"Oferta": st.column_config.LinkColumn("Oferta")})
    else:
        st.info("Todavía no hay Ofertas para mostrar.")


def show_search_cart(items) -> None:
    with st.expander("Carrito en CLP"):
        shipping = st.number_input("Envío por Tienda", min_value=0, value=4000, step=500)
        orders = [Order(item.quantity, item.name) for item in items]
        found = {}
        for item in items:
            eligible = [offer for offer in item.offers if offer.currency == "CLP"
                        and offer.stock_status != "unavailable" and not offer.suspicious]
            found.setdefault(item.name.lower(), []).extend(Offer(
                store=offer.store, card_name=item.name, title=offer.card_name,
                price_clp=offer.amount, url=offer.url,
            ) for offer in eligible)
        plan = optimizer.build_optimal_plan(orders, found, int(shipping))
        st.caption("Usa Ofertas en CLP sin alertas de Precio ni Stock agotado. "
                   "Las demás Monedas se muestran con su valor original.")
        st.metric("Total con Envíos", f"CLP {Decimal(plan.total):,.2f}")
        if plan.lines:
            st.dataframe([vars(line) for line in plan.lines], hide_index=True)
        if plan.missing:
            st.caption("Sin Oferta apta: " + ", ".join(plan.missing))


def cancel_search(search_id: str) -> None:
    key = st.session_state.setdefault(f"cancel_{search_id}", str(uuid4()))
    try:
        state = build_muchi().searches.cancel_search(search_id, key)
    except QueryFailed as error:
        st.error(str(error))
        return
    st.session_state["search_state"] = state
    st.session_state["terminal_results"] = False
    st.rerun()


def refresh_search(search_id: str) -> None:
    state = st.session_state.get("search_state")
    if state is not None and state.done and st.session_state.get("terminal_results"):
        return
    state = build_muchi().searches.read_search(search_id)
    items = build_muchi().searches.read_results(search_id)
    was_active = st.session_state.get("search_state")
    st.session_state["search_state"] = state
    st.session_state["search_items"] = items
    st.session_state["terminal_results"] = state.done
    if state.done and (was_active is None or not was_active.done):
        st.rerun()


def show_search_progress() -> None:
    search_id = st.session_state.get("search_id")
    if not search_id:
        return
    try:
        refresh_search(search_id)
    except QueryFailed as error:
        st.error(str(error))
        st.caption("Los Resultados recibidos se conservan. Se reintentará la Consulta.")
    state = st.session_state.get("search_state")
    if state:
        st.progress(min(max(state.processed / max(state.total, 1), 0), 1),
                    text=f"{state.status}: {state.processed} de {state.total} Cartas")
        st.caption(f"Búsqueda: {search_id}")
        if state.status == "failed":
            st.error("La Búsqueda falló en el Servicio. Puedes crear otra.")
        if not state.done and st.button("Cancelar Búsqueda"):
            cancel_search(search_id)
    items = st.session_state.get("search_items", ())
    show_search_results(items)
    if items:
        show_search_cart(items)


def show_sources() -> None:
    if st.button("Consultar Estado de Fuentes"):
        try:
            st.dataframe(build_muchi().searches.read_sources(), hide_index=True)
        except QueryFailed as error:
            st.error(str(error))


try:
    settings = load_api_settings()
except (ValueError, OSError) as error:
    st.error(str(error))
    st.stop()

st.markdown(style.paint_hero("Muchi", "Busca Cartas y cotiza tu Lista", CAT),
            unsafe_allow_html=True)
with st.sidebar:
    MUCHI_MESSENGER = show_sidebar_muchi()
show_pending_muchi()

if "search_id" not in st.session_state and st.query_params.get("search"):
    st.session_state["search_id"] = st.query_params["search"]

show_search_form()
show_search_resume()
current_state = st.session_state.get("search_state")
poll_interval = None if st.session_state.get("terminal_results") else settings.poll_seconds
st.fragment(run_every=poll_interval)(show_search_progress)()
with st.expander("Fuentes"):
    show_sources()
