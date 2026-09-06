"""Umbrales de negocio ajustables sin tocar la logica de Muchi."""

# Una cola baja se considera sospechosa si el siguiente precio es al menos
# esta cantidad de veces mayor. Ejemplo: $152 -> $1.462 supera 4x.
SUSPICIOUS_PRICE_GAP_RATIO = 4
SUSPICIOUS_PRICE_MIN_OFFERS = 5
SUSPICIOUS_PRICE_MAX_LOW_OFFERS = 2
SUSPICIOUS_PRICE_MAX_LOW_SHARE = 0.20

# Estas fuentes informan disponibilidad, pero Muchi no puede comprobarla
# contra la tienda en el momento de mostrar cada oferta.
UNVERIFIED_STOCK_SOURCES = frozenset({"scry", "directo", "muchi-api"})

# Verificacion en vivo de las ofertas que probablemente recibiran el clic.
STOCK_VERIFY_CHEAPEST_OFFERS = 5
STOCK_VERIFY_TIMEOUT_SECONDS = 10.0
STOCK_VERIFY_CACHE_SECONDS = 60

# Sólo se buscan dentro de señales estructuradas o controles de compra; nunca
# en todo el texto de la página, donde podrían aparecer en políticas o reseñas.
OUT_OF_STOCK_MARKERS = (
    "fuera de stock",
    "sin stock",
    "agotado",
    "sold out",
    "out of stock",
)

# CLP por dolar para convertir las ofertas de muchi-api (Scryfall: TCGPlayer,
# Cardmarket, Cardhoarder cotizan en USD). Misma tasa por defecto que
# sources/moxfield.py usa para CardKingdom.
MUCHI_API_USD_CLP_RATE = 700
