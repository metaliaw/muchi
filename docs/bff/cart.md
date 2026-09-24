[English](cart.md) · [Español](cart.es.md)

# Cart Reading Separates Advice from Choice

`GET /api/searches/{id}/cart` returns the suggested allocation across offers
and stores. `POST` totals quantities the shopper explicitly selected without
replacing that choice with a newly optimized plan.

Both paths use the same currency conversion, shipping input and presentation
model. The distinction lets the interface offer a starting plan, then report
the cost of the person's own selections.

The routes are implemented by [`read_cart`](../../server/main.py),
[`read_chosen_cart`](../../server/main.py), and [`build_cart`](../../server/presenter.py).
