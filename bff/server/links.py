"""Las Redes de Muchi: un Archivo público, leído una sola vez.

Antes cada Red era una Variable de Entorno, declarada cinco veces —en el
Ejemplo, en el Script de Deploy, en la Plantilla de Build, en el Servidor y en
su Prueba— para guardar una Dirección que cualquiera puede leer en la Página.
Eso es Configuración de Producto, no un Secreto: vive en `config/socials.yaml`,
a la vista, y se cambia sin recordar el Nombre de ninguna Variable.
"""
from __future__ import annotations

from functools import lru_cache

import yaml

from muchi.paths import ROOT

SOCIALS_FILE = ROOT / "config" / "socials.yaml"
# Una Dirección que no es https no es una Red: es un Enlace roto esperando.
LINK_SCHEME = "https://"


def validate_networks(value) -> list[dict]:
    """Un Archivo malo Detiene el Arranque; no Sirve media Barra."""
    if not isinstance(value, dict) or not isinstance(value.get("networks"), list):
        raise ValueError("socials.yaml must declare a list of networks")
    networks = []
    for row in value["networks"]:
        if not isinstance(row, dict) or set(row) - {"name", "icon", "url"}:
            raise ValueError("Unknown network fields")
        name, icon = str(row.get("name", "")).strip(), str(row.get("icon", "")).strip()
        url = str(row.get("url") or "").strip()
        if not name or not icon:
            raise ValueError("A network needs a name and an icon")
        if url and not url.startswith(LINK_SCHEME):
            raise ValueError(f"{name} must link over https")
        networks.append({"name": name, "icon": icon, "url": url})
    return networks


@lru_cache(maxsize=1)
def load_socials() -> tuple[dict, ...]:
    """Carga las Redes una vez al Arrancar, en el Orden del Archivo."""
    text = SOCIALS_FILE.read_text(encoding="utf-8")
    return tuple(validate_networks(yaml.safe_load(text) or {}))


def read_socials() -> list[dict]:
    """Solo las que Llevan a alguna parte: una Ranura vacía no es una Red."""
    return [dict(red) for red in load_socials() if red["url"]]
