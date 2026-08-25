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
from ..ports import CommanderNotFound
from ..ports import Recommendation as Recommendation
from ..text import normalize_name

BASE = "https://json.edhrec.com/pages/commanders/"

# EDHREC lista las basicas entre las recomendaciones (Mountain aparece en el
# 97.6% de los Krenko). Como sugerencia de compra no aportan nada, asi que por
# defecto quedan fuera.
_BASIC_LANDS = {"plains", "island", "swamp", "mountain", "forest", "wastes"}


def is_basic_land(name: str) -> bool:
    n = name.strip().lower()
    for prefix in ("snow-covered ",):
        if n.startswith(prefix):
            n = n[len(prefix):]
    return n in _BASIC_LANDS


class CommanderPageMissing(LookupError):
    """No hay pagina de EDHREC para ese nombre."""


@dataclass(frozen=True)
class RawRecommendation:
    name: str
    category: str      # "Top Cards", "High Synergy Cards", ...
    tag: str            # "topcards", "highsynergycards", ...
    inclusion: float    # 0..1
    synergy: float
    num_decks: int

    @property
    def inclusion_pct(self) -> float:
        return round(self.inclusion * 100, 1)


def fetch_recommendations(sess: PoliteSession, commander: str,
                          include_basics: bool = False) -> list[RawRecommendation]:
    """Todas las cartas que EDHREC sugiere para ese comandante.

    Vienen ordenadas por inclusion descendente y sin duplicados: una misma
    carta puede aparecer en "Top Cards" y en "Creatures" a la vez, y nos
    quedamos con la categoria mas especifica que la nombra primero.
    """
    url = f"{BASE}{normalize_name(commander)}.json"
    try:
        data = sess.get(url).json()
    except requests.HTTPError as e:
        # Un slug inexistente responde 403 (no 404). Tambien seria 403 si nos
        # bloquearan, pero desde la app lo primero es muchisimo mas probable.
        code = getattr(e.response, "status_code", None)
        if code in (403, 404):
            raise CommanderPageMissing(commander) from e
        raise

    lists = (data.get("container") or {}).get("json_dict", {}).get("cardlists") or []

    out: list[RawRecommendation] = []
    seen: set[str] = set()

    for card_list in lists:
        category = card_list.get("header") or "?"
        tag = card_list.get("tag") or ""
        for c in card_list.get("cardviews") or []:
            name = (c.get("name") or "").strip()
            if not name or name.lower() in seen:
                continue
            if not include_basics and is_basic_land(name):
                continue
            seen.add(name.lower())

            potential = c.get("potential_decks") or 0
            used = c.get("num_decks") or 0
            out.append(RawRecommendation(
                name=name,
                category=category,
                tag=tag,
                inclusion=(used / potential) if potential else 0.0,
                synergy=float(c.get("synergy") or 0.0),
                num_decks=used,
            ))

    return sorted(out, key=lambda r: -r.inclusion)


@dataclass
class EdhrecAdvisor:
    """Cumple RecomendadorMazo. Traduce EDHREC al vocabulario del nucleo.

    Contrato: https://json.edhrec.com/pages/commanders/<slug>.json
    """
    sess: PoliteSession

    def recommend_cards(self, commander: str) -> list[Recommendation]:
        try:
            raw = fetch_recommendations(self.sess, commander)
        except CommanderPageMissing as e:
            raise CommanderNotFound(commander) from e

        return [
            Recommendation(name=r.name, category=r.category,
                           inclusion=r.inclusion, synergy=r.synergy)
            for r in raw
        ]
