[English](supported-games.md) · [Español](supported-games.es.md)

# Supported Games Reflect Configured Stores

`GET /api/supported-games` returns games available from configured stores, with an optional `kind` filter for single cards or sealed products. The interface offers only capabilities the active search composition can serve.

The route is implemented by [`read_supported_games`](../../server/main.py).
