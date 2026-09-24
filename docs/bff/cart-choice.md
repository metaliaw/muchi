[English](cart-choice.md) · [Español](cart-choice.es.md)

# Cart Choice Totals the Shopper's Own Plan

`POST /api/searches/{id}/cart` accepts quantities selected by the shopper and returns totals using the same currency conversion and shipping model as the suggested cart. It reports the person's plan without silently optimizing over it.

The route is implemented by [`read_chosen_cart`](../../server/main.py) and [`build_cart`](../../server/presenter.py).
