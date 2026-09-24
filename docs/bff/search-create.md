[English](search-create.md) · [Español](search-create.es.md)

# Search Creation Turns a List into Idempotent Work

`POST /api/searches` parses and validates the entire list, applies the public
card and quantity limits, and creates the search through the shared Muchi
domain. The idempotency key means a retry after an uncertain response refers to
the same intent.

The BFF returns an initial visible state and queued rows immediately. Slow
source queries continue behind the API boundary, so the browser can begin
polling without holding an HTTP request open. Rejected lines and invalid list
sizes are returned before any search is created.

The route is implemented by [`create_search`](../../server/main.py); the API's
queue and worker behavior belongs to its
[architecture guide](https://github.com/cangrejometralleta/muchi-api/blob/main/docs/architecture.md).
