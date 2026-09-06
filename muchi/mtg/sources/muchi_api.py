"""Cliente de muchi-api, el backend propio de busqueda y precios.

muchi-api ya resuelve una carta contra Scryfall (enlaces a TCGPlayer,
Cardmarket, Cardhoarder) y contra las tiendas que declara en su propio
config/stores.yaml. Consultarlo suma esas ofertas sin reimplementar esa
logica aca.

Los precios de Scryfall llegan en USD (ver internal/scryfall/client.go de
muchi-api). Como Muchi cotiza siempre en pesos, se convierten con una tasa
fija -- igual idea que sources/moxfield.py usa para CardKingdom.

Contrato: GET {MUCHI_API_URL}/v1/cards/offers?name=<carta>
Header:   Authorization: Bearer <MUCHI_API_TOKEN>

Sin las variables de entorno, esto no hace nada: la fuente es opcional, igual
que store_api y moxfield.
"""
from __future__ import annotations

import os
from dataclasses import dataclass

from .. import constants
from ..http import PoliteSession
from ..models import Offer


def convert_to_clp(amount: str, currency: str, rate: int) -> int | None:
    """CLP entero, o None si no sabemos convertir esa moneda."""
    try:
        value = float(amount)
    except (TypeError, ValueError):
        return None
    if currency == "CLP":
        return int(round(value))
    if currency == "USD":
        return int(round(value * rate))
    return None


def build_offers(reply: dict, rate: int) -> list[Offer]:
    """Traduce la respuesta de /v1/cards/offers a Oferta."""
    fallback_name = reply.get("name") or ""
    out: list[Offer] = []

    for row in reply.get("offers") or []:
        price = convert_to_clp(row.get("price_amount", ""), row.get("price_currency", ""), rate)
        if price is None or price <= 0:
            continue

        name = row.get("card_name") or fallback_name
        if not name:
            continue

        title = name
        detail = " / ".join(x for x in (row.get("condition"), row.get("finish")) if x)
        if detail:
            title += f" - {detail}"

        out.append(Offer(
            store=row.get("store") or "?",
            card_name=name,
            title=title,
            price_clp=price,
            url=row.get("url") or "",
            finish=row.get("finish") or "",
            condition=row.get("condition") or "",
            language=row.get("language") or "",
            source="muchi-api",
            marketplace=False,
            key=row.get("id") or row.get("variant_id")
                or f"{row.get('store')}:{row.get('url')}",
        ))

    return sorted(out, key=lambda o: o.price_clp)


def query_offers(sess: PoliteSession, base_url: str, token: str,
                 name: str, rate: int) -> list[Offer]:
    """Consulta una carta. Solo lectura: GET, nunca escribe."""
    r = sess.get(
        f"{base_url}/v1/cards/offers",
        params={"name": name},
        headers={"Authorization": f"Bearer {token}", "Accept": "application/json"},
    )
    return build_offers(r.json(), rate)


@dataclass
class MuchiApiSource:
    """Cumple FuenteOfertas contra el backend propio muchi-api."""
    sess: PoliteSession
    base_url: str
    token: str
    rate: int = constants.MUCHI_API_USD_CLP_RATE
    name: str = "muchi-api"

    def find_offers(self, card_name: str) -> list[Offer]:
        return query_offers(self.sess, self.base_url, self.token, card_name, self.rate)


def build_source(sess: PoliteSession) -> MuchiApiSource | None:
    """Arma la fuente si estan MUCHI_API_URL y MUCHI_API_TOKEN. Sin ellas, nada."""
    url = os.getenv("MUCHI_API_URL", "").rstrip("/")
    token = os.getenv("MUCHI_API_TOKEN", "")
    if not url or not token:
        return None
    return MuchiApiSource(sess, url, token)
