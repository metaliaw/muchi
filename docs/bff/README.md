[English](README.md) · [Español](README.es.md)

# BFF Entry Points

The BFF has two jobs: keep the API credential on the server, and translate the
API's contract into the public shapes the Frontend needs. Each door below has a
reason to exist. The API's own entry points and queue are documented in its
[architecture guide](https://github.com/cangrejometralleta/muchi-api/blob/main/docs/architecture.md).

## Serve the Application

- [`GET /{path}` and `/assets/*`](spa.md) serves built files and Vue history
  routes from the same origin as the BFF.
- [`GET /api/health`](health.md) gives Cloud Run and operators a cheap liveness
  signal without calling the API.
- [`GET /ads.txt`](ads.md) publishes the authorized AdSense seller record only
  when an AdSense publisher is configured.

## Describe the Application

- [`GET /api/config`](config.md) gives the Frontend public runtime settings,
  limits, links and feature switches without exposing credentials.
- [`GET /api/muchi`](muchi.md) serves the curated phrase and help catalogue the
  interface presents.
- [`GET /api/sources`](sources.md) reports which offer sources the current
  search composition can use.
- [`GET /api/languages`](languages.md) exposes the translator's supported
  languages.
- [`GET /api/supported-games`](supported-games.md) exposes games offered by
  configured stores.

## Resolve Card Input

- [`POST /api/decklist`](decklist.md) previews how pasted text parses before a
  paid or slow search is created.
- [`GET /api/card`](card-names.md) translates an entered card name.
- [`GET /api/card/suggestions`](card-suggestions.md) corrects a name prefix.
- [`GET /api/card/autocomplete`](card-autocomplete.md) supplies game-specific
  catalog matches.
- [`GET /api/card/metadata`](card-metadata.md) resolves printing metadata.
- [`GET /api/card/art`](card-art.md) resolves the selected printing's artwork
  without making the browser depend on translator internals.

## Run and Present Searches

- [`POST /api/searches`](search-create.md) validates the complete list and
  creates an idempotent search.
- [`GET /api/searches/{id}`](search-read.md) combines state and a page of
  presented results for one polling cycle.
- [`GET /api/searches/{id}/stock`](stock.md) checks candidates through the
  configured services.
- [`POST /api/searches/{id}/stock`](stock-browser.md) combines browser
  observations with optional service checks.
- [`POST /api/searches/{id}/cancel`](search-cancel.md) cancels the search using
  its idempotency key.
- [`GET /api/searches/{id}/cart`](cart.md) returns a suggested allocation.
- [`POST /api/searches/{id}/cart`](cart-choice.md) totals the shopper's chosen
  quantities without replacing them with a new recommendation.
