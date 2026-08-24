"""Pruebas de la logica que no depende de la red.

Corre con pytest, o directo:  python tests/test_muchi.py
"""
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from mtgcl import decklist, optimizer  # noqa: E402
from mtgcl.models import Offer, Pedido  # noqa: E402
from mtgcl.sources import edhrec, scry, shopify  # noqa: E402


def test_decklist_formatos():
    pedidos, ignoradas = decklist.parse(
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
    encontrado = {p.nombre: p.cantidad for p in pedidos}
    # El bug clasico: sin el lookahead (?=\d), "Sol Ring" se parseaba "Sol"
    # y "Lightning Bolt" quedaba en "Lightning".
    assert encontrado == {
        "Lightning Bolt": 4,
        "Sol Ring": 2,
        "Ragavan, Nimble Pilferer": 1,
        "Counterspell": 1,
        "Fire": 3,
    }, encontrado
    assert not ignoradas, ignoradas


def test_decklist_suma_repetidos():
    pedidos, _ = decklist.parse("2 Sol Ring\n1 Sol Ring")
    assert len(pedidos) == 1 and pedidos[0].cantidad == 3


def _oferta(tienda, carta, precio):
    return Offer(store=tienda, card_name=carta, title=carta, price_clp=precio,
                 url="http://x", key=f"{tienda}-{carta}")


def test_optimizador_prefiere_concentrar():
    """Con envio caro, conviene una sola tienda aunque las cartas salgan mas."""
    pedidos = [Pedido(1, "A"), Pedido(1, "B")]
    ofertas = {
        "a": [_oferta("T1", "A", 1000), _oferta("T2", "A", 900)],
        "b": [_oferta("T1", "B", 1000), _oferta("T3", "B", 900)],
    }

    barato = optimizer.plan_mas_barato(pedidos, ofertas, envio_por_tienda=5000)
    # Ingenuo: 900 + 900 + 2 envios = 11800, en dos tiendas distintas
    assert len(barato.tiendas) == 2 and barato.total == 11800, barato.total

    optimo = optimizer.plan_optimo(pedidos, ofertas, envio_por_tienda=5000)
    # Optimo: todo en T1 = 2000 + 1 envio = 7000
    assert optimo.tiendas == ["T1"], optimo.tiendas
    assert optimo.total == 7000, optimo.total
    assert optimo.total < barato.total


def test_optimizador_separa_si_el_envio_es_barato():
    pedidos = [Pedido(1, "A"), Pedido(1, "B")]
    ofertas = {
        "a": [_oferta("T1", "A", 5000), _oferta("T2", "A", 100)],
        "b": [_oferta("T1", "B", 5000), _oferta("T3", "B", 100)],
    }
    plan = optimizer.plan_optimo(pedidos, ofertas, envio_por_tienda=200)
    assert sorted(plan.tiendas) == ["T2", "T3"], plan.tiendas
    assert plan.total == 600, plan.total


def test_optimizador_reporta_faltantes():
    pedidos = [Pedido(1, "A"), Pedido(2, "Inexistente")]
    ofertas = {"a": [_oferta("T1", "A", 1000)]}
    plan = optimizer.plan_optimo(pedidos, ofertas, envio_por_tienda=0)
    assert plan.faltantes == ["Inexistente"], plan.faltantes
    assert plan.total == 1000


def test_optimizador_respeta_cantidades():
    pedidos = [Pedido(4, "A")]
    ofertas = {"a": [_oferta("T1", "A", 250)]}
    plan = optimizer.plan_optimo(pedidos, ofertas, envio_por_tienda=0)
    assert plan.costo_cartas == 1000, plan.costo_cartas


def test_slug():
    assert scry.slug("Ragavan, Nimble Pilferer") == "ragavan-nimble-pilferer"
    assert scry.slug("Jotun Grunt") == "jotun-grunt"
    assert scry.slug("Sol Ring") == "sol-ring"


def test_parse_offers_desde_html_real():
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
    ofertas = scry._ofertas_desde_html(html)
    assert len(ofertas) == 2
    assert ofertas[0].store == "CatLotus" and ofertas[0].price_clp == 99000  # ordenado
    assert ofertas[1].es_foil is True
    assert ofertas[0].es_foil is False
    assert ofertas[1].condition == "Near Mint"
    assert ofertas[0].edicion == "MH2"


def test_marketplace_vs_tienda():
    """Los particulares de scry cuelgan de marketplace.scry.cl; las tiendas no."""
    assert scry.es_marketplace("https://marketplace.scry.cl/magic-master/sol-ring") is True
    assert scry.es_marketplace("https://scry.cl/card/sol-ring") is True
    assert scry.es_marketplace("https://catlotus.cl/carta/123") is False
    assert scry.es_marketplace("https://www.paytowin.cl/products/x") is False
    assert scry.es_marketplace("https://gameofmagicsingles.cl/products/y") is False
    # No debe confundirse con un dominio que apenas contenga la cadena
    assert scry.es_marketplace("https://noscry.cl/x") is False


def test_marca_marketplace_al_parsear():
    html = (
        '<a data-track-type="store_offer_click" data-store-name="Magic Master" '
        'data-offer-title="Sol Ring - NM" data-price-clp="1791" data-variant-key="k1" '
        'data-product-url="https://marketplace.scry.cl/magic-master/sol-ring">Ver</a>'
        '<a data-track-type="store_offer_click" data-store-name="CatLotus" '
        'data-offer-title="Sol Ring - NM" data-price-clp="2000" data-variant-key="k2" '
        'data-product-url="https://catlotus.cl/x">Ver</a>'
    )
    ofertas = scry._ofertas_desde_html(html)
    por_tienda = {o.store: o.marketplace for o in ofertas}
    assert por_tienda == {"Magic Master": True, "CatLotus": False}, por_tienda


def _rec(nombre, cat="Top Cards", inc=0.5, sin=0.1):
    return edhrec.Recomendacion(nombre=nombre, categoria=cat, tag=cat.lower(),
                                inclusion=inc, sinergia=sin, num_decks=10)


def test_edhrec_slug_comandantes():
    assert edhrec.slug("Atraxa, Praetors' Voice") == "atraxa-praetors-voice"
    assert edhrec.slug("Kenrith, the Returned King") == "kenrith-the-returned-king"
    assert edhrec.slug("Edgar Markov") == "edgar-markov"


def test_faltantes_descuenta_lo_que_ya_tengo():
    recs = [_rec("Sol Ring"), _rec("Skullclamp"), _rec("Arcane Signet")]
    # La comparacion es normalizada: mayusculas y puntuacion no deben importar
    faltan = edhrec.faltantes(recs, {"sol ring", "ARCANE SIGNET"})
    assert [r.nombre for r in faltan] == ["Skullclamp"]


def test_faltantes_sin_mazo_devuelve_todo():
    recs = [_rec("Sol Ring"), _rec("Skullclamp")]
    assert len(edhrec.faltantes(recs, set())) == 2


def test_categorias_conserva_el_orden():
    recs = [_rec("A", "Top Cards"), _rec("B", "Creatures"),
            _rec("C", "Top Cards"), _rec("D", "Instants")]
    assert edhrec.categorias(recs) == ["Top Cards", "Creatures", "Instants"]


def test_basicas_se_reconocen():
    for n in ("Mountain", "island", "Snow-Covered Forest", "Wastes"):
        assert edhrec.es_basica(n) is True, n
    for n in ("Mountain Valley", "Goblin Warchief", "Islandia"):
        assert edhrec.es_basica(n) is False, n


def test_inclusion_pct():
    assert _rec("X", inc=0.6127).inclusion_pct == 61.3


def test_parse_titulo_shopify():
    d = shopify.parse_titulo("Ragavan, Nimble Pilferer (Borderless) [MH2 - 138]")
    assert d["nombre"] == "Ragavan, Nimble Pilferer"
    assert d["variante"] == "Borderless"
    assert d["set"] == "MH2" and d["cn"] == "138"

    d2 = shopify.parse_titulo("Growth Spiral (7054) [SLD - 7054]")
    assert d2["nombre"] == "Growth Spiral" and d2["set"] == "SLD"


def test_ofertas_de_producto_shopify():
    producto = {
        "title": "Sol Ring [C21 - 263]",
        "handle": "sol-ring-c21",
        "variants": [
            {"id": 1, "title": "Near Mint / English / Normal", "price": "3500", "available": True},
            {"id": 2, "title": "Near Mint / English / Foil", "price": "9000", "available": False},
        ],
    }
    ofertas = shopify.ofertas_de_producto(producto, "PayToWin", "https://www.paytowin.cl")
    assert len(ofertas) == 1, "las variantes sin stock se descartan"
    assert ofertas[0].price_clp == 3500
    assert ofertas[0].language == "English"
    assert "variant=1" in ofertas[0].url


if __name__ == "__main__":
    fallos = 0
    for nombre, fn in sorted(globals().items()):
        if not nombre.startswith("test_") or not callable(fn):
            continue
        try:
            fn()
            print(f"  ok   {nombre}")
        except AssertionError as e:
            fallos += 1
            print(f"  FALLA {nombre}: {e}")
    print("\nTodo verde" if not fallos else f"\n{fallos} pruebas fallaron")
    sys.exit(1 if fallos else 0)
