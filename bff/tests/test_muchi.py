"""Tests for logic that does not touch the network.

Run with pytest, or directly:  python tests/test_muchi.py
"""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from muchi.mtg import decklist, phrases, optimizer  # noqa: E402
from muchi.mtg.models import Offer, Order  # noqa: E402


def test_decklist_formats():
    orders, ignored = decklist.parse_decklist(
        """
        # mi mazo
        4 Lightning Bolt
        2x Sol Ring
        1 Ragavan, Nimble Pilferer (MH2) 138
        Counterspell
        Sideboard:
        3 Fire // Ice
        """
    )
    found_by_card = {p.name: p.quantity for p in orders}
    # The classic bug: without the (?=\d) lookahead, "Sol Ring" was parsed as "Sol"
    # and "Lightning Bolt" came out as "Lightning".
    assert found_by_card == {
        "Lightning Bolt": 4,
        "Sol Ring": 2,
        "Ragavan, Nimble Pilferer": 1,
        "Counterspell": 1,
        "Fire": 3,
    }, found_by_card
    assert not ignored, ignored


def test_decklist_sums_duplicates():
    orders, _ = decklist.parse_decklist("2 Sol Ring\n1 Sol Ring")
    assert len(orders) == 1 and orders[0].quantity == 3








def _offer(store, card_name, price):
    return Offer(store=store, card_name=card_name, title=card_name, price_clp=price,
                 url="http://x", key=f"{store}-{card_name}")


def test_optimizer_prefers_concentrating():
    """When shipping is expensive, a single store wins even if cards cost more."""
    orders = [Order(1, "A"), Order(1, "B")]
    offers = {
        "a": [_offer("T1", "A", 1000), _offer("T2", "A", 900)],
        "b": [_offer("T1", "B", 1000), _offer("T3", "B", 900)],
    }

    naive = optimizer.build_naive_plan(orders, offers, shipping_per_store=5000)
    # Naive: 900 + 900 + 2 shipments = 11800 across two stores
    assert len(naive.stores) == 2 and naive.total == 11800, naive.total

    optimal = optimizer.build_optimal_plan(orders, offers, shipping_per_store=5000)
    # Optimal: everything at T1 = 2000 + 1 shipment = 7000
    assert optimal.stores == ["T1"], optimal.stores
    assert optimal.total == 7000, optimal.total
    assert optimal.total < naive.total


def test_optimizer_splits_when_shipping_is_cheap():
    orders = [Order(1, "A"), Order(1, "B")]
    offers = {
        "a": [_offer("T1", "A", 5000), _offer("T2", "A", 100)],
        "b": [_offer("T1", "B", 5000), _offer("T3", "B", 100)],
    }
    plan = optimizer.build_optimal_plan(orders, offers, shipping_per_store=200)
    assert sorted(plan.stores) == ["T2", "T3"], plan.stores
    assert plan.total == 600, plan.total


def test_optimizer_reports_missing():
    orders = [Order(1, "A"), Order(2, "Inexistente")]
    offers = {"a": [_offer("T1", "A", 1000)]}
    plan = optimizer.build_optimal_plan(orders, offers, shipping_per_store=0)
    assert plan.missing == ["Inexistente"], plan.missing
    assert plan.total == 1000


def test_optimizer_respects_quantities():
    orders = [Order(4, "A")]
    offers = {"a": [_offer("T1", "A", 250)]}
    plan = optimizer.build_optimal_plan(orders, offers, shipping_per_store=0)
    assert plan.cards_cost == 1000, plan.cards_cost


































def test_greetings_and_help_not_empty():
    assert phrases.read_phrases().greetings and all(p.text.strip() for p in phrases.read_phrases().greetings)
    for title, detail in phrases.read_phrases().help_topics:
        assert title.strip() and len(detail) > 30, title








# ------------------------------------------------------------------ phrases
def test_phrase_book_reads_catalog():
    from muchi.mtg import phrases

    book = phrases.read_phrases()
    assert book.every == 10
    assert book.phrases, "the phrases catalog must not be empty"
    assert all(p.text.strip() for p in book.phrases)
    states = {p.state for p in book.phrases}
    assert states <= {"idle", "talk", "happy", "alert", "angry"}, states


def test_phrase_book_missing_file_is_harmless():
    from muchi.mtg import phrases

    book = phrases.read_phrases(Path("no-such-phrases.json"))
    assert book.every == 0 and book.phrases == ()






def test_pick_phrase_returns_one_of_the_list():
    from muchi.mtg import phrases

    pool = (phrases.Phrase("a", "happy"), phrases.Phrase("b", "alert"))
    assert phrases.pick_phrase(pool) in pool
    assert phrases.pick_phrase(()) is None


def test_speaks_now_on_the_boundary():
    from muchi.mtg import phrases

    assert phrases.speaks_now(10, 10) is True
    assert phrases.speaks_now(20, 10) is True
    assert phrases.speaks_now(9, 10) is False
    assert phrases.speaks_now(0, 10) is False
    assert phrases.speaks_now(5, 0) is False


def test_phrases_file_ships_with_example():
    from muchi.mtg import phrases

    assert phrases.PHRASES_PATH.exists(), "constants/phrases.json is missing"
    doc = json.loads(phrases.PHRASES_PATH.read_text(encoding="utf-8"))
    assert set(doc) == {"every", "phrases", "greetings", "help", "dark", "light"}


def test_normalize_name():
    from muchi.mtg.text import normalize_name
    assert normalize_name("Ragavan, Nimble Pilferer") == "ragavan-nimble-pilferer"
    assert normalize_name("Jotun Grunt") == "jotun-grunt"
    assert normalize_name("Sol Ring") == "sol-ring"










































# ----------------------------------------------------------------- the ports


def test_only_the_cast_imports_sources():
    """The core depends on the shape. If this fails, a vendor leaked in."""
    root = Path(__file__).resolve().parent.parent
    revisados = (list((root / "muchi" / "mtg").glob("*.py"))
                 + list((root / "server").glob("*.py")))
    # If the package moves again, this test would pass looking at zero files.
    assert len(revisados) > 10, f"the package path is wrong: {revisados}"

    offenders = []
    for py in revisados:
        if py.name == "cast.py":
            continue
        for line in py.read_text(encoding="utf-8").splitlines():
            if line.startswith(("from .sources", "from muchi.mtg.sources")):
                offenders.append(f"{py.name}: {line}")
    assert offenders == [], offenders


# ---------------------------------------------------------------- the offers
class _FakeSource:
    def __init__(self, name, offers=(), explodes=False):
        self.name = name
        self._offers = list(offers)
        self._explodes = explodes

    def find_offers(self, card_name):
        if self._explodes:
            raise RuntimeError("the store went down")
        return list(self._offers)


class _FakePrimary(_FakeSource):
    def identify_card(self, name):
        return "id-1", list(self._offers)

    def suggest_names(self, text):
        return []

    def refresh_offers(self, card_id):
        return iter(())


















# -------------------------------------------------------------------- the deck


# ----------------------------------------------------------------- the stores




# ------------------------------------------------------- decklist: copy-paste
def test_decklist_rejects_malformed_lines():
    """Whatever no card name can possibly be, report it instead of letting it slip."""
    orders, ignored = decklist.parse_decklist(
        "4 Lightning Bolt\n"
        "esto no se entiende ###\n"
        "https://moxfield.com/decks/abc123\n"
        "Total: $45.000\n"
        "100% completo\n"
        "42"
    )
    assert [p.name for p in orders] == ["Lightning Bolt"]
    assert len(ignored) == 4, ignored
    assert "esto no se entiende ###" in ignored


def test_decklist_accepts_odd_real_names():
    """The filter must not eat cards that actually exist."""
    odd_names = [
        'Kongming, "Sleeping Dragon"',   # double quotes
        "+2 Mace",                        # starts with sign
        "Sword of Dungeons & Dragons",   # ampersand
        "Jotun Grunt",
        "Mr. Orfeo, the Boulder",        # dot
        "Ach! Hans, Run!",               # exclamation marks
        "Borrowing 100,000 Arrows",      # digits and comma
        "Yawgmoth's Will",               # apostrophe
        "Lim-Dul the Necromancer",       # hyphen
        "Question Elemental?",           # question mark
    ]
    orders, ignored = decklist.parse_decklist("\n".join(odd_names))
    assert ignored == [], ignored
    assert [p.name for p in orders] == odd_names


def test_decklist_accents_and_ligatures():
    orders, ignored = decklist.parse_decklist("1 Jotun Grunt\n1 Aether Vial\n1 Seance")
    assert ignored == []
    assert len(orders) == 3


def test_decklist_strips_exporter_noise():
    """Moxfield, Arena, Archidekt and deckstats hang extra tokens at the end."""
    orders, ignored = decklist.parse_decklist(
        "1 Sol Ring (LTC) 344 *F*\n"       # Moxfield with foil
        "1x Arcane Signet (c21) 263\n"     # Archidekt
        "1 Command Tower #!Commander\n"    # deckstats
        "1 Cultivate [Ramp]\n"             # Archidekt category
        "4 Forest (UNF) 235"               # Arena, basic land
    )
    assert ignored == [], ignored
    assert {p.name: p.quantity for p in orders} == {
        "Sol Ring": 1, "Arcane Signet": 1, "Command Tower": 1,
        "Cultivate": 1, "Forest": 4,
    }


def test_decklist_cleans_clipboard_noise():
    """Curly quotes, non-breaking space, double space and markdown bullets."""
    orders, ignored = decklist.parse_decklist(
        "1 Yawgmoth\u2019s Will\n"        # curly apostrophe
        "2 Sol\u00a0Ring\n"               # non-breaking space
        "3 Lightning  Bolt\n"             # double space
        "- 4 Counterspell\n"              # list bullet
        "\u2022 1 Brainstorm"
    )
    assert ignored == [], ignored
    assert {p.name: p.quantity for p in orders} == {
        "Yawgmoth's Will": 1, "Sol Ring": 2, "Lightning Bolt": 3,
        "Counterspell": 4, "Brainstorm": 1,
    }


def test_decklist_sums_by_flattened_name():
    """Pasting the same card with different punctuation must not duplicate the line."""
    orders, _ = decklist.parse_decklist(
        "2 Yawgmoth's Will\n1 Yawgmoths Will\n1 YAWGMOTH'S WILL"
    )
    assert len(orders) == 1, [p.name for p in orders]
    assert orders[0].quantity == 4
    assert orders[0].name == "Yawgmoth's Will", "keeps the first form seen"


def test_decklist_headers_are_not_cards():
    orders, ignored = decklist.parse_decklist(
        "Deck\n4 Lightning Bolt\nCreatures (12)\nSideboard:\n"
        "Rampa:\n2 Sol Ring\nCommander\n1 Atraxa"
    )
    assert ignored == [], ignored
    assert {p.name for p in orders} == {"Lightning Bolt", "Sol Ring", "Atraxa"}


def test_decklist_basic_lands_are_cards():
    """'Land' is a header; 'Forest' is a card. Don't confuse them."""
    orders, _ = decklist.parse_decklist(
        "Lands\n10 Forest\n5 Island\n1 Swamp\n1 Mountain\n1 Plains"
    )
    assert {p.name for p in orders} == {
        "Forest", "Island", "Swamp", "Mountain", "Plains"}


def test_decklist_keeps_arrival_order():
    orders, _ = decklist.parse_decklist("1 Zur\n1 Alesha\n1 Muldrotha")
    assert [p.name for p in orders] == ["Zur", "Alesha", "Muldrotha"]


# ------------------------------------------------------------------- paths
def test_root_paths_match_repo():
    """muchi.paths counts levels; this test counts them separately and compares.

    If both used the same helper, a counting error would be invisible.
    """
    from muchi import paths

    root = Path(__file__).resolve().parent.parent
    assert paths.ROOT == root, f"{paths.ROOT} != {root}"
    assert paths.ASSETS == root / "assets"
    assert paths.DATA == root / "data"





# ------------------------------------------------------------- el oraculo
# El puente es->en y la escalera de pedidos. Es texto puro y nucleo puro: se
# prueba entero sin red, que es justo lo que lo hace confiable.

























# -------------------------------------------------------------- el catalogo
# Traduccion de pedidos a sintaxis de Scryfall y parseo de sus respuestas.

CARD_JSON = {
    "name": "Lightning Bolt", "lang": "en",
    "oracle_id": "4457ed35-7c10-48c8-9776-456485fdf070",
    "oracle_text": "Lightning Bolt deals 3 damage to any target.",
    "type_line": "Instant", "mana_cost": "{R}", "rarity": "common",
    "image_uris": {"normal": "https://img/bolt.jpg"},
    "scryfall_uri": "https://scryfall.com/card/lea/161",
}


















if __name__ == "__main__":
    failures = 0
    for name, fn in sorted(globals().items()):
        if not name.startswith("test_") or not callable(fn):
            continue
        try:
            fn()
            print(f"  ok   {name}")
        except AssertionError as e:
            failures += 1
            print(f"  FAIL {name}: {e}")
    print("\nAll green" if not failures else f"\n{failures} tests failed")
    sys.exit(1 if failures else 0)
