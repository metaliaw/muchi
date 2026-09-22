"""Adapta el Contrato HTTP de Muchi a Búsquedas del Dominio."""
from __future__ import annotations

from dataclasses import dataclass, field
from decimal import Decimal, InvalidOperation
from urllib.parse import quote

import requests

from muchi.mtg.models import Order
from muchi.mtg.ports import QueryFailed, SearchRejected
from muchi.mtg.search import (SearchItem, SearchOffer, SearchResults, SearchState,
                              StockCheck)


def build_search(reply: dict) -> SearchState:
    return SearchState(
        id=reply["id"], status=reply["status"], total=reply["total"],
        processed=reply["processed"], found=reply["found"],
        errors=reply["errors"], current_card=reply.get("current_card") or "",
        game=reply.get("game") or "",
    )


def read_metadata(row: dict, field: str) -> str:
    metadata = row.get("metadata") or {}
    return str(metadata.get(field) or "")


# La Edicion no es un Campo del Contrato: cae en metadata, y cada Tienda la
# nombra a su manera. Se toma la primera Llave que traiga algo.
EDITION_FIELDS = ("set_code", "set", "edition", "expansion")


def read_edition(row: dict) -> str:
    for field in EDITION_FIELDS:
        if found := read_metadata(row, field):
            return found
    return ""


def read_locations(row: dict) -> tuple[str, ...]:
    locations = row.get("locations") or []
    if not isinstance(locations, list):
        raise ValueError("Invalid locations")
    return tuple(str(location).strip() for location in locations
                 if str(location).strip())


def read_quantity(row: dict) -> int | None:
    """Las Unidades declaradas. Ausente es "no se Pregunto", no es Cero."""
    quantity = row.get("stock_quantity")
    if quantity is None:
        return None
    if type(quantity) is not int or quantity < 0:
        raise ValueError("Invalid stock quantity")
    return quantity


def build_offer(row: dict) -> SearchOffer:
    amount = Decimal(row["price_amount"])
    if not amount.is_finite() or amount < 0:
        raise ValueError("Invalid price")
    return SearchOffer(
        card_name=row["card_name"], store=row["store"], amount=amount,
        currency=row["price_currency"], url=row["url"],
        stock_status=row["stock_status"], suspicious=row["suspicious"],
        source=row["source"], finish=row.get("finish") or "",
        language=row.get("language") or "", condition=row.get("condition") or "",
        variant=read_metadata(row, "variant"), title=read_metadata(row, "title"),
        edition=read_edition(row), card_key=row.get("card_key") or "",
        locations=read_locations(row),
        offer_id=str(row.get("id") or ""),
        stock_quantity=read_quantity(row),
        image=row.get("image") or "",
        suspicious_reason=row.get("suspicious_reason") or "",
        metadata={str(key): str(value) for key, value in (row.get("metadata") or {}).items()},
    )


def order_offers(offers) -> tuple[SearchOffer, ...]:
    """Ordena por Precio dentro de cada Moneda.

    La API agrupa las Ofertas por Fuente y solo ordena dentro del Grupo, así
    que la más barata de todas puede llegar en cualquier Posición. Las Monedas
    no se comparan entre sí: un Peso y un Dólar no son el mismo Número.
    """
    return tuple(sorted(offers, key=lambda offer: (offer.currency, offer.amount)))


def read_faults(row: dict) -> tuple[str, ...]:
    """Solo el Nombre de la Fuente. La Razon es para el Log, no para quien Busca."""
    names = []
    for fault in row.get("faults") or ():
        name = str(fault.get("source") or "").strip()
        if name and name not in names:
            names.append(name)
    return tuple(names)


def build_results(reply: dict) -> SearchResults:
    """Lee una Página de Resultados.

    Dos Campos Llegan ausentes aunque el Contrato los Declare obligatorios: la
    API los Marca `omitempty`, así que un `sequence` en cero Desaparece y un
    Item sin Ofertas Manda `null` en vez de una Lista vacía. Ambos son Estados
    normales de un Item que aún no Termina o que no se Encontró, y ninguno
    Merece Tumbar el Ciclo entero.
    """
    ordered = sorted(reply["items"] or (), key=lambda row: row["position"])
    items = tuple(SearchItem(
        name=row["original_name"], quantity=row["quantity"], status=row["status"],
        offers=order_offers(build_offer(offer) for offer in row["offers"] or ()),
        error_message=row.get("error_message") or "",
        id=row["id"], position=row["position"], sequence=row.get("sequence") or 0,
        game=row.get("game") or "",
        faults=read_faults(row),
    ) for row in ordered)
    return SearchResults(items, reply["cursor"], reply["has_more"])


def name_contract_fault(error: Exception) -> str:
    """Qué Parte de la Respuesta no se Pudo Leer.

    Sin esto una Respuesta que Cambió de Forma se Cuenta igual que una rota:
    el Mensaje Decía que el Contrato no se Cumple y Callaba en qué Campo, que
    es lo único que Habría Servido para Arreglarlo.
    """
    if isinstance(error, KeyError):
        return f"falta el Campo «{error.args[0]}»."
    detail = str(error).strip() or type(error).__name__
    return f"{detail[:120]}."


def build_stock_checks(reply: dict) -> tuple[StockCheck, ...]:
    return tuple(StockCheck(
        offer_id=str(row["id"]),
        stock_status=row["stock_status"],
        stock_quantity=read_quantity(row),
    ) for row in reply["offers"])


@dataclass
class SearchProvider:
    base_url: str
    token: str = field(repr=False)
    timeout_seconds: float = 10
    session: requests.Session = field(default_factory=requests.Session, repr=False)

    def request_reply(self, method: str, path: str, *, payload=None,
                      key: str | None = None, params=None) -> dict:
        headers = {"Authorization": f"Bearer {self.token}", "Accept": "application/json"}
        if key:
            headers["Idempotency-Key"] = key
        try:
            response = self.session.request(
                method, f"{self.base_url}{path}", headers=headers,
                json=payload, params=params, timeout=self.timeout_seconds,
                allow_redirects=False,
            )
            with response:
                if response.status_code in (401, 403):
                    raise SearchRejected("La API rechazó el Código de Seguridad.")
                if response.status_code == 409:
                    raise SearchRejected("La API informó un Conflicto de Estado o Idempotencia.")
                if 400 <= response.status_code < 500 and response.status_code not in (408, 429):
                    raise SearchRejected(f"La API rechazó el Pedido: HTTP {response.status_code}.")
                if not 200 <= response.status_code < 300:
                    raise QueryFailed(f"La API respondió HTTP {response.status_code}.")
                reply = response.json()
                if not isinstance(reply, dict):
                    raise ValueError("Expected object")
                return reply
        except requests.Timeout:
            raise QueryFailed("La API agotó el Tiempo de Espera. Se puede reintentar la Consulta.") from None
        except requests.ConnectionError:
            raise QueryFailed("No se pudo conectar con la API. Comprueba que el Servicio esté levantado y MUCHI_API_URL sea correcta.") from None
        except (requests.RequestException, ValueError):
            raise QueryFailed("No se pudo obtener una Respuesta válida de la API.") from None

    def parse_reply(self, parser, reply):
        try:
            return parser(reply)
        except (KeyError, TypeError, ValueError, InvalidOperation) as error:
            raise QueryFailed(
                "La Respuesta no cumple el Contrato de la API: "
                f"{name_contract_fault(error)}") from None

    def create_search(self, orders: list[Order], verify_stock: bool,
                      key: str, game: str = "magic",
                      match: str = "exact",
                      kind: str = "single") -> SearchState:
        if not 1 <= len(orders) <= 500:
            raise QueryFailed("La Búsqueda admite entre 1 y 500 Cartas.")
        if any(not order.name.strip() or not 1 <= order.quantity <= 99 for order in orders):
            raise QueryFailed("Cada Carta requiere Nombre y Cantidad entre 1 y 99.")
        payload = {
            "game": game,
            "cards": [{"name": order.name, "quantity": order.quantity} for order in orders],
            "options": {"verify_stock": verify_stock, "match": match,
                        "kind": kind},
        }
        reply = self.request_reply("POST", "/searches", payload=payload, key=key)
        return self.parse_reply(build_search, reply)

    def read_search(self, search_id: str) -> SearchState:
        reply = self.request_reply("GET", f"/searches/{quote(search_id, safe='')}")
        return self.parse_reply(build_search, reply)

    def read_results(self, search_id: str, after: int = 0) -> SearchResults:
        reply = self.request_reply(
            "GET", f"/searches/{quote(search_id, safe='')}/results",
            params={"after": after, "limit": 50},
        )
        return self.parse_reply(build_results, reply)

    def cancel_search(self, search_id: str, key: str) -> SearchState:
        reply = self.request_reply(
            "POST", f"/searches/{quote(search_id, safe='')}/cancel", key=key,
        )
        return self.parse_reply(build_search, reply)

    def check_stock(self, search_id: str,
                    offer_ids: tuple[str, ...]) -> tuple[StockCheck, ...]:
        """Le Pide a la API que vuelva a mirar el Stock de estas Ofertas.

        La Consulta a la Tienda vive del otro Lado de la Frontera: el Worker ya
        Sabe hablarle a cada Proveedor, y repetir esa Logica aca la publicaria
        dos veces. El BFF solo Decide a quien preguntar y en que Orden.
        """
        if not offer_ids:
            return ()
        reply = self.request_reply(
            "POST", f"/searches/{quote(search_id, safe='')}/stock",
            payload={"offers": list(offer_ids)},
        )
        return self.parse_reply(build_stock_checks, reply)

    def read_sources(self) -> list[dict]:
        reply = self.request_reply("GET", "/health/sources")
        return self.parse_reply(lambda value: value["sources"], reply)

    def read_supported_games(self, kind: str = "") -> list[dict]:
        """Los Juegos que se Pueden buscar, y para qué Tipo.

        Cada Juego Trae sus dos Marcas. `kind` Recorta la Lista a los que
        Contestan esa Pregunta, para quien Dibuja un Selector de un Tipo solo.
        """
        reply = self.request_reply("GET", "/supported-games",
                                   params={"kind": kind} if kind else None)
        return self.parse_reply(lambda value: value["games"], reply)

    def read_card_metadata(self, game: str, name: str, language: str = "",
                           edition: str = "", foil: bool = False) -> dict:
        return self.request_reply("GET", "/cards/metadata", params={
            "game": game, "name": name, "language": language,
            "edition": edition, "foil": str(foil).lower(),
        })

    def autocomplete_cards(self, game: str, name: str, language: str = "") -> list[str]:
        reply = self.request_reply("GET", "/cards/autocomplete", params={
            "game": game, "name": name, "language": language,
        })
        return self.parse_reply(lambda value: value["suggestions"], reply)

    def find_offers(self, name: str) -> tuple[SearchOffer, ...]:
        reply = self.request_reply("GET", "/cards/offers", params={"name": name})
        return self.parse_reply(
            lambda value: tuple(build_offer(row) for row in value["offers"]), reply,
        )
