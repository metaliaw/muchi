"""El Catálogo define los Textos y la Frecuencia sin Defaults duplicados."""
import pytest
import yaml

from muchi.mtg import phrases


def test_custom_catalog_controls_messages(tmp_path):
    document = {
        "every": 7,
        "phrases": [{"state": "happy", "phrases": ["Caricia"]}],
        "greetings": [{"text": "Bienvenido", "state": "idle"}],
        "help": [{"title": "Ayuda", "detail": "Consulta una Carta"}],
    }
    path = tmp_path / "phrases.yaml"
    path.write_text(yaml.safe_dump(document))
    book = phrases.read_phrases(path)
    assert book.every == 7
    assert phrases.speaks_now(7, book.every)
    assert not phrases.speaks_now(10, book.every)
    assert phrases.select_greeting(book, 10) == phrases.Phrase("Bienvenido", "idle")
    assert book.help_topics == (("Ayuda", "Consulta una Carta"),)


@pytest.mark.parametrize("field,value", [
    ("every", None), ("every", True), ("every", 0),
    ("greetings", []), ("help", []),
    ("phrases", [{"state": "anxiety", "phrases": ["Texto"]}]),
    ("phrases", [{"state": "talk", "phrases": "Texto"}]),
    ("greetings", [{"text": "", "state": "talk"}]),
])
def test_rejects_invalid_catalog(tmp_path, field, value):
    document = yaml.safe_load(phrases.PHRASES_PATH.read_text())
    document[field] = value
    path = tmp_path / "phrases.yaml"
    path.write_text(yaml.safe_dump(document))
    with pytest.raises(ValueError):
        phrases.read_phrases(path)


def test_missing_frequency_has_no_fallback(tmp_path):
    document = yaml.safe_load(phrases.PHRASES_PATH.read_text())
    del document["every"]
    path = tmp_path / "phrases.yaml"
    path.write_text(yaml.safe_dump(document))
    with pytest.raises(ValueError, match="every"):
        phrases.read_phrases(path)


def test_absent_catalog_has_no_greeting(tmp_path):
    book = phrases.read_phrases(tmp_path / "missing.yaml")
    assert phrases.select_greeting(book) is None
    assert book.help_topics == ()
