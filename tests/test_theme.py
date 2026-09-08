"""Prueba que la Elección de Tema sobreviva a la Visita."""
from muchi.mtg import theme


def test_theme_opens_with_the_light_on_without_a_cookie():
    assert theme.read_theme_choice({}) == "claro"


def test_theme_reads_the_dark_choice_from_the_cookie():
    assert theme.read_theme_choice({"muchi_tema": "oscuro"}) == "oscuro"


def test_theme_script_saves_the_choice_for_a_year():
    script = theme.build_cookie_script("oscuro")
    assert "muchi_tema=oscuro" in script
    assert "max-age=31536000" in script
    assert "path=/" in script


def test_dark_theme_repaints_the_palette_and_light_leaves_it_alone():
    assert "--mu-papel" in theme.paint_theme(dark=True)
    assert "--mu-papel" not in theme.paint_theme(dark=False)
