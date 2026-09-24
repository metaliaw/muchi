[English](spa.md) · [Español](spa.es.md)

# The SPA Shares an Origin with the BFF

When a compiled Frontend is present, `/assets/*` serves its hashed assets and `GET /{path}` serves existing public files or the Vue application shell for a history route. Runtime HTML receives the configured AdSense tag before it is sent.

Serving the interface and `/api/*` from one Cloud Run service gives the browser one origin and avoids a separate CORS boundary. The path is resolved under the build directory so a requested file cannot escape it.

The mount and fallback are implemented in [`server/main.py`](../../server/main.py).
