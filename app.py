"""Muchi Presenta Búsquedas y Resultados persistidos por la API."""
from __future__ import annotations

import html
import re
from uuid import uuid4
from urllib.parse import quote
from decimal import Decimal
from datetime import datetime, timezone

import pandas
import streamlit as st
from dotenv import load_dotenv

load_dotenv()

from muchi.api.settings import load_api_settings
from muchi.mtg.settings import load_rate_settings
from muchi.mtg import decklist, style, sprites, phrases, messaging, optimizer
from muchi.mtg import theme
from muchi.mtg import treatment
from muchi.mtg import cast as muchi_cast
from muchi.mtg.models import Offer, Order
from muchi.mtg.ports import QueryFailed, SearchRejected

CAT = "🐱"
PAW = "🐾"
CLICKS_BEFORE_COUNTER = 3
# La API marca "unknown" cuando la Fuente no publica Stock: Agregadores como
# scry.cl indexan Precios, no Inventario. No es Ausencia de Carta.
STOCK_LABELS = {"available": "En Stock", "unavailable": "Agotado",
                "unknown": "No confirmado"}
# El Contrato pide ambas Opciones en cada Pedido. Se envían fijas: la API
# comprueba el Stock de las Ofertas más baratas y hoy ignora "stores_only".
VERIFY_STOCK = True
STORES_ONLY = True
SUSPICIOUS_REASON = re.compile(r"price_below_(\d+)_percent_median")
# El Muchi Dólar solo convierte Dólares. Una Oferta en otra Moneda se muestra
# con su Valor original y queda fuera del Carrito, porque nadie sabe cuánto es.
MUCHI_DOLAR_CURRENCY = "USD"
DARK_PHRASE = "se apaga la luz , baila como pokemon en cOnVerS3"
LIGHT_PHRASE = "oh no prendieron las luces, no me vean estoy gordo"
MUCHI_MESSENGER = None

st.set_page_config(page_title="Muchi.cl", page_icon=CAT, layout="wide")
st.markdown(style.CSS, unsafe_allow_html=True)
st.markdown(sprites.build_sprite_css(), unsafe_allow_html=True)


def paint_dark_mode() -> None:
    """Pinta la Paleta elegida y guarda la Elección para la próxima Visita."""
    if "muchi_oscuro" not in st.session_state:
        cookie = theme.read_theme_choice(st.context.cookies)
        st.session_state["muchi_oscuro"] = cookie == theme.DARK_THEME
    dark = st.session_state["muchi_oscuro"]
    st.markdown(theme.paint_theme(dark), unsafe_allow_html=True)
    with st.container(key=theme.COOKIE_SLOT):
        st.iframe(theme.build_cookie_script(theme.name_theme(dark)),
                  height=theme.COOKIE_HEIGHT)


def toggle_dark_mode() -> None:
    """Muchi comenta la Luz en la Corrida siguiente.

    El Toggle ya dejó su Valor nuevo en session_state cuando llega acá.
    """
    dark = st.session_state["muchi_oscuro"]
    remember_muchi(html.escape(DARK_PHRASE if dark else LIGHT_PHRASE),
                   "happy" if dark else "alert")


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

    st.toggle("Modo Oscuro", key="muchi_oscuro", on_change=toggle_dark_mode,
              help="Apaga la Luz")

    if st.button("Muchi, ayudame!", key="muchi", use_container_width=True):
        st.session_state["muchi_habla"] = True
        st.session_state["muchi_veces"] = st.session_state.get("muchi_veces", 0) + 1
    if st.session_state.get("muchi_habla"):
        show_muchi_help()
    return MUCHI_MESSENGER

def remember_search(state, label: str = "") -> None:
    history = st.session_state.setdefault("search_history", {})
    previous = history.get(state.id, {})
    history[state.id] = {"state": state, "label": label or previous.get("label", state.id)}


def select_search(state) -> None:
    remember_search(state)
    st.session_state.pop("search_unavailable", None)
    st.session_state.pop("search_checked", None)
    st.session_state["terminal_results"] = False
    st.session_state["search_id"] = state.id
    st.session_state["search_state"] = state
    st.session_state.pop("search_items", None)
    st.query_params["search"] = state.id


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
    remember_search(state, ", ".join(order.name for order in pending["orders"]))
    select_search(state)
    st.session_state.pop("pending_search", None)
    st.rerun()


def show_search_form() -> None:
    error = st.session_state.pop("search_error", None)
    if error:
        st.error(error)
    state = st.session_state.get("search_state")
    active = state is not None and not state.done and not st.session_state.get("search_unavailable")
    pending = "pending_search" in st.session_state
    with st.form("search_form"):
        text = st.text_area(
            "Una Carta o tu Lista", placeholder="Sol Ring\n4 Lightning Bolt",
        )
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
            orders=orders, verify_stock=VERIFY_STOCK, stores_only=STORES_ONLY,
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
            select_search(state)
            st.rerun()

    with st.expander("Búsquedas Recientes"):
        st.caption("Búsquedas abiertas en esta Sesión. Guarda el Enlace para retomarlas después.")
        history = st.session_state.get("search_history", {})
        for identifier, entry in reversed(list(history.items())):
            state = entry["state"]
            st.write(f"{entry['label']} · {state.status}")
            st.markdown(f"[Enlace a la Búsqueda](?search={quote(identifier, safe='')})")
            if st.button("Abrir Búsqueda", key=f"resume_{identifier}",
                         disabled="pending_search" in st.session_state):
                select_search(state)
                st.rerun()
        if not history:
            st.caption("Todavía no hay Búsquedas en esta Sesión.")


def read_suspicious_note(reason: str) -> str:
    """Traduce el Código de la API. Hoy solo emite uno; el resto pasa crudo."""
    if match := SUSPICIOUS_REASON.fullmatch(reason):
        return f"Precio bajo el {match[1]}% de la Mediana de su Moneda"
    return reason or "Precio fuera de Rango"


def build_offer_row(offer) -> dict:
    return {
        "Carta": offer.card_name, "Tienda": offer.store,
        "Tratamiento": treatment.build_treatment(offer),
        "Precio": float(offer.amount), "Moneda": offer.currency,
        "Stock": STOCK_LABELS.get(offer.stock_status, offer.stock_status),
        "Sospechoso": read_suspicious_note(offer.suspicious_reason)
        if offer.suspicious else "",
        "Oferta": offer.url,
    }


def order_rows(rows: list[dict]) -> list[dict]:
    """Abre por Precio, barata primero, mezclando todas las Cartas.

    La Tabla es ordenable por Encabezado, así que esto es solo el Orden de
    partida: el que sirve para responder "cuánto sale lo más barato".
    """
    return sorted(rows, key=lambda row: (row["Moneda"], row["Precio"]))


def build_price_format(rows: list[dict]):
    """Escribe el Precio a la Chilena: Punto de Miles, Coma de Decimales.

    Los Decimales aparecen solo si alguna Oferta los trae. Los Pesos no llevan
    Centavos y la Columna se lee mejor sin ellos, pero 3,49 tampoco es 3.

    Se aplica como Formato de pandas y no como `format` de la Columna: así la
    Celda se lee 1.791 mientras el Valor sigue siendo el Número 1791, que es
    lo que la Tabla ordena cuando alguien pincha el Encabezado.
    """
    decimals = 0 if all(float(row["Precio"]).is_integer() for row in rows) else 2
    swap = str.maketrans(",.", ".,")
    return lambda value: f"{value:,.{decimals}f}".translate(swap)


def show_search_results(items) -> None:
    rows = []
    for item in items:
        if item.status == "source_error":
            st.warning(f"{item.name}: no se pudo completar la Consulta.")
        elif item.status == "not_found":
            st.caption(f"{item.name}: sin Ofertas.")
        rows.extend(build_offer_row(offer) for offer in item.offers)
    if rows:
        rows = order_rows(rows)
        frame = pandas.DataFrame(rows)
        st.dataframe(
            frame.style.format({"Precio": build_price_format(rows)}),
            hide_index=True, use_container_width=True,
            column_config={
                "Oferta": st.column_config.LinkColumn("Oferta"),
                "Sospechoso": st.column_config.TextColumn(
                    "Sospechoso",
                    help="Por qué Muchi desconfía del Precio. El Carrito las descarta.",
                ),
            },
        )
    else:
        state = st.session_state.get("search_state")
        if st.session_state.get("search_unavailable"):
            st.info("No hay Ofertas recibidas para mostrar.")
        elif st.session_state.get("terminal_results"):
            st.info("No hay Ofertas para mostrar.")
        elif state and state.status == "queued":
            st.info("Búsqueda en Cola. Esperando que el Servicio la procese.")
        else:
            st.info("Consultando Ofertas. Una Carta puede tardar varios minutos; "
                    "los Resultados aparecen cuando la API termina de consultarla.")


def convert_to_clp(offer, muchi_dolar: int) -> Decimal | None:
    """Lleva una Oferta a Pesos. Devuelve None si su Moneda no tiene Cambio."""
    if offer.currency == "CLP":
        return offer.amount
    if offer.currency == MUCHI_DOLAR_CURRENCY:
        return offer.amount * muchi_dolar
    return None


def show_search_cart(items) -> None:
    muchi_dolar = load_rate_settings().muchi_dolar
    with st.expander("Carrito en CLP"):
        shipping = st.number_input("Envío por Tienda", min_value=0, value=4000, step=500)
        orders = [Order(item.quantity, item.name) for item in items]
        found = {}
        converted = 0
        for item in items:
            for offer in item.offers:
                if offer.stock_status == "unavailable" or offer.suspicious:
                    continue
                price = convert_to_clp(offer, muchi_dolar)
                if price is None:
                    continue
                converted += offer.currency != "CLP"
                found.setdefault(item.name.lower(), []).append(Offer(
                    store=offer.store, card_name=item.name, title=offer.card_name,
                    price_clp=price, url=offer.url,
                ))
        plan = optimizer.build_optimal_plan(orders, found, int(shipping))
        st.caption("Usa Ofertas sin alertas de Precio ni Stock agotado. "
                   f"**Muchi Dólar: 1 USD = CLP {muchi_dolar:,}**, el Cambio "
                   "de Muchi con Costos de Compra incluidos, no el del Mercado.")
        if converted:
            st.caption(f"{converted} Ofertas en USD entraron convertidas.")
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
    if st.session_state.get("search_unavailable"):
        return
    state = st.session_state.get("search_state")
    if state is not None and state.done and st.session_state.get("terminal_results"):
        return
    state = build_muchi().searches.read_search(search_id)
    st.session_state["search_state"] = state
    remember_search(state)
    st.session_state["search_checked"] = datetime.now(timezone.utc).strftime("%H:%M:%S UTC")
    items = build_muchi().searches.read_results(search_id)
    st.session_state["search_items"] = items
    st.session_state["terminal_results"] = state.done
    if state.done:
        st.rerun()


def show_search_progress() -> None:
    search_id = st.session_state.get("search_id")
    if not search_id:
        return
    try:
        refresh_search(search_id)
    except SearchRejected as error:
        st.session_state["search_unavailable"] = str(error)
        st.rerun()
    except QueryFailed as error:
        st.error(str(error))
        st.caption("Los Resultados recibidos se conservan. Se reintentará la Consulta.")
    unavailable = st.session_state.get("search_unavailable")
    if unavailable:
        st.error(unavailable)
        st.caption("La Consulta automática se detuvo. Puedes retomar otra Búsqueda o crear una nueva.")
    state = st.session_state.get("search_state")
    if state:
        st.progress(min(max(state.processed / max(state.total, 1), 0), 1),
                    text=f"{state.status}: {state.processed} de {state.total} Cartas")
        st.caption(f"Búsqueda: {search_id}")
        if checked := st.session_state.get("search_checked"):
            st.caption(f"Último Estado recibido: {checked}")
        if not st.session_state.get("terminal_results") and not unavailable:
            st.caption(f"Consulta automática cada {settings.poll_seconds:g} segundos.")
        if state.current_card and not state.done:
            st.caption(f"Consultando: {state.current_card}")
        if state.status == "failed":
            st.error("La Búsqueda falló en el Servicio. Puedes crear otra.")
        if not state.done and not unavailable and st.button("Cancelar Búsqueda"):
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

paint_dark_mode()
st.markdown(style.paint_hero("Muchi", "Busca Cartas y cotiza tu Lista", CAT),
            unsafe_allow_html=True)
with st.sidebar:
    MUCHI_MESSENGER = show_sidebar_muchi()
show_pending_muchi()

if "search_id" not in st.session_state and st.query_params.get("search"):
    st.session_state["search_id"] = st.query_params["search"]

show_search_form()
show_search_resume()
poll_interval = (settings.poll_seconds if st.session_state.get("search_id")
                 and not st.session_state.get("terminal_results")
                 and not st.session_state.get("search_unavailable") else None)
st.fragment(run_every=poll_interval)(show_search_progress)()
with st.expander("Fuentes"):
    show_sources()
