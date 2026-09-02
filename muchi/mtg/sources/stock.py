"""Verificación puntual de stock en la página final de una oferta."""
from __future__ import annotations

import json
from dataclasses import dataclass
from urllib.parse import parse_qs, urlparse

from bs4 import BeautifulSoup

from .. import constants
from ..http import PoliteSession
from ..models import Offer
from .shopify import product_json_url, read_variant_availability


def _schema_availability(value) -> bool | None:
    """Busca availability dentro de JSON-LD, incluso si viene anidado."""
    if isinstance(value, dict):
        availability = str(value.get("availability") or "").lower()
        if availability.endswith("outofstock"):
            return False
        if availability.endswith("instock"):
            return True
        for child in value.values():
            result = _schema_availability(child)
            if result is not None:
                return result
    elif isinstance(value, list):
        for child in value:
            result = _schema_availability(child)
            if result is not None:
                return result
    return None


def _woo_variation_availability(form, selected: dict[str, str]) -> bool | None:
    """Lee la variante WooCommerce exacta cuando viene embebida en el form."""
    raw = form.get("data-product_variations")
    if raw is None:
        return None
    try:
        variations = json.loads(raw)
    except (json.JSONDecodeError, TypeError):
        return None
    if not isinstance(variations, list):
        return None
    if not variations:
        return False

    for variation in variations:
        attributes = variation.get("attributes") or {}
        if selected and all(str(attributes.get(key, "")) == value
                            for key, value in selected.items()):
            value = variation.get("is_in_stock")
            return value if isinstance(value, bool) else None
    return None


def read_page_availability(html: str,
                           selected: dict[str, str] | None = None) -> bool | None:
    """Interpreta señales acotadas; una frase suelta global no es evidencia."""
    soup = BeautifulSoup(html, "lxml")
    selected = selected or {}

    # WooCommerce mezcla recomendaciones disponibles con el producto agotado.
    # Todo este bloque queda deliberadamente limitado al producto principal.
    product = soup.select_one('main div.product[id^="product-"]') \
        or soup.select_one('div.product[id^="product-"]')
    if product is not None:
        variation_form = product.select_one("form.variations_form")
        if variation_form is not None:
            result = _woo_variation_availability(variation_form, selected)
            if result is not None:
                return result
        if "outofstock" in (product.get("class") or []):
            return False
        if product.select_one("p.stock.out-of-stock") is not None:
            return False
        if product.select_one("p.stock.in-stock") is not None:
            return True

    for tag in soup.select('script[type="application/ld+json"]'):
        try:
            result = _schema_availability(json.loads(tag.string or tag.get_text()))
        except (json.JSONDecodeError, TypeError):
            continue
        if result is not None:
            return result

    for tag in soup.select('[itemprop="availability"]'):
        value = str(tag.get("content") or tag.get("href") or tag.get_text()).lower()
        if value.endswith("outofstock") or "outofstock" in value:
            return False
        if value.endswith("instock") or "instock" in value:
            return True

    selectors = ('main [class*="stock" i], main [id*="stock" i], '
                 'main [class*="inventory" i], main [id*="inventory" i], '
                 'main [class*="availability" i], main [id*="availability" i], '
                 'main button[disabled]')
    for tag in soup.select(selectors):
        text = " ".join(tag.get_text(" ", strip=True).lower().split())
        if any(marker in text for marker in constants.OUT_OF_STOCK_MARKERS):
            return False
    return None


@dataclass
class StorePageStockVerifier:
    """Prueba Shopify primero y cae a inspeccionar la página HTML final."""
    sess: PoliteSession

    def verify_stock(self, offer: Offer) -> bool | None:
        target = product_json_url(offer.url)
        if target:
            endpoint, variant_id = target
            try:
                data = self.sess.get(endpoint).json()
                result = (read_variant_availability(data, variant_id)
                          if isinstance(data, dict) else None)
                if result is not None:
                    return result
            except Exception:
                pass

        try:
            selected = {
                key: values[0]
                for key, values in parse_qs(urlparse(offer.url).query).items()
                if key.startswith("attribute_") and values
            }
            return read_page_availability(self.sess.get(offer.url).text, selected)
        except Exception:
            return None
