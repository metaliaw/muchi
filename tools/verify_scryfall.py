"""Chequea contra la API real que todo lo que le pedimos al catalogo existe.

Por que hace falta un script aparte: la sintaxis de Scryfall no es un contrato
estable. Los slugs de `otag:` los mantiene la comunidad y pueden renombrarse, y
un operador mal escrito **no falla ruidosamente** --- devuelve cero resultados,
que desde la app se ve identico a "no existen cartas asi". Los tests no pueden
distinguir esos dos casos porque no tocan la red. Esto si.

Correrlo antes de tocar oracle.py o el diccionario TAGS:

    python tools/verify_scryfall.py

No necesita streamlit: solo requests.
"""
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from muchi.mtg import oracle  # noqa: E402
from muchi.mtg.http import PoliteSession  # noqa: E402
from muchi.mtg.ports import CardRequest, QueryFailed  # noqa: E402
from muchi.mtg.sources import scryfall  # noqa: E402

OK, FAIL, HMM = "  ok  ", " FALLA", " ojo  "


def count_cards(catalog, query: str, multilingual: bool = False) -> int | None:
    """Cuantas cartas devuelve una query cruda. None si no se pudo preguntar."""
    request = CardRequest(literal_text=query) if multilingual else None
    try:
        if request is not None:
            return catalog.search_cards(request).total
        found = scryfall.ask_scryfall(catalog.sess, "/cards/search", {
            "q": query, "unique": "cards", "order": "edhrec", "dir": "asc",
        })
    except QueryFailed as e:
        print(f"        {e}")
        return None
    return int((found or {}).get("total_cards") or 0)


def title(text: str) -> None:
    print(f"\n{text}\n" + "-" * len(text))


def check_operators(catalog) -> int:
    title("Operadores que arma render_query")
    problems = 0
    for label, query in [
        ("id<=", "id<=wu t:creature"),
        ("t:", "t:creature"),
        ("f:", "f:commander t:creature"),
        ("mv<=", "mv<=2 t:instant"),
        ("o: frase", 'o:"destroy target creature"'),
        ("o: palabra", "o:flying"),
        ("! exacto", '!"Sol Ring"'),
        ("or", '(!"Sol Ring" or !"Counterspell")'),
    ]:
        n = count_cards(catalog, query)
        problems += 0 if n else 1
        print(f"[{OK if n else FAIL}] {label:<12} {query:<45} {n}")
    return problems


def check_tags(catalog) -> int:
    title("Slugs de Scryfall Tagger (los chips de intencion)")
    print("Si alguno sale en cero, corregi el slug en scryfall.TAGS o borralo:")
    print("el respaldo del nucleo lo cubre igual, pero da peores resultados.\n")

    problems = 0
    for intent in oracle.INTENTS:
        tag = scryfall.TAGS.get(intent.key)
        if not tag:
            print(f"[{HMM}] {intent.key:<20} sin etiqueta, va por texto siempre")
            continue
        n = count_cards(catalog, tag)
        fallback = count_cards(
            catalog, " ".join(scryfall.quote_term(p) for p in intent.fallback))
        problems += 0 if n else 1
        print(f"[{OK if n else FAIL}] {tag:<26} {str(n):>7} cartas   "
              f"(respaldo: {fallback} cartas)")
    return problems


def check_spanish(catalog) -> None:
    title("La pregunta abierta: busca `o:` sobre el texto impreso en espanol?")
    print("Scryfall documenta `o:` sobre el Oracle en INGLES. Si esto devuelve")
    print("cartas, el diccionario de oracle.py pasa a ser un refuerzo y no la")
    print("unica via.\n")
    for text in ("destruye la criatura objetivo", "roba una carta"):
        n = count_cards(catalog, text, multilingual=True)
        print(f"[{OK if n else HMM}] {text:<34} {n} cartas (multilingue)")
    print("\n  ok    -> se puede buscar en espanol directo. Anotalo en el README.")
    print("  ojo   -> es lo esperado: el diccionario es obligatorio.")


def check_ladder(catalog) -> int:
    title("La escalera completa, de punta a punta")
    cases = [
        ("destruye la criatura objetivo", {}),
        ("roba una carta cuando muere una criatura", {"format_name": "commander"}),
        ("gana vida cada vez que algo muere", {"colors": ("w", "b"), "max_mana": 4}),
        ("counter target spell", {"card_type": "instant"}),
        ("", {"intents": ("removal",), "colors": ("w",), "max_mana": 2}),
    ]
    problems = 0
    for text, filters in cases:
        print(f"\n  > {text or '(solo chips y filtros)'}")
        for step, request in enumerate(oracle.build_requests(text, **filters), 1):
            page = catalog.search_cards(request)
            note = f"  [{request.note}]" if request.note else ""
            mark = "<-- gana" if page.cards else ""
            print(f"     {step}. {page.total:>7} cartas  {page.explain}{note} {mark}")
            if page.cards:
                break
        else:
            problems += 1
            print("     ningun escalon devolvio nada")
    return problems


def check_translations(catalog) -> None:
    title("Traducciones al espanol (translate_cards)")
    page = catalog.search_cards(CardRequest(words=("flying",),
                                            card_type="creature"))
    cards = catalog.translate_cards(list(page.cards)[:5], "es")
    for card in cards:
        mark = f"-> {card.local_name}" if card.is_translated else "-> (sin impresion ES)"
        print(f"  {card.name:<34} {mark}")
    done = sum(1 for c in cards if c.is_translated)
    print(f"\n  {done} de {len(cards)} tienen impresion en espanol. Que no sea "
          "todas es normal: por eso siempre cae al ingles.")


def main() -> int:
    catalog = scryfall.ScryfallCatalog(
        PoliteSession(min_interval=0.12, timeout=10.0, max_retries=1))

    problems = check_operators(catalog)
    problems += check_tags(catalog)
    check_spanish(catalog)
    problems += check_ladder(catalog)
    check_translations(catalog)

    print("\n" + "=" * 62)
    print(f"{problems} cosas que revisar (mira las lineas con FALLA)." if problems
          else "Todo lo que la app le pregunta al catalogo funciona.")
    return 1 if problems else 0


if __name__ == "__main__":
    raise SystemExit(main())
