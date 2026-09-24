[English](sources.md) · [Español](sources.es.md)

# Sources Reports the Active Search Composition

`GET /api/sources` exposes the offer sources assembled by the current Muchi
search configuration. The interface can explain where results may come from
without maintaining a second, drifting list of stores.

The list is derived from the configured search service, so disabled or
unsupported sources are not asserted by the browser as available.

The route is implemented by [`read_sources`](../../server/main.py).
