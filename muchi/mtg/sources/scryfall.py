"""Cliente de Scryfall: el catalogo global de Magic, no los precios chilenos.

Cumple CatalogoCartas. Ojo con el nombre, que se parece peligrosamente al del
vecino: **scry.cl** (sources/scry.py) es el agregador chileno que sabe cuanto
vale una carta aca; **Scryfall** sabe que cartas existen y que dice cada una.
La app las usa juntas --- se elige una carta con este y se cotiza con aquel ---
pero son necesidades distintas.

Este es el unico archivo del repo que escribe sintaxis de Scryfall. El nucleo
manda PedidoCartas, que son sustantivos del dominio, y aca se vuelven `o:`,
`otag:`, `id<=`. Si manana el catalogo fuera otro, se cambia una linea de
cast.py y este archivo.

Cortesia: Scryfall pide 50-100 ms entre requests (maximo ~10/s), bastante mas
suelto que scry.cl. Por eso cast.py le arma su propia sesion en vez de reusar
la de 1.5s, que haria sentir roto al buscador sin ninguna necesidad.
"""
from __future__ import annotations

from dataclasses import dataclass

import requests

from ..http import PoliteSession
from ..ports import Card, CardPage, CardRequest, QueryFailed

BASE = "https://api.scryfall.com"
_JSON = {"Accept": "application/json"}

# Scryfall pagina de a 175 y nosotros mostramos una docena: pedir mas es
# gastarle ancho de banda a un servicio gratis.
PAGE_SIZE = 12

# Las etiquetas de Scryfall Tagger, por intencion del nucleo. Las mantiene la
# comunidad y pueden renombrarse, asi que viven aca y no en el dominio: si una
# muere, se corrige esta linea y el nucleo ni se entera --- ya trae su propio
# respaldo en texto de reglas.
TAGS = {
    "removal": "otag:removal",
    "board-wipe": "otag:board-wipe",
    "card-draw": "otag:card-advantage",
    "ramp": "otag:ramp",
    "tutor": "otag:tutor",
    "counterspell": "otag:counterspell",
    "lifegain": "otag:lifegain",
    "token-generation": "otag:repeatable-token-generator",
    "recursion": "otag:recursion",
}


def read_field(d: dict, field: str) -> str:
    """El campo de la carta, uniendo las caras si es de doble cara."""
    if d.get(field):
        return str(d[field])
    faces = [f[field] for f in d.get("card_faces") or [] if f.get(field)]
    return "\n//\n".join(faces)


def read_image(d: dict, size: str = "normal") -> str:
    uris = d.get("image_uris") or {}
    if not uris:
        for face in d.get("card_faces") or []:
            if face.get("image_uris"):
                uris = face["image_uris"]
                break
    return uris.get(size) or uris.get("large") or uris.get("small") or ""


def parse_card(d: dict) -> Card:
    """Arma una Carta desde el objeto card de Scryfall.

    Si el objeto ES una impresion traducida, el nombre canonico igual sale de
    `name`: Scryfall guarda ahi el ingles y deja el traducido en `printed_name`.
    Eso es justo lo que necesitamos --- al carrito tiene que viajar el ingles.
    """
    translated = (d.get("lang") or "en") != "en"
    return Card(
        name=read_field(d, "name"),
        oracle_id=d.get("oracle_id") or "",
        text=read_field(d, "oracle_text"),
        type_line=read_field(d, "type_line"),
        mana_cost=read_field(d, "mana_cost"),
        image=read_image(d),
        url=d.get("scryfall_uri") or "",
        rarity=d.get("rarity") or "",
        local_name=read_field(d, "printed_name") if translated else "",
        local_text=read_field(d, "printed_text") if translated else "",
        local_type=read_field(d, "printed_type_line") if translated else "",
        local_image=read_image(d) if translated else "",
    )


def quote_term(phrase: str) -> str:
    """Una frase va entre comillas; una palabra suelta, pelada."""
    return f'o:"{phrase}"' if " " in phrase else f"o:{phrase}"


def render_query(request: CardRequest) -> str:
    """Traduce un pedido del nucleo a la sintaxis de Scryfall.

    Los terminos se combinan con Y, que es lo que hace util a la escalera de
    oracle.build_requests(): cada termino de mas achica el resultado, y por eso
    hay que tener a mano una version mas suelta del mismo pedido.
    """
    parts: list[str] = []

    if request.literal_text:
        parts.append(f'o:"{request.literal_text}"')
    else:
        parts += [quote_term(p) for p in request.phrases]
        parts += [quote_term(w) for w in request.words]
        parts += [TAGS[k] for k in request.intents if k in TAGS]

    if request.colors:
        # id<= es "cabe en un mazo de esta identidad", que es lo que quiere
        # alguien armando un Commander. c: seria el color de la carta: mas
        # literal, y deja fuera justo lo que se estaba buscando.
        parts.append("id<=" + "".join(request.colors))
    if request.card_type:
        parts.append(f"t:{request.card_type}")
    if request.format_name:
        parts.append(f"f:{request.format_name}")
    if request.max_mana is not None:
        parts.append(f"mv<={int(request.max_mana)}")

    return " ".join(parts)


def ask_scryfall(sess: PoliteSession, path: str, params: dict) -> dict | None:
    """GET a Scryfall. None cuando no hay resultados.

    Scryfall responde **404 cuando una busqueda no encuentra nada**, no una
    lista vacia, y 400 cuando no entiende la query. Las dos son respuestas
    normales para nosotros: la escalera sigue con el escalon siguiente.

    Lo que si es un error es que la red se caiga, y ahi sale QueryFailed: un
    requests.HTTPError cruzando el puerto seria una fuga del vendor.
    """
    try:
        r = sess.get(f"{BASE}{path}", params=params, headers=_JSON)
    except requests.HTTPError as e:
        if e.response is not None and e.response.status_code in (400, 404):
            return None
        raise QueryFailed(f"Scryfall respondio {e}") from e
    except requests.RequestException as e:
        raise QueryFailed(f"No pude hablar con el catalogo: {e}") from e
    return r.json()


def build_name_query(names: list[str]) -> str:
    """(!"A" or !"B" ...) --- el `!` de Scryfall es match exacto de nombre."""
    parts = [f'!"{n}"' for n in names if n and '"' not in n]
    return "(" + " or ".join(parts) + ")" if parts else ""


def merge_translation(card: Card, other: Card) -> Card:
    """La carta canonica con la impresion traducida pegada al lado."""
    return Card(
        name=card.name, oracle_id=card.oracle_id, text=card.text,
        type_line=card.type_line, mana_cost=card.mana_cost, image=card.image,
        url=card.url, rarity=card.rarity,
        local_name=other.local_name or other.name,
        local_text=other.local_text,
        local_type=other.local_type,
        local_image=other.local_image,
    )


@dataclass
class ScryfallCatalog:
    """Cumple CatalogoCartas. Unico lugar que sabe como habla Scryfall.

    Contrato: https://scryfall.com/docs/api
    """
    sess: PoliteSession
    name: str = "scryfall"

    def search_cards(self, request: CardRequest) -> CardPage:
        """Un pedido, una pagina de cartas.

        El orden es por que tan jugada es la carta. Para alguien que no sabe
        que busca eso se lee como relevancia; ordenar por nombre le pondria
        arriba la carta mas oscura del catalogo.
        """
        query = render_query(request)
        if not query:
            return CardPage()

        params = {"q": query, "unique": "cards", "order": "edhrec", "dir": "asc"}
        if request.literal_text:
            # El texto crudo puede venir en espanol: sin esto no habria contra
            # que matchear. Es el unico escalon que lo necesita, y por eso el
            # unico que carga con el ruido de traer cada carta N veces.
            params["include_multilingual"] = "true"

        found = ask_scryfall(self.sess, "/cards/search", params)
        if not found:
            return CardPage(explain=query)

        cards, seen = [], set()
        for item in found.get("data", []):
            card = parse_card(item)
            key = card.oracle_id or card.name
            if key in seen:
                continue
            seen.add(key)
            cards.append(card)
            if len(cards) >= PAGE_SIZE:
                break

        return CardPage(cards=tuple(cards),
                        total=int(found.get("total_cards") or 0),
                        explain=query)

    def suggest_names(self, text: str) -> list[str]:
        """Nombres que empiezan como lo escrito. Para cuando si sabe el nombre."""
        if len(text.strip()) < 2:
            return []
        try:
            found = ask_scryfall(self.sess, "/cards/autocomplete", {"q": text})
        except QueryFailed:
            return []
        return list((found or {}).get("data", []))[:8]

    def resolve_name(self, text: str) -> Card | None:
        """Una carta a partir de un nombre mal escrito ('lightnig bot')."""
        if len(text.strip()) < 3:
            return None
        found = ask_scryfall(self.sess, "/cards/named", {"fuzzy": text})
        return parse_card(found) if found else None

    def translate_cards(self, cards: list[Card], language: str) -> list[Card]:
        """Le pega a cada carta su impresion traducida, en UN solo request.

        Buscar en ingles y despues traer las traducciones es mejor que buscar
        derecho en el idioma: media biblioteca no tiene impresion traducida
        --- reprints, precons, sets viejos --- asi que hacerlo al reves achicaria
        el resultado por el idioma de la vista, que no tiene nada que ver.

        Si el request falla, devuelve las cartas tal cual: la traduccion es un
        lujo y no puede tumbar la busqueda.
        """
        if not cards or language == "en":
            return cards

        query = build_name_query([c.name for c in cards])
        if not query:
            return cards

        try:
            found = ask_scryfall(self.sess, "/cards/search", {
                "q": f"{query} lang:{language}", "unique": "prints",
                "include_multilingual": "true",
                "order": "released", "dir": "desc",
            })
        except QueryFailed:
            return cards
        if not found:
            return cards

        # unique=prints trae todas las reimpresiones; nos quedamos con la
        # primera de cada nombre, que por el orden es la mas reciente.
        by_name: dict[str, Card] = {}
        for item in found.get("data", []):
            card = parse_card(item)
            by_name.setdefault(card.name, card)

        return [merge_translation(c, by_name[c.name]) if c.name in by_name else c
                for c in cards]
