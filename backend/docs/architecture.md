# Muchi API Architecture

**English** | [Español](architecture.es.md)

Muchi is free software across two repositories. Search collection, persistence,
and processing live here in
[cangrejometralleta/muchi-api](https://github.com/cangrejometralleta/muchi-api),
alongside the [OpenAPI contract](../openapi.yaml). The interface, its
presentation rules, and the BFF that consumes that contract live in
[metaliaw/muchi](https://github.com/metaliaw/muchi); its
[architecture](https://github.com/metaliaw/muchi/blob/main/docs/architecture.es.md)
is documented there.

This document covers the API side: HTTP entry points, queue, worker, persistence,
and expiration. Diagrams are not copied between the repositories; a copy grows
stale without anyone noticing.

The three executable doors are documented separately in the
[entry-point guide](entrypoints.md): `ServeAPI`, `ProcessSearch`, and
`SweepQueue`. The [sweeper guide](queue/sweeper.md) explains why task wake-ups can be
spent, how bounded reconciliation restores them, and what the sweeper does not
own.

## Overview

Muchi API separates receiving searches from the slower work of querying sources.
The API persists each request and responds without waiting for results. Cloud
Tasks wakes private workers that claim one card at a time. A periodic sweeper
restores execution opportunities when pending work and task wakeups fall out of
sync. It counts ready work and adds wake-ups; it does not process items or
delete expired data.

```mermaid
flowchart TB
    user[Web Client or Integration]

    subgraph edge[HTTP Entry]
        api[ServeAPI<br/>Public Cloud Function]
        auth[Bearer Token<br/>Idempotency Key]
        health[Public Health]
      cards[Game, Metadata,<br/>and Offer Queries]
      purchase[Checkout, Quote,<br/>and Order Routes]
        hook[WooCommerce Webhook<br/>HMAC Signature]
        auth --> api
        health --> api
      cards --> api
      purchase --> api
        hook --> api
    end

    subgraph control[Control Plane]
        tasks[Cloud Tasks<br/>1 dispatch per second<br/>2 concurrent]
        scheduler[Cloud Scheduler<br/>every 5 minutes]
        sweeper[SweepQueue<br/>bounded reconciliation]
        scheduler -->|OIDC| sweeper
        sweeper -->|restores wakeups| tasks
        hourly[Cloud Scheduler<br/>hourly]
        releaser[ReleaseOrders<br/>frees stale pending orders]
        hourly -->|OIDC| releaser
    end

    subgraph work[Work Plane]
        worker[ProcessSearch<br/>Private Cloud Function]
        claim[Lease Claim<br/>and Transactional Cleanup]
      search[Game Search Service]
      compose[Providers + SourcesByGame<br/>from stores.yaml]
        worker --> claim --> search
      compose --> search
    end

    subgraph data[Persistence]
        firestore[(Firestore)]
        searches[(searches)]
        items[(items)]
        offers[(item_offers)]
        cache[(offer_cache)]
        healthStore[(source_health)]
        ordersStore[(orders)]
        firestore --- searches
        firestore --- items
        firestore --- offers
        firestore --- cache
        firestore --- healthStore
        firestore --- ordersStore
    end

    subgraph providers[External Sources]
      aggregators[scry.cl and TCGMatch<br/>Aggregators]
      stores[WooCommerce, Shopify,<br/>Jumpseller, and PrestaShop]
        lists[Moxfield Lists]
    end

    subgraph security[Identity and Secrets]
        secret[Secret Manager<br/>muchi-api-token latest]
        taskIdentity[Cloud Tasks Account]
        apiIdentity[API Account]
        workerIdentity[Worker Account]
    end

    user -->|HTTPS| auth
    user -->|GET health| health
    api -->|stores Search and Items| firestore
    api -->|one wakeup per Card| tasks
    api -->|reads Status and Results| firestore
    tasks -->|POST with OIDC| worker
    claim <-->|transaction and lease| firestore
    search <-->|cache and health| firestore
    search --> aggregators
    search --> stores
    search --> lists
    secret -.-> api
    secret -.-> worker
    taskIdentity -.-> tasks
    taskIdentity -.-> worker
    apiIdentity -.-> api
    api -->|cart, quote, bacs checkout| stores
    stores -.->|order.updated| hook
    api -->|orders: create, confirm, release| firestore
    releaser -->|pending to released| firestore
    workerIdentity -.-> worker
```

## Responsibilities

| Component | Responsibility | Trust Boundary |
| --- | --- | --- |
| `ServeAPI` | Validates the contract, authentication, and idempotency; persists and dispatches. | Public entry; business routes require a Bearer Token. |
| Cloud Tasks | Delivers wakeups with rate control and retries. | Invokes the worker through OIDC identity. |
| `ProcessSearch` | Claims a card, queries offers, checks stock, and saves the result. | Private function; anonymous invocation is not allowed. |
| `SweepQueue` | Counts ready work and restores lost wakeups up to a limit. | Private function invoked by Cloud Scheduler. |
| `ReleaseOrders` | Moves `pending` orders past their TTL to `released`, up to a limit per run. | Private function invoked hourly by Cloud Scheduler. |
| Firestore | Stores searches, items, offers, orders, idempotency, cache, and health. | Access is limited to authorized service accounts. |
| Secret Manager | Provides the current token version when an instance starts. | The secret is never included in images or responses. |
| Sources | Provide catalogs and availability through independent formats. | Untrusted external network; uses timeouts, retries, and body limits. |

## Search Flow

1. The client sends a list and an idempotency key.
2. The API validates limits and creates the search and its items in Firestore.
3. The API adds one Cloud Tasks wakeup per card and responds with `202`.
4. Cloud Tasks invokes the private worker using OIDC.
5. The worker claims the next available item in a transaction.
6. The service resolves the game's aggregator and stores from
   `config/stores.yaml`; it queries up to four stores concurrently and combines
   all offers.
7. The worker saves offers and progress in persisted state.
8. The client polls status and results until the search reaches a terminal state.

The task carries a signal, not an item ID. This lets competing consumers and
lease recovery make progress, but requires the claim to remove dead work so FIFO
progress is preserved. The [orphaned items incident](queue/orphaned-queue-items.md)
explains this invariant.

## Purchase Flow

Purchases are in progress: WooCommerce is the only platform that reaches a
real order, and no order has run against a live store yet. The
[agentless checkout guide](checkout/agentless-checkout.md) holds the detail
and the manual pilot checklist.

```mermaid
sequenceDiagram
    participant C as Client
    participant A as ServeAPI
    participant S as search.Service
    participant W as WooCommerce Store
    participant F as Firestore
    participant R as ReleaseOrders

    C->>A: POST /searches/{id}/checkout
    A->>S: CheckoutLinks
    S-->>C: per-store cart link or product pages
    C->>A: POST /searches/{id}/checkout with shipping
    S->>W: Store API cart, no order
    W-->>C: quote with shipping and payment methods
    C->>A: POST /searches/{id}/orders + Idempotency-Key
    S->>W: fill cart, checkout by bacs
    W-->>S: store order id
    S->>F: CreateOrder, status pending
    A-->>C: 201 Order
    W->>A: order.updated webhook, HMAC signed
    A->>F: pending to confirmed or released
    R->>F: pending past TTL to released
    C->>A: GET /searches/{id}/orders/{order_id}
    A-->>C: current status
```

```mermaid
stateDiagram-v2
    [*] --> pending: PlaceOrder
    pending --> confirmed: webhook processing or completed
    pending --> released: webhook cancelled, failed, refunded
    pending --> released: ReleaseOrders after TTL
    [*] --> linked: buyer follows the store's cart link
    linked --> reported: buyer gives the store's order number
    reported --> [*]
    confirmed --> [*]
    released --> [*]
```

Every move names the status it expects to leave, so a late webhook loses to a
release that landed first, and the other way around.

## Code Layers

```mermaid
flowchart LR
    function[function.go<br/>Cloud Function Entry Points]
    transport[internal/httpapi<br/>Handlers and HTTP DTOs]
    application[internal/application<br/>Composition]
    catalog[internal/catalog<br/>Source Construction]
    domain[internal/search and model<br/>Search Service]
    orderer[internal/stores<br/>Orderer and woocommerce.Client]
    releaser[internal/orders<br/>Releaser]
    repository[SearchRepository<br/>Persistence Interface]
    repositories[internal/repositories<br/>Repository Implementations]
    datasources[internal/datasources<br/>Generic Database Contract]
    database[internal/db<br/>Firebase Adapter]
    worker[search.Worker<br/>Background Work]
    config[config/stores.yaml<br/>Games, Origins, and Stores]
    adapters[Adapters<br/>Firestore, Tasks, and Sources]

    function --> transport
    function --> application
    transport --> domain
    worker --> domain
    domain --> repository
    repository --> repositories
    repositories --> datasources
    datasources --> database
    application --> database
    application --> repositories
    config --> catalog
    catalog --> application
    application --> domain
    application --> adapters
    domain -->|OrderPlacer| orderer
    function --> releaser
    releaser -->|Expirer| repositories
    adapters --> domain
```

The HTTP handlers decode request DTOs and pass business inputs to the search
service. The worker calls the same service without HTTP. The service reaches
persistence through `SearchRepository`, implemented by `internal/repositories`.
Repositories use the provider-neutral `datasources.Database` contract; the
Firebase-backed `internal/db` package supplies its current implementation.

`internal/model` groups the domain types for offers, carts, searches, and source
health. These types carry no JSON or Firestore tags. `internal/httpapi` owns
request and response DTOs; `internal/db` owns database records and stored
JSON payloads. `internal/application` selects the Firebase-backed adapter and
passes it to repositories through `datasources.Database`. The Firebase SDK
stays inside `internal/db`. Explicit mappings at each boundary preserve the API and persisted
field names independently. Search idempotency uses its own stable fingerprint.

The domain also declares needs such as `TaskQueue`, `Provider`,
`OfferSource`, `StockChecker`, and `OfferCache`. `internal/catalog` builds the
aggregators in `Providers` and groups stores in `SourcesByGame` from the games,
origins, platforms, and enabled states in YAML. It also gathers print and set
catalogs, plus the product-kind limits. `internal/application` opens connections
and assembles the `Runtime` from those parts. Provider
packages implement the ports, so search rules do not import Firestore, Cloud
Tasks, or concrete HTTP client types.

Store configuration and stock checking live in `internal/stores`. Its
`jumpseller`, `shopify`, `prestashop`, `woocommerce`, and `moxfield` subpackages
contain the platform-specific clients. Offer aggregators live separately in
`internal/aggregators`.

The [game provider flow](search/game-search-providers.md) details this composition. The
[card identity guide](search/card-identity-games-sets.md) separates searches,
printings, and commercial variants.

### The Store Boundary

`internal/application` is the composition root that selects the database
provider. Production packages outside it depend on repository ports and do not
import `internal/db` or `internal/taskqueue`; the `Runtime` exposes ports, not
concrete types.

- `datasources.Database` is the provider-neutral contract implemented by the
  Firebase adapter. `internal/repositories` implements search and worker ports
  through that contract.
- `Vault` gathers the five roles currently served by one repository: saving searches,
  caching offers, reporting its health, counting waiting work, and pacing
  traffic to each source. They are listed separately because they are
  independent; one repository satisfying all five is a composition coincidence,
  not a domain assumption.
- `Dispatcher` gathers work dispatch and the wakeup that restores the sweeper.
- `Teller` is optional: a store with something to say receives a logger, while
  one that stays quiet simply does not implement it.

The assertions in `internal/application/vault_test.go` hold this boundary at
compile time.

One assumption is absent from the interfaces and should be explicit: **expiration
is delegated to the provider**. Firestore applies TTL to `expires_at` according
to `firestore.indexes.json`; the code rejects expired records because deletion
is eventual. A store without native TTL must sweep records itself. That is the
costliest piece to port, not the queries.

The [provider-agnostic storage plan](stores/provider-agnostic-store-plan.md) describes how to
test this boundary with a second adapter.

## Availability and Recovery

- Leases recover an item when a worker dies mid-task.
- Cloud Tasks retries transient failures and limits pressure on sources.
- Caching reduces repeated queries and keeps positive and negative TTLs separate.
- The sweeper reconciles ready work with wakeups, with a limit of 50 per cycle.
- Claims transactionally remove orphans, with at most 20 discards and one final
  claim per turn.
- Firestore applies TTL to searches, items, offers, idempotency, and cache; the
  code also rejects expired records because deletion is eventual.

## Security

- The API function is publicly reachable, but authenticates business routes with
  a constant-time Bearer Token comparison.
- The basic health endpoint remains public for operations and load balancing.
- The worker and sweeper require authenticated invocation.
- Cloud Tasks and Cloud Scheduler use dedicated accounts and OIDC tokens whose
  audience matches the destination URL.
- The API and worker use separate accounts with minimum permissions for
  Firestore, Cloud Tasks, and Secret Manager.
- `muchi-api-token:latest` allows rotation without storing the value in files,
  deployment arguments, or images.

## Deployment

The topology is created in dependency order:

1. `deploy-infra.sh`: services, identities, IAM, Firestore, TTL, and queue.
2. `deploy-worker.sh`: private worker and OIDC invocation permission.
3. `deploy-sweeper.sh`: private sweeper and scheduled invocation.
4. `deploy-api.sh`: public API connected to the actual worker URL.

`deploy-order-release.sh` deploys `ReleaseOrders` and its hourly job.
`deploy-infra.sh` also creates the `orders` index its query needs.

`deploy.sh` orchestrates all five steps, the order release after the sweeper. Each component can be deployed
separately to reduce time and change surface when fixing an issue.
