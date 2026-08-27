"""Del "quiero algo que mate una criatura" a un pedido que el catalogo entienda.

Este modulo es nucleo: arma CardRequest, que son puros sustantivos del dominio.
No sabe como se escribe una busqueda en Scryfall ni en ningun otro lado --- eso
Vive detras del puerto, en sources/. Aca se resuelven dos problemas que son
nuestros y no del proveedor:

**El idioma.** El texto de reglas canonico de Magic es en ingles. El jugador
chileno escribe en espanol, en spanglish, o mal. PHRASES es una capa de
normalizacion, no un traductor: lo que matchea se reemplaza, lo que no, pasa
igual. Por eso "destruye target creature" funciona igual de bien que cualquiera
de los dos idiomas puros, que es como se escribe de verdad.

**El todo o nada.** Un solo pedido estricto es una apuesta: si el usuario metio
una palabra rara, cero resultados y chao. build_requests() devuelve varios, del
mas estricto al mas suelto, cada uno contando en su `note` que tuvo que soltar.
La app prueba en orden y se queda con el primero que trae algo.
"""
from __future__ import annotations

import re
import unicodedata
from dataclasses import dataclass

from .ports import CardRequest

# --------------------------------------------------------------- diccionario
# Claves sin tilde y en minuscula: la entrada se aplana antes de buscar. Se
# matchean de la frase mas larga a la mas corta, si no "roba una carta" se
# partiria en "roba" y perderiamos la frase entera.
PHRASES: dict[str, str] = {
    # --- lo que hace la carta
    "destruye la criatura objetivo": "destroy target creature",
    "destruir la criatura objetivo": "destroy target creature",
    "destruye todas las criaturas": "destroy all creatures",
    "destruye todo": "destroy all",
    "destruir todo": "destroy all",
    "destruye": "destroy",
    "destruir": "destroy",
    "exiliar": "exile",
    "exilia": "exile",
    "roba una carta": "draw a card",
    "robar una carta": "draw a card",
    "roba dos cartas": "draw two cards",
    "robar cartas": "draw a card",
    "roba cartas": "draw a card",
    "robar": "draw",
    "roba": "draw",
    "descartar": "discard",
    "descarta": "discard",
    "contrarresta el hechizo objetivo": "counter target spell",
    "contrarrestar hechizo": "counter target spell",
    "contrarrestar": "counter target spell",
    "contrarresta": "counter target spell",
    "sacrificar": "sacrifice",
    "sacrifica": "sacrifice",
    "ganar vida": "gain life",
    "gana vida": "gain life",
    "pierde vida": "loses life",
    "perder vida": "loses life",
    "inflige dano": "deals damage",
    "hace dano": "deals damage",
    "dano": "damage",
    "crea una ficha": "create a token",
    "crear fichas": "create a token",
    "crea fichas": "create a token",
    "ficha": "token",
    "fichas": "token",
    "busca en tu biblioteca": "search your library",
    "buscar en tu biblioteca": "search your library",
    "tutor": "search your library",
    "devuelve a la mano": "return to its owner's hand",
    "vuelve a la mano": "return to its owner's hand",
    "devuelve": "return",
    "gana el control": "gain control",
    "roba el control": "gain control",
    "copia": "copy",
    "turno adicional": "take an extra turn",
    "turno extra": "take an extra turn",
    "agrega mana": "add",
    "anade mana": "add",
    "no puede ser bloqueada": "can't be blocked",
    "no puede ser bloqueado": "can't be blocked",
    "no puede ser contrarrestado": "can't be countered",
    "sin pagar su costo": "without paying its mana cost",
    "sin costo de mana": "without paying its mana cost",
    "cuesta menos": "costs less to cast",
    "contador +1/+1": "+1/+1 counter",
    "contadores +1/+1": "+1/+1 counter",
    # --- disparos
    "cada vez que": "whenever",
    "siempre que": "whenever",
    "cuando entra al campo de batalla": "enters the battlefield",
    "entra al campo de batalla": "enters the battlefield",
    "al comienzo de tu mantenimiento": "at the beginning of your upkeep",
    "mantenimiento": "upkeep",
    "final del turno": "end step",
    "paso final": "end step",
    "cuando muere": "dies",
    "muere": "dies",
    "cuando ataca": "attacks",
    "ataca": "attacks",
    "bloquea": "blocks",
    # --- zonas y objetos
    "campo de batalla": "battlefield",
    "cementerio": "graveyard",
    "biblioteca": "library",
    "mazo": "library",
    "mano": "hand",
    "cada oponente": "each opponent",
    "oponentes": "opponents",
    "oponente": "opponent",
    "todos los jugadores": "each player",
    "jugador": "player",
    "hechizos": "spells",
    "hechizo": "spell",
    "criaturas": "creatures",
    "criatura": "creature",
    "artefacto": "artifact",
    "encantamiento": "enchantment",
    "instantaneo": "instant",
    "conjuro": "sorcery",
    "tierras": "lands",
    "tierra": "land",
    "vidas": "life",
    "vida": "life",
    # --- habilidades con nombre
    "toque mortal": "deathtouch",
    "toque letal": "deathtouch",
    "vinculo vital": "lifelink",
    "volar": "flying",
    "vuela": "flying",
    "vigilancia": "vigilance",
    "arrollar": "trample",
    "arrolla": "trample",
    "prisa": "haste",
    "amenaza": "menace",
    "alcance": "reach",
    "defensor": "defender",
    "indestructible": "indestructible",
    "proteccion": "protection",
    "equipar": "equip",
    "equipo": "equipment",
}

# Lo que sobra despues de traducir. Si se colara como termino de busqueda --- que
# se combinan con Y --- el pedido no devolveria nada nunca. Hay chilenismos
# adentro a proposito: aparecen seguido y no dicen nada de la carta.
FILLER = {
    "que", "de", "del", "la", "el", "los", "las", "un", "una", "unos", "unas",
    "y", "o", "a", "al", "en", "con", "por", "para", "su", "sus", "mi", "mis",
    "tu", "tus", "me", "te", "se", "lo", "algo", "carta", "cartas", "quiero",
    "busco", "necesito", "sea", "haga", "hacer", "poder", "pueda", "cualquier",
    "todos", "todas", "todo", "toda", "cuando", "si", "no", "mas", "menos",
    "the", "an", "of", "to", "i", "want", "some", "card", "cards", "that",
    "pa", "pal", "po", "cachai", "wea", "weas", "onda", "media", "bkn", "pulenta",
}

# Al partir una frase en palabras, estas solo estrechan sin aportar.
FILLER_EN = {"a", "an", "the", "of", "to", "its", "your", "and", "or", "for"}


@dataclass(frozen=True)
class Intent:
    """Un chip de "para que la quiero".

    `fallback` son las frases que describen lo mismo en texto de reglas. Existe
    porque una fuente puede tener etiquetas semanticas mucho mejores que buscar
    texto --- y puede tambien renombrarlas o no tenerlas. La fuente decide con
    que responde; el nucleo se asegura de que siempre haya un plan B.
    """
    key: str
    label: str
    fallback: tuple[str, ...]


INTENTS: list[Intent] = [
    Intent("removal", "Sacar algo del campo", ("destroy target",)),
    Intent("board-wipe", "Barrer el campo entero", ("destroy all",)),
    Intent("card-draw", "Robar cartas", ("draw a card",)),
    Intent("ramp", "Hacer mas mana", ("search your library for a", "land")),
    Intent("tutor", "Buscar una carta en el mazo", ("search your library",)),
    Intent("counterspell", "Contrarrestar hechizos", ("counter target",)),
    Intent("lifegain", "Ganar vida", ("gain", "life")),
    Intent("token-generation", "Hacer fichas", ("create", "token")),
    Intent("recursion", "Sacar cosas del cementerio", ("from your graveyard",)),
]

BY_KEY = {i.key: i for i in INTENTS}

CARD_TYPES = {
    "Cualquiera": "",
    "Criatura": "creature",
    "Instantaneo": "instant",
    "Conjuro": "sorcery",
    "Artefacto": "artifact",
    "Encantamiento": "enchantment",
    "Planeswalker": "planeswalker",
    "Tierra": "land",
}

FORMATS = {
    "Cualquiera": "",
    "Commander": "commander",
    "Modern": "modern",
    "Pioneer": "pioneer",
    "Standard": "standard",
    "Legacy": "legacy",
    "Pauper": "pauper",
}

COLORS = {"Blanco": "w", "Azul": "u", "Negro": "b", "Rojo": "r", "Verde": "g"}


def flatten_text(text: str) -> str:
    """Minusculas y sin tildes, conservando los espacios."""
    s = unicodedata.normalize("NFD", text or "")
    s = "".join(c for c in s if unicodedata.category(c) != "Mn")
    return re.sub(r"\s+", " ", s.lower()).strip()


def parse_request(text: str) -> tuple[list[str], list[str]]:
    """Parte el texto libre en (frases traducidas, palabras que no reconoci).

    Las frases se van tachando del texto a medida que matchean, de la mas larga
    a la mas corta, para que una no se cuente dos veces por una sub-frase suya.
    """
    remaining = flatten_text(text).replace('"', " ")
    if not remaining:
        return [], []

    found: list[str] = []
    for spanish in sorted(PHRASES, key=len, reverse=True):
        if spanish in remaining:
            english = PHRASES[spanish]
            if english not in found:
                found.append(english)
            remaining = remaining.replace(spanish, " ")

    loose = [w for w in re.findall(r"[a-z0-9'+/-]{2,}", remaining)
             if w not in FILLER and not w.isdigit()]
    return found, loose


def split_phrases(phrases: list[str]) -> list[str]:
    """Las frases hechas palabras sueltas, sin el relleno del ingles.

    Sirve porque una frase exacta es fragil: "destroy target creature" no
    aparece en "destroy target *attacking* creature", pero las tres palabras
    por separado si.
    """
    words: list[str] = []
    for phrase in phrases:
        for word in phrase.split():
            word = word.strip(".,")
            if word and word not in FILLER_EN and word not in words:
                words.append(word)
    return words


def list_fallbacks(keys: tuple[str, ...]) -> tuple[str, ...]:
    """Las frases con que se describe una intencion sin usar su etiqueta.

    Es el plan B de los chips: una fuente puede tener etiquetas semanticas
    mejores que buscar texto, pero tambien puede renombrarlas o no tenerlas, y
    un chip muerto no falla ruidoso --- devuelve cero, que se lee igual que
    "no existen cartas asi".
    """
    out: list[str] = []
    for key in keys:
        intent = BY_KEY.get(key)
        if not intent:
            continue
        out += [p for p in intent.fallback if p not in out]
    return tuple(out)


def build_requests(text: str, *, intents: tuple[str, ...] = (),
                   colors: tuple[str, ...] = (), card_type: str = "",
                   format_name: str = "", max_mana: int | None = None,
                   ) -> list[CardRequest]:
    """La escalera de pedidos, del mas estricto al mas suelto.

    La app los prueba en orden y se queda con el primero que devuelva cartas.
    """
    phrases, loose = parse_request(text)
    intents = tuple(k for k in intents if k in BY_KEY)
    literal = flatten_text(text).replace('"', " ").strip()

    filters = {"colors": colors, "card_type": card_type,
               "format_name": format_name, "max_mana": max_mana}
    has_filters = bool(colors or card_type or format_name or max_mana is not None)

    steps: list[CardRequest] = []

    def add(request: CardRequest) -> None:
        if request.is_empty:
            return
        same = (request.phrases, request.words, request.intents,
                request.literal_text)
        if all(same != (s.phrases, s.words, s.intents, s.literal_text)
               for s in steps):
            steps.append(request)

    # 1. todo junto
    add(CardRequest(phrases=tuple(phrases), words=tuple(loose),
                    intents=intents, **filters))

    # 2. sin las palabras que no supimos traducir. Solo si queda algo que
    #    buscar: soltarlas todas no es aflojar, es devolver el catalogo entero
    #    filtrado por tipo.
    if loose and (phrases or intents):
        add(CardRequest(phrases=tuple(phrases), intents=intents, **filters,
                        note="dejando de lado las palabras que no reconoci"))

    # 3. las frases partidas en palabras
    if any(" " in p for p in phrases):
        add(CardRequest(words=tuple(split_phrases(phrases)), intents=intents,
                        **filters, note="buscando las palabras por separado"))

    # 4. el chip por su texto en vez de por su etiqueta
    if intents:
        add(CardRequest(phrases=tuple(phrases) + list_fallbacks(intents),
                        **filters,
                        note="buscando el texto en vez de la etiqueta"))

    # 5. el texto crudo, tal cual lo escribio. Ultimo recurso a proposito: que
    #    una fuente sepa buscar sobre el texto traducido no esta garantizado,
    #    asi que nada de lo de arriba depende de esto.
    if literal and " " in literal:
        add(CardRequest(literal_text=literal, **filters,
                        note="probando tu texto tal cual"))

    # 6. solo los filtros y la intencion
    if has_filters or intents:
        add(CardRequest(intents=intents, **filters, note="solo con los filtros"))
        add(CardRequest(phrases=list_fallbacks(intents), **filters,
                        note="solo con los filtros"))

    return steps
