"""El BFF traduce el Dominio a JSON y nunca deja escapar el Token."""
from decimal import Decimal

import pytest
from fastapi.testclient import TestClient

from muchi.mtg.ports import CardNotFound, QueryFailed, SearchRejected, TranslationFailed
from muchi.mtg.search import SearchItem, SearchOffer, SearchState
from server import main


def build_offer(**changes) -> SearchOffer:
    values = dict(
        card_name="Sol Ring", store="Tienda", amount=Decimal("1000"),
        currency="CLP", url="https://tienda.cl/sol-ring", stock_status="available",
        suspicious=False, source="scry", variant="Foil NM Inglés",
    )
    values.update(changes)
    return SearchOffer(**values)


class FakeSearches:
    def __init__(self, state=None, items=()):
        self.state = state or SearchState("abc", "running", 2, 1, 1, 0, "Sol Ring")
        self.items = items
        self.created = []

    def create_search(self, orders, verify_stock, stores_only, key):
        self.created.append((orders, key))
        return self.state

    def read_search(self, search_id):
        return self.state

    def read_results(self, search_id):
        return self.items

    def cancel_search(self, search_id, key):
        return SearchState("abc", "cancelled", 2, 1, 1, 0)

    def read_sources(self):
        return [{"source": "scry", "status": "ok"}]


@pytest.fixture
def client(monkeypatch):
    searches = FakeSearches(items=(
        SearchItem("Sol Ring", 2, "found", (
            build_offer(amount=Decimal("4000")),
            build_offer(store="Otra", amount=Decimal("1500")),
            build_offer(store="Dudosa", amount=Decimal("10"), suspicious=True,
                        suspicious_reason="price_below_40_percent_median"),
            build_offer(store="Gringa", amount=Decimal("3"), currency="USD"),
        )),
        SearchItem("Black Lotus", 1, "not_found", ()),
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
    assert best == ["Otra"]
    assert reply["summary"] == {"lowest_clp": 10.0, "offers": 4, "stores": 4}
    assert any("Black Lotus" in notice["text"] for notice in reply["notices"])


def test_suspicious_offer_explains_itself_in_words(client):
    http, _ = client
    reply = http.get("/api/searches/abc").json()
    dudosa = next(offer for offer in reply["offers"] if offer["store"] == "Dudosa")
    assert "⚠ Precio bajo el 40% de la Mediana de su Moneda" in [
        pill["text"] for pill in dudosa["pills"]
    ]
    assert dudosa["note"] and dudosa["action"] == "Verificar"


def test_cart_skips_suspicious_and_converts_dollars(client):
    http, _ = client
    plan = http.get("/api/searches/abc/cart?shipping=4000").json()
    assert plan["muchi_dolar"] == 1000
    assert plan["converted_offers"] == 1
    # La Oferta sospechosa de 10 pesos no entra: el Carrito toma la de 1500.
    assert plan["cards_cost"] == 3000
    assert plan["total"] == 7000
    assert plan["missing"] == ["Black Lotus"]


def test_create_search_parses_the_decklist(client):
    http, searches = client
    reply = http.post("/api/searches", json={"text": "4 Lightning Bolt", "key": "k" * 10})
    assert reply.status_code == 200
    orders, key = searches.created[0]
    assert (orders[0].quantity, orders[0].name, key) == (4, "Lightning Bolt", "k" * 10)


def test_create_search_rejects_lines_it_cannot_read(client):
    http, _ = client
    reply = http.post("/api/searches", json={"text": "https://tienda.cl/x", "key": "k" * 10})
    assert reply.status_code == 422


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

    def find_art(self, name, language=""):
        self.looked.append((name, language))
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
    assert translator.looked == [("Sol Ring", "")]


def test_a_card_without_art_is_a_not_found(translating):
    client, _ = translating(FakeTranslator(art={}))
    reply = client.get("/api/card/art", params={"name": "zzzz"})
    assert reply.status_code == 404


def test_art_refuses_a_language_that_is_not_offered(translating):
    client, translator = translating(FakeTranslator())
    reply = client.get("/api/card/art", params={"name": "Sol Ring", "language": "xx"})
    assert reply.status_code == 400
    assert translator.looked == []
