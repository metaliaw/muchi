"""Reparto de una decklist entre tiendas.

Comprar cada carta en la tienda mas barata suele ser una mala idea: si eso te
deja comprando en 9 tiendas, pagas 9 envios y terminas gastando mas. Esto
resuelve el trade-off real: precio de las cartas + costo de los envios.

Es un problema tipo set-cover (NP-dificil), asi que usamos una heuristica:
greedy de construccion + busqueda local de mejora. Con ~30 tiendas y ~100
cartas da resultados muy buenos en milisegundos, pero no garantiza el optimo.
"""
from __future__ import annotations

from dataclasses import dataclass, field

from .models import Offer, Pedido


@dataclass
class Linea:
    carta: str
    cantidad: int
    tienda: str
    precio_unitario: int
    url: str
    titulo: str

    @property
    def subtotal(self) -> int:
        return self.precio_unitario * self.cantidad


@dataclass
class Plan:
    lineas: list[Linea] = field(default_factory=list)
    faltantes: list[str] = field(default_factory=list)
    envio_por_tienda: int = 0

    @property
    def tiendas(self) -> list[str]:
        return sorted({l.tienda for l in self.lineas})

    @property
    def costo_cartas(self) -> int:
        return sum(l.subtotal for l in self.lineas)

    @property
    def costo_envios(self) -> int:
        return self.envio_por_tienda * len(self.tiendas)

    @property
    def total(self) -> int:
        return self.costo_cartas + self.costo_envios


def _mejor_por_tienda(ofertas: list[Offer]) -> dict[str, Offer]:
    """De cada tienda nos quedamos con su oferta mas barata para esa carta."""
    mejor: dict[str, Offer] = {}
    for o in ofertas:
        actual = mejor.get(o.store)
        if actual is None or o.price_clp < actual.price_clp:
            mejor[o.store] = o
    return mejor


def plan_mas_barato(pedidos: list[Pedido],
                    ofertas_por_carta: dict[str, list[Offer]],
                    envio_por_tienda: int = 0) -> Plan:
    """Baseline ingenuo: cada carta en su tienda mas barata, sin mirar envios."""
    plan = Plan(envio_por_tienda=envio_por_tienda)
    for p in pedidos:
        ofertas = ofertas_por_carta.get(p.nombre.lower()) or []
        if not ofertas:
            plan.faltantes.append(p.nombre)
            continue
        o = min(ofertas, key=lambda x: x.price_clp)
        plan.lineas.append(Linea(p.nombre, p.cantidad, o.store, o.price_clp, o.url, o.title))
    return plan


def plan_optimo(pedidos: list[Pedido],
                ofertas_por_carta: dict[str, list[Offer]],
                envio_por_tienda: int = 0,
                max_tiendas: int | None = None) -> Plan:
    """Minimiza (cartas + envios) eligiendo en que tiendas comprar."""
    # precios[i][tienda] = oferta mas barata de esa carta en esa tienda
    precios: list[dict[str, Offer]] = []
    cantidades: list[int] = []
    nombres: list[str] = []
    faltantes: list[str] = []

    for p in pedidos:
        ofertas = ofertas_por_carta.get(p.nombre.lower()) or []
        if not ofertas:
            faltantes.append(p.nombre)
            continue
        precios.append(_mejor_por_tienda(ofertas))
        cantidades.append(p.cantidad)
        nombres.append(p.nombre)

    if not precios:
        return Plan(faltantes=faltantes, envio_por_tienda=envio_por_tienda)

    candidatas = sorted({t for m in precios for t in m})
    # Penalizacion por carta no cubierta: alta como para forzar cobertura,
    # pero finita para que el algoritmo no explote si algo es incomprable.
    tope = max(o.price_clp for m in precios for o in m.values())
    PENA = tope * 4 + envio_por_tienda * 10

    def costo(seleccion: frozenset[str]) -> int:
        if not seleccion:
            return PENA * sum(cantidades)
        total = envio_por_tienda * len(seleccion)
        for i, mapa in enumerate(precios):
            disponibles = [mapa[t].price_clp for t in seleccion if t in mapa]
            total += (min(disponibles) if disponibles else PENA) * cantidades[i]
        return total

    # --- construccion greedy ---
    actual: frozenset[str] = frozenset()
    mejor_costo = costo(actual)
    while max_tiendas is None or len(actual) < max_tiendas:
        candidata, candidata_costo = None, mejor_costo
        for t in candidatas:
            if t in actual:
                continue
            c = costo(actual | {t})
            if c < candidata_costo:
                candidata, candidata_costo = t, c
        if candidata is None:
            break
        actual = actual | {candidata}
        mejor_costo = candidata_costo

    # --- busqueda local: sacar y permutar tiendas ---
    mejoro = True
    while mejoro:
        mejoro = False
        for t in list(actual):
            c = costo(actual - {t})
            if c < mejor_costo:
                actual, mejor_costo, mejoro = actual - {t}, c, True
                break
        if mejoro:
            continue
        for dentro in list(actual):
            for fuera in candidatas:
                if fuera in actual:
                    continue
                nueva = (actual - {dentro}) | {fuera}
                c = costo(nueva)
                if c < mejor_costo:
                    actual, mejor_costo, mejoro = nueva, c, True
                    break
            if mejoro:
                break

    # --- materializar el plan ---
    plan = Plan(faltantes=faltantes, envio_por_tienda=envio_por_tienda)
    for i, mapa in enumerate(precios):
        opciones = [mapa[t] for t in actual if t in mapa]
        if not opciones:
            plan.faltantes.append(nombres[i])
            continue
        o = min(opciones, key=lambda x: x.price_clp)
        plan.lineas.append(Linea(nombres[i], cantidades[i], o.store, o.price_clp, o.url, o.title))

    return plan
