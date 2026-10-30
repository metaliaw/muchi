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
