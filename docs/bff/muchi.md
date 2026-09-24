[English](muchi.md) · [Español](muchi.es.md)

# Muchi Serves the Curated Voice of the Interface

`GET /api/muchi` gives the interface its phrases, greetings, help topics and
context-specific responses from the shared phrase catalogue. Keeping this
content behind one route lets the client render a consistent voice without
copying the source data into Vue bundles.

The response groups phrases by their use and preserves each phrase's state.
It does not choose when to speak; the interface chooses among the available
lines.

The route is implemented by [`read_muchi`](../../server/main.py).
