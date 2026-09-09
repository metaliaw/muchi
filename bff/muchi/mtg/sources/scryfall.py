"""Scryfall: el Invitado que sabe Nombres en Todos los Idiomas.

Dos Puertas. /cards/named con fuzzy adivina el Idioma solo y alcanza casi
siempre. /cards/search con lang: acota a un Idioma cuando el Nombre existe en
varios y el fuzzy elige mal; named ignora lang en silencio, por eso no sirve.
"""
from __future__ import annotations

import requests

from muchi.mtg.ports import CardNotFound, TranslationFailed

NAMED_URL = "https://api.scryfall.com/cards/named"
SEARCH_URL = "https://api.scryfall.com/cards/search"
# Scryfall responde 400 a quien llega con el User-Agent de fabrica de su
# Libreria: pide que la Aplicacion se nombre. El Accept lo pide su Guia.
HEADERS = {"User-Agent": "Muchi/1.0", "Accept": "application/json"}

# El Codigo que Scryfall entiende, el Nombre que el Front muestra, y como se
# escribe Sol Ring ahi: el Ejemplo ensena que Idioma espera el Campo. Los
# Nombres salen de la propia Fuente, no de una Traduccion inventada.
LANGUAGES = (
    ("es", "Español", "Anillo solar"),
    ("pt", "Portugués", "Anel Solar"),
    ("en", "Inglés", "Sol Ring"),
    ("fr", "Francés", "Anneau solaire"),
    ("de", "Alemán", "Sonnenring"),
    ("it", "Italiano", "Anello Solare"),
    ("ja", "Japonés", "太陽の指輪"),
)
LANGUAGE_CODES = frozenset(code for code, _, _ in LANGUAGES)


class NameTranslator:
    """Traduce un Nombre escrito en cualquier Idioma al Nombre Canónico."""

    # El Core pregunta los Idiomas al Traductor; nadie mas nombra a Scryfall.
    languages = LANGUAGES
    codes = LANGUAGE_CODES

    def __init__(self, timeout_seconds: float = 10) -> None:
        self.timeout_seconds = timeout_seconds

    def translate_name(self, name: str, language: str = "") -> str:
        if language:
            return self._search_in_language(name, language)
        return self._named(name)

    def _named(self, name: str) -> str:
        reply = self._ask(NAMED_URL, {"fuzzy": name})
        if reply.status_code == 404:
            raise CardNotFound(name)
        self._check(reply)
        return str(reply.json()["name"])

    def suggest_names(self, text: str, language: str,
                      limit: int = 3) -> tuple[str, ...]:
        """Nombres que empiezan como lo escrito. Vacio si no hay ninguno.

        Sugerir es un Favor, no un Servicio: un Fracaso de la Fuente devuelve
        Nada y quien escribe ni se entera.
        """
        try:
            reply = self._ask(SEARCH_URL, {
                "q": f'lang:{language} "{text}"',
                "include_multilingual": "true",
                "unique": "cards",
            })
        except TranslationFailed:
            return ()
        if not reply.ok:
            return ()
        try:
            found = reply.json().get("data") or []
        except ValueError:
            return ()
        # printed_name es el Nombre en ese Idioma; name es el Canonico, que
        # aparece cuando la Impresion no trae Nombre traducido.
        names = []
        for card in found:
            written = card.get("printed_name") or card.get("name") or ""
            if written and written not in names:
                names.append(written)
            if len(names) == limit:
                break
        return tuple(names)

    def find_art(self, name: str, language: str = "",
                 edition: str = "", foil: bool = False) -> dict:
        """La Imagen de una Carta y su Enlace a Scryfall.

        Con Edicion se pide esa Impresion y no otra: la Carta que se mira es
        la que esta en venta, con su Ilustracion y su Marco. Si esa Impresion
        no aparece, se cae a la Carta a secas antes que dejar el Panel vacio.

        Mirar es un Favor, como sugerir: si la Fuente falla, devuelve Nada.
        La Imagen la sirve Scryfall directo al Navegador; el BFF solo pasa
        la Direccion, y ningun Byte de Carta cruza por aca.
        """
        if edition:
            exact = self._art_of_printing(name, edition, foil)
            if exact:
                return exact
        return self._art_of_card(name, language)

    def _art_of_printing(self, name: str, edition: str, foil: bool) -> dict:
        """Una Impresion concreta. El Nombre va exacto: ! no admite parecidos."""
        query = f'!"{name}" set:{edition}'
        if foil:
            query += " is:foil"
        card = self._first_card(SEARCH_URL, {"q": query, "unique": "prints"})
        return self._art_of(card) if card else {}

    def _art_of_card(self, name: str, language: str) -> dict:
        if language:
            card = self._first_card(SEARCH_URL, {
                "q": f'lang:{language} "{name}"',
                "include_multilingual": "true", "unique": "cards"})
        else:
            card = self._first_card(NAMED_URL, {"fuzzy": name})
        return self._art_of(card) if card else {}

    def _first_card(self, url: str, params: dict) -> dict:
        """La primera Carta que devuelva la Puerta, o Nada si algo sale mal."""
        try:
            reply = self._ask(url, params)
        except TranslationFailed:
            return {}
        if not reply.ok:
            return {}
        try:
            body = reply.json()
        except ValueError:
            return {}
        if "data" in body:
            found = body.get("data") or []
            return found[0] if found else {}
        return body

    @staticmethod
    def _art_of(card: dict) -> dict:
        # Una Carta de dos Caras no trae image_uris arriba: la Cara si.
        images = card.get("image_uris")
        if not images:
            faces = card.get("card_faces") or []
            images = (faces[0].get("image_uris") if faces else None) or {}
        picture = images.get("normal") or images.get("large") or images.get("small")
        if not picture:
            return {}
        return {"name": card.get("name", ""),
                "printed_name": card.get("printed_name") or "",
                "image": picture,
                "url": card.get("scryfall_uri", "")}

    def _search_in_language(self, name: str, language: str) -> str:
        # Las Comillas piden la Frase entera: sin ellas cada Palabra busca sola.
        reply = self._ask(SEARCH_URL, {
            "q": f'lang:{language} "{name}"',
            "include_multilingual": "true",
            "order": "released",
        })
        if reply.status_code == 404:
            raise CardNotFound(name)
        self._check(reply)
        found = reply.json().get("data") or []
        if not found:
            raise CardNotFound(name)
        return str(found[0]["name"])

    def _ask(self, url: str, params: dict):
        try:
            return requests.get(url, params=params, headers=HEADERS,
                                timeout=self.timeout_seconds)
        except requests.RequestException as error:
            raise TranslationFailed(str(error)) from error

    @staticmethod
    def _check(reply) -> None:
        if reply.ok:
            return
        # El Detalle de Scryfall dice por que; sin el, un 400 no se puede leer.
        try:
            detail = str(reply.json().get("details", ""))
        except ValueError:
            detail = ""
        raise TranslationFailed(
            f"scryfall {reply.status_code}: {detail}" if detail
            else f"scryfall {reply.status_code}")
