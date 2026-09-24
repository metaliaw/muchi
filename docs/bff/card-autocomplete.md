[English](card-autocomplete.md) · [Español](card-autocomplete.es.md)

# Card Autocomplete Searches the Selected Game Catalog

`GET /api/card/autocomplete` takes a game, partial name and optional language, then returns catalog candidates while the person types. It keeps game-specific catalog behavior out of Vue and does not start a store search.

The route is implemented by [`autocomplete_cards`](../../server/main.py).
