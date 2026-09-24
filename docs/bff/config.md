[English](config.md) · [Español](config.es.md)

# Config Describes the Public Runtime

`GET /api/config` lets the Vue application learn the current public limits,
polling interval, exchange rate, feature switches, links and social networks
from the same deployment that serves it.

The response deliberately excludes the backend bearer token. The browser needs
product configuration to render consistently, while the BFF alone needs the
credential to cross the service boundary. Limits are returned by the server so
the interface can guide input without becoming their authority.

The route is implemented by [`read_config`](../../server/main.py).
