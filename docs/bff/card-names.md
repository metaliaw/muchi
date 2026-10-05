[English](card-names.md) · [Español](card-names.es.md)

# Card Names Resolve the Language at the Boundary

`GET /api/card` translates the entered name into the canonical card name. It takes `name`, an optional `game` (Magic when absent) and an optional `language`.

The browser accepts names in the person's language while the search domain works with canonical identities. muchi-api reads the name through the game's language provider and refuses a `language` the game does not declare in `supported-games`. A card the catalog does not know answers 404, apart from a catalog that did not answer, which is a failure and can be retried.

The route is implemented by [`read_card`](../../server/main.py).
