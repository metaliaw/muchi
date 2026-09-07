"""Verifica el Contrato de Configuración y el Catálogo de Stock."""
import yaml

import pytest

from muchi.mtg import constants, settings


@pytest.fixture(autouse=True)
def reset_settings(monkeypatch):
    for variable in (
        "MUCHI_ENV", "MUCHI_STOCK_TIMEOUT_SECONDS", "MUCHI_STOCK_CACHE_SECONDS",
        "MUCHI_UNVERIFIED_STOCK_SOURCES", "MUCHI_OUT_OF_STOCK_MARKERS",
    ):
        monkeypatch.delenv(variable, raising=False)
    settings.load_stock_settings.cache_clear()
    yield
    settings.load_stock_settings.cache_clear()


def test_requires_environment(monkeypatch):
    with pytest.raises(ValueError, match="MUCHI_ENV"):
        settings.load_stock_settings()
    monkeypatch.setenv("MUCHI_ENV", "unknown")
    with pytest.raises(ValueError, match="MUCHI_ENV"):
        settings.load_stock_settings()


def test_applies_precedence(tmp_path, monkeypatch):
    folder = tmp_path / "config"
    folder.mkdir()
    (folder / "stock.defaults.yaml").write_text(
        yaml.safe_dump({"timeout_seconds": 10, "cache_seconds": 60})
    )
    (folder / "stock.development.yaml").write_text('timeout_seconds: 8\n')
    monkeypatch.setattr(settings, "ROOT", tmp_path)
    monkeypatch.setenv("MUCHI_ENV", "development")
    assert settings.load_stock_settings() == settings.StockSettings(8, 60)
    settings.load_stock_settings.cache_clear()
    monkeypatch.setenv("MUCHI_STOCK_TIMEOUT_SECONDS", "3")
    assert settings.load_stock_settings() == settings.StockSettings(3, 60)
    monkeypatch.setenv("MUCHI_STOCK_TIMEOUT_SECONDS", "4")
    assert settings.load_stock_settings().timeout_seconds == 3


@pytest.mark.parametrize("value", ["0", "-1", "nan", "inf", "bad"])
def test_rejects_override(value, monkeypatch):
    monkeypatch.setenv("MUCHI_ENV", "production")
    monkeypatch.setenv("MUCHI_STOCK_TIMEOUT_SECONDS", value)
    with pytest.raises(ValueError):
        settings.load_stock_settings()


@pytest.mark.parametrize("value", [None, True, "10", -1, 0])
def test_rejects_file_value(value, tmp_path, monkeypatch):
    folder = tmp_path / "config"
    folder.mkdir()
    (folder / "stock.defaults.yaml").write_text(
        yaml.safe_dump({"timeout_seconds": value, "cache_seconds": 60})
    )
    monkeypatch.setattr(settings, "ROOT", tmp_path)
    with pytest.raises(ValueError):
        settings.read_stock_file("defaults")


def test_rejects_global_override(monkeypatch):
    monkeypatch.setenv("MUCHI_ENV", "production")
    monkeypatch.setenv("MUCHI_OUT_OF_STOCK_MARKERS", "available")
    with pytest.raises(ValueError, match="cannot be overridden"):
        settings.load_stock_settings()


def test_rejects_unknown_key(tmp_path, monkeypatch):
    (tmp_path / "config").mkdir()
    (tmp_path / "config/stock.production.yaml").write_text('unverified_sources: []\n')
    monkeypatch.setattr(settings, "ROOT", tmp_path)
    with pytest.raises(ValueError, match="Unknown"):
        settings.read_stock_file("production")


def test_requires_environment_file(tmp_path, monkeypatch):
    (tmp_path / "config").mkdir()
    (tmp_path / "config/stock.defaults.yaml").write_text('{}')
    monkeypatch.setattr(settings, "ROOT", tmp_path)
    monkeypatch.setenv("MUCHI_ENV", "production")
    with pytest.raises(FileNotFoundError):
        settings.load_stock_settings()


def test_validates_catalog(tmp_path, monkeypatch):
    (tmp_path / "constants").mkdir()
    target = tmp_path / "constants/stock.yaml"
    target.write_text('unverified_sources: [1]\nout_of_stock_markers: [sold out]\n')
    monkeypatch.setattr(constants, "ROOT", tmp_path)
    with pytest.raises(ValueError, match="strings"):
        constants.read_stock_catalog()


def test_requires_complete_settings(tmp_path, monkeypatch):
    folder = tmp_path / "config"
    folder.mkdir()
    (folder / "stock.defaults.yaml").write_text('{}')
    (folder / "stock.production.yaml").write_text('{}')
    monkeypatch.setattr(settings, "ROOT", tmp_path)
    monkeypatch.setenv("MUCHI_ENV", "production")
    with pytest.raises(ValueError, match="Missing required"):
        settings.load_stock_settings()


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


def test_offer_thresholds_change_results():
    from dataclasses import replace
    from muchi.mtg import offers
    from muchi.mtg.models import Offer

    config = settings.OfferSettings(**settings.read_offer_file("defaults"))
    found = [Offer(store="test", card_name="Card", title="Card", price_clp=price,
                   url=f"https://example.com/{price}") for price in [10, 50, 60, 70, 80]]
    assert offers.find_suspicious_prices(found, config) == {found[0]}
    assert offers.find_suspicious_prices(found, replace(config, price_gap_ratio=6)) == set()
    assert offers.find_suspicious_prices(found[:1], replace(config, minimum_offers=1)) == set()

    class Verifier:
        def verify_stock(self, offer):
            return True

    checks = offers.verify_cheapest_stock(Verifier(), found, replace(config, stock_check_limit=1))
    assert checks == {found[1]: True}
