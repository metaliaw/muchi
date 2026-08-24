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

La config vive en tiendas-api.json en la raiz (ver tiendas-api.ejemplo.json).
Si el archivo no existe, esto no hace nada: es opcional.
"""
from __future__ import annotations

import json
import os
from dataclasses import dataclass, field
from pathlib import Path

from ..http import PoliteSession
from ..models import Offer

CONFIG = Path(__file__).resolve().parent.parent.parent / "tiendas-api.json"


@dataclass(frozen=True)
class TiendaAPI:
    nombre: str
    url: str
    tipo: str = "json"              # "json" | "postgrest"
    env_apikey: str = ""            # nombre de la variable de entorno, no la clave
    campos: dict = field(default_factory=dict)

    @property
    def apikey(self) -> str:
        return os.getenv(self.env_apikey, "") if self.env_apikey else ""


def cargar(ruta: Path | str = CONFIG) -> list[TiendaAPI]:
    """Lee la config. Sin archivo devuelve lista vacia: la feature es opcional."""
    ruta = Path(ruta)
    if not ruta.exists():
        return []
    datos = json.loads(ruta.read_text(encoding="utf-8"))
    return [TiendaAPI(**t) for t in datos.get("tiendas", [])]


def _texto(fila: dict, clave: str, defecto: str = "") -> str:
    valor = fila.get(clave)
    return str(valor).strip() if valor not in (None, "") else defecto


def _precio(fila: dict, clave: str) -> int | None:
    crudo = fila.get(clave)
    if crudo in (None, ""):
        return None
    try:
        return int(round(float(str(crudo).replace("$", "").replace(".", "").strip())))
    except (TypeError, ValueError):
        return None


def a_ofertas(tienda: TiendaAPI, filas: list[dict]) -> list[Offer]:
    """Traduce las filas crudas a Offer segun el mapeo de campos configurado."""
    c = tienda.campos
    salida: list[Offer] = []

    for fila in filas:
        precio = _precio(fila, c.get("precio", "precio"))
        if not precio or precio <= 0:
            continue

        # Si declaran stock, respetamos que sea > 0; si no lo declaran, asumimos
        # que lo publicado esta disponible.
        campo_stock = c.get("stock")
        if campo_stock:
            try:
                if int(fila.get(campo_stock) or 0) <= 0:
                    continue
            except (TypeError, ValueError):
                pass

        nombre = _texto(fila, c.get("nombre", "nombre"))
        if not nombre:
            continue

        edicion = _texto(fila, c.get("edicion", ""), "")
        condicion = _texto(fila, c.get("condicion", ""), "")
        titulo = nombre + (f" [{edicion}]" if edicion else "")
        if condicion:
            titulo += f" - {condicion}"

        salida.append(Offer(
            store=tienda.nombre,
            card_name=nombre,
            title=titulo,
            price_clp=precio,
            url=_texto(fila, c.get("url", ""), tienda.url),
            finish=_texto(fila, c.get("acabado", ""), "Normal"),
            condition=condicion,
            language=_texto(fila, c.get("idioma", ""), ""),
            source="api",
            marketplace=False,
            key=f"{tienda.nombre}:{_texto(fila, c.get('id', 'id'), nombre)}",
        ))

    return sorted(salida, key=lambda o: o.price_clp)


def buscar(sess: PoliteSession, tienda: TiendaAPI, nombre: str) -> list[Offer]:
    """Consulta una tienda. Solo lectura: nunca escribe ni autentica usuarios."""
    campo = tienda.campos.get("nombre", "nombre")

    if tienda.tipo == "postgrest":
        cabeceras = {"Accept": "application/json"}
        if tienda.apikey:
            cabeceras["apikey"] = tienda.apikey
            cabeceras["Authorization"] = f"Bearer {tienda.apikey}"
        r = sess.get(
            tienda.url,
            params={"select": "*", campo: f"ilike.*{nombre}*", "limit": 100},
            headers=cabeceras,
        )
    else:
        r = sess.get(tienda.url.format(q=nombre), headers={"Accept": "application/json"})

    datos = r.json()
    if isinstance(datos, dict):
        # Endpoints que envuelven la lista, p.ej. {"resultados": [...]}
        for llave in (tienda.campos.get("raiz"), "data", "results", "resultados", "items"):
            if llave and isinstance(datos.get(llave), list):
                datos = datos[llave]
                break
        else:
            datos = []

    return a_ofertas(tienda, datos if isinstance(datos, list) else [])
