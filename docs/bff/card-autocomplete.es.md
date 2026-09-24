[English](card-autocomplete.md) · **Español**

# Card Autocomplete Consulta el Catálogo del Juego

`GET /api/card/autocomplete` recibe juego, nombre parcial e idioma opcional y devuelve candidatos mientras la Persona escribe. Mantiene fuera de Vue las reglas del catálogo y no inicia una búsqueda en tiendas.

La ruta está implementada por [`autocomplete_cards`](../../server/main.py).
