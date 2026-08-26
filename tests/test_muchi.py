"""Pruebas de la logica que no depende de la red.

Corre con pytest, o directo:  python tests/test_muchi.py
"""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from muchi.mtg import catalog, decklist, mascot, deck, offers, optimizer  # noqa: E402
from muchi.mtg.models import Offer, Order  # noqa: E402
from muchi.mtg.sources import store_api, edhrec, moxfield, scry, shopify  # noqa: E402


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
    # El bug clasico: sin el lookahead (?=\d), "Sol Ring" se parseaba "Sol"
    # y "Lightning Bolt" quedaba en "Lightning".
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
    """Con envio caro, conviene una sola tienda aunque las cartas salgan mas."""
    orders = [Order(1, "A"), Order(1, "B")]
    offers = {
        "a": [_offer("T1", "A", 1000), _offer("T2", "A", 900)],
        "b": [_offer("T1", "B", 1000), _offer("T3", "B", 900)],
    }

    naive = optimizer.build_naive_plan(orders, offers, shipping_per_store=5000)
    # Ingenuo: 900 + 900 + 2 envios = 11800, en dos tiendas distintas
    assert len(naive.stores) == 2 and naive.total == 11800, naive.total

    optimal = optimizer.build_optimal_plan(orders, offers, shipping_per_store=5000)
    # Optimo: todo en T1 = 2000 + 1 envio = 7000
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
    # nunca son marketplace: son tiendas con sitio propio
    assert all(not o.marketplace for o in offers)


def test_pda_chile_configured():
    assert "PDA Chile" in shopify.STORES
    assert shopify.STORES["PDA Chile"].startswith("https://")
    # las inalcanzables se declaran con motivo, no se omiten en silencio
    for store, (url, reason) in shopify.BLOCKED_STORES.items():
        assert url.startswith("https://") and len(reason) > 20, store


def test_store_api_without_config():
    """La feature es opcional: sin archivo, lista vacia y cero peticiones."""
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
        {"carta": "Black Lotus", "precio_clp": 999999, "cantidad": 0},  # sin stock
        {"carta": "Roto", "precio_clp": None, "cantidad": 2},           # sin precio
        {"carta": "Brainstorm", "precio_clp": "4.750", "cantidad": 1},  # precio con punto
    ]
    offers = store_api.build_offers(store, rows)

    assert [o.card_name for o in offers] == ["Sol Ring", "Brainstorm"]
    o = offers[0]
    assert o.price_clp == 3500 and o.store == "Wombat"
    assert o.title == "Sol Ring [C21] - NM"
    assert o.url == "https://x/sol-ring"
    assert o.source == "api" and not o.marketplace
    assert offers[1].price_clp == 4750, "debe parsear '4.750' como 4750"


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
    # el ejemplo no debe traer ninguna clave de verdad
    for t in stores:
        assert not t.api_key, f"{t.name} trae una clave en el archivo"


def test_moxfield_list_id():
    assert moxfield.extract_list_id(
        "https://moxfield.com/decks/yqdRPdoUlEiFz21qKYHGMA") == "yqdRPdoUlEiFz21qKYHGMA"
    assert moxfield.extract_list_id(
        "https://www.moxfield.com/decks/l-hvQOWPTEKJ0rMjMUHJxg/") == "l-hvQOWPTEKJ0rMjMUHJxg"
    assert moxfield.extract_list_id("l_sw58BtJUaWZj9n6VNCDw") == "l_sw58BtJUaWZj9n6VNCDw"
    for broken in ("", "https://ejemplo.cl/algo/otro"):
        try:
            moxfield.extract_list_id(broken)
            raise AssertionError(f"deberia rechazar {broken!r}")
        except ValueError:
            pass


def test_moxfield_foil_uses_ck_foil():
    """El bug caro: cotizar un foil con el precio no-foil."""
    prices = {"ck": 1.50, "ck_foil": 17.99}
    assert moxfield.read_ck_price(prices, is_foil=True) == 17.99
    assert moxfield.read_ck_price(prices, is_foil=False) == 1.50
    # si falta el del acabado pedido, NO se cae al otro
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
    # la lista de foils japoneses cotiza distinto y eso no debe perderse
    off_rate = [l for l in lists if l["tasa"] != 700]
    assert len(off_rate) == 1 and off_rate[0]["tasa"] == 500, off_rate


def test_muchi_sprite_is_a_gif():
    path = Path(__file__).resolve().parent.parent / "assets" / "muchi.gif"
    assert path.exists(), "falta assets/muchi.gif (corre tools/generate_muchi.py)"
    assert path.read_bytes()[:6] in (b"GIF87a", b"GIF89a")
    uri = mascot.read_sprite_datauri(path)
    assert uri.startswith("data:image/gif;base64,")


def test_missing_sprite_is_harmless():
    """Si falta el GIF cae a un emoji en vez de reventar la app."""
    assert mascot.read_sprite_datauri("no-existe.gif") is None
    html = mascot.build_cat_html(None)
    assert "mu-gato" in html and "img" not in html


def test_hearts_vary_between_clicks():
    first = mascot.build_hearts_html(seed=1)
    second = mascot.build_hearts_html(seed=2)
    # Cuidado: el contenedor se llama "mu-corazones" y contiene la subcadena.
    assert first.count('class="mu-corazon"') == 9
    assert first != second, "con la misma posicion siempre se notaria que es la misma animacion"


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
    """Formato exacto que server-rendera scry.cl."""
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
    assert offers[0].store == "CatLotus" and offers[0].price_clp == 99000  # ordenado
    assert offers[1].is_foil is True
    assert offers[0].is_foil is False
    assert offers[1].condition == "Near Mint"
    assert offers[0].edition == "MH2"


def test_marketplace_vs_store():
    """Los particulares de scry cuelgan de marketplace.scry.cl; las tiendas no."""
    assert scry.is_marketplace("https://marketplace.scry.cl/magic-master/sol-ring") is True
    assert scry.is_marketplace("https://scry.cl/card/sol-ring") is True
    assert scry.is_marketplace("https://catlotus.cl/carta/123") is False
    assert scry.is_marketplace("https://www.paytowin.cl/products/x") is False
    assert scry.is_marketplace("https://gameofmagicsingles.cl/products/y") is False
    # No debe confundirse con un dominio que apenas contenga la cadena
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
    # La comparacion es normalizada: mayusculas y puntuacion no deben importar
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
    assert len(offers) == 1, "las variantes sin stock se descartan"
    assert offers[0].price_clp == 3500
    assert offers[0].language == "English"
    assert "variant=1" in offers[0].url


# --------------------------------------------------------------- los puertos
def test_every_source_meets_its_port():
    """Un adaptador que deja de cumplir su puerto rompe aca, no en produccion."""
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
    """El nucleo depende de la forma. Si esto falla, un vendor se filtro."""
    root = Path(__file__).resolve().parent.parent
    revisados = list((root / "muchi" / "mtg").glob("*.py")) + [root / "app.py"]
    # Si el paquete se mueve otra vez, este test pasaria mirando cero archivos.
    assert len(revisados) > 10, f"la ruta del paquete quedo mal: {revisados}"

    offenders = []
    for py in revisados:
        if py.name == "cast.py":
            continue
        for line in py.read_text(encoding="utf-8").splitlines():
            if line.startswith(("from .sources", "from muchi.mtg.sources")):
                offenders.append(f"{py.name}: {line}")
    assert offenders == [], offenders


# -------------------------------------------------------------- las ofertas
class _FakeSource:
    def __init__(self, name, offers=(), explodes=False):
        self.name = name
        self._offers = list(offers)
        self._explodes = explodes

    def find_offers(self, card_name):
        if self._explodes:
            raise RuntimeError("la tienda se cayo")
        return list(self._offers)


class _FakePrimary(_FakeSource):
    def identify_card(self, name):
        return "id-1", list(self._offers)

    def suggest_names(self, text):
        return []

    def refresh_offers(self, card_id):
        return iter(())


def test_find_offers_sorts_and_dedupes():
    """Si la fuente principal ya cubre una tienda, la secundaria no la repite."""
    primary = _FakePrimary("scry", [_offer("CatLotus", "Sol Ring", 5000)])
    local = _FakeSource("indice local", [
        _offer("CatLotus", "Sol Ring", 4000),   # ya cubierta: se descarta
        _offer("PDA Chile", "Sol Ring", 3000),  # nueva: entra
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


# ------------------------------------------------------------------ el mazo
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


# --------------------------------------------------------------- las tiendas
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
    """Lo que ningun nombre de carta puede ser, se reporta en vez de colarse."""
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
    """El filtro no puede comerse cartas que existen de verdad."""
    odd_names = [
        'Kongming, "Sleeping Dragon"',   # comillas dobles
        "+2 Mace",                        # empieza con signo
        "Sword of Dungeons & Dragons",   # ampersand
        "Jotun Grunt",
        "Mr. Orfeo, the Boulder",        # punto
        "Ach! Hans, Run!",               # exclamaciones
        "Borrowing 100,000 Arrows",      # digitos y comas
        "Yawgmoth's Will",               # apostrofo
        "Lim-Dul the Necromancer",       # guion
        "Question Elemental?",           # interrogacion
    ]
    orders, ignored = decklist.parse_decklist("\n".join(odd_names))
    assert ignored == [], ignored
    assert [p.name for p in orders] == odd_names


def test_decklist_accents_and_ligatures():
    orders, ignored = decklist.parse_decklist("1 Jotun Grunt\n1 Aether Vial\n1 Seance")
    assert ignored == []
    assert len(orders) == 3


def test_decklist_strips_exporter_noise():
    """Moxfield, Arena, Archidekt y deckstats cuelgan cosas al final."""
    orders, ignored = decklist.parse_decklist(
        "1 Sol Ring (LTC) 344 *F*\n"       # Moxfield con foil
        "1x Arcane Signet (c21) 263\n"     # Archidekt
        "1 Command Tower #!Commander\n"    # deckstats
        "1 Cultivate [Ramp]\n"             # categoria de Archidekt
        "4 Forest (UNF) 235"                # Arena, tierra basica
    )
    assert ignored == [], ignored
    assert {p.name: p.quantity for p in orders} == {
        "Sol Ring": 1, "Arcane Signet": 1, "Command Tower": 1,
        "Cultivate": 1, "Forest": 4,
    }


def test_decklist_cleans_clipboard_noise():
    """Comillas curvas, espacio duro, doble espacio y vinetas de markdown."""
    orders, ignored = decklist.parse_decklist(
        "1 Yawgmoth\u2019s Will\n"        # apostrofo curvo
        "2 Sol\u00a0Ring\n"               # espacio duro
        "3 Lightning  Bolt\n"             # doble espacio
        "- 4 Counterspell\n"              # vineta de lista
        "\u2022 1 Brainstorm"
    )
    assert ignored == [], ignored
    assert {p.name: p.quantity for p in orders} == {
        "Yawgmoth's Will": 1, "Sol Ring": 2, "Lightning Bolt": 3,
        "Counterspell": 4, "Brainstorm": 1,
    }


def test_decklist_sums_by_flattened_name():
    """Pegar la misma carta con otra puntuacion no debe duplicar la linea."""
    orders, _ = decklist.parse_decklist(
        "2 Yawgmoth's Will\n1 Yawgmoths Will\n1 YAWGMOTH'S WILL"
    )
    assert len(orders) == 1, [p.name for p in orders]
    assert orders[0].quantity == 4
    assert orders[0].name == "Yawgmoth's Will", "se conserva la primera forma"


def test_decklist_headers_are_not_cards():
    orders, ignored = decklist.parse_decklist(
        "Deck\n4 Lightning Bolt\nCreatures (12)\nSideboard:\n"
        "Rampa:\n2 Sol Ring\nCommander\n1 Atraxa"
    )
    assert ignored == [], ignored
    assert {p.name for p in orders} == {"Lightning Bolt", "Sol Ring", "Atraxa"}


def test_decklist_basic_lands_are_cards():
    """'Land' es encabezado; 'Forest' es una carta. No confundirlos."""
    orders, _ = decklist.parse_decklist(
        "Lands\n10 Forest\n5 Island\n1 Swamp\n1 Mountain\n1 Plains"
    )
    assert {p.name for p in orders} == {
        "Forest", "Island", "Swamp", "Mountain", "Plains"}


def test_decklist_keeps_arrival_order():
    orders, _ = decklist.parse_decklist("1 Zur\n1 Alesha\n1 Muldrotha")
    assert [p.name for p in orders] == ["Zur", "Alesha", "Muldrotha"]


# ------------------------------------------------------------------ rutas
def test_root_coincide_con_la_raiz_real():
    """muchi.paths cuenta niveles; este test los cuenta aparte y compara.

    Si los dos usaran el mismo helper, un error en la cuenta seria invisible.
    """
    from muchi import paths

    root = Path(__file__).resolve().parent.parent
    assert paths.ROOT == root, f"{paths.ROOT} != {root}"
    assert paths.ASSETS == root / "assets"
    assert paths.DATA == root / "data"


def test_las_rutas_del_paquete_caen_dentro_del_repo():
    """Una ruta que sube un nivel de mas no explota: apunta a otro lado."""
    from muchi import paths
    from muchi.mtg import db, mascot
    from muchi.mtg.sources import moxfield, store_api

    for label, path in [("db.PATH", db.PATH), ("mascot.SPRITE", mascot.SPRITE),
                        ("store_api.CONFIG", store_api.CONFIG),
                        ("moxfield.CONFIG", moxfield.CONFIG)]:
        assert paths.ROOT in path.parents, f"{label} se salio del repo: {path}"

    assert mascot.SPRITE.exists(), "el sprite deberia estar donde apunta"
    assert moxfield.CONFIG.exists(), "la config de inventarios deberia estar"


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
            print(f"  FALLA {name}: {e}")
    print("\nTodo verde" if not failures else f"\n{failures} pruebas fallaron")
    sys.exit(1 if failures else 0)


