"""Etiquetas de Tratamiento leídas de Campos y de Textos reales de Tiendas."""
import pytest

from muchi.mtg.search import SearchOffer
from muchi.mtg import treatment


def make_offer(finish="", language="", condition="", variant="", title=""):
    return SearchOffer("Sol Ring", "Store", 0, "CLP", "https://example.com",
                       "unknown", False, "source", finish, language, condition,
                       variant, title)


@pytest.mark.parametrize("offer, expected", [
    (make_offer(finish="nonfoil", language="en"), "No Foil · Inglés"),
    (make_offer(finish="foil", language="en"), "Foil · Inglés"),
    (make_offer(language="Inglés", condition="Near Mint"), "Inglés · NM"),
    (make_offer(variant="Near Mint / Inglés / Normal"), "No Foil · Inglés · NM"),
    (make_offer(variant="Inglés Surge Foil NM"), "Foil · Inglés · NM"),
    (make_offer(variant="Lightly Played Foil"), "Foil · LP"),
    (make_offer(variant="NM Italian"), "Italiano · NM"),
    (make_offer(title="Sol Ring [SLD] #910 ENG FOIL"), "Foil · Inglés"),
    (make_offer(title="Sol Ring [Inglés / NM / No Foil]"), "No Foil · Inglés · NM"),
    (make_offer(title="Sol Ring [Idioma: Español, Estado: NM]"), "Español · NM"),
    (make_offer(title="Sol Ring — Near Mint · Spanish"), "Español · NM"),
    (make_offer(title="Anillo solar | ES | NM"), "Español · NM"),
    (make_offer(title="Sol Ring (Retro Frame) | Español | NM | BRC"), "Español · NM"),
    (make_offer(title="Sol Ring"), ""),
    (make_offer(), ""),
])
def test_reads_treatment_from_stores(offer, expected):
    assert treatment.build_treatment(offer) == expected


@pytest.mark.parametrize("title", [
    "Anillo de sol | ES | NM",       # "de" no delata Alemán
    "Sol Ring en Español NM",        # "en" no delata Inglés
    "Sol Ring [C20] Español NM it",  # "it" en minúscula no delata Italiano
])
def test_lowercase_words_are_not_codes(title):
    assert treatment.build_treatment(make_offer(title=title)) == "Español · NM"


def test_contract_fields_win_over_the_title():
    offer = make_offer(finish="foil", title="Sol Ring [Inglés / NM / No Foil]")
    assert treatment.build_treatment(offer) == "Foil · Inglés · NM"
