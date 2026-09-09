"""Prueba la Identidad: Precios a la Chilena y Tarjetas bien armadas."""
from decimal import Decimal

from muchi.mtg import style


def test_pesos_are_written_without_cents():
    assert style.format_amount(Decimal("1234567"), "CLP") == "$1.234.567"


def test_other_currencies_keep_their_cents_and_their_name():
    assert style.format_amount(Decimal("3.49"), "USD") == "USD 3,49"


def test_offer_card_shows_its_pills_and_its_button():
    card = style.paint_offer_card("Sol Ring", "$1.500", "https://example.com",
                                  [("tienda", "LaCripta"), ("foil", "Foil")])
    assert 'class="mu-pill tienda">LaCripta<' in card
    assert 'class="mu-pill foil">Foil<' in card
    assert 'href="https://example.com"' in card and ">Ver<" in card
    assert "mu-card mejor" not in card


def test_the_cheapest_offer_wears_the_periwinkle():
    card = style.paint_offer_card("Sol Ring", "$1.500", "#", best=True)
    assert "mu-card mejor" in card and "mu-precio mejor" in card


def test_a_suspicious_offer_explains_itself_and_asks_to_be_checked():
    card = style.paint_offer_card("Sol Ring", "$152", "#", note="Muy por debajo.",
                                  action="Verificar")
    assert '<div class="mu-sub">Muy por debajo.</div>' in card
    assert ">Verificar<" in card


def test_the_card_escapes_what_the_store_wrote():
    card = style.paint_offer_card('<b>Sol</b>', "$1", "#", [("cond", "<i>NM</i>")])
    assert "<b>" not in card and "<i>" not in card
