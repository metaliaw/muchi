[English](card-names.md) · [Español](card-names.es.md)

# Card Names Resolve Language at the Boundary

`GET /api/card` translates the entered name into the canonical card name.

The browser can accept names in the person's language while the search domain works with canonical identities. Translation errors remain distinct from a missing card, so the interface can explain a temporary failure separately from an unknown name.

The route is implemented by [`read_card`](../../server/main.py).
