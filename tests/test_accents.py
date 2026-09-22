"""El Acento no Parte una Carta en dos."""
from muchi.mtg.search import SearchOffer
from server import presenter


def build_offer(name: str) -> SearchOffer:
    return SearchOffer(card_name=name, store="Casa", amount=1, currency="CLP",
                       url="https://casamyl.cl/x", stock_status="unknown",
                       suspicious=False, source="tcgmatch")


def test_a_tilde_is_not_another_card():
    """Mitos y Leyendas Escribe con Tilde; su Índice Archiva sin ella."""
    assert presenter.names_same_card("dragon de magma", "Dragón de Magma")
    assert presenter.names_same_card("Dragón de Magma [Imperio]", "dragon de magma")


def test_the_card_type_folds_its_accent():
    """Las dos Escrituras Caen en el mismo Grupo, con una sola Corona."""
    offer = build_offer("Dragón de Magma")
    assert presenter.read_card_type(offer, "dragon de magma", presenter.MATCH_EXACT) \
        == presenter.read_card_type(offer, "Dragón de Magma", presenter.MATCH_EXACT)
