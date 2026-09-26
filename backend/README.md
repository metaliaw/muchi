# Muchi API

[English](README.md) | [Español](README.es.md)

A Go API that searches and compares TCG card offers from Chilean stores,
checks stock, and processes resumable search lists. It supports Magic: The
Gathering, Pokémon, and Yu-Gi-Oh!, with prices in Chilean pesos.

Our community believes in us; we believe in the community. This public
repository shares how Muchi works and invites people to build it with us.

Start with local development, read the [OpenAPI contract](openapi.yaml), or try
the [Bruno requests](collections/README.md). The guides below document the API
and development decisions.

## Development Documentation

### Overview

- [Architecture](docs/architecture.md)
- [API Entry Points](docs/entrypoints.md)

### Search

- [Game Search, Aggregators, and Stores](docs/search/game-search-providers.md)
- [Source Pacing](docs/search/source-pacing.md)
- [Search Findings](docs/search/search-findings.md)
- [Card Identity Across Games and Sets](docs/search/card-identity-games-sets.md)
- [Cursor Pagination and Consistency](docs/search/cursor-pagination-consistency.md)
- [Sealed Products](docs/search/sealed-products.md)

### Stores

- [Provider-Agnostic Store Plan](docs/stores/provider-agnostic-store-plan.md)
- [Candidate Stores from Sol Ring](docs/stores/candidate-stores-sol-ring.md)
- [A Store Down for Eight Days](docs/stores/store-down-eight-days.md)

### Queue

- [Queue Sweeper](docs/queue/sweeper.md)
- [Orphaned Queue Items](docs/queue/orphaned-queue-items.md)

### Checkout

- [Web Agent Purchases Proposal](docs/checkout/web-agent-purchases-proposal.md)
- [Agentic Commerce Protocols](docs/checkout/agentic-commerce-protocols.md)
- [Checkout Without Agents](docs/checkout/agentless-checkout.md)
- [AP2 and ACP: Fit Analysis for Muchi](docs/checkout/ap2-acp-fit-analysis.md)

### Operations

- [Google Cloud Startup Credits](docs/operations/google-cloud-startup-credits.md)
- [Deployment, Revisions, and Secret Rotation](docs/operations/deployment-revisions-secrets.md)
- [Deployment with Cats](docs/operations/deployment.md)

## Local Development

You need Docker Compose to run the API, worker, and local Firestore emulator;
this flow does not require a Google Cloud account. From the repository root, run:

```sh
docker compose up --build
```

The API is available at `http://localhost:8081`; the local token is
`local-development-token`. Prometheus metrics are at `/metrics`.

`./start.sh [serve|work]` uses Docker Compose to build and start the API
(`serve` by default) or worker (`work`), along with the Firestore emulator. The
services run in the background while the script follows their logs. Ctrl+C stops
following logs; `docker compose down` stops the services. This avoids the
Podman Compose attached startup path when reusing active containers. Use the
command above to start all services. `./build.sh` builds a local binary after
formatting, vetting, and testing. On Windows, `start.cmd` and `build.cmd` do the
same. Building outside Docker requires Go 1.26 or later.

```sh
curl -X POST http://localhost:8081/v1/searches \
  -H 'Authorization: Bearer local-development-token' \
  -H 'Idempotency-Key: demo-1' \
  -H 'Content-Type: application/json' \
  -d '{"cards":[{"name":"Sol Ring","quantity":1}],"options":{"verify_stock":true,"stores_only":true}}'
```

The response returns an `id`. Replace `SEARCH_ID` with that value to read
results while the worker processes the request:

```sh
curl http://localhost:8081/v1/searches/SEARCH_ID/results \
  -H 'Authorization: Bearer local-development-token'
```

Use `GET /v1/searches/SEARCH_ID` to check search status. The example token is
for local development only.

## Sources and Prices

Offer sources are [scry.cl](https://scry.cl) and the stores enabled in
`config/stores.yaml`: WooCommerce, Shopify, and Jumpseller catalogs, plus
inventories published through Moxfield. Direct connections to Magic4Ever,
Cartas La Fortaleza, ChronoMagic, and GameQuest are currently disabled because
their Jumpseller pagination is slow. scry.cl remains enabled and can still show its
published offers. `enabled: false` also prevents direct stock checks. To
reactivate a store, change that value in `config/stores.yaml` and restart the API
and worker.

Muchi reads published prices from scry.cl pages in CLP, along with the store and
variant. Stock verification queries configured stores; a published price does
not confirm stock. scry.cl's URL, enabled state, and community setting are configured
under `search_providers` in `config/stores.yaml`. The reader uses scry.cl's public
HTML and does not force a cache refresh.

The [game search flow](docs/search/game-search-providers.md) explains how YAML combines
aggregators and stores, selects each adapter, and handles timing and duplicates.
The [card identity guide](docs/search/card-identity-games-sets.md) explains
what identifies a search, a printing, and a commercial variant. The
[search findings](docs/search/search-findings.md) track defects at the seam
between a store's product title and the card inferred by the code. The
[sealed product guide](docs/search/sealed-products.md) explains box identity, source
selection, the `sealed` property, and why HTTP 429 is not a source failure. The
[eight-day store outage](docs/stores/store-down-eight-days.md) covers why a fixed
one-minute circuit breaker was insufficient and what remains open: `www.deckscards.cl`
is slow, not down.

The [source pacing policy](docs/search/source-pacing.md) describes what counts as a
failure, how long sources are paused, which honest responses never count as
failures, and why uncertainty should favor the caller.

Moxfield lists are resolved automatically through API v3 from each
`lists[].url`. To add a store, set `name`, `enabled: true`, and its lists with
`label`, `url`, and `clp_per_ck_usd`, then restart the service. Manual export and
import are not required.

Each list retains its published sets, finishes, and quantities. The price is
Card Kingdom USD multiplied by the list rate and rounded to the nearest peso
(half values round up). Foil uses `ck_foil`, and etched uses `ck_etched`; entries
without a matching price are omitted and logged. Quantity is stored in
`metadata.quantity`; stock remains unconfirmed until checked with the store.

The full inventory cache in Firestore shares `MUCHI_OFFER_CACHE_TTL_SECONDS`
(86400 seconds, one day by default) and refreshes on demand after expiration.
Per-card result caching can extend how long changes remain invisible by another
TTL period. A blocked or invalid Moxfield response is treated as an error, never
as an empty list. Other sources remain available.

To check the nine configured public lists:

```sh
MUCHI_TEST_MOXFIELD_LIVE=1 go test ./internal/stores/moxfield -run TestPublicLists -v
```

## Commands

- `muchi-api serve`: serves HTTP and metrics.
- `muchi-api work`: processes pending items with renewable leases.

Worker navigation: local `work` enters through
[`cmd/muchi-api/main.go`](cmd/muchi-api/main.go); Cloud Tasks enters through
[`ProcessSearch`](function.go), selected by
[`deploy-worker.sh`](deploy-worker.sh). Both build the worker in
[`internal/application/worker.go`](internal/application/worker.go) and run
[`internal/search/worker.go`](internal/search/worker.go), which calls the shared
search service.

Firestore retains searches and results for 24 hours. Offer caching keeps
independent positive and negative TTLs. Cloud Tasks invokes a private function
for each card, without a resident worker.

The [orphaned queue items incident](docs/queue/orphaned-queue-items.md) explains how
an active queue stopped progressing, the claim pattern that fixed it, and the
invariants needed to reproduce this architecture safely. The
[cursor pagination guide](docs/search/cursor-pagination-consistency.md) explains how
clients read incremental results while workers finish cards out of order.

## Google Cloud

The [architecture guide](docs/architecture.md) shows boundaries among HTTP
entry points, queue, workers, persistence, sources, identities, and secrets. The
[deployment, revisions, and secrets incident](docs/operations/deployment-revisions-secrets.md)
explains why a correct rotation must also move traffic and update every
consumer. The [provider-agnostic storage plan](docs/stores/provider-agnostic-store-plan.md)
describes the remaining work to run without a cloud provider and keep provider
replacement an explicit choice.

The public function uses the `ServeAPI` entry point. The private Cloud Tasks
function uses `ProcessSearch` and must reject unauthenticated invocations.

Required variables: `GOOGLE_CLOUD_PROJECT`, `MUCHI_API_TOKEN`,
`MUCHI_TASK_REGION`, `MUCHI_TASK_QUEUE`, `MUCHI_TASK_URL`, and
`MUCHI_TASK_SERVICE_ACCOUNT`.

Enable a TTL policy on `expires_at` for the `searches`, `items`, `item_offers`,
`idempotency`, and `offer_cache` collection groups. The code rejects expired
documents even if Firestore has not deleted them yet.

## Deployment

The [deployment guide](docs/operations/deployment.md) explains how `deploy.sh` orchestrates
the four components, what each script does, required permissions, shared
`config/deploy.env` options, and token handling.

```sh
./deploy.sh --project my-project
```

Deployments run from a machine authenticated with `gcloud auth login`; GitHub
does not deploy automatically.

## Contributing

Help by reporting an incorrect search, suggesting a store, or submitting a code
or documentation improvement. Open an
[issue](https://github.com/cangrejometralleta/muchi-api/issues) with the game,
card, store, and expected result so the issue can be reproduced. Do not include
tokens, credentials, or personal information.

To contribute code, fork the repository and open a pull request describing the
problem and how you checked the change. Run `./build.sh` or `build.cmd` to check
formatting, analysis, and tests. Tests against live stores are optional and make
external requests; enable them only when reviewing that integration.

## License

Muchi API is free software under the **GNU Affero General Public License v3.0 or
later**. You may use, read, modify, and redistribute it. The Affero adds one
condition to the GPL: anyone operating this API, or a modified version, as a
network service must offer its users the corresponding source code. Forks are
welcome; a closed fork operated elsewhere is not.

```text
Copyright (C) 2026 Muchi

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
```

The [Muchi Front](https://github.com/metaliaw/muchi) uses the same license.
Two repositories, one rule.
