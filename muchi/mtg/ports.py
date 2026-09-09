"""Los Fallos que el Núcleo declara y que algo de afuera origina.

Los Errores son del Dominio. Quien llama atrapa SearchRejected, jamás el
requests.HTTPError que lo originó: un Error del Vendor que cruza el Puerto es
una fuga, igual que un Tipo del Vendor.

El Puerto que los levanta es SearchService, en search.py. Acá vivían también
los Puertos de las Fuentes propias —Catálogos, Inventarios, Verificadores de
Stock—; hoy las Ofertas las trae la API y su Worker las consigue, así que el
Núcleo ya no declara con quién hablar para juntarlas.
"""
from __future__ import annotations


class QueryFailed(RuntimeError):
    """La Fuente no respondió, o respondió algo que no entendemos."""


class SearchRejected(QueryFailed):
    """El Servicio rechazó el Pedido; corregirlo permite un nuevo Envío."""
