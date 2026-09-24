[English](card-metadata.md) · [Español](card-metadata.es.md)

# Card Metadata Resolves a Printing for Display

`GET /api/card/metadata` returns game, language, edition and foil-specific metadata for a selected card identity. The interface can display a printing from the same catalog rules used by search.

The route is implemented by [`read_card_metadata`](../../server/main.py).
