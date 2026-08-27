"""Tests for logic that does not touch the network.

Run with pytest, or directly:  python tests/test_muchi.py
"""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from muchi.mtg import catalog, decklist, mascot, deck, offers, optimizer, oracle  # noqa: E402
from muchi.mtg.models import Offer, Order  # noqa: E402
from muchi.mtg.ports import CardRequest  # noqa: E402
from muchi.mtg.sources import (  # noqa: E402
    store_api, edhrec, moxfield, scry, scryfall, shopify,
)


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


def _memory_db():
    import sqlite3
    cx = sqlite3.connect(":memory:")
    cx.row_factory = sqlite3.Row
    catalog.ensure_tables(cx)
    return cx


def _seed_row(cx, store, card_name, price, url):
    from muchi.mtg.text import normalize_name as _slug
    cx.execute(
        "INSERT INTO catalogo (clave, tienda, carta_slug, carta, titulo, precio,"
        " url, acabado, condicion, idioma) VALUES (?,?,?,?,?,?,?,?,?,?)",
        (url, store, _slug(card_name), card_name, f"{card_name} [XYZ - 1]", price, url,
         "Normal", "Near Mint", "English"),
    )
    cx.commit()


def test_catalog_matches_by_slug():
    cx = _memory_db()
    _seed_row(cx, "PDA Chile", "Ragavan, Nimble Pilferer", 90000, "http://x/1")
    # distinta capitalizacion y puntuacion deben encontrar lo mismo
    assert len(catalog.find_local_offers(cx, "ragavan nimble pilferer")) == 1
    assert len(catalog.find_local_offers(cx, "RAGAVAN, NIMBLE PILFERER")) == 1
    assert catalog.find_local_offers(cx, "Otra Carta") == []


def test_catalog_sorts_by_price():
    cx = _memory_db()
    _seed_row(cx, "PDA Chile", "Sol Ring", 5000, "http://x/caro")
    _seed_row(cx, "PDA Chile", "Sol Ring", 1200, "http://x/barato")
    offers = catalog.find_local_offers(cx, "Sol Ring")
    assert [o.price_clp for o in offers] == [1200, 5000]
    assert all(o.source == "directo" for o in offers)
    # they are never marketplace: these are stores with their own site
    assert all(not o.marketplace for o in offers)


def test_pda_chile_configured():
    assert "PDA Chile" in shopify.STORES
    assert shopify.STORES["PDA Chile"].startswith("https://")
    # unreachable stores are declared with a reason, never silently omitted
    for store, (url, reason) in shopify.BLOCKED_STORES.items():
        assert url.startswith("https://") and len(reason) > 20, store


def test_store_api_without_config():
    """The feature is optional: no file, empty list and zero requests."""
    assert store_api.load_stores("no-existe-este-archivo.json") == []


def test_store_api_maps_fields():
    store = store_api.StoreApi(
        name="Wombat", url="https://x/rest/v1/stock", kind="postgrest",
        fields={"nombre": "carta", "precio": "precio_clp", "stock": "cantidad",
                "edicion": "set_codigo", "condicion": "estado", "url": "link"},
    )
    rows = [
        {"carta": "Sol Ring", "precio_clp": 3500, "cantidad": 4,
         "set_codigo": "C21", "estado": "NM", "link": "https://x/sol-ring"},
        {"carta": "Black Lotus", "precio_clp": 999999, "cantidad": 0},  # out of stock
        {"carta": "Roto", "precio_clp": None, "cantidad": 2},           # no price
        {"carta": "Brainstorm", "precio_clp": "4.750", "cantidad": 1},  # price with dot
    ]
    offers = store_api.build_offers(store, rows)

    assert [o.card_name for o in offers] == ["Sol Ring", "Brainstorm"]
    o = offers[0]
    assert o.price_clp == 3500 and o.store == "Wombat"
    assert o.title == "Sol Ring [C21] - NM"
    assert o.url == "https://x/sol-ring"
    assert o.source == "api" and not o.marketplace
    assert offers[1].price_clp == 4750, "must parse '4.750' as 4750"


def test_store_api_key_comes_from_env():
    store = store_api.StoreApi(name="X", url="https://x", env_apikey="MUCHI_TEST_KEY")
    os.environ.pop("MUCHI_TEST_KEY", None)
    assert store.api_key == ""
    os.environ["MUCHI_TEST_KEY"] = "secreta"
    try:
        assert store.api_key == "secreta"
    finally:
        os.environ.pop("MUCHI_TEST_KEY", None)


def test_example_config_is_valid_json():
    path = Path(__file__).resolve().parent.parent / "store-api.example.json"
    data = json.loads(path.read_text(encoding="utf-8"))
    stores = [store_api.build_store(row) for row in data["tiendas"]]
    assert len(stores) == 2
    # the example must not carry any real API key
    for t in stores:
        assert not t.api_key, f"{t.name} ships an API key in the example"


def test_moxfield_list_id():
    assert moxfield.extract_list_id(
        "https://moxfield.com/decks/yqdRPdoUlEiFz21qKYHGMA") == "yqdRPdoUlEiFz21qKYHGMA"
    assert moxfield.extract_list_id(
        "https://www.moxfield.com/decks/l-hvQOWPTEKJ0rMjMUHJxg/") == "l-hvQOWPTEKJ0rMjMUHJxg"
    assert moxfield.extract_list_id("l_sw58BtJUaWZj9n6VNCDw") == "l_sw58BtJUaWZj9n6VNCDw"
    for broken in ("", "https://ejemplo.cl/algo/otro"):
        try:
            moxfield.extract_list_id(broken)
            raise AssertionError(f"should reject {broken!r}")
        except ValueError:
            pass


def test_moxfield_foil_uses_ck_foil():
    """The expensive bug: quoting a foil at the non-foil price."""
    prices = {"ck": 1.50, "ck_foil": 17.99}
    assert moxfield.read_ck_price(prices, is_foil=True) == 17.99
    assert moxfield.read_ck_price(prices, is_foil=False) == 1.50
    # if the requested finish is missing, do NOT fall through to the other one
    assert moxfield.read_ck_price({"ck": 1.50}, is_foil=True) is None
    assert moxfield.read_ck_price({"ck_foil": 17.99}, is_foil=False) is None
    assert moxfield.read_ck_price({}, is_foil=False) is None


def test_inventory_config_is_coherent():
    path = Path(__file__).resolve().parent.parent / "moxfield-inventories.json"
    cfg = json.loads(path.read_text(encoding="utf-8"))
    lists = cfg["listas"]
    assert len(lists) == 9
    for l in lists:
        assert moxfield.extract_list_id(l["url"])
        assert 100 <= l["tasa"] <= 2000, l
    # the japanese foils list quotes differently, and that must not get lost
    off_rate = [l for l in lists if l["tasa"] != 700]
    assert len(off_rate) == 1 and off_rate[0]["tasa"] == 500, off_rate


def test_hearts_vary_between_clicks():
    first = mascot.build_hearts_html(seed=1)
    second = mascot.build_hearts_html(seed=2)
    # Careful: the container class is "mu-corazones" which contains "mu-corazon".
    assert first.count('class="mu-corazon"') == 9
    assert first != second, "same positions every time would look like a canned animation"


def test_greetings_and_help_not_empty():
    assert mascot.GREETINGS and all(s.strip() for s in mascot.GREETINGS)
    for title, detail in mascot.HELP_TOPICS:
        assert title.strip() and len(detail) > 30, title


# ------------------------------------------------------------------ phrases
def test_phrase_book_reads_yaml():
    from muchi.mtg import phrases

    book = phrases.read_phrases()
    assert book.every == 10
    assert book.phrases, "the phrases YAML must not be empty"
    assert all(p.text.strip() for p in book.phrases)
    states = {p.state for p in book.phrases}
    assert states <= {"idle", "talk", "happy", "alert", "angry"}, states


def test_phrase_book_missing_file_is_harmless():
    from muchi.mtg import phrases

    book = phrases.read_phrases(Path("no-such-phrases.yaml"))
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

    assert phrases.PHRASES_PATH.exists(), "muchi-phrases.yaml is missing from the repo root"
    text = phrases.PHRASES_PATH.read_text(encoding="utf-8")
    assert "amsiedad" in text


def test_normalize_name():
    from muchi.mtg.text import normalize_name
    assert normalize_name("Ragavan, Nimble Pilferer") == "ragavan-nimble-pilferer"
    assert normalize_name("Jotun Grunt") == "jotun-grunt"
    assert normalize_name("Sol Ring") == "sol-ring"


def test_parse_offers_from_real_html():
    """Exact format that scry.cl server-renders."""
    html = (
        '<a data-track-type="store_offer_click" data-store-name="PayToWin" '
        'data-card-name="Ragavan, Nimble Pilferer" '
        'data-offer-title="Ragavan, Nimble Pilferer [MH2] - Near Mint Foil" '
        'data-price-clp="118400" data-variant-key="k1" '
        'data-product-url="https://www.paytowin.cl/products/x?variant=1">Ver</a>'
        '<a data-track-type="store_offer_click" data-store-name="CatLotus" '
        'data-offer-title="Ragavan, Nimble Pilferer [MH2] - Near Mint" '
        'data-price-clp="99000" data-variant-key="k2" '
        'data-product-url="https://catlotus.cl/y">Ver</a>'
    )
    offers = scry.parse_offers_html(html)
    assert len(offers) == 2
    assert offers[0].store == "CatLotus" and offers[0].price_clp == 99000  # sorted by price
    assert offers[1].is_foil is True
    assert offers[0].is_foil is False
    assert offers[1].condition == "Near Mint"
    assert offers[0].edition == "MH2"


def test_marketplace_vs_store():
    """Scry marketplace offers come from marketplace.scry.cl; stores don't."""
    assert scry.is_marketplace("https://marketplace.scry.cl/magic-master/sol-ring") is True
    assert scry.is_marketplace("https://scry.cl/card/sol-ring") is True
    assert scry.is_marketplace("https://catlotus.cl/carta/123") is False
    assert scry.is_marketplace("https://www.paytowin.cl/products/x") is False
    assert scry.is_marketplace("https://gameofmagicsingles.cl/products/y") is False
    # Must not match a domain that just happens to contain the substring
    assert scry.is_marketplace("https://noscry.cl/x") is False


def test_marks_marketplace_on_parse():
    html = (
        '<a data-track-type="store_offer_click" data-store-name="Magic Master" '
        'data-offer-title="Sol Ring - NM" data-price-clp="1791" data-variant-key="k1" '
        'data-product-url="https://marketplace.scry.cl/magic-master/sol-ring">Ver</a>'
        '<a data-track-type="store_offer_click" data-store-name="CatLotus" '
        'data-offer-title="Sol Ring - NM" data-price-clp="2000" data-variant-key="k2" '
        'data-product-url="https://catlotus.cl/x">Ver</a>'
    )
    offers = scry.parse_offers_html(html)
    by_store = {o.store: o.marketplace for o in offers}
    assert by_store == {"Magic Master": True, "CatLotus": False}, by_store


def _rec(name, category="Top Cards", inc=0.5, syn=0.1):
    return edhrec.RawRecommendation(name=name, category=category, tag=category.lower(),
                                    inclusion=inc, synergy=syn, num_decks=10)


def test_edhrec_normalizes_commanders():
    assert edhrec.normalize_name("Atraxa, Praetors' Voice") == "atraxa-praetors-voice"
    assert edhrec.normalize_name("Kenrith, the Returned King") == "kenrith-the-returned-king"
    assert edhrec.normalize_name("Edgar Markov") == "edgar-markov"


def test_subtracts_cards_already_owned():
    recs = [_rec("Sol Ring"), _rec("Skullclamp"), _rec("Arcane Signet")]
    # The comparison is normalised: case and punctuation must not matter
    missing_recs = deck.subtract_owned_cards(recs, {"sol ring", "ARCANE SIGNET"})
    assert [r.name for r in missing_recs] == ["Skullclamp"]


def test_without_a_deck_returns_all():
    recs = [_rec("Sol Ring"), _rec("Skullclamp")]
    assert len(deck.subtract_owned_cards(recs, set())) == 2


def test_categories_keep_their_order():
    recs = [_rec("A", "Top Cards"), _rec("B", "Creatures"),
            _rec("C", "Top Cards"), _rec("D", "Instants")]
    assert deck.list_categories(recs) == ["Top Cards", "Creatures", "Instants"]


def test_basic_lands_are_recognised():
    for n in ("Mountain", "island", "Snow-Covered Forest", "Wastes"):
        assert edhrec.is_basic_land(n) is True, n
    for n in ("Mountain Valley", "Goblin Warchief", "Islandia"):
        assert edhrec.is_basic_land(n) is False, n


def test_inclusion_pct():
    assert _rec("X", inc=0.6127).inclusion_pct == 61.3


def test_shopify_parses_title():
    d = shopify.parse_title("Ragavan, Nimble Pilferer (Borderless) [MH2 - 138]")
    assert d["nombre"] == "Ragavan, Nimble Pilferer"
    assert d["variante"] == "Borderless"
    assert d["set"] == "MH2" and d["cn"] == "138"

    d2 = shopify.parse_title("Growth Spiral (7054) [SLD - 7054]")
    assert d2["nombre"] == "Growth Spiral" and d2["set"] == "SLD"


def test_shopify_product_offers():
    product = {
        "title": "Sol Ring [C21 - 263]",
        "handle": "sol-ring-c21",
        "variants": [
            {"id": 1, "title": "Near Mint / English / Normal", "price": "3500", "available": True},
            {"id": 2, "title": "Near Mint / English / Foil", "price": "9000", "available": False},
        ],
    }
    offers = shopify.build_product_offers(product, "PayToWin", "https://www.paytowin.cl")
    assert len(offers) == 1, "out-of-stock variants are discarded"
    assert offers[0].price_clp == 3500
    assert offers[0].language == "English"
    assert "variant=1" in offers[0].url


# ----------------------------------------------------------------- the ports
def test_every_source_meets_its_port():
    """An adapter that stops fulfilling its port breaks here, not in production."""
    from muchi.mtg import ports
    from muchi.mtg.http import PoliteSession
    from muchi.mtg.sources.store_api import StoreApiSource
    from muchi.mtg.sources.edhrec import EdhrecAdvisor
    from muchi.mtg.sources.scry import ScrySource
    from muchi.mtg.sources.shopify import ShopifyCatalog

    sess = PoliteSession()
    assert isinstance(ScrySource(sess), ports.PrimarySource)
    assert isinstance(ScrySource(sess), ports.OfferSource)
    assert isinstance(catalog.IndexedOffers(_memory_db()), ports.OfferSource)
    assert isinstance(StoreApiSource(sess, store_api.StoreApi("X", "u")),
                      ports.OfferSource)
    assert isinstance(EdhrecAdvisor(sess), ports.DeckAdvisor)
    assert isinstance(ShopifyCatalog(sess), ports.StoreCatalog)


def test_only_the_cast_imports_sources():
    """The core depends on the shape. If this fails, a vendor leaked in."""
    root = Path(__file__).resolve().parent.parent
    revisados = list((root / "muchi" / "mtg").glob("*.py")) + [root / "app.py"]
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


def test_find_offers_sorts_and_dedupes():
    """When the primary source already covers a store, the secondary skips it."""
    primary = _FakePrimary("scry", [_offer("CatLotus", "Sol Ring", 5000)])
    local = _FakeSource("local index", [
        _offer("CatLotus", "Sol Ring", 4000),   # already covered: discarded
        _offer("PDA Chile", "Sol Ring", 3000),  # new: enters the list
    ])

    card_id, found = offers.find_offers(primary, [local], "Sol Ring")

    assert card_id == "id-1"
    assert [(o.store, o.price_clp) for o in found] == [
        ("PDA Chile", 3000), ("CatLotus", 5000),
    ]


def test_a_failing_source_does_not_sink_search():
    primary = _FakePrimary("scry", [_offer("CatLotus", "Sol Ring", 5000)])
    broken_source = _FakeSource("Wombat", explodes=True)
    healthy = _FakeSource("PDA Chile", [_offer("PDA Chile", "Sol Ring", 3000)])

    _, found = offers.find_offers(primary, [broken_source, healthy], "Sol Ring")
    assert [o.store for o in found] == ["PDA Chile", "CatLotus"]


def test_filter_hides_sellers_and_foils():
    from muchi.mtg.models import Offer
    normal = Offer("CatLotus", "Sol Ring", "Sol Ring", 1000, "u", finish="Normal")
    foil = Offer("PDA Chile", "Sol Ring", "Sol Ring", 2000, "u", finish="Foil")
    loose = Offer("Pepito", "Sol Ring", "Sol Ring", 500, "u", marketplace=True)
    all_offers = [normal, foil, loose]

    assert [o.store for o in offers.filter_offers(all_offers)] == ["CatLotus", "PDA Chile"]
    assert len(offers.filter_offers(all_offers, stores_only=False)) == 3
    assert [o.store for o in offers.filter_offers(all_offers, "Solo foil")] == ["PDA Chile"]
    assert [o.store for o in offers.filter_offers(all_offers, "Solo normal")] == ["CatLotus"]
    assert [o.store for o in offers.filter_offers(all_offers, stores=["PDA Chile"])] \
        == ["PDA Chile"]


# -------------------------------------------------------------------- the deck
def test_deck_subtracts_and_lists_categories():
    from muchi.mtg.ports import Recommendation

    recs = [
        Recommendation("Sol Ring", "Top Cards", 0.9, 0.1),
        Recommendation("Arcane Signet", "Mana Artifacts", 0.8, 0.2),
    ]
    assert [r.name for r in deck.subtract_owned_cards(recs, {"sol ring"})] \
        == ["Arcane Signet"]
    assert deck.list_categories(recs) == ["Top Cards", "Mana Artifacts"]
    assert recs[0].inclusion_pct == 90.0


# ----------------------------------------------------------------- the stores
class _FakeCatalog:
    def list_indexable_stores(self):
        return {"PDA Chile": "https://www.pdachile.cl"}

    def list_blocked_stores(self):
        return {}

    def download_offers(self, store, url, progress=None):
        return [_offer(store, "Sol Ring", 3000)], 1


def test_index_store_saves_and_reports():
    from muchi.mtg import stores as store_index

    cx = _memory_db()
    before = store_index.read_stores_status(cx, _FakeCatalog())
    assert before[0].store == "PDA Chile" and before[0].indexed is False

    assert store_index.index_store(cx, _FakeCatalog(), "PDA Chile", "https://x") == 1

    after = store_index.read_stores_status(cx, _FakeCatalog())
    assert after[0].indexed is True
    assert after[0].offers == 1 and after[0].products == 1
    assert catalog.find_local_offers(cx, "Sol Ring")[0].price_clp == 3000


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


def test_package_paths_stay_within_repo():
    """A path that goes one level too far doesn't explode: it points elsewhere."""
    from muchi import paths
    from muchi.mtg import db
    from muchi.mtg.sources import moxfield, store_api

    for label, path in [("db.PATH", db.PATH),
                        ("store_api.CONFIG", store_api.CONFIG),
                        ("moxfield.CONFIG", moxfield.CONFIG)]:
        assert paths.ROOT in path.parents, f"{label} escaped the repo: {path}"

    assert moxfield.CONFIG.exists(), "the inventory config should be where the path points"


# ------------------------------------------------------------- el oraculo
# El puente es->en y la escalera de pedidos. Es texto puro y nucleo puro: se
# prueba entero sin red, que es justo lo que lo hace confiable.

def test_oracle_translates_a_known_phrase():
    phrases, loose = oracle.parse_request("destruye la criatura objetivo")
    assert phrases == ["destroy target creature"]
    assert loose == [], "'objetivo' ya viene dentro de la frase traducida"


def test_oracle_prefers_the_longest_phrase():
    phrases, _ = oracle.parse_request("roba una carta")
    assert phrases == ["draw a card"], "no debe partirse en 'draw' mas relleno"


def test_oracle_accepts_spanglish():
    """Asi se escribe de verdad: media frase en cada idioma."""
    phrases, loose = oracle.parse_request("destruye target creature")
    assert "destroy" in phrases
    assert set(loose) == {"target", "creature"}


def test_oracle_ignores_accents_and_case():
    assert (oracle.parse_request("DESTRUYE la Criatura")[0]
            == oracle.parse_request("destruye la criatura")[0])
    assert oracle.parse_request("hace daño")[0] == ["deals damage"], \
        "la tilde y la enie no pueden dejar la frase sin traducir"


def test_oracle_drops_filler_words():
    _, loose = oracle.parse_request("quiero una carta que roba cartas")
    assert loose == [], "'quiero', 'una', 'que' no pueden llegar al pedido"


def test_oracle_without_anything_invents_nothing():
    assert oracle.build_requests("") == []


def test_oracle_first_request_is_the_strictest():
    steps = oracle.build_requests("roba una carta", card_type="creature")
    assert steps[0].phrases == ("draw a card",)
    assert steps[0].card_type == "creature"
    assert steps[0].note == "", "la primera no afloja nada, no hay que avisar"


def test_oracle_ladder_loosens_step_by_step():
    """Con una palabra que no conocemos, el segundo escalon la deja fuera."""
    steps = oracle.build_requests("destruye chuchunco")
    assert steps[0].words == ("chuchunco",)
    assert steps[1].words == () and steps[1].phrases == ("destroy",)
    assert steps[1].note, "cuando afloja tiene que poder explicarlo"
    assert not any(s.is_empty for s in steps), "ningun pedido puede salir vacio"


def test_oracle_never_drops_every_word():
    """Soltar todas las palabras no es aflojar: es devolver el catalogo entero.

    "counter target spell" no matchea ninguna frase del diccionario --- ya viene
    en ingles --- asi que las tres palabras son lo unico que dice que buscar.
    """
    steps = oracle.build_requests("counter target spell", card_type="instant")
    assert steps[0].words == ("counter", "target", "spell")
    assert all(s.words or s.phrases or s.literal_text or s.note == "solo con los filtros"
               for s in steps)
    assert steps[1].literal_text, "antes de rendirse prueba el texto tal cual"


def test_oracle_intent_carries_a_fallback():
    steps = oracle.build_requests("", intents=("removal",))
    assert steps[0].intents == ("removal",)
    assert any("destroy target" in p for s in steps for p in s.phrases), \
        "si la etiqueta del proveedor muere, tiene que quedar un plan B"


def test_oracle_raw_spanish_is_the_last_resort():
    """Buscar el texto en espanol puede no funcionar: nunca va primero."""
    steps = oracle.build_requests("destruye la criatura objetivo")
    literal = [i for i, s in enumerate(steps) if s.literal_text]
    assert literal and literal[0] > 0


def test_oracle_declares_no_vendor_syntax():
    """El nucleo pide en sustantivos; la sintaxis vive detras del puerto."""
    source = (Path(__file__).resolve().parent.parent
              / "muchi" / "mtg" / "oracle.py").read_text(encoding="utf-8")
    for leak in ("otag:", "o:\"", "id<=", "mv<=", "scryfall"):
        assert leak not in source, f"se filtro sintaxis del proveedor: {leak}"


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


def test_scryfall_renders_a_request():
    query = scryfall.render_query(CardRequest(
        phrases=("draw a card",), words=("dies",), intents=("removal",),
        colors=("w", "u"), card_type="creature", format_name="commander",
        max_mana=3,
    ))
    assert query == ('o:"draw a card" o:dies otag:removal id<=wu '
                     't:creature f:commander mv<=3')


def test_scryfall_ignores_unknown_intents():
    """Un chip que el proveedor no tiene no puede ensuciar la query."""
    query = scryfall.render_query(CardRequest(phrases=("draw a card",),
                                              intents=("no-existe",)))
    assert query == 'o:"draw a card"'


def test_scryfall_parses_a_card():
    card = scryfall.parse_card(CARD_JSON)
    assert card.name == "Lightning Bolt"
    assert card.image == "https://img/bolt.jpg"
    assert card.mana_cost == "{R}"
    assert not card.is_translated


def test_scryfall_keeps_the_english_name_on_translations():
    """Lo mas importante del modulo: al carrito va 'Lightning Bolt', no 'Rayo'.

    Las tiendas chilenas indexan por el nombre en ingles. Si la clave canonica
    se contaminara con el traducido, la cotizacion no encontraria nada.
    """
    card = scryfall.parse_card({
        **CARD_JSON, "lang": "es", "printed_name": "Rayo",
        "printed_text": "Rayo hace 3 puntos de dano a cualquier objetivo.",
        "printed_type_line": "Instantaneo",
    })
    assert card.name == "Lightning Bolt"
    assert card.local_name == "Rayo"
    assert card.show_as("es")[0] == "Rayo"
    assert card.show_as("en")[0] == "Lightning Bolt"


def test_scryfall_falls_back_to_english():
    card = scryfall.parse_card(CARD_JSON)
    name, type_line, text, image = card.show_as("es")
    assert name == "Lightning Bolt"
    assert type_line == "Instant" and text.startswith("Lightning Bolt deals")
    assert image == "https://img/bolt.jpg"


def test_scryfall_joins_both_faces():
    card = scryfall.parse_card({
        "name": "Delver of Secrets // Insectile Aberration", "lang": "en",
        "card_faces": [
            {"name": "Delver of Secrets",
             "oracle_text": "At the beginning of your upkeep...",
             "image_uris": {"normal": "https://img/delver.jpg"}},
            {"name": "Insectile Aberration", "oracle_text": "Flying"},
        ],
    })
    assert "Flying" in card.text and "upkeep" in card.text
    assert card.image == "https://img/delver.jpg", "la imagen sale de la cara que la tenga"


def test_scryfall_builds_an_exact_name_query():
    assert scryfall.build_name_query(["Sol Ring", "Counterspell"]) \
        == '(!"Sol Ring" or !"Counterspell")'
    assert scryfall.build_name_query([]) == ""


def test_scryfall_fulfils_the_catalog_port():
    from muchi.mtg.ports import CardCatalog
    assert isinstance(scryfall.ScryfallCatalog(sess=None), CardCatalog)


def test_muchi_mentions_the_new_finder():
    titles = " ".join(t for t, _ in mascot.HELP_TOPICS).lower()
    assert "no sabes que carta" in titles


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

