[English](health.md) · [Español](health.es.md)

# Health Reports the BFF's Availability

`GET /api/health` answers whether this BFF process can serve requests. Cloud
Run can check it without depending on the search API or external stores.

It returns `{"status":"ok"}` and performs no downstream request. That makes
the signal cheap and keeps an API outage distinct from a BFF process outage.
It does not promise that searches are available; those dependencies are
observed through request failures and service monitoring.

The route is implemented by [`read_health`](../../server/main.py).
