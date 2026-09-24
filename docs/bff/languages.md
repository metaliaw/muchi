[English](languages.md) · [Español](languages.es.md)

# Languages Reports Translation Capability

`GET /api/languages` returns the translator's supported language codes, labels and examples. The interface uses that catalogue rather than guessing which localized card names can resolve.

The route is implemented by [`read_languages`](../../server/main.py).
