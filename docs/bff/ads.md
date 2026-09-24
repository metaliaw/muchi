[English](ads.md) · [Español](ads.es.md)

# ads.txt Declares an Active Publisher

`GET /ads.txt` returns the authorized seller line required by AdSense when a
valid publisher ID is configured. With no valid ID, the route returns 404 so
the site does not claim an advertising relationship it does not have.

The publisher ID is runtime configuration. The BFF can change it without
rebuilding Vue, and the seller authority identifier remains part of the
published format.

The route is implemented by [`read_ads_txt`](../../server/main.py).
