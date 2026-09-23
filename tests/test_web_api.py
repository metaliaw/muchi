"""El BFF traduce el Dominio a JSON y nunca deja escapar el Token."""
import importlib
from decimal import Decimal

import pytest
from fastapi.testclient import TestClient

from muchi.mtg.ports import CardNotFound, QueryFailed, SearchRejected, TranslationFailed
from muchi.mtg.search import (SearchItem, SearchOffer, SearchResults, SearchState,
                              StockCheck)
from server import links, main, presenter


def build_offer(**changes) -> SearchOffer:
    values = dict(
        card_name="Sol Ring", store="Tienda", amount=Decimal("1000"),
        currency="CLP", url="https://tienda.cl/sol-ring", stock_status="available",
        suspicious=False, source="scry", variant="Foil NM Inglés",
        finish="foil", edition="c21",
    )
    values.update(changes)
    return SearchOffer(**values)


class FakeSearches:
    def __init__(self, state=None, items=()):
        self.state = state or SearchState("abc", "running", 2, 1, 1, 0, "Sol Ring", "magic")
        self.items = items
        self.created = []
        # Lo que cada Oferta contesta cuando se le pregunta, y lo que se preguntó.
        self.stock = {}
        self.asked = []

    def create_search(self, orders, verify_stock, key, game="magic", match="exact",
                      kind="single"):
        self.created.append((orders, key, game, match, kind))
        return self.state

    def read_search(self, search_id):
        return self.state

    def read_results(self, search_id, after=0):
        remaining = tuple(item for item in self.items if item.sequence > after)
        cursor = remaining[-1].sequence if remaining else after
        return SearchResults(remaining, cursor, False)

    def cancel_search(self, search_id, key):
        return SearchState("abc", "cancelled", 2, 1, 1, 0, "", "magic")

    def check_stock(self, search_id, offer_ids):
        self.asked.append(offer_ids)
        return tuple(StockCheck(offer_id, *self.stock.get(offer_id, ("available", None)))
                     for offer_id in offer_ids)

    def read_sources(self):
        return [{"source": "scry", "status": "ok"}]

    def read_supported_games(self, kind=""):
        self.asked_kind = kind
        games = [
            {"name": "Magic: The Gathering", "reference_key": "magic",
             "singles": True, "sealed": True},
            {"name": "Mitos y Leyendas", "reference_key": "mitos-y-leyendas",
             "singles": True, "sealed": False},
        ]
        if kind == "sealed":
            return [game for game in games if game["sealed"]]
        return games

    def read_card_metadata(self, game, name, language="", edition="", foil=False):
        return {
            "name": name, "edition": edition or "SVP",
            "image": f"https://images.example/{game}/{name}.jpg",
        }

    def autocomplete_cards(self, game, name, language=""):
        return [name, f"{name} VMAX"]


@pytest.fixture
def client(monkeypatch):
    searches = FakeSearches(items=(
        SearchItem("Sol Ring", 2, "found", (
            build_offer(amount=Decimal("4000"), offer_id="of-4000"),
            build_offer(store="Otra", amount=Decimal("1500"), offer_id="of-1500"),
            build_offer(store="Dudosa", amount=Decimal("10"), suspicious=True,
                        suspicious_reason="price_below_40_percent_median",
                        offer_id="of-10"),
            build_offer(store="Gringa", amount=Decimal("3"), currency="USD",
                        offer_id="of-usd"),
        ), id="item-1", position=0, sequence=1, game="magic"),
        SearchItem("Black Lotus", 1, "not_found", (),
               id="item-2", position=1, sequence=2, game="magic"),
    ))
    monkeypatch.setattr(main, "build_muchi", lambda: type("Cast", (), {"searches": searches})())
    return TestClient(main.app), searches


def test_read_search_orders_offers_and_marks_cheapest(client):
    http, _ = client
    reply = http.get("/api/searches/abc").json()
    assert reply["state"]["done"] is False
    amounts = [(offer["currency"], offer["amount"]) for offer in reply["offers"]]
    assert amounts == [("CLP", "10"), ("CLP", "1500"), ("CLP", "4000"), ("USD", "3")]
    best = [offer["store"] for offer in reply["offers"] if offer["best"]]
    assert best == ["Dudosa"]
    assert reply["summary"] == {"lowest_clp": 10.0, "offers": 4, "stores": 4}
    assert reply["items"] == [
        {"id": "item-1", "position": 0, "sequence": 1,
         "name": "Sol Ring", "quantity": 2, "status": "found", "offers": 4,
         "game": "magic"},
        {"id": "item-2", "position": 1, "sequence": 2,
         "name": "Black Lotus", "quantity": 1, "status": "not_found", "offers": 0,
         "game": "magic"},
    ]
    assert (reply["cursor"], reply["has_more"]) == (2, False)
    assert any("Black Lotus" in notice["text"] for notice in reply["notices"])


def test_suspicious_measure_is_disabled(client):
    http, _ = client
    reply = http.get("/api/searches/abc").json()
    dudosa = next(offer for offer in reply["offers"] if offer["store"] == "Dudosa")
    assert not dudosa["suspicious"]
    assert dudosa["note"] == "" and dudosa["action"] == "Ver"


def test_unverified_offers_show_no_stock_badge(client):
    """Sin Re-verificación, el Stock no se Declara: ni Pastilla ni Etiqueta."""
    from server import presenter

    rows = presenter.build_results(
        (SearchItem("Sol Ring", 1, "found", (
            build_offer(amount=Decimal("4000")),), id="i-1", position=0, sequence=1),),
        muchi_dolar=1000, verified=False)
    offer, = rows["offers"]
    assert offer["stock_label"] == ""
    assert all(pill["text"] not in presenter.STOCK_LABELS.values()
               for pill in offer["pills"])


def test_cart_uses_low_prices_and_converts_dollars(client):
    http, _ = client
    plan = http.get("/api/searches/abc/cart?shipping=4000").json()
    assert plan["muchi_dolar"] == 1000
    assert plan["converted_offers"] == 1
    assert plan["cards_cost"] == 20
    assert plan["total"] == 4020
    assert plan["missing"] == ["Black Lotus"]


def test_api_warning_is_ignored():
    """La Bandera recibida se conserva fuera de la Presentación y el Carrito."""
    items = (SearchItem("Sol Ring", 1, "found", (
        build_offer(amount=Decimal("1000"), suspicious=True),
        build_offer(store="Otra", amount=Decimal("1200")),
    ), id="item-1", position=0, sequence=1, game="magic"),)
    rows = presenter.build_results(items, 1000)["offers"]
    assert all(not row["suspicious"] for row in rows)


def test_create_search_parses_the_decklist(client):
    http, searches = client
    reply = http.post("/api/searches", json={
        "text": "4 Lightning Bolt", "game": "pokemon", "key": "k" * 10,
    })
    assert reply.status_code == 200
    orders, key, game, match, kind = searches.created[0]
    assert (orders[0].quantity, orders[0].name, key, game, match, kind) == (
        4, "Lightning Bolt", "k" * 10, "pokemon", "exact", "single",
    )
    assert reply.json()["items"] == [
        {"name": "Lightning Bolt", "quantity": 4, "position": 0,
         "sequence": 0, "status": "queued", "offers": 0},
    ]


def test_create_search_rejects_lines_it_cannot_read(client):
    http, _ = client
    reply = http.post("/api/searches", json={
        "text": "https://tienda.cl/x", "game": "magic", "key": "k" * 10,
    })
    assert reply.status_code == 422


def test_supported_games_come_from_the_api(client):
    """Cada Juego Trae sus dos Marcas: qué se le Puede pedir."""
    http, _ = client
    assert http.get("/api/supported-games").json() == {"games": [
        {"name": "Magic: The Gathering", "reference_key": "magic",
         "singles": True, "sealed": True},
        {"name": "Mitos y Leyendas", "reference_key": "mitos-y-leyendas",
         "singles": True, "sealed": False},
    ]}


def test_supported_games_answer_by_kind(client):
    """Un Selector de Cajas Dibuja solo los Juegos que Tienen Cajas."""
    http, _ = client

    games = http.get("/api/supported-games", params={"kind": "sealed"}).json()["games"]

    assert [game["reference_key"] for game in games] == ["magic"]


def test_card_metadata_uses_the_selected_game(client):
    http, _ = client
    reply = http.get("/api/card/metadata", params={"game": "pokemon", "name": "Pikachu"})
    assert reply.json() == {
        "name": "Pikachu", "edition": "SVP",
        "image": "https://images.example/pokemon/Pikachu.jpg",
    }


def test_card_autocomplete_uses_the_selected_game(client):
    http, _ = client
    reply = http.get("/api/card/autocomplete", params={
        "game": "pokemon", "name": "Pika",
    })
    assert reply.json() == {"suggestions": ["Pika", "Pika VMAX"]}


def test_rejected_search_stops_the_polling(client, monkeypatch):
    http, searches = client
    def reject(search_id):
        raise SearchRejected("La API rechazó el Código de Seguridad.")
    monkeypatch.setattr(searches, "read_search", reject)
    reply = http.get("/api/searches/abc")
    assert reply.status_code == 409 and reply.json()["retriable"] is False


def test_failed_query_can_be_retried(client, monkeypatch):
    http, searches = client
    def fail(search_id):
        raise QueryFailed("La API agotó el Tiempo de Espera.")
    monkeypatch.setattr(searches, "read_search", fail)
    reply = http.get("/api/searches/abc")
    assert reply.status_code == 502 and reply.json()["retriable"] is True


def test_config_never_publishes_the_token(client):
    http, _ = client
    body = http.get("/api/config").text
    assert "token" not in body.lower()


def test_config_publishes_support_links(client, monkeypatch):
    http, _ = client
    monkeypatch.setenv("MUCHI_DONATION_URL", "https://apoyo.example/muchi")
    monkeypatch.setenv("MUCHI_SPONSOR_NAME", "La Guarida")
    monkeypatch.setenv("MUCHI_SPONSOR_TEXT", "Cartas y accesorios.")
    monkeypatch.setenv("MUCHI_SPONSOR_URL", "https://guarida.example")
    monkeypatch.setenv("MUCHI_ADSENSE_CLIENT", "ca-pub-1234567890123456")
    monkeypatch.setenv("MUCHI_ADSENSE_SLOT", "1234567890")

    reply = http.get("/api/config").json()

    assert reply["donation_url"] == "https://apoyo.example/muchi"
    assert reply["sponsor_name"] == "La Guarida"
    assert reply["sponsor_text"] == "Cartas y accesorios."
    assert reply["sponsor_url"] == "https://guarida.example"
    assert reply["adsense_client"] == "ca-pub-1234567890123456"
    assert reply["adsense_slot"] == "1234567890"


def test_ads_txt_authorizes_configured_publisher(monkeypatch):
    monkeypatch.setenv("MUCHI_ADSENSE_CLIENT", "ca-pub-1234567890123456")

    content = main.read_ads_txt()

    assert content == "google.com, pub-1234567890123456, DIRECT, f08c47fec0942fa0\n"


# ------------------------------------------------------------- la Traducción
class FakeTranslator:
    """Traduce sin Red: anota lo que le piden y responde lo que se le dijo."""

    languages = (("es", "Español", "Anillo solar"), ("ja", "Japonés", "太陽の指輪"))
    codes = frozenset({"es", "ja"})

    ART = {"name": "Sol Ring", "printed_name": "",
           "image": "https://cards.scryfall.io/normal/sol.jpg",
           "url": "https://scryfall.com/card/x"}

    def __init__(self, answer="Sol Ring", explodes=None,
                 suggestions=("Anillo solar", "Anillo brillante"), art=None):
        self.answer = answer
        self.explodes = explodes
        self.suggestions = suggestions
        self.art = self.ART if art is None else art
        self.asked = []
        self.suggested = []
        self.looked = []

    def find_art(self, name, language="", edition="", foil=False):
        self.looked.append((name, language, edition, foil))
        return self.art

    def suggest_names(self, text, language, limit=3):
        self.suggested.append((text, language))
        return self.suggestions

    def translate_name(self, name, language=""):
        self.asked.append((name, language))
        if self.explodes:
            raise self.explodes
        return self.answer


@pytest.fixture
def translating(monkeypatch):
    def install(translator):
        monkeypatch.setattr(
            main, "build_muchi",
            lambda: type("Cast", (), {"translator": translator})())
        return TestClient(main.app), translator
    return install


def test_card_translates_the_name_the_front_wrote(translating):
    client, translator = translating(FakeTranslator())
    reply = client.get("/api/card", params={"name": "Anillo Solar", "language": "es"})
    assert reply.status_code == 200
    assert reply.json() == {"name": "Anillo Solar", "canonical_name": "Sol Ring",
                            "language": "es"}
    assert translator.asked == [("Anillo Solar", "es")]


def test_card_without_language_lets_the_source_guess(translating):
    client, translator = translating(FakeTranslator())
    client.get("/api/card", params={"name": "Anillo Solar"})
    assert translator.asked == [("Anillo Solar", "")]


def test_card_rejects_a_language_that_is_not_offered(translating):
    """El Código inventado se corta aquí: Scryfall ni se entera."""
    client, translator = translating(FakeTranslator())
    reply = client.get("/api/card", params={"name": "Anillo Solar", "language": "xx"})
    assert reply.status_code == 400
    assert "xx" in reply.json()["detail"]
    assert translator.asked == []


def test_card_not_found_answers_with_muchis_voice(translating):
    client, _ = translating(FakeTranslator(explodes=CardNotFound("Anillo Solar")))
    reply = client.get("/api/card", params={"name": "Anillo Solar"})
    assert reply.status_code == 404
    assert reply.json()["detail"].strip()


def test_a_broken_source_is_a_bad_gateway(translating):
    client, _ = translating(FakeTranslator(explodes=TranslationFailed("scryfall 503")))
    reply = client.get("/api/card", params={"name": "Anillo Solar"})
    assert reply.status_code == 502
    assert "503" in reply.json()["detail"]


def test_card_refuses_an_empty_name(translating):
    client, translator = translating(FakeTranslator())
    assert client.get("/api/card", params={"name": ""}).status_code == 422
    assert translator.asked == []


def test_languages_come_from_the_translator_not_the_front(translating):
    client, _ = translating(FakeTranslator())
    rows = client.get("/api/languages").json()["languages"]
    assert rows[0] == {"code": "es", "label": "Español", "example": "Anillo solar"}
    # El Ejemplo viaja con el Idioma: el Front no escribe ningun Nombre.
    assert {"code": "ja", "label": "Japonés", "example": "太陽の指輪"} in rows


def test_suggestions_reach_the_front_with_the_language_chosen(translating):
    client, translator = translating(FakeTranslator())
    reply = client.get("/api/card/suggestions",
                       params={"name": "Anillo", "language": "es"})
    assert reply.status_code == 200
    assert reply.json() == {"suggestions": ["Anillo solar", "Anillo brillante"]}
    assert translator.suggested == [("Anillo", "es")]


def test_suggestions_refuse_a_language_that_is_not_offered(translating):
    client, translator = translating(FakeTranslator())
    reply = client.get("/api/card/suggestions",
                       params={"name": "Anillo", "language": "xx"})
    assert reply.status_code == 400
    assert translator.suggested == []


def test_suggestions_need_a_language_to_look_in(translating):
    """Sin Idioma no hay Sugerencia: el Campo siempre tiene uno elegido."""
    client, translator = translating(FakeTranslator())
    reply = client.get("/api/card/suggestions", params={"name": "Anillo"})
    assert reply.status_code == 422
    assert translator.suggested == []


def test_no_suggestion_is_an_empty_answer_not_a_failure(translating):
    client, _ = translating(FakeTranslator(suggestions=()))
    reply = client.get("/api/card/suggestions",
                       params={"name": "zzzz", "language": "es"})
    assert reply.status_code == 200
    assert reply.json() == {"suggestions": []}


def test_art_answers_with_the_address_not_the_bytes(translating):
    """La Imagen la sirve Scryfall al Navegador; el BFF solo pasa la Dirección."""
    client, translator = translating(FakeTranslator())
    reply = client.get("/api/card/art", params={"name": "Sol Ring"})
    assert reply.status_code == 200
    assert reply.json()["image"].startswith("https://cards.scryfall.io/")
    assert translator.looked == [("Sol Ring", "", "", False)]


def test_a_card_without_art_is_a_not_found(translating):
    client, _ = translating(FakeTranslator(art={}))
    reply = client.get("/api/card/art", params={"name": "zzzz"})
    assert reply.status_code == 404


def test_art_refuses_a_language_that_is_not_offered(translating):
    client, translator = translating(FakeTranslator())
    reply = client.get("/api/card/art", params={"name": "Sol Ring", "language": "xx"})
    assert reply.status_code == 400
    assert translator.looked == []


# -------------------------------------------------------------- el Carrito
def test_the_cart_warns_until_the_purchase_is_ready(client):
    """La Compra todavía no Está: el Carrito se Muestra, pero Avisando."""
    reply, _ = client
    assert reply.get("/api/config").json()["cart_ready"] is False


def test_the_warning_goes_away_from_the_environment(monkeypatch):
    """Retirar el Aviso es una Línea en el Entorno, no un Deploy de la Interfaz."""
    monkeypatch.setenv("MUCHI_CART_READY", "1")
    importlib.reload(main)
    try:
        assert main.CART_READY is True
    finally:
        monkeypatch.delenv("MUCHI_CART_READY")
        importlib.reload(main)


# --------------------------------------------------------------- las Redes
def test_config_names_the_repository_for_the_footer(client):
    reply, _ = client
    assert reply.get("/api/config").json()["repository_url"].startswith("https://github.com/")


def test_config_names_both_repositories_because_muchi_is_open_source(client):
    """La Interfaz y la API son Libres: el Panel abre las dos Puertas."""
    reply, _ = client
    config = reply.get("/api/config").json()
    assert config["repository_url"] == "https://github.com/metaliaw/muchi"
    assert config["api_repository_url"] == "https://github.com/cangrejometralleta/muchi-api"


def test_a_network_without_an_address_does_not_exist():
    """La Barra solo muestra las Redes que Llevan a alguna parte."""
    redes = links.validate_networks({"networks": [
        {"name": "Instagram", "icon": "📸", "url": "https://instagram.com/muchi"},
        {"name": "Discord", "icon": "💬", "url": ""},
    ]})

    assert [red["name"] for red in redes if red["url"]] == ["Instagram"]


def test_the_networks_keep_the_order_of_the_file():
    """El Archivo Dice el Orden; la Barra no lo Reordena por su cuenta."""
    assert [red["name"] for red in links.load_socials()][:2] == ["Instagram", "TikTok"]


def test_a_network_out_of_https_stops_the_start():
    """Media Barra no Sirve: un Archivo malo se Nota al Arrancar, no al Mirarla."""
    for broken in ({"networks": [{"name": "X", "icon": "𝕏", "url": "http://x.com"}]},
                   {"networks": [{"name": "", "icon": "📸", "url": ""}]},
                   {"networks": [{"name": "X", "icon": "𝕏", "handle": "@muchi"}]},
                   {"redes": []}):
        with pytest.raises(ValueError):
            links.validate_networks(broken)


def test_configured_networks_reach_the_bar(client):
    """Lo que el Archivo Declara es lo que el Front Recibe."""
    reply, _ = client

    rows = reply.get("/api/config").json()["socials"]

    assert rows == [dict(red) for red in links.load_socials() if red["url"]]
    assert all(row["url"].startswith("https://") for row in rows)


def test_art_carries_the_edition_of_the_offer_that_was_clicked(translating):
    client, translator = translating(FakeTranslator())
    client.get("/api/card/art",
               params={"name": "Sol Ring", "edition": "c21", "foil": "true"})
    assert translator.looked == [("Sol Ring", "", "c21", True)]


def test_an_offer_publishes_what_identifies_its_printing(client):
    """Sin Edición ni Acabado, el Front no puede pedir la Impresión exacta."""
    reply, _ = client
    row = reply.get("/api/searches/abc").json()["offers"][0]
    assert "edition" in row and "finish" in row


def test_an_offer_wears_its_edition_as_a_pill():
    """La Edición Distingue una Impresión de otra: va en una Pastilla propia."""
    offer = build_offer(edition="Modern Horizons 3 Commander")

    row = presenter.build_offer(offer, muchi_dolar=1000)

    assert {"kind": "edicion", "text": "Modern Horizons 3 Commander"} in row["pills"]


def test_an_edition_code_is_read_in_capitals():
    """`c21` no es una Palabra: es el Código de una Edición."""
    row = presenter.build_offer(build_offer(edition="c21"), muchi_dolar=1000)

    assert {"kind": "edicion", "text": "C21"} in row["pills"]


def test_a_catalog_name_is_not_an_edition():
    """`MTG Single` Nombra el Catálogo de la Tienda, no la Edición de la Carta."""
    for noise in ("MTG Single", "Magic: The Gathering Singles", ""):
        row = presenter.build_offer(build_offer(edition=noise), muchi_dolar=1000)
        assert not [pill for pill in row["pills"] if pill["kind"] == "edicion"]


def test_an_offer_publishes_its_pickup_locations():
    offer = build_offer(locations=("Santiago - Providencia", "Santiago - Las Condes"))

    row = presenter.build_offer(offer, muchi_dolar=1000)

    assert row["locations"] == ["Santiago - Providencia", "Santiago - Las Condes"]


# ------------------------------------------------------- el Stock comprobado
def test_the_crown_moves_to_the_next_offer_with_stock(client):
    """La barata sin Stock no Recomienda: la Corona pasa a la siguiente barata."""
    http, searches = client
    searches.stock = {"of-10": ("unavailable", 0), "of-1500": ("available", 2)}

    answer = http.get("/api/searches/abc/stock").json()

    assert searches.asked == [("of-10",), ("of-1500",)]
    assert answer["best"] == [{"card_type": "sol ring", "offer_id": "of-1500"}]
    assert answer["uncrowned"] == []
    coronada = next(row for row in answer["offers"] if row["offer_id"] == "of-1500")
    assert coronada["stock_label"] == "2 unidades"
    assert {"kind": "tienda", "text": "2 unidades"} in coronada["pills"]
    agotada = next(row for row in answer["offers"] if row["offer_id"] == "of-10")
    assert agotada["stock_label"] == "Agotado"


def test_the_cart_recommends_before_anyone_chooses(client):
    """Sin Elecciones, el Carrito es el Reparto que Muchi Haría."""
    http, _ = client

    plan = http.get("/api/searches/abc/cart?shipping=0").json()

    assert [(row["store"], row["cards"]) for row in plan["stores"]] == [("Dudosa", 2)]
    # La Línea Nombra su Oferta: con eso el Front Llena los Selectores.
    assert plan["stores"][0]["lines"][0]["offer_id"] == "of-10"


def test_the_cart_names_the_printing_it_bought(client):
    """Dos Líneas con el mismo Nombre y distinta Edición no son la misma Compra."""
    http, _ = client

    line = http.get("/api/searches/abc/cart?shipping=0").json()["stores"][0]["lines"][0]

    assert (line["edition"], line["finish"]) == ("c21", "foil")


def test_the_cart_sums_what_the_person_chose(client):
    """Elegir dos Tiendas es Comprar en dos Tiendas, aunque Salga más caro."""
    http, _ = client

    plan = http.post("/api/searches/abc/cart?shipping=0", json={"picks": [
        {"offer_id": "of-10", "units": 1},
        {"offer_id": "of-1500", "units": 1},
    ]}).json()

    assert sorted((row["store"], row["cards"]) for row in plan["stores"]) \
        == [("Dudosa", 1), ("Otra", 1)]
    assert plan["cards_cost"] == 1510
    assert plan["short"] == []


def test_the_cart_says_what_the_choice_leaves_out(client):
    """Se Piden dos Copias y se Elige una: el Carrito lo Dice."""
    http, _ = client

    plan = http.post("/api/searches/abc/cart?shipping=0",
                     json={"picks": [{"offer_id": "of-10", "units": 1}]}).json()

    assert plan["short"] == [{"card_name": "Sol Ring", "units": 1}]


def test_a_pick_of_zero_buys_nothing(client):
    """Cero no es una Compra: la Oferta no Entra al Carrito."""
    http, _ = client

    plan = http.post("/api/searches/abc/cart?shipping=0", json={"picks": [
        {"offer_id": "of-10", "units": 0},
        {"offer_id": "of-1500", "units": 2},
    ]}).json()

    assert [(row["store"], row["cards"]) for row in plan["stores"]] == [("Otra", 2)]


def test_a_pick_outside_the_search_is_ignored(client):
    """Se Compra de esta Búsqueda, no del Catálogo entero."""
    http, _ = client

    plan = http.post("/api/searches/abc/cart?shipping=0", json={"picks": [
        {"offer_id": "of-ajena", "units": 3},
        {"offer_id": "of-10", "units": 2},
    ]}).json()

    assert [(row["store"], row["cards"]) for row in plan["stores"]] == [("Dudosa", 2)]


def test_the_config_says_how_long_a_stock_lasts(client):
    """El Front Guarda lo Confirmado con su Hora; el Tope lo Dice el Servidor."""
    http, _ = client

    assert http.get("/api/config").json()["stock_fresh_seconds"] == 600


def test_the_browser_check_saves_the_visit(client):
    """Lo que el Navegador Confirmó no se le Pregunta al Servicio otra vez."""
    http, searches = client

    answer = http.post("/api/searches/abc/stock",
                       json={"checks": [{"offer_id": "of-10", "available": True}]}).json()

    assert searches.asked == []
    assert answer["best"] == [{"card_type": "sol ring", "offer_id": "of-10"}]


def test_the_browser_no_passes_the_crown(client):
    """Un Agotado visto en el Navegador Mueve la Corona sin Visitar la barata."""
    http, searches = client

    answer = http.post("/api/searches/abc/stock",
                       json={"checks": [{"offer_id": "of-10", "available": False}]}).json()

    assert searches.asked == [("of-1500",)]
    assert answer["best"] == [{"card_type": "sol ring", "offer_id": "of-1500"}]


def test_a_single_touch_visits_nobody(client):
    """Un Toque sobre una Oferta no Desata una Ronda nuestra a las Tiendas."""
    http, searches = client

    answer = http.post("/api/searches/abc/stock?ask=false",
                       json={"checks": [{"offer_id": "of-1500", "available": True}]}).json()

    assert searches.asked == []
    assert answer["best"] == [{"card_type": "sol ring", "offer_id": "of-1500"}]


def test_a_touch_the_browser_cannot_reach_asks_only_that_store(client):
    """Una Tienda sin Catálogo abierto se Pregunta desde acá, y solo ella.

    Las Tiendas leídas de Listas no Sirven un JSON que el Navegador pueda Leer.
    Antes ese Toque Quedaba sin Respuesta; ahora Preguntamos nosotros, pero por
    esa Oferta y por nadie más: un Toque no Vale una Ronda.
    """
    http, searches = client

    answer = http.post("/api/searches/abc/stock?ask=false",
                       json={"checks": [], "asking": ["of-1500"]}).json()

    assert searches.asked == [("of-1500",)]
    assert [row["offer_id"] for row in answer["offers"]] == ["of-1500"]


def test_an_offer_outside_the_search_is_ignored(client):
    """El Navegador Informa sobre esta Búsqueda: lo demás no Corona nada."""
    http, searches = client

    answer = http.post("/api/searches/abc/stock",
                       json={"checks": [{"offer_id": "of-ajena", "available": True}]}).json()

    assert searches.asked == [("of-10",)]
    assert all(row["offer_id"] != "of-ajena" for row in answer["offers"])


def test_a_browser_yes_crowns_from_beyond_the_limit(client):
    """El Tope es de la Pregunta, no de la Corona.

    `stock_check_limit` Acota las Visitas que Hacemos nosotros. Una Oferta que
    el Navegador Confirmó no nos Costó ninguna, así que Compite aunque esté
    fuera del Plan — si no, la Duda más barata se Queda la Corona teniendo un
    Sí más arriba.
    """
    http, searches = client
    searches.stock = {name: ("unknown", None) for name in ("of-10", "of-1500", "of-4000")}

    # `of-usd` es la cuarta de la Lista: el Plan de tres no la Nombra.
    answer = http.post("/api/searches/abc/stock",
                       json={"checks": [{"offer_id": "of-usd", "available": True}]}).json()

    assert searches.asked == [("of-10",), ("of-1500",), ("of-4000",)]
    assert answer["best"] == [{"card_type": "sol ring", "offer_id": "of-usd"}]


def test_a_checked_offer_stops_the_round(client):
    """Si la más barata Tiene, nadie más recibe una Visita."""
    http, searches = client

    answer = http.get("/api/searches/abc/stock").json()

    assert searches.asked == [("of-10",)]
    assert answer["best"] == [{"card_type": "sol ring", "offer_id": "of-10"}]


def test_a_confirmed_yes_beats_a_cheaper_doubt(client):
    """Una Tienda que no Declara Stock no Niega, pero tampoco Confirma.

    La barata queda en Duda y la siguiente Confirma: la Corona es del Sí. Una
    Carta que no llega no es una Compra barata.
    """
    http, searches = client
    searches.stock = {"of-10": ("unknown", None), "of-1500": ("available", 1)}

    answer = http.get("/api/searches/abc/stock").json()

    assert searches.asked == [("of-10",), ("of-1500",)]
    assert answer["best"] == [{"card_type": "sol ring", "offer_id": "of-1500"}]


def test_a_doubt_still_crowns_when_nobody_confirms(client):
    """Si ninguna Tienda Confirma, la Duda más barata sigue siendo la Corona."""
    http, searches = client
    searches.stock = {"of-10": ("unknown", None), "of-1500": ("unavailable", 0),
                      "of-4000": ("unknown", None), "of-usd": ("unknown", None)}

    answer = http.get("/api/searches/abc/stock").json()

    assert answer["best"] == [{"card_type": "sol ring", "offer_id": "of-10"}]


def test_a_card_without_stock_anywhere_loses_its_crown(client):
    """Ninguna Oferta con Stock es Ninguna Recomendación, no la menos mala."""
    http, searches = client
    searches.stock = {name: ("unavailable", 0)
                      for name in ("of-10", "of-1500", "of-4000", "of-usd")}

    answer = http.get("/api/searches/abc/stock").json()

    assert answer["best"] == []
    assert answer["uncrowned"] == ["sol ring"]


def test_only_offers_that_can_be_compared_are_asked():
    """Sin Cambio a Pesos no hay Corona que defender: a esa Oferta no se la molesta.

    Tampoco a la que ya se declaró Agotada, ni a la que la API no supo Nombrar:
    volver a preguntar por ellas gasta una Visita a la Tienda sin cambiar nada.
    """
    item = SearchItem("Sol Ring", 1, "found", (
        build_offer(store="Euro", amount=Decimal("2"), currency="EUR", offer_id="of-eur"),
        build_offer(store="Agotada", amount=Decimal("100"), offer_id="of-cero",
                    stock_status="unavailable"),
        build_offer(store="Anonima", amount=Decimal("200"), offer_id=""),
        build_offer(store="Tienda", amount=Decimal("300"), offer_id="of-300"),
    ), id="i-1", position=0, sequence=1)

    plan = presenter.plan_stock_checks((item,), muchi_dolar=1000)

    assert plan == {"sol ring": ["of-300"]}


def test_a_checked_offer_declares_its_units(client):
    """Comprobar de a una Declara Stock aunque la Re-verificación esté apagada."""
    row = presenter.build_offer(build_offer(stock_quantity=3), muchi_dolar=1000,
                                verified=False)

    assert row["stock_quantity"] == 3 and row["stock_label"] == "3 unidades"
    assert {"kind": "tienda", "text": "3 unidades"} in row["pills"]


def test_an_offer_tells_the_front_where_its_catalog_lives():
    row = presenter.build_offer(build_offer(source="moxfield"), muchi_dolar=1000)

    assert row["source"] == "moxfield"


def test_zero_units_is_sold_out_whatever_the_status_says(client):
    """Cero Unidades es Agotado: la Cantidad manda sobre la Etiqueta."""
    row = presenter.build_offer(build_offer(stock_quantity=0), muchi_dolar=1000)

    assert row["stock_label"] == "Agotado"


# ---------------------------------------------------------------- los Topes
def build_list(count: int) -> str:
    return "\n".join(f"1 Carta{index:04d}" for index in range(count))


def test_the_limits_come_from_the_commander_deck(client):
    """Cien Cartas tiene el Mazo: 99 copias más el Comandante.

    Los Topes salen del Formato, no de una Preferencia. Cambiarlos sin cambiar
    esa Razón es lo que este Test pregunta en voz alta.
    """
    reply, _ = client
    limits = reply.get("/api/config").json()["limits"]
    assert limits == {"max_cards": 100, "max_quantity": 99}


def test_a_bulk_search_stops_at_the_limit(client):
    """El Tope lo publica /api/config y lo aplica la misma Constante."""
    reply, _ = client
    limit = reply.get("/api/config").json()["limits"]["max_cards"]

    assert reply.post("/api/searches",
                      json={"text": build_list(limit), "key": "en-el-tope"}
                      ).status_code == 200

    refused = reply.post("/api/searches",
                         json={"text": build_list(limit + 1), "key": "pasado-el-tope"})
    assert refused.status_code == 422
    # El Mensaje dice el Número de verdad: escrito a mano se despegaría.
    assert str(limit) in refused.json()["detail"]["detail"]


def test_the_polling_rhythm_reaches_the_front(client):
    """El Front no elige el Ritmo: lo lee de la Configuración."""
    reply, _ = client
    assert reply.get("/api/config").json()["poll_seconds"] == 3


def test_create_search_carries_the_match_mode(client):
    """El Modo ancho Viaja a la API; sin Pedirlo, la Búsqueda es angosta."""
    http, searches = client
    http.post("/api/searches", json={
        "text": "Kuriboh", "game": "yugioh", "key": "k" * 10, "match": "includes",
    })
    assert searches.created[0][3] == "includes"


def test_create_search_carries_the_sealed_kind(client):
    """Pedir Sellado Viaja a la API; sin Pedirlo, se Buscan Cartas sueltas."""
    http, searches = client
    http.post("/api/searches", json={
        "text": "Caja de Sobres Bloomburrow", "game": "magic", "key": "k" * 10,
        "kind": "sealed",
    })
    assert searches.created[0][4] == "sealed"

    http.post("/api/searches", json={
        "text": "Sol Ring", "game": "magic", "key": "j" * 10,
    })
    assert searches.created[1][4] == "single"


def test_create_search_refuses_an_unknown_kind(client):
    http, _ = client
    reply = http.post("/api/searches", json={
        "text": "Sol Ring", "game": "magic", "key": "k" * 10, "kind": "booster",
    })
    assert reply.status_code == 422


def test_create_search_refuses_an_unknown_match_mode(client):
    http, _ = client
    reply = http.post("/api/searches", json={
        "text": "Kuriboh", "game": "yugioh", "key": "k" * 10, "match": "contains",
    })
    assert reply.status_code == 422


def kuriboh_items():
    """Una Búsqueda ancha: la Carta pedida y dos Derivados que no son ella."""
    return (SearchItem("Kuriboh", 3, "found", (
        build_offer(card_name="Kuriboh", store="Uno", amount=Decimal("900")),
        build_offer(card_name="Kuriboh", store="Dos", amount=Decimal("1200")),
        build_offer(card_name="Winged Kuriboh", store="Tres", amount=Decimal("300")),
        build_offer(card_name="Winged Kuriboh", store="Cuatro", amount=Decimal("500")),
        build_offer(card_name="Linkuriboh", store="Cinco", amount=Decimal("100")),
    ), id="item-1", position=0, sequence=1, game="yugioh"),)


def test_each_card_type_crowns_its_own_cheapest():
    """La más barata es por Tipo de Carta: un Linkuriboh no gana a un Kuriboh."""
    results = presenter.build_results(kuriboh_items(), 1000, match="includes")
    best = {row["card_name"] for row in results["offers"] if row["best"]}
    assert best == {"Kuriboh", "Winged Kuriboh", "Linkuriboh"}
    crowned = {(row["card_name"], row["store"]) for row in results["offers"] if row["best"]}
    assert crowned == {("Kuriboh", "Uno"), ("Winged Kuriboh", "Tres"), ("Linkuriboh", "Cinco")}


def test_offers_sort_by_price_inside_each_card_type():
    """Los Tipos no se Mezclan, y adentro de cada uno manda el Precio."""
    results = presenter.build_results(kuriboh_items(), 1000, match="includes")
    shown = [(row["card_type"], Decimal(row["amount"])) for row in results["offers"]]
    assert [card for card, _ in shown] == [
        "kuriboh", "kuriboh", "winged kuriboh", "winged kuriboh", "linkuriboh",
    ]
    assert [price for _, price in shown] == [
        Decimal("900"), Decimal("1200"), Decimal("300"), Decimal("500"), Decimal("100"),
    ]


def test_an_exact_search_keeps_its_printings_together():
    """En `exact` las Impresiones son la misma Carta y compiten entre ellas."""
    items = (SearchItem("Kuriboh", 1, "found", (
        build_offer(card_name="Kuriboh", store="Uno", amount=Decimal("900")),
        build_offer(card_name="Kuriboh (C)", store="Dos", amount=Decimal("400")),
    ), id="item-1", position=0, sequence=1, game="yugioh"),)
    results = presenter.build_results(items, 1000, match="exact")
    assert {row["card_type"] for row in results["offers"]} == {"kuriboh"}
    assert sum(1 for row in results["offers"] if row["best"]) == 1


def test_one_pokemon_card_groups_editions():
    """La Carta funcional reúne Ediciones antes de Comparar."""
    items = (SearchItem("Slowpoke", 1, "found", (
        build_offer(card_name="Slowpoke", card_key="slowpoke", edition="SV1",
                    store="Uno", amount=Decimal("900")),
        build_offer(card_name="Slowpoke", card_key="slowpoke", edition="SV2",
                    store="Dos", amount=Decimal("700")),
        build_offer(card_name="Slowpoke ex", card_key="slowpoke ex", edition="SV3",
                    store="Tres", amount=Decimal("400")),
    ), id="item-1", position=0, sequence=1, game="pokemon"),)
    results = presenter.build_results(items, 1000, match="exact")
    assert {row["card_type"] for row in results["offers"]} == {
        "slowpoke", "slowpoke ex",
    }
    first_type = [row for row in results["offers"]
                  if row["card_type"] == "slowpoke"]
    assert {row["edition"] for row in first_type} == {"SV1", "SV2"}
    assert {(row["card_type"], row["store"])
            for row in results["offers"] if row["best"]} == {
        ("slowpoke", "Dos"), ("slowpoke ex", "Tres"),
    }


def test_pokemon_metadata_does_not_split_one_functional_card():
    """La Edición y el Tipo no Fragmentan la Identidad que publicó la API."""
    items = (SearchItem("Slowpoke", 1, "found", (
        build_offer(card_name="Slowpoke", card_key="slowpoke",
                    metadata={"pokemon_type": "water"}),
        build_offer(card_name="Slowpoke", card_key="slowpoke", store="Otra",
                    metadata={"pokemon_type": "psychic"}),
    ), id="item-1", position=0, sequence=1, game="pokemon"),)
    rows = presenter.build_results(items, 1000)["offers"]
    assert {row["card_type"] for row in rows} == {"slowpoke"}
    assert {row["card_label"] for row in rows} == {"Slowpoke"}


def test_pokemon_treatment_and_language_share_a_section():
    """Idioma y Tratamiento no Fragmentan una misma Carta funcional."""
    items = (SearchItem("Pikachu", 1, "found", (
        build_offer(card_name="Pikachu", card_key="pikachu", language="en",
                    finish="foil", variant="Near Mint Foil",
                    metadata={"functional_id": "25", "type": "lightning"}),
        build_offer(card_name="Pikachu", card_key="pikachu", language="es",
                    finish="", variant="Played", store="Otra",
                    metadata={"functional_id": "25", "type": "lightning"}),
    ), id="item-1", position=0, sequence=1, game="pokemon"),)
    rows = presenter.build_results(items, 1000)["offers"]
    assert {row["card_type"] for row in rows} == {"pikachu"}
    assert sum(row["best"] for row in rows) == 1


def test_the_cart_buys_the_card_that_was_asked_for():
    """Pedir 3 Kuriboh y recibir un Linkuriboh barato no es un Carrito."""
    cart = presenter.build_cart(kuriboh_items(), 0, 1000, match="includes")
    bought = {line["title"] for store in cart["stores"] for line in store["lines"]}
    assert bought == {"Kuriboh"}


def test_one_card_from_three_sources_is_one_group():
    """Cada Fuente Escribe el Título a su manera; la API dice cuál Carta es."""
    items = (SearchItem("Kuriboh", 1, "found", (
        build_offer(card_name="Winged Kuriboh", card_key="winged kuriboh",
                    store="Uno", amount=Decimal("300")),
        build_offer(card_name='LDS3-EN100 "Winged Kuriboh" Common',
                    card_key="winged kuriboh", store="Dos", amount=Decimal("400")),
        build_offer(card_name="Winged Kuriboh (PUR)", card_key="winged kuriboh",
                    store="Tres", amount=Decimal("500")),
    ), id="item-1", position=0, sequence=1, game="yugioh"),)
    results = presenter.build_results(items, 1000, match="includes")
    assert {row["card_type"] for row in results["offers"]} == {"winged kuriboh"}
    assert sum(1 for row in results["offers"] if row["best"]) == 1


def test_an_offer_without_a_key_falls_back_to_its_title():
    """Una API anterior no manda `card_key`; el Título sigue sirviendo."""
    items = (SearchItem("Kuriboh", 1, "found", (
        build_offer(card_name="Winged Kuriboh", store="Uno", amount=Decimal("300")),
        build_offer(card_name="Linkuriboh", store="Dos", amount=Decimal("100")),
    ), id="item-1", position=0, sequence=1, game="yugioh"),)
    results = presenter.build_results(items, 1000, match="includes")
    assert {row["card_type"] for row in results["offers"]} == {"winged kuriboh", "linkuriboh"}


def test_a_fallen_store_is_named_to_whoever_searched():
    """Una Carta con Ofertas y una Tienda caída no es una Carta completa."""
    items = (SearchItem("Kuriboh", 1, "found", (
        build_offer(card_name="Kuriboh", store="Uno", amount=Decimal("900")),
    ), id="item-1", position=0, sequence=1, game="yugioh",
        faults=("v3.netdecker.cl",)),)
    notices = presenter.build_results(items, 1000)["notices"]
    warnings = [n for n in notices if n["level"] == "warning"]
    assert len(warnings) == 1
    assert "v3.netdecker.cl" in warnings[0]["text"]
    assert "<!doctype" not in warnings[0]["text"]


def test_a_complete_answer_warns_about_nobody():
    items = (SearchItem("Kuriboh", 1, "found", (
        build_offer(card_name="Kuriboh", store="Uno", amount=Decimal("900")),
    ), id="item-1", position=0, sequence=1, game="yugioh"),)
    notices = presenter.build_results(items, 1000)["notices"]
    assert [n for n in notices if n["level"] == "warning"] == []


def test_an_offer_carries_the_picture_its_store_published():
    """La Foto de la Tienda Llega entera: una Caja sellada no Tiene otra."""
    from muchi.api.client import build_offer

    offer = build_offer({
        "id": "1", "card_name": "Play Booster Display", "store": "Oasis",
        "price_amount": "194990", "price_currency": "CLP",
        "url": "https://oasisgames.cl/box", "source": "www.oasisgames.cl",
        "stock_status": "available", "suspicious": False,
        "image": "https://cdn.shopify.com/box.png",
    })
    assert offer.image == "https://cdn.shopify.com/box.png"

    row = presenter.build_offer(offer, 1000, verified=False)
    assert row["image"] == "https://cdn.shopify.com/box.png"
