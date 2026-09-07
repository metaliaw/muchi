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
from decimal import Decimal

from .models import Offer, Order


@dataclass
class Line:
    card_name: str
    quantity: int
    store: str
    unit_price: int | Decimal
    url: str
    title: str

    @property
    def subtotal(self) -> int | Decimal:
        return self.unit_price * self.quantity


@dataclass
class Plan:
    lines: list[Line] = field(default_factory=list)
    missing: list[str] = field(default_factory=list)
    shipping_per_store: int = 0

    @property
    def stores(self) -> list[str]:
        return sorted({l.store for l in self.lines})

    @property
    def cards_cost(self) -> int | Decimal:
        return sum(l.subtotal for l in self.lines)

    @property
    def shipping_cost(self) -> int | Decimal:
        return self.shipping_per_store * len(self.stores)

    @property
    def total(self) -> int | Decimal:
        return self.cards_cost + self.shipping_cost


def pick_best_per_store(offers: list[Offer]) -> dict[str, Offer]:
    """De cada tienda nos quedamos con su oferta mas barata para esa carta."""
    best: dict[str, Offer] = {}
    for o in offers:
        current = best.get(o.store)
        if current is None or o.price_clp < current.price_clp:
            best[o.store] = o
    return best


def build_naive_plan(orders: list[Order],
                     offers_by_card: dict[str, list[Offer]],
                     shipping_per_store: int = 0) -> Plan:
    """Baseline ingenuo: cada carta en su tienda mas barata, sin mirar envios."""
    plan = Plan(shipping_per_store=shipping_per_store)
    for p in orders:
        offers = offers_by_card.get(p.name.lower()) or []
        if not offers:
            plan.missing.append(p.name)
            continue
        o = min(offers, key=lambda x: x.price_clp)
        plan.lines.append(
            Line(p.name, p.quantity, o.store, o.price_clp, o.url, o.title))
    return plan


def build_optimal_plan(orders: list[Order],
                       offers_by_card: dict[str, list[Offer]],
                       shipping_per_store: int = 0,
                       max_stores: int | None = None) -> Plan:
    """Minimiza (cartas + envios) eligiendo en que tiendas comprar."""
    # precios[i][tienda] = oferta mas barata de esa carta en esa tienda
    prices: list[dict[str, Offer]] = []
    quantities: list[int] = []
    names: list[str] = []
    missing: list[str] = []

    for p in orders:
        offers = offers_by_card.get(p.name.lower()) or []
        if not offers:
            missing.append(p.name)
            continue
        prices.append(pick_best_per_store(offers))
        quantities.append(p.quantity)
        names.append(p.name)

    if not prices:
        return Plan(missing=missing, shipping_per_store=shipping_per_store)

    candidates = sorted({t for m in prices for t in m})
    # Penalizacion por carta no cubierta: alta como para forzar cobertura,
    # pero finita para que el algoritmo no explote si algo es incomprable.
    ceiling = max(o.price_clp for m in prices for o in m.values())
    PENALTY = ceiling * 4 + shipping_per_store * 10

    def cost_selection(selection: frozenset[str]) -> int | Decimal:
        if not selection:
            return PENALTY * sum(quantities)
        total = shipping_per_store * len(selection)
        for i, by_store in enumerate(prices):
            available = [by_store[t].price_clp for t in selection if t in by_store]
            total += (min(available) if available else PENALTY) * quantities[i]
        return total

    # --- construccion greedy ---
    current: frozenset[str] = frozenset()
    best_cost = cost_selection(current)
    while max_stores is None or len(current) < max_stores:
        candidate, candidate_cost = None, best_cost
        for t in candidates:
            if t in current:
                continue
            c = cost_selection(current | {t})
            if c < candidate_cost:
                candidate, candidate_cost = t, c
        if candidate is None:
            break
        current = current | {candidate}
        best_cost = candidate_cost

    # --- busqueda local: sacar y permutar tiendas ---
    improved = True
    while improved:
        improved = False
        for t in list(current):
            c = cost_selection(current - {t})
            if c < best_cost:
                current, best_cost, improved = current - {t}, c, True
                break
        if improved:
            continue
        for inside in list(current):
            for outside in candidates:
                if outside in current:
                    continue
                new_set = (current - {inside}) | {outside}
                c = cost_selection(new_set)
                if c < best_cost:
                    current, best_cost, improved = new_set, c, True
                    break
            if improved:
                break

    # --- materializar el plan ---
    plan = Plan(missing=missing, shipping_per_store=shipping_per_store)
    for i, by_store in enumerate(prices):
        options = [by_store[t] for t in current if t in by_store]
        if not options:
            plan.missing.append(names[i])
            continue
        o = min(options, key=lambda x: x.price_clp)
        plan.lines.append(
            Line(names[i], quantities[i], o.store, o.price_clp, o.url, o.title))

    return plan
