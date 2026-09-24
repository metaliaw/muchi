# Plan: Prove the Store Boundary with a Second Adapter

[English](provider-agnostic-store-plan.md) | [Español](provider-agnostic-store-plan.es.md)

This document describes work **that has not been done**. It is a plan to prove
that the boundary described in [Architecture](architecture.md#the-store-boundary)
survives a provider change. Today the boundary is a well-formed intention: the
ports exist, but only one adapter implements them, and nobody has checked
whether another one can.

## The Problem It Solves

Muchi API does not run from a cold start on a clean machine. Starting the project
requires a Google Cloud project with Firestore and Cloud Tasks because the only
adapters talk to those services. This raises the cost of three separate tasks:

- **Contributing.** Someone fixing a store parser needs cloud credentials before
  running the first test.
- **Testing.** Storage integration tests are skipped unless an emulator is
  available.
- **Changing providers.** Nobody knows the cost until they try.

The goal is not to abandon Firestore. It is to keep using it as a choice instead
of letting omission turn it into a foregone conclusion.

## What to Build

### One. A Shared Contract Test

A single test suite should accept a `Vault` and check the rules the domain
assumes, without knowing the provider:

- A created search can be retrieved with the items it declared.
- Two creations with the same idempotency key return the same search; a repeated
  key with a different body returns a conflict.
- A claim returns at most one item per turn, and a lease prevents two owners
  from claiming the same one.
- An expired lease becomes available again.
- Cursor pagination neither repeats nor skips items when results arrive between
  pages.
- An offer saved with a positive TTL can be read; an expired offer cannot.
- The pending-work counter respects its limit.

This suite runs today against the Firestore adapter with an emulator, and later
against any other adapter. It makes the boundary verifiable and should be
written **first**: without it, a second adapter proves nothing.

### Two. An In-Memory Adapter

Implement `Vault` and `Dispatcher` over in-memory structures, with its own sweep
for expiration. It serves three purposes:

- Run the complete suite without cloud services or an emulator.
- Start the local API with `MUCHI_STORE=memory` to work on parsers and the HTTP
  contract.
- Become the second subject of the contract tests, which is the actual proof
  that the ports do not leak Firestore assumptions.

It is never deployed. An in-memory adapter that nobody uses will rot, so it
should be part of the default local development path, not left in a corner.

### Three. Choose the Store during Composition

`BuildRuntime` selects the adapter from configuration and remains the only place
in the code that names a provider. The rest of the program already receives
ports and does not change.

## Questions to Resolve Along the Way

**Expiration.** This is the deepest assumption and the only one missing from all
signatures. Firestore deletes data by TTL policy; the code rejects expired
records because deletion is eventual. An adapter without native TTL must sweep
records itself. That sweep is part of the contract even though no interface
declares it. The contract test must require it explicitly.

**Transactions.** Item claims, idempotency, and cancellation rely on
transactions. A store without them—a cache or file—cannot implement `Vault`
without silently becoming incorrect. The contract should fail loudly rather
than accept a weak implementation.

**Dispatch.** `Dispatcher` is simpler: in memory, it can be a goroutine with a
queue. Cloud Tasks retry with backoff, which cannot be emulated, so the local
equivalent must be clear about what it does not do.

## Suggested Order

1. Run the contract suite against the current adapter with the Firestore
   emulator.
2. Build the in-memory adapter until it passes the same contract.
3. Select the adapter by configuration and enable local tests without cloud.
4. Only then, if a real candidate provider appears, use the contract to test a
   third adapter.

## What This Plan Does Not Propose

It does not propose a new abstraction layer or an ORM. The ports already exist
and the domain is clean; what is missing is evidence, not architecture.
