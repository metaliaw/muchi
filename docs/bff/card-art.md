[English](card-art.md) · [Español](card-art.es.md)

# Card Art Resolves the Selected Printing

`GET /api/card/art` asks the translator for artwork by name, language, edition and foil choice. It gives the browser a single boundary for art lookup while keeping language validation and provider-specific resolution on the server.

The image itself can be delivered directly to the browser by its returned source URL; the BFF does not proxy large image bytes. Missing art and unsupported languages receive explicit client errors.

The route is implemented by [`read_card_art`](../../server/main.py).
