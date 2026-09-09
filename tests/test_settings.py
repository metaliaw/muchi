"""Verifica el Contrato de Configuración de Ofertas y Tasas."""
import yaml

import pytest

from muchi.mtg import settings


@pytest.fixture(autouse=True)
def reset_settings(monkeypatch):
    monkeypatch.delenv("MUCHI_ENV", raising=False)
    settings.load_offer_settings.cache_clear()
    settings.load_rate_settings.cache_clear()
    yield
    settings.load_offer_settings.cache_clear()
    settings.load_rate_settings.cache_clear()






















@pytest.mark.parametrize("values", [
    {"minimum_offers": True}, {"minimum_offers": 1.5},
    {"maximum_low_share": 1}, {"price_gap_ratio": 1},
    {"stock_check_limit": 0}, {"price_gap_ratio": float("nan")},
    {"unknown": 5},
])
def test_rejects_offer_values(values):
    with pytest.raises(ValueError):
        settings.validate_offer_values(values)


def test_applies_offer_overrides(tmp_path, monkeypatch):
    defaults = settings.read_offer_file("defaults")
    folder = tmp_path / "config"
    folder.mkdir()
    (folder / "offers.defaults.yaml").write_text(yaml.safe_dump(defaults))
    (folder / "offers.development.yaml").write_text('stock_check_limit: 3\n')
    monkeypatch.setattr(settings, "ROOT", tmp_path)
    monkeypatch.setenv("MUCHI_ENV", "development")
    settings.load_offer_settings.cache_clear()
    try:
        assert settings.load_offer_settings().stock_check_limit == 3
        settings.load_offer_settings.cache_clear()
        monkeypatch.setenv("MUCHI_OFFERS_STOCK_CHECK_LIMIT", "1")
        assert settings.load_offer_settings().stock_check_limit == 1
    finally:
        settings.load_offer_settings.cache_clear()




def test_muchi_dolar_is_public_and_positive(monkeypatch):
    monkeypatch.setenv("MUCHI_ENV", "development")
    settings.load_rate_settings.cache_clear()
    assert settings.load_rate_settings().muchi_dolar > 0
    settings.load_rate_settings.cache_clear()


def test_muchi_dolar_accepts_an_override(monkeypatch):
    monkeypatch.setenv("MUCHI_ENV", "development")
    monkeypatch.setenv("MUCHI_RATES_MUCHI_DOLAR", "1234")
    settings.load_rate_settings.cache_clear()
    assert settings.load_rate_settings().muchi_dolar == 1234
    settings.load_rate_settings.cache_clear()


@pytest.mark.parametrize("values", [{"muchi_dolar": 0}, {"muchi_dolar": 9.5},
                                    {"dolar": 900}, []])
def test_muchi_dolar_rejects_bad_values(values):
    with pytest.raises(ValueError):
        settings.validate_rate_values(values)


# ------------------------------------------------------- el Tope de Cartas
@pytest.fixture
def api_environment(monkeypatch):
    """El Servicio pide URL y Token antes de contar Cartas."""
    from muchi.api import settings as api_settings

    monkeypatch.setenv("MUCHI_ENV", "development")
    monkeypatch.setenv("MUCHI_API_URL", "http://127.0.0.1:8081")
    monkeypatch.setenv("MUCHI_API_TOKEN", "un-token-de-prueba")
    monkeypatch.delenv("MUCHI_API_MAX_CARDS", raising=False)
    api_settings.load_api_settings.cache_clear()
    yield api_settings
    api_settings.load_api_settings.cache_clear()


def test_the_card_limit_comes_from_the_configuration(api_environment):
    """Sale del Archivo, no de un Número escrito en el Código."""
    written = yaml.safe_load(
        (settings.ROOT / "config" / "api.defaults.yaml").read_text(encoding="utf-8"))
    assert api_environment.load_api_settings().max_cards == written["max_cards"]


def test_a_variable_overrides_the_card_limit(api_environment, monkeypatch):
    monkeypatch.setenv("MUCHI_API_MAX_CARDS", "42")
    api_environment.load_api_settings.cache_clear()
    assert api_environment.load_api_settings().max_cards == 42


@pytest.mark.parametrize("value", ["0", "-5", "abc", "500.5", ""])
def test_a_card_limit_that_is_not_a_whole_count_is_refused(
        api_environment, monkeypatch, value):
    """Un Tope a medias no se redondea: se rechaza al arrancar."""
    monkeypatch.setenv("MUCHI_API_MAX_CARDS", value)
    api_environment.load_api_settings.cache_clear()
    with pytest.raises(ValueError):
        api_environment.load_api_settings()


@pytest.mark.parametrize("written", ["max_cards: 0", "max_cards: 12.5",
                                     "max_cards: '500'", "cartas: 500"])
def test_the_configuration_file_refuses_a_bad_card_limit(tmp_path, monkeypatch, written):
    """El Archivo se valida al leerlo, no cuando alguien busca 500 Cartas."""
    from muchi.api import settings as api_settings

    folder = tmp_path / "config"
    folder.mkdir()
    (folder / "api.defaults.yaml").write_text(written, encoding="utf-8")
    monkeypatch.setattr(api_settings, "ROOT", tmp_path)

    with pytest.raises(ValueError):
        api_settings.read_api_file("defaults")


def test_the_configuration_file_accepts_a_whole_count(tmp_path, monkeypatch):
    from muchi.api import settings as api_settings

    folder = tmp_path / "config"
    folder.mkdir()
    (folder / "api.defaults.yaml").write_text("max_cards: 500", encoding="utf-8")
    monkeypatch.setattr(api_settings, "ROOT", tmp_path)

    assert api_settings.read_api_file("defaults") == {"max_cards": 500}
