"""Conector generico para tiendas que exponen su stock por una API de solo lectura.

Existe para el caso del Wombat y de cualquier tienda chica que no tenga
e-commerce: si publican un endpoint de lectura, Muchi los consulta como a
cualquier otra tienda, sin scrapear nada.

Dos perfiles:

  "postgrest"  Supabase y cualquier PostgREST. La consulta se arma sola:
                   GET {url}?select=*&{campo_nombre}=ilike.*termino*
               Las claves NUNCA van en el archivo de config: se leen de una
               variable de entorno (ver `env_apikey`).

  "json"       Cualquier endpoint que devuelva una lista de objetos JSON.
               El termino de busqueda se interpola en `url` con {q}.

La config vive en store-api.json en la raiz (ver store-api.example.json).
Si el archivo no existe, esto no hace nada: es opcional.
"""
from __future__ import annotations

import json
import os
from dataclasses import dataclass, field
from pathlib import Path

from muchi.paths import ROOT
from ..http import PoliteSession
from ..models import Offer

CONFIG = ROOT / "store-api.json"


@dataclass(frozen=True)
class StoreApi:
    name: str
    url: str
    kind: str = "json"              # "json" | "postgrest"
    env_apikey: str = ""            # nombre de la variable de entorno, no la clave
    fields: dict = field(default_factory=dict)

    @property
    def api_key(self) -> str:
        return os.getenv(self.env_apikey, "") if self.env_apikey else ""


def build_store(row: dict) -> StoreApi:
    """Traduce una entrada del archivo al vocabulario del nucleo.

    Las claves del JSON siguen en espanol a proposito: ese archivo lo escribe
    el duenio de la tienda, y su documentacion tambien esta en espanol. La
    traduccion vive aca, que es el borde, y no se filtra hacia adentro.
    """
    return StoreApi(
        name=row.get("nombre", ""),
        url=row.get("url", ""),
        kind=row.get("tipo", "json"),
        env_apikey=row.get("env_apikey", ""),
        fields=row.get("campos") or {},
    )


def load_stores(path: Path | str = CONFIG) -> list[StoreApi]:
    """Lee la config. Sin archivo devuelve lista vacia: la feature es opcional."""
    path = Path(path)
    if not path.exists():
        return []

    data = json.loads(path.read_text(encoding="utf-8"))
    return [build_store(row) for row in data.get("tiendas", [])]


def read_field_text(row: dict, key: str, default: str = "") -> str:
    value = row.get(key)
    return str(value).strip() if value not in (None, "") else default


def read_price_field(row: dict, key: str) -> int | None:
    raw_value = row.get(key)
    if raw_value in (None, ""):
        return None
    try:
        return int(round(float(str(raw_value).replace("$", "").replace(".", "").strip())))
    except (TypeError, ValueError):
        return None


def build_offers(store: StoreApi, rows: list[dict]) -> list[Offer]:
    """Traduce las filas crudas a Oferta segun el mapeo de campos configurado."""
    c = store.fields
    out: list[Offer] = []

    for row in rows:
        price = read_price_field(row, c.get("precio", "precio"))
        if not price or price <= 0:
            continue

        # Si declaran stock, respetamos que sea > 0; si no lo declaran, asumimos
        # que lo publicado esta disponible.
        stock_field = c.get("stock")
        if stock_field:
            try:
                if int(row.get(stock_field) or 0) <= 0:
                    continue
            except (TypeError, ValueError):
                pass

        name = read_field_text(row, c.get("nombre", "nombre"))
        if not name:
            continue

        edition = read_field_text(row, c.get("edicion", ""), "")
        condition = read_field_text(row, c.get("condicion", ""), "")
        title = name + (f" [{edition}]" if edition else "")
        if condition:
            title += f" - {condition}"

        out.append(Offer(
            store=store.name,
            card_name=name,
            title=title,
            price_clp=price,
            url=read_field_text(row, c.get("url", ""), store.url),
            finish=read_field_text(row, c.get("acabado", ""), "Normal"),
            condition=condition,
            language=read_field_text(row, c.get("idioma", ""), ""),
            source="api",
            marketplace=False,
            key=f"{store.name}:{read_field_text(row, c.get('id', 'id'), name)}",
        ))

    return sorted(out, key=lambda o: o.price_clp)


def query_store(sess: PoliteSession, store: StoreApi, name: str) -> list[Offer]:
    """Consulta una tienda. Solo lectura: nunca escribe ni autentica usuarios."""
    field = store.fields.get("nombre", "nombre")

    if store.kind == "postgrest":
        headers = {"Accept": "application/json"}
        if store.api_key:
            headers["apikey"] = store.api_key
            headers["Authorization"] = f"Bearer {store.api_key}"
        r = sess.get(
            store.url,
            params={"select": "*", field: f"ilike.*{name}*", "limit": 100},
            headers=headers,
        )
    else:
        r = sess.get(store.url.format(q=name), headers={"Accept": "application/json"})

    data = r.json()
    if isinstance(data, dict):
        # Endpoints que envuelven la lista, p.ej. {"resultados": [...]}
        for wrapper_key in (store.fields.get("raiz"), "data", "results", "resultados", "items"):
            if wrapper_key and isinstance(data.get(wrapper_key), list):
                data = data[wrapper_key]
                break
        else:
            data = []

    return build_offers(store, data if isinstance(data, list) else [])


@dataclass
class StoreApiSource:
    """Cumple FuenteOfertas para una tienda con endpoint de solo lectura."""
    sess: PoliteSession
    store: StoreApi

    @property
    def name(self) -> str:
        return self.store.name

    def find_offers(self, card_name: str) -> list[Offer]:
        return query_store(self.sess, self.store, card_name)


def build_api_sources(sess: PoliteSession,
                      path: Path | str = CONFIG) -> list[StoreApiSource]:
    """Una fuente por tienda declarada. Sin config, ninguna: es opcional."""
    return [StoreApiSource(sess, t) for t in load_stores(path)]
