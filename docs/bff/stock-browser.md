[English](stock-browser.md) · [Español](stock-browser.es.md)

# Browser Stock Checks Add Store Observations

`POST /api/searches/{id}/stock` accepts availability observed by the shopper's browser for offers in this search. The BFF validates offer IDs, combines those observations with optional server-side checks, then computes the recommendation.

This route exists for stores whose public catalogs can be read by the browser. It avoids repeating a store visit from the server while keeping ranking and recommendation decisions on the server.

The route is implemented by [`check_stock_with_browser`](../../server/main.py) and `answer_stock`.
