"""Recomendaciones de EDHREC para un comandante.

EDHREC publica el mismo JSON que consume su propio sitio, una peticion por
comandante:

    https://json.edhrec.com/pages/commanders/<slug>.json

De ahi salen ~13 listas (Top Cards, High Synergy, Creatures, Mana Artifacts...)
con dos metricas que valen la pena distinguir:

  inclusion = num_decks / potential_decks
      Que tan comun es la carta. "El 61% de los Krenko juegan Skullclamp."
  sinergia
      Cuanto MAS aparece con este comandante que en el resto del formato. Una
      carta con inclusion alta y sinergia baja es un staple generico (Sol Ring);
      una con sinergia alta es especifica de este mazo.

Su robots.txt permite /commanders (solo bloquea previews de articulos y
/deckpreview/).
"""
from __future__ import annotations

from dataclasses import dataclass

import requests

from ..http import PoliteSession
from ..texto import slug

BASE = "https://json.edhrec.com/pages/commanders/"

# EDHREC lista las basicas entre las recomendaciones (Mountain aparece en el
# 97.6% de los Krenko). Como sugerencia de compra no aportan nada, asi que por
# defecto quedan fuera.
_BASICAS = {"plains", "island", "swamp", "mountain", "forest", "wastes"}


def es_basica(nombre: str) -> bool:
    n = nombre.strip().lower()
    for prefijo in ("snow-covered ",):
        if n.startswith(prefijo):
            n = n[len(prefijo):]
    return n in _BASICAS


class ComandanteNoEncontrado(LookupError):
    """No hay pagina de EDHREC para ese nombre."""


@dataclass(frozen=True)
class Recomendacion:
    nombre: str
    categoria: str      # "Top Cards", "High Synergy Cards", ...
    tag: str            # "topcards", "highsynergycards", ...
    inclusion: float    # 0..1
    sinergia: float
    num_decks: int

    @property
    def inclusion_pct(self) -> float:
        return round(self.inclusion * 100, 1)


def recomendaciones(sess: PoliteSession, comandante: str,
                    incluir_basicas: bool = False) -> list[Recomendacion]:
    """Todas las cartas que EDHREC sugiere para ese comandante.

    Vienen ordenadas por inclusion descendente y sin duplicados: una misma
    carta puede aparecer en "Top Cards" y en "Creatures" a la vez, y nos
    quedamos con la categoria mas especifica que la nombra primero.
    """
    url = f"{BASE}{slug(comandante)}.json"
    try:
        datos = sess.get(url).json()
    except requests.HTTPError as e:
        # Un slug inexistente responde 403 (no 404). Tambien seria 403 si nos
        # bloquearan, pero desde la app lo primero es muchisimo mas probable.
        codigo = getattr(e.response, "status_code", None)
        if codigo in (403, 404):
            raise ComandanteNoEncontrado(comandante) from e
        raise

    listas = (datos.get("container") or {}).get("json_dict", {}).get("cardlists") or []

    salida: list[Recomendacion] = []
    vistas: set[str] = set()

    for lista in listas:
        categoria = lista.get("header") or "?"
        tag = lista.get("tag") or ""
        for c in lista.get("cardviews") or []:
            nombre = (c.get("name") or "").strip()
            if not nombre or nombre.lower() in vistas:
                continue
            if not incluir_basicas and es_basica(nombre):
                continue
            vistas.add(nombre.lower())

            potenciales = c.get("potential_decks") or 0
            usados = c.get("num_decks") or 0
            salida.append(Recomendacion(
                nombre=nombre,
                categoria=categoria,
                tag=tag,
                inclusion=(usados / potenciales) if potenciales else 0.0,
                sinergia=float(c.get("synergy") or 0.0),
                num_decks=usados,
            ))

    return sorted(salida, key=lambda r: -r.inclusion)


def faltantes(recs: list[Recomendacion], ya_tengo: set[str]) -> list[Recomendacion]:
    """Filtra las que ya estan en tu mazo. Compara normalizado, no literal."""
    tengo = {slug(n) for n in ya_tengo}
    return [r for r in recs if slug(r.nombre) not in tengo]


def categorias(recs: list[Recomendacion]) -> list[str]:
    """Categorias presentes, en el orden en que EDHREC las publica."""
    vistas: list[str] = []
    for r in recs:
        if r.categoria not in vistas:
            vistas.append(r.categoria)
    return vistas
