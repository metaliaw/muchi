[English](supported-games.md) · [Español](supported-games.es.md)

# Supported Games Reflect Configured Stores

`GET /api/supported-games` returns games available from configured stores, with an optional `kind` filter for single cards or sealed products. The interface offers only capabilities the active search composition can serve.

Games that declare languages carry them in `languages`, each with a `code`, `label` and `example`. `backend/config/stores.yaml` declares them, validates the language of every card query, and picks the provider that reads the names. The interface only announces what arrives: a game without `languages` shows no note.

The route is implemented by [`read_supported_games`](../../bff/server/main.py).
