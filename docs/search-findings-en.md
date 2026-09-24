[English](search-findings-en.md) · [Español](search-findings.md)

# Search Findings

Muchi displayed a List of Offers and crowned the cheapest. That worked
while every Search returned one Card and its Printings, the only thing
the API provided.

When the API learned `match=includes`, a Search began returning different
Cards: "Kuriboh" also finds Winged Kuriboh, Linkuriboh and Token: Kuriboh.
Four unwritten Assumptions fell apart; until then, they had been true.

The fifth Finding was discovered along the way.

Defects across the Boundary — in the Sources doing the searching — live in
`docs/search-findings.md` in the
[muchi-api Repository](https://github.com/cangrejometralleta/muchi-api).
Each Repository keeps its own Findings: copying the other Document would
age like the copied Contract did, which is precisely F5.

## Where Each Decision Lives

`server/presenter.py` decides, and `web/src/search.js` is its Twin across
the Boundary. The BFF sends one Page per Cycle and the Frontend accumulates
them, so the same Decision is written twice. Four of these five Findings
touched both Twins.

```text
API response            card_key · faults · offers
        │
        ▼
server/presenter.py      groups by Card type, crowns, builds the Cart
        │  one Page per Cycle
        ▼
web/src/search.js        accumulates Pages and rebuilds the Summary
        │
        ▼
components/OfferList     renders, does not decide
```

## The Findings

| ID | Finding | Status |
| --- | --- | --- |
| F1 | One "cheapest" for the entire Page | Closed `a1bc9d8` |
| F2 | The Cart bought the wrong Card | Closed `a1bc9d8` |
| F3 | Grouping by Text splits one Card into several | Closed `57e571e` |
| F4 | Nobody knew a Store was missing | Closed `b972a65` |
| F5 | The copied Contract was 130 lines behind | Closed `03a4d92` |

### F1. One "Cheapest" for the Entire Page

`order_offers` said it in its Docstring: "Opens by Price, cheapest first,
mixing all Cards." And `pick_cheapest` crowned one Offer for the whole Response.

```text
 100 CLP  Linkuriboh        🐾 cheapest
 300 CLP  Winged Kuriboh
 900 CLP  Kuriboh           ← what was requested
```

Comparing Linkuriboh's Price with Kuriboh's says nothing: they are different
Cards. Each Card type crowns its own Offer, and the View separates types
with Headings when more than one appears.

Card type depends on Mode. In `exact`, the type is the requested Card:
its Printings are the same Card and compete. In `includes`, each Title is
a type. This is one Rule expressed in Domain language.

### F2. The Cart Bought the Wrong Card

`build_cart` grouped by the requested Name and admitted every Offer for the
Item. With derivatives present, the Optimizer bought the cheaper derivative.

```text
before the fix  total 4700   1× Winged Kuriboh LV9   ✗
after the fix   total 5000   1× Kuriboh              ✓
```

Derivatives are for browsing, not buying three copies of. In `includes`,
the Cart returns to the requested Card by filtering with `names_same_card`,
the Twin of `offer.MatchesCard`.

That is a third Twin and the only one crossing the API boundary: it repeats
a Rule the API already applies. It exists because the Cart must narrow
what a broad Search returned.

### F3. Grouping by Text Splits One Card into Several

`read_card_type` lowercased the Title and used it as the Card identity.
The Frontend used Text to answer a Question only the API could answer,
because the API knows the Matching rules.

```text
Winged Kuriboh                                     ┐
LDS3-EN100 “Winged Kuriboh” Common Effect Monster  ├→ three Groups
Winged Kuriboh (PUR)                               ┘
```

Each Source writes Titles differently. Grouping by that Text split a Card
into as many Groups as spellings, each with its own "cheapest" — effectively none.

The API now sends `card_key` and answers once. The Frontend reads it and
falls back to the Title when empty, which is what an older API version
returns: both sides cannot always be deployed together.

### F4. Nobody Knew a Store Was Missing

The API already reported failed Sources, but the Frontend ignored them.
A Search missing a Store arrived as `found` with its Offers and looked complete.

```text
⚠ Kuriboh: could not query v3.netdecker.cl; its offers are missing.
```

The View did not change. `notices` already existed and rendered with its
Notice styling; only its contents were missing. That is the best possible
outcome for a change like this.

The Frontend keeps the Source name and discards the Reason. The Reason
includes the Response body; a Store under maintenance returns an HTML page.
That belongs in the Log. Searchers only need to know this Price was compared
against one fewer Store. The Test requires that `<!doctype` never appear
in visible Text.

### F5. The Copied Contract Was 130 Lines Behind

`docs/api/openapi.yaml` stated on its first line that it was a Copy synced
from `muchi-api`. It lacked `/supported-games`, Results pagination, `match`,
`card_key`, `faults` and `SourceFault`, and still required `stores_only`
after that field had disappeared.

A Copy aging silently misleads its Reader into believing it is the Contract.
First, the full source was copied. Then `muchi-api` became public and the
Copy was deleted along with the Bruno copy. Now the Contract is linked
where it lives, a synchronization that cannot be forgotten.

This Document exists for the same reason in reverse: API Findings are not
copied here.

## What Remains Open

1. The Frontend has no JS test runner. `web/src/search.js` is the Twin of
   `server/presenter.py`, but only Python is tested. Nothing prevents the
   Twins from drifting; three Copies of the Matching rule drifted that
   way in the API without a Test noticing.
2. `deploy.sh` runs neither `pytest` nor the Frontend build. Both must be
   invoked manually before deployment.
3. No View was inspected in a Browser. The Mode selector, Group headings
   and failed-Source Notice were verified through API requests against
   both deployed Services and through Tests, without viewing the Page.
4. The Cart repeats an API rule (`names_same_card`). If Matching changes
   there but not here, the Cart silently starts buying differently.

## What This Round Taught Us about Method

- Change the Twins together or separate them. Every Finding touching
  `presenter.py` also touched `search.js` in the same round.
- Test through Mutation. Every Fix has a case that fails if reverted:
  without the cheapest Offer per type, Linkuriboh wins; without the Cart
  filter, Linkuriboh is purchased.
- Deploy the Consumer before the Contract. When removing `stores_only`,
  the Frontend stopped sending it and was deployed first. The API uses
  `DisallowUnknownFields`; reversing the order would return `400` for
  every Search.
