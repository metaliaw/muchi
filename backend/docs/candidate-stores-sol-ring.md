# Candidate Stores from Sol Ring

[English](candidate-stores-sol-ring.md) | [Español](stores-sol-ring.md)

Verification: September 7, 2026. Live GCP search:
`search_1bad0a3b9c8aacc035c23576`. It completed with one card found and zero
errors. It returned **208 offers**: 198 from scry.cl, 4 direct offers from La
Cripta, and 6 from Moxfield.

API: https://muchi-serve-api-c2ce6c7oga-rj.a.run.app. Health: HTTP 200. Worker
without authentication: HTTP 403. The Cloud Tasks flow completed successfully
in 2.71 seconds using the cache from the earlier direct query.

**OnPlay, the three Shopify stores, and the four Jumpseller stores are enabled
in `config/stores.yaml`.** The five remaining stores need more work. La Cripta
and El Wombat Rabioso are already configured.

The counts below are offers present in this Scry search, not each store's full
catalog size or confirmed stock.

| Store | Domain | Offers | Platform | Next Step |
| --- | --- | ---: | --- | --- |
| OnPlay Singles | [onplay.cl](https://onplay.cl) | 8 | WooCommerce | Added to stores.yaml |
| Rhystic Bazaar | [rhysticbazaar.cl](https://rhysticbazaar.cl) | 17 | WooCommerce | Adjust name matching |
| CardNexus | [cardnexus.cl](https://cardnexus.cl) | 26 | WooCommerce | Resolve reader access |
| Game of Magic Singles | [gameofmagicsingles.cl](https://gameofmagicsingles.cl) | 14 | Shopify | Added to stores.yaml |
| Collector Center | [singles.collectorcenter.cl](https://singles.collectorcenter.cl) | 16 | Shopify | Added to stores.yaml |
| CardSouls | [www.cardsouls.cl](https://www.cardsouls.cl) | 16 | Shopify | Added to stores.yaml |
| Cartas La Fortaleza | [www.cartaslafortaleza.cl](https://www.cartaslafortaleza.cl) | 3 | Jumpseller | Added to stores.yaml |
| ChronoMagic | [www.chronomagic.cl](https://www.chronomagic.cl) | 1 | Jumpseller | Added to stores.yaml |
| GameQuest | [gamequest.cl](https://gamequest.cl) | 3 | Jumpseller | Added to stores.yaml |
| Magic4Ever | [www.magic4ever.cl](https://www.magic4ever.cl) | 3 | Jumpseller | Added to stores.yaml |
| CatLotus | [catlotus.cl](https://catlotus.cl) | 39 | No compatible API verified | Investigate catalog |
| Dominio Arcano | [dominioarcano.cl](https://dominioarcano.cl) | 2 | No compatible API verified | Investigate catalog |
| HunterCard TCG | [www.huntercardtcg.com](https://www.huntercardtcg.com) | 1 | No compatible API verified | Review URL and catalog |

## Evidence by Store

- **OnPlay Singles:** The real reader completed the query: 8 CLP offers, all
  marked available by the catalog. [Observed product](https://onplay.cl/product/sol-ring-5/).
  [Queried catalog](https://onplay.cl/wp-json/wc/store/v1/products?search=Sol%20Ring&per_page=100).
- **Rhystic Bazaar:** Its public API is accessible. It uses names such as
  “Sol Ring — Near Mint,” which current exact equality excludes.
  [Observed product](https://rhysticbazaar.cl/product/sol-ring-near-mint-7/).
  [Queried catalog](https://rhysticbazaar.cl/wp-json/wc/store/v1/products?search=Sol%20Ring&per_page=100).
- **CardNexus:** An exploratory query returned 100 products with exact names in
  CLP. The real Go reader received HTTP 403 with browser verification; complete
  pagination remains unverified.
  [Observed product](https://cardnexus.cl/product/sol-ring-36/).
  [Queried catalog](https://cardnexus.cl/wp-json/wc/store/v1/products?search=Sol%20Ring&per_page=100).
- **Game of Magic Singles:** Shopify reader enabled, with paginated search and
  stock by variant. [Observed product](https://gameofmagicsingles.cl/products/sol-ring-2683-secret-lair-drop-series?variant=54369284817206).
  [Verified product JSON](https://gameofmagicsingles.cl/products/sol-ring-2683-secret-lair-drop-series.js).
- **Collector Center:** Shopify reader enabled, with paginated search and stock
  by variant. [Observed product](https://singles.collectorcenter.cl/products/stcg-card-92a2e8c4-dc36-4443-b95e-3f8f26d4b84b?variant=51967974768928).
  [Verified product JSON](https://singles.collectorcenter.cl/products/stcg-card-92a2e8c4-dc36-4443-b95e-3f8f26d4b84b.js).
- **CardSouls:** Shopify reader enabled, with paginated search and stock by
  variant. [Observed product](https://www.cardsouls.cl/products/sol-ring-0356-fic-356?variant=51342901575903).
  [Verified product JSON](https://www.cardsouls.cl/products/sol-ring-0356-fic-356.js).
- **Cartas La Fortaleza:** Platform identified on the product page. The
  WooCommerce route returns 404. [Observed product](https://www.cartaslafortaleza.cl/sol-ring-ingles-nm-cma).
- **ChronoMagic:** Platform identified on the product page. The WooCommerce
  route returns 404. [Observed product](https://www.chronomagic.cl/sol-ring-lcc-313-en-nm).
- **GameQuest:** Platform identified on the product page. The WooCommerce route
  returns 404. [Observed product](https://gamequest.cl/sol-ring-espanol-nm-afc).
- **Magic4Ever:** Platform identified on the product page. The WooCommerce route
  returns 404. [Observed product](https://www.magic4ever.cl/sol-ring-7?variant=121035117).
- **CatLotus:** Product page accessible; the WooCommerce route returns 404.
  [Observed product](https://catlotus.cl/cartas/C21/Sol%20Ring?name=Sol+Ring&set=C21).
- **Dominio Arcano:** Product page accessible. The tested route returns 200 but
  not the expected WooCommerce product JSON.
  [Observed product](https://dominioarcano.cl/mtg/single/sol-ring-c20-252).
- **HunterCard TCG:** The product URL returned by Scry returns 404. The tested
  route does not return a compatible WooCommerce catalog.
  [Observed product](https://www.huntercardtcg.com/producto/sol-ring-showcase-promo-316-ingles/).

## OnPlay Store Entry

Enabled entry under `stores:` in `config/stores.yaml`:

```yaml
  onplay.cl:
    name: "OnPlay Singles"
    platform: woocommerce
    unavailable_selectors:
      - "div.product.outofstock"
      - "p.stock.out-of-stock"
    unavailable_text:
      - "Sold out"
      - "Agotado"
    scope_selector: "div.product"
    timeout_seconds: 10
    allow_redirects: true
    enabled: true
```

The check confirmed the catalog, CLP prices, and availability reported by the
WooCommerce API. The selectors follow La Cripta's existing pattern; an out-of-
stock OnPlay page was not verified. HTML stock checks retain the current reader's
limitations.

## Marketplace Sellers

The search also returned **46 offers from 20 sellers** on `marketplace.scry.cl`.
Their links do not provide a standalone domain that could be added as separate
WooCommerce catalogs.

| Seller | Offers |
| --- | ---: |
| Bazar de Waldosky | 1 |
| Cartas del Nexo | 1 |
| El Bazar de Olguis | 1 |
| Fallen Angel | 2 |
| Grafitos | 8 |
| Ineko Card Shop | 4 |
| Kydroukair | 3 |
| La resistencia de Gerrard | 1 |
| Magic Master | 1 |
| Mana Surge | 5 |
| Mox & Mana | 1 |
| Naipes Embrujados TCG | 1 |
| Posada Canto de Guerra | 3 |
| The Stack | 4 |
| Tiendálogo | 1 |
| Wither and Bloom | 1 |
| Woodedcard | 1 |
| Zanahoria | 2 |
| laipp's store | 2 |
| moxes | 3 |

These results sample one card and the state observed on this date; they are not
an exhaustive list of stores in Chile.

## Shopify Reader

Queries `/search` with the name in quotes, product type, and availability filter.
It follows pagination and reads `/products/{handle}.js` for available variants,
prices, and options. It queries `/cart.js` to verify currency and converts
hundredths to currency units. It accepts set suffixes and rejects names of other
cards. Each offer retains a URL with `?variant=ID`; stock checks query that
variant, even when it is sold out.

Optional live test: `MUCHI_TEST_SHOPIFY_LIVE=1 go test ./internal/stores/shopify -run TestLiveSolRing -v`.
These configuration and reader changes need a new deployment to reach GCP.

Direct validation with the Go reader found **74 available Sol Ring offers**:
34 from Game of Magic Singles, 19 from Collector Center, and 21 from CardSouls.
All three queries completed; every offer reported CLP, and one available variant
per store was checked. Total time: 28.36 seconds, without production Firestore
coordination. These counts are from this direct test, not the historical GCP
search above.

## Jumpseller Reader

Cartas La Fortaleza, ChronoMagic, GameQuest, and Magic4Ever use
`platform: jumpseller`. The reader follows `/api/search/{name}?page=...` with the
storefront's AJAX headers, filters card names (including set, language, and
condition suffixes), and queries only matching products. Jumpseller performs
approximate searches: quotes do not remove unrelated results, and some queries
have many pages. The search does not stop at a page without matches. Pagination
identifies products by ID because links can repeat across distinct products. It
ends when the API returns an empty page and rejects pages that contain no new ID.

The themes of all four stores publish `product-form-json` and `product-json` in
HTML. The reader checks currency in product metadata, reads prices in currency
units, and subtracts the published discount. Variants retain `?variant_id=ID`,
language, condition, and finish. Availability combines status with quantity or
unlimited stock: `available` with zero quantity still means sold out.

Queries are serialized within each process, spaced four seconds apart, and keep
the existing retries and source coordination. Errors, including HTTP 429, are
propagated so an incomplete search is not cached as a final result. Large
searches may take several minutes.

Reference contracts: [Jumpseller Liquid](https://jumpseller.com/support/liquid/)
and [Product Search](https://jumpseller.com/support/search/). The public API
requires storefront headers (`X-Requested-With` and `Referer`). Requests without
them returned HTTP 403. Search uses this API; prices and stock are read from the
JSON data published on product pages.

Optional test:
`MUCHI_TEST_JUMPSELLER_LIVE=1 go test ./internal/stores/jumpseller -run TestLiveSolRing -v -timeout 90m`.
Fixtures for the four stores retain relevant public fragments for checking price
and stock without network access. Deployment to GCP is still required.

The deployment and HTTP task deadline was extended to 1800 seconds to allow full
pagination with pauses. The local asynchronous flow avoids the local API's HTTP
write limit for these searches. [Cloud Tasks supports HTTP deadlines up to 30
minutes](https://docs.cloud.google.com/tasks/docs/creating-http-target-tasks).

### Jumpseller Validation Status

Full HTML queries confirmed 17 available offers at Cartas La Fortaleza, 2 at
ChronoMagic, and 3 at GameQuest, all in CLP. The final version with AJAX search
confirmed the 2 from ChronoMagic again. Magic4Ever passed page 60 using product
IDs, but returned HTTP 429 while other stores were being tested simultaneously;
its full search was not validated. The latest AJAX queries to Cartas La Fortaleza
and GameQuest also received 429.

The final four-second pause and local serialization were added after that test.
Another full sweep against the active limit has not been repeated. Existing
Firestore coordination still operates per domain; local serialization does not
guarantee a global limit across instances. 429 errors remain visible to the
aggregator and prevent incomplete aggregate results from being cached.
