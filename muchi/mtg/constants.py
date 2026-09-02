"""Umbrales de negocio ajustables sin tocar la logica de Muchi."""

# Una cola baja se considera sospechosa si el siguiente precio es al menos
# esta cantidad de veces mayor. Ejemplo: $152 -> $1.462 supera 4x.
SUSPICIOUS_PRICE_GAP_RATIO = 4
SUSPICIOUS_PRICE_MIN_OFFERS = 5
SUSPICIOUS_PRICE_MAX_LOW_OFFERS = 2
SUSPICIOUS_PRICE_MAX_LOW_SHARE = 0.20

# Estas fuentes informan disponibilidad, pero Muchi no puede comprobarla
# contra la tienda en el momento de mostrar cada oferta.
UNVERIFIED_STOCK_SOURCES = frozenset({"scry", "directo"})
