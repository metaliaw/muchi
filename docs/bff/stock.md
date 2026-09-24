[English](stock.md) · [Español](stock.es.md)

# Service Stock Checks Rank Candidates

`GET /api/searches/{id}/stock` asks configured stores about ranked candidates
for this search.

The BFF owns candidate ranking and the recommendation. Checks proceed in price
order and stop when the cheapest confirmed candidate for each card type is
known or the configured visit limit is reached.

The route is implemented by [`check_stock`](../../server/main.py) and
`answer_stock`.
