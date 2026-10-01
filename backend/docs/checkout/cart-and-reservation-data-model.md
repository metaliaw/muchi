# Cart and Reservation Data Model

**English** | [Español](cart-and-reservation-data-model.es.md)

This document describes the offer lookup added to the API and the proposed
Firestore shape for carts and stock reservations. The offer lookup is
implemented; cart persistence and stock reservation are not.

## Offer Lookup

`offer_cache/{hash(query_key)}` remains the shared, query-keyed cache. Saving a
non-empty offer list also writes each offer snapshot to
`offers/{hash(offer_id)}`. The document payload retains the source's stable
`offer_id` and expires with that cache write. A later cache write containing
the same offer refreshes its lookup snapshot. Cache reads backfill missing offer
index documents, so entries written before this index was introduced can still
be looked up while their query cache remains live.

`GET /v1/offers/{offer_id}` returns that latest indexed snapshot, or `404` when
the ID was never indexed or its snapshot expired. It does not call the source
and does not verify current stock; use the search stock route for a live check.
The endpoint requires the API bearer token.

The offer ID is a logical reference, while the Firestore document ID is its
SHA-256 hash. This keeps source IDs with punctuation safe as Firestore paths.
The payload ID is checked after lookup.

## Proposed Cart Shape

The front end may keep its cart snapshot, but the API owns expiry and checkout
validation. If carts need server persistence, use:

```text
carts/{cart_id}
  status: active | expired | checked_out
  created_at: timestamp
  updated_at: timestamp
  expires_at: timestamp

carts/{cart_id}/lines/{line_id}
  offer_id: string
  quantity: integer
  offer_snapshot: object
  status: current | stale | unavailable
  checked_at: timestamp
```

Adding a line does not hold stock. The API checks `expires_at` when reading,
updating, or checking out; Firestore TTL is only cleanup after logical expiry.
On refresh, the API resolves the same `offer_id` and updates its snapshot. If it
cannot find that ID, it keeps the old snapshot and marks the line `stale`.
Unavailable stock is a separate state from a missing offer. Checkout rejects
the whole cart if any line is stale or unavailable, leaving removal or
replacement to the buyer.

## Proposed Stock and Reservation Shape

An inventory source records stock owned or controlled by Muchi. Each document
represents one offer at one final source; its stable ID can hash the offer ID
and source identity. One offer may have stock at multiple final sources:

```text
stock_sources/{source_id}
  offer_id: string
  final_source: { type: string, id: string }
  on_hand: integer
  reserved: integer
  revision: integer
  updated_at: timestamp
```

The available amount is `on_hand - reserved`. At cash checkout, a Firestore
transaction checks availability, increments `reserved`, and creates the order
and its allocations together. Each allocation identifies the offer, source,
and quantity it holds:

```text
orders/{order_id}/reservations/{allocation_id}
  offer_id: string
  source_id: string
  quantity: integer
  final_source_snapshot: { type: string, id: string }
```

If one offer is fulfilled from two sources, it has two allocation documents.
The buyer's cart does not select the source; the API chooses it from current
stock. Aggregate quantities for the same offer and source into one allocation.
Changes to `on_hand` and `reserved` must be transactional so concurrent carts
cannot reserve the same units.

The post-payment expiry and release policy is undecided. Until that decision,
the schema must not invent an automatic reservation expiry. External stores
remain authoritative for their own stock and reservations.

## Collections and Expiry

| Collection | Purpose | Expiry |
| --- | --- | --- |
| `offer_cache` | Shared lists keyed by normalized search query | Existing cache TTL |
| `offers` | Offer snapshots indexed by hashed offer ID | Same TTL as the write that refreshed the snapshot |
| `item_offers` | Offers attached to a completed search item | Search expiry |
| `carts` | Proposed server-side cart state | Logical `expires_at`, then optional TTL cleanup |
| `stock_sources` | Proposed controlled inventory counts | No TTL |
| `orders` | Purchase lifecycle | Existing order status policy |
| `orders/{id}/reservations` | Proposed allocations held by a purchase | Released by an explicit order transition |

Enable Firestore TTL on the `offers` collection group's `expires_at` field.
The API checks expiry itself; TTL deletion is asynchronous.
