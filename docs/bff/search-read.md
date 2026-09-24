[English](search-read.md) · [Español](search-read.es.md)

# Search Reading Joins State and a Result Page

`GET /api/searches/{search_id}` gives the browser one polling response with
search state, a page of results, and a continuation cursor. The BFF shapes
offers, stock labels, currency ordering and visible cart recommendations so
Vue does not reproduce business presentation rules.

The `after` cursor bounds each read as a search grows. The `match` option tells
the presenter how to group related card names; it does not alter persisted API
results. The browser can stop polling on terminal states and retain its last
valid response when a later read fails.

The route is implemented by [`read_search`](../../server/main.py) and
[`presenter.py`](../../server/presenter.py).
