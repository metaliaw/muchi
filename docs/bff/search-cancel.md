[English](search-cancel.md) · [Español](search-cancel.es.md)

# Search Cancellation Uses the Original Intent Key

`POST /api/searches/{id}/cancel` delegates cancellation with the idempotency
key used to create the search. This lets the API verify that the cancellation
belongs to the same caller intent and makes repeated cancellation requests
safe.

The BFF returns the updated visible state. Queue cleanup and terminal-state
rules remain the API's responsibility.

The route is implemented by [`cancel_search`](../../server/main.py).
