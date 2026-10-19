# Memory

Index of the English docs. Spanish versions sit beside each one as `.es.md`.
Work not yet distilled into them lives in [STORY.md](STORY.md).

## Overview

- [Muchi API Architecture](docs/architecture.md)
- [API Entry Points](docs/entrypoints.md)

## Checkout

- [Cart and Reservation Data Model](docs/checkout/cart-and-reservation-data-model.md)
- [Agentic Commerce Protocols](docs/checkout/agentic-commerce-protocols.md)
- [Checkout Without Agents](docs/checkout/agentless-checkout.md)
- [AP2 and ACP: Fit Analysis for Muchi](docs/checkout/ap2-acp-fit-analysis.md)
- [Web Agent Purchases 🐈](docs/checkout/web-agent-purchases-proposal.md)

## Paired Repositories

These repositories are commonly worked on locally together. Paths are
relative to this repository:

- `.` — `cangrejometralleta/muchi-api`, API contract and Firestore data (this
  repository).
- `../../metaliaw/muchi` — browser interface and BFF.
- `../OneTwoThree` — the manifesto: Rules, Values and
  Patterns that govern how this repository is read and written. A Pattern or
  Rule that belongs to every project is written there, not here.

For changes across their boundary, inspect and update both repositories using
these paths. Do not ask the user for their locations again when both checkouts
are present.

## Entrypoints

- [ProcessSearch Turns a Wake-up into One Claimed Unit](docs/entrypoints/process-search.md)
- [ServeAPI Exposes the Search Contract](docs/entrypoints/serve-api.md)

## Operations

- [Deployment with Cats](docs/operations/deployment.md)
- [Deployment, Revisions, and Secret Rotation](docs/operations/deployment-revisions-secrets.md)
- [Google Cloud Startup Credits](docs/operations/google-cloud-startup-credits.md)

## Queue

- [Orphaned Queue Items Incident](docs/queue/orphaned-queue-items.md)
- [The Sweeper Replaces Spent Queue Wake-ups](docs/queue/sweeper.md)

## Search

- [Card Identity Across Games and Sets](docs/search/card-identity-games-sets.md)
- [Cursor Pagination and Consistency](docs/search/cursor-pagination-consistency.md)
- [Game Search, Aggregators, and Stores](docs/search/game-search-providers.md)
- [Sealed Products](docs/search/sealed-products.md)
- [Search Findings](docs/search/search-findings.md)
- [Source Pacing](docs/search/source-pacing.md)

## Stores

- [Candidate Stores from Sol Ring](docs/stores/candidate-stores-sol-ring.md)
- [Plan: Prove the Store Boundary with a Second Adapter](docs/stores/provider-agnostic-store-plan.md)
- [A Store Down for Eight Days](docs/stores/store-down-eight-days.md)
