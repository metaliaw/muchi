[English](optional-reverification-en.md) · [Español](optional-reverification.md)

# Why Stock Reverification Is Optional

September 2026 · `deploy.next` branch, expanded September 20

## The Load Chain

A Search for 50 Cards queries Sources once per Card. Up to that point,
Load grows with the List as expected.

Reverification (`verify_stock`) adds a second Pass: for every cheap Offer
found, the Service visits the Store again to confirm Stock remains
available. The chain becomes:

```text
1 Search = N Price queries + M Stock queries
           (one per Card)    (one per candidate Offer)
```

M has no Limit proportional to N: a common Card available cheaply at twenty
Stores generates twenty extra Visits. A Commander deck — a hundred Cards,
repeated Lands — is exactly the case that rechecks most.

## The Effect on the System

- Searches take longer to finish. The User watches a stalled Progress bar
  while the Service checks Stock that may no longer matter: the Cart only
  takes the cheapest per Store.
- Stores receive twice the Traffic. We are Guests in their Catalogs;
  doubling Visits for fleeting Certainty creates a Debt someone will notice.
- Certainty expires immediately. Stock confirmed at 21:00 can sell out at
  21:01. Reverification buys minutes of Certainty with minutes of Waiting.

## The Incident That Made It Visible

The Frontend fell back to *Retry submission* without Results. The Failure chain:

1. The BFF took too long — reverification inflates every Cycle — and ended
   with a 5xx.
2. The Client marked the Failure as retriable and retained the Submission.
3. Without `searchId`, Polling never started; only the Button remained.

Retrying is a symptom. Each Retry resends the whole Search; if Load caused
the Failure, it adds more Load. An overloaded System that keeps retrying
gets in its own way.

## The Decision

`MUCHI_VERIFY_STOCK` defaults to off. The Cart trusts the Price seen on the
first Pass, like any Comparator. Consistently, while the Flag is off, no
Offer declares Stock: without Certainty, it shows no Badge, neither
"In stock" nor "Unconfirmed".

Enable it with one Environment line (`MUCHI_VERIFY_STOCK=1`), without a
Deploy. It is reserved for times when Certainty matters more than Load,
such as a large overnight Sale where Stock changes every minute.

## The Pass We Do Not Pay For

Load was the argument. Nobody said Stock confirmation was unnecessary;
we said we could not pay for that second Pass within the Search cycle.

There is another path. A Shopify Store serves its Catalog at
`/products/<handle>.js` with `Access-Control-Allow-Origin: *`. The Buyer's
Browser reads it directly: it does not use our IP, does not enter the Search
cycle, and happens once when someone presses the Button. Waiting, duplicate
Traffic and fleeting Certainty are addressed in different ways.

### How Far It Reaches

Measured against Offers saved in `data/precios.db`, the Browser reaches
one in five. The remainder still depend on the Service:

| Platform | Offers | From the Browser |
| --- | --- | --- |
| Shopify | ~1,880 | Yes — public JSON with open CORS |
| WooCommerce | ~2,440 | No — Store API responds without `Allow-Origin` |
| scry.cl (Marketplace) | ~2,050 | No — `consultar_stock` requires a CSRF cookie |
| Jumpseller | ~580 | No — no public Product JSON |
| Custom | ~2,010 | No |

WooCommerce is the most frustrating: the Data is served and complete;
only a Header outside our control is missing. A known Store adding it
would unlock more Offers than all the Shopify coverage gained so far.

### The Boundary

The Frontend investigates; the Server awards the Crown.

```text
Browser → /products/<handle>.js          (reachable Stores)
        → POST /api/searches/{id}/stock {"checks": [...]}
Server  → asks the Service               (only for what is missing)
        → crown_checked_offers          (the Crown, as always)
```

The Browser asks about what will be Bought: Offers with selected Copies,
initially placed where allocation recommends them. If nothing is selected,
it asks about the cheapest Offers, the ones someone would buy anyway.
It proceeds cheapest first and stops at the first yes: everything further
up costs more. `browser_check_limit` — currently 5 — caps each Card, so a
hundred-Card List makes a hundred Queries when the cheapest has Stock,
and five hundred only when none respond.

`answer_stock` takes known Results and starts Rounds there. An Offer the
Browser confirmed is not checked elsewhere; an Offer absent from the Plan
is discarded — the Browser reports on this Search, not the entire Catalog.
A Confirmation still counts beyond `stock_check_limit`, which limits our
Visits, rather than eligibility for the Crown. `server/presenter.py` still
decides who gets the cheapest marker under the same rules as
[The Stock Request](api/stock-order-en.md). That Decision stays on the
Server; the Frontend does not recommend.

`GET` still exists and follows the same path without prior Knowledge.

### Data Is Saved with Its Timestamp

Confirmation survives a Reload with the Time the Store supplied it.
It returns only while fresh under `stock_fresh_seconds`, currently 600 in
[`config/offers.defaults.yaml`](../config/offers.defaults.yaml).
After that Limit, nothing returns and the Button reappears.

Ten minutes covers building a Cart and returning within a Shopping session;
nobody carries today's Certainty into tomorrow. The displayed Age is that
of the oldest Confirmation. Showing the newest would imply Freshness that
half the Rows lack.

The latest Search is saved to avoid visiting Stores again, because
yesterday's Price still informs. Stock is saved to tell us when to ask
again. Keeping it without a Timestamp would preserve an "available" that
ages into a lie.

## What Remains

- `MUCHI_VERIFY_STOCK` remains off for the same reasons. This confirms
  afterward, on request; it does not reverify during the Search.
- Retry retains its Idempotency key, so resubmission does not duplicate
  Searches. That behavior stays intact.
- The Network error now explains what happened ("The request did not reach
  the service") instead of showing a silent `TypeError`.
- The Frontend detects changes in each Cycle (`stateChanges`), leaving a
  path toward partial Deltas without requiring anything new from the Network.
- Still open: measure how often cheap Shopify Offers actually change Stock.
  Ten minutes is a Judgment, not a Measurement.
- `cartasmagicsur.cl` still cannot be checked: Vercel returns `429` to
  non-Browser Clients and provides no `Allow-Origin`, so the Buyer's Browser
  cannot reach it either. Only another Store's Confirmation displaces it.
