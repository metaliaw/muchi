[English](decklist.md) · [Español](decklist.es.md)

# Decklist Previews Input Before a Search

`POST /api/decklist` parses pasted text and returns recognized card quantities
and ignored lines. The interface can show the person exactly what a search
would contain before it starts backend work.

This preview shares the production parser, including sealed-product mode. It
does not create a search, reserve work or query stores. Invalid lines remain
visible to the user instead of silently becoming a different list.

The route is implemented by [`read_decklist`](../../server/main.py).
