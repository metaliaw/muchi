[English](repository-evolution.md) · [Español](repository-evolution.es.md)

# Repository Structure Evolution

Muchi changed how it distributes code and opens it to the public. The history
has three stages; dates are omitted because the available documentation
records the decisions, but not when they happened.

## Two Repositories, One Private

At first, Muchi lived in two separate repositories. One was private to protect
business logic if that ever became necessary. That was the reason for keeping
it private; we found no list of rules or modules intended to remain private.

## Two Public Repositories

Later, both repositories became public. The interface and its BFF lived in
`metaliaw/muchi`; the API, its workers and persistence lived in the other
repository. Each side had a visible place for its contracts and
responsibilities. The API documentation explained opening the code as a way
to share how Muchi works and build it with the people who use it.

## One Repository for the Work

The API's code and documentation now live here with the interface and BFF.
The former API repository was deprecated. The API and BFF may deploy as
separate services, but their code has one source in this repository.

## Reasons We Know

- **Protect business logic:** this was the reason to keep one repository
  private at first, according to the project's recollection. This reason is
  recorded from the user's clarification.
- **Share and build with the community:** the project documentation recorded
  this reason for making the code public.
- **Separate responsibilities:** the earlier design assigned the experience
  and BFF to this repository, and search, workers and persistence to the API
  repository. The code is consolidated here; runtime roles may remain
  separate.
- **Work from one repository:** the separate repository was deprecated so
  future work, contracts and documentation have one source here.

The review searched the available documentation for references to privacy,
visibility, business logic and repositories. Mentions of "private" functions
and `chaos/` describe service access controls and sensitive data, not the
historical decision to make the repository private.

The current structure is documented in:

- [API Integration](api/README.md) — client, settings and boundary.
- [Muchi Architecture](architecture.md) — request flow, security and deploy.
- [Search Findings](search-findings.md) — decisions at the boundary.

## What Changed

| Stage | Visible code | Source of work |
| --- | --- | --- |
| Two repositories, one private | Partial | Split |
| Two public repositories | Complete in both | Split between the two |
| One active repository | Consolidated in `muchi` | This repository |

The history moves from a private boundary to a public one, then to a single
source of code, documentation and future work. The API and BFF can still run
as separate services.
