"""El Traductor cruza la Frontera con Scryfall: aquí se simula la Respuesta.

Ninguna Prueba sale a la Red. Lo que se vigila es el Contrato con la Fuente
—la Puerta, los Parámetros y la Cabecera— porque romperlo no falla al escribir
Código, falla en Producción con un 400 que nadie sabe leer.
"""
import pytest

from muchi.mtg.ports import CardNotFound, TranslationFailed
from muchi.mtg.sources import scryfall
from muchi.mtg.sources.scryfall import NameTranslator


class FakeReply:
    def __init__(self, status_code=200, body=None):
        self.status_code = status_code
        self.ok = 200 <= status_code < 300
        self._body = body

    def json(self):
        if self._body is None:
            raise ValueError("sin cuerpo")
        return self._body


class FakeScryfall:
    """Anota cada Pedido y devuelve la Respuesta que la Prueba preparó."""

    def __init__(self, *replies):
        self.replies = list(replies)
        self.asked = []

    def get(self, url, params=None, headers=None, timeout=None):
        self.asked.append({"url": url, "params": params, "headers": headers,
                           "timeout": timeout})
        return self.replies.pop(0)


@pytest.fixture
def scry(monkeypatch):
    def install(*replies):
        fake = FakeScryfall(*replies)
        monkeypatch.setattr(scryfall.requests, "get", fake.get)
        return fake
    return install


# --------------------------------------------------------------- sin Idioma
def test_fuzzy_translates_when_no_language_is_chosen(scry):
    fake = scry(FakeReply(200, {"name": "Sol Ring"}))

    assert NameTranslator().translate_name("Anillo Solar") == "Sol Ring"

    asked = fake.asked[0]
    assert asked["url"] == scryfall.NAMED_URL
    assert asked["params"] == {"fuzzy": "Anillo Solar"}


def test_named_reports_the_card_it_could_not_find(scry):
    scry(FakeReply(404))
    with pytest.raises(CardNotFound, match="Anillo Solar"):
        NameTranslator().translate_name("Anillo Solar")


# --------------------------------------------------------------- con Idioma
def test_language_switches_to_the_search_door(scry):
    """named ignora lang en silencio; por eso un Idioma cambia de Puerta."""
    fake = scry(FakeReply(200, {"data": [{"name": "Sol Ring"}]}))

    assert NameTranslator().translate_name("Anillo Solar", "es") == "Sol Ring"

    asked = fake.asked[0]
    assert asked["url"] == scryfall.SEARCH_URL
    # Las Comillas piden la Frase entera: sin ellas cada Palabra busca sola.
    assert asked["params"]["q"] == 'lang:es "Anillo Solar"'
    assert asked["params"]["include_multilingual"] == "true"


def test_search_without_matches_is_a_card_not_found(scry):
    """Scryfall contesta 200 con data vacía; para Muchi es el mismo Fracaso."""
    scry(FakeReply(200, {"data": []}))
    with pytest.raises(CardNotFound):
        NameTranslator().translate_name("Anillo Solar", "ja")


def test_search_keeps_the_first_of_many(scry):
    scry(FakeReply(200, {"data": [{"name": "Sol Ring"}, {"name": "Otra"}]}))
    assert NameTranslator().translate_name("Rayo", "es") == "Sol Ring"


# ------------------------------------------------------------- los Fracasos
def test_a_default_user_agent_is_what_scryfall_rejects():
    """Scryfall responde 400 a quien no se nombra. La Cabecera es el Contrato."""
    assert scryfall.HEADERS["User-Agent"] == "Muchi/1.0"


def test_every_door_carries_the_headers(scry):
    fake = scry(FakeReply(200, {"name": "Sol Ring"}),
                FakeReply(200, {"data": [{"name": "Sol Ring"}]}))
    translator = NameTranslator()
    translator.translate_name("Anillo Solar")
    translator.translate_name("Anillo Solar", "es")

    for asked in fake.asked:
        assert asked["headers"] == scryfall.HEADERS


def test_failure_carries_the_reason_scryfall_gave(scry):
    """Sin el Detalle, «scryfall 400» no se puede diagnosticar."""
    scry(FakeReply(400, {"details": "Your User-Agent string is a default value"}))
    with pytest.raises(TranslationFailed, match="default value"):
        NameTranslator().translate_name("Anillo Solar")


def test_failure_survives_a_reply_that_is_not_json(scry):
    scry(FakeReply(503))
    with pytest.raises(TranslationFailed, match="503"):
        NameTranslator().translate_name("Anillo Solar")


def test_a_network_that_never_answers_is_a_controlled_failure(scry, monkeypatch):
    def explode(*args, **kwargs):
        raise scryfall.requests.RequestException("se cayó la Red")

    monkeypatch.setattr(scryfall.requests, "get", explode)
    with pytest.raises(TranslationFailed, match="se cayó la Red"):
        NameTranslator().translate_name("Anillo Solar")


def test_the_translator_names_the_languages_it_offers():
    """El Front no escribe la Lista: la pide a quien traduce."""
    labels = {code: label for code, label, _ in NameTranslator.languages}
    assert labels["es"] == "Español" and labels["ja"] == "Japonés"
    assert "" not in NameTranslator.codes
    assert NameTranslator.codes == set(labels)


def test_every_language_shows_how_sol_ring_is_written_there():
    """El Ejemplo es el que ensena que Idioma espera el Campo."""
    for code, label, example in NameTranslator.languages:
        assert code and label.strip() and example.strip(), code
    examples = [example for _, _, example in NameTranslator.languages]
    # Un Ejemplo repetido delataria un Idioma copiado de otro.
    assert len(set(examples)) == len(examples), examples


# ----------------------------------------------------------- las Sugerencias
def test_suggestions_answer_with_the_names_written_in_that_language(scry):
    fake = scry(FakeReply(200, {"data": [
        {"printed_name": "Anillo solar", "name": "Sol Ring"},
        {"printed_name": "Anillo de Ozolith", "name": "The Ozolith"},
    ]}))

    found = NameTranslator().suggest_names("Anillo", "es")

    assert found == ("Anillo solar", "Anillo de Ozolith")
    assert fake.asked[0]["params"]["q"] == 'lang:es "Anillo"'


def test_suggestions_fall_back_to_the_canonical_name(scry):
    """Una Impresion sin Nombre traducido igual sirve de Pista."""
    scry(FakeReply(200, {"data": [{"name": "Sol Ring"}]}))
    assert NameTranslator().suggest_names("Sol", "en") == ("Sol Ring",)


def test_suggestions_stop_at_the_limit_and_never_repeat(scry):
    scry(FakeReply(200, {"data": [
        {"printed_name": "Anillo solar"}, {"printed_name": "Anillo solar"},
        {"printed_name": "Anillo de Ozolith"}, {"printed_name": "Anillo brillante"},
        {"printed_name": "Anillo sombrío"},
    ]}))
    found = NameTranslator().suggest_names("Anillo", "es")
    assert found == ("Anillo solar", "Anillo de Ozolith", "Anillo brillante")


def test_suggesting_is_a_favour_so_a_failure_stays_quiet(scry):
    """Traducir levanta el Fracaso; sugerir se lo traga y devuelve Nada."""
    scry(FakeReply(404))
    assert NameTranslator().suggest_names("zzzz", "es") == ()

    scry(FakeReply(503))
    assert NameTranslator().suggest_names("Anillo", "es") == ()


def test_a_network_that_falls_never_reaches_whoever_is_typing(scry, monkeypatch):
    def explode(*args, **kwargs):
        raise scryfall.requests.RequestException("se cayó la Red")

    monkeypatch.setattr(scryfall.requests, "get", explode)
    assert NameTranslator().suggest_names("Anillo", "es") == ()


# ------------------------------------------------------------- la Imagen
def test_art_asks_named_when_no_language_is_chosen(scry):
    fake = scry(FakeReply(200, {
        "name": "Sol Ring", "scryfall_uri": "https://scryfall.com/x",
        "image_uris": {"normal": "https://cards.scryfall.io/sol.jpg"},
    }))

    art = NameTranslator().find_art("Sol Ring")

    assert art["image"] == "https://cards.scryfall.io/sol.jpg"
    assert art["name"] == "Sol Ring" and art["url"] == "https://scryfall.com/x"
    assert fake.asked[0]["url"] == scryfall.NAMED_URL


def test_art_in_a_language_keeps_the_printed_name(scry):
    scry(FakeReply(200, {"data": [{
        "name": "Sol Ring", "printed_name": "Anillo solar",
        "scryfall_uri": "https://scryfall.com/es",
        "image_uris": {"normal": "https://cards.scryfall.io/anillo.jpg"},
    }]}))

    art = NameTranslator().find_art("Anillo solar", "es")

    assert art["printed_name"] == "Anillo solar"
    assert art["image"] == "https://cards.scryfall.io/anillo.jpg"


def test_a_two_faced_card_shows_its_front(scry):
    """Sin image_uris arriba, la Imagen vive en la primera Cara."""
    scry(FakeReply(200, {
        "name": "Delver of Secrets // Insectile Aberration",
        "card_faces": [
            {"image_uris": {"normal": "https://cards.scryfall.io/delver.jpg"}},
            {"image_uris": {"normal": "https://cards.scryfall.io/aberration.jpg"}},
        ],
    }))
    assert NameTranslator().find_art("Delver")["image"] \
        == "https://cards.scryfall.io/delver.jpg"


def test_a_card_without_any_image_is_nothing_to_show(scry):
    scry(FakeReply(200, {"name": "Sol Ring", "image_uris": {}}))
    assert NameTranslator().find_art("Sol Ring") == {}


def test_looking_is_a_favour_so_a_failure_stays_quiet(scry):
    scry(FakeReply(404))
    assert NameTranslator().find_art("zzzz") == {}

    scry(FakeReply(503))
    assert NameTranslator().find_art("Sol Ring") == {}
