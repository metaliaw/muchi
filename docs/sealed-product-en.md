[English](sealed-product-en.md) · [Español](sealed-product.md)

# Sealed Products

Muchi searched for single Cards. A Booster box seemed like the same thing
at another Price, but its Naming, Grouping and Image source are different.

The API learned `kind=sealed`. This Document explains what changed on this
side of the Boundary and, especially, what broke during Testing. Almost
none of those Failures belonged to sealed Products. They already existed,
hidden by the fact that a single Card has a Catalog to query and a Box does not.

Defects across the Boundary — in the Contract and Sources — live in
`docs/sealed-product.md` in the
[muchi-api Repository](https://github.com/cangrejometralleta/muchi-api).
Each Repository keeps its own Findings for the same reason described in
[Search Findings](search-findings-en.md): copied Documents age poorly.

## The Catalog Is Chosen

The Form has a "What to search" `fieldset` with two Options, alongside
the existing Match selector:

```text
Game:  [ Magic ▾ ]
What to search:  ( • Singles )  ( ○ Sealed products )
```

A Search contains only Cards or only Boxes. Mixing them would require
deciding what each line means; Text alone does not distinguish "Booster box"
from "Sol Ring".

`kind` travels in the URL with `match`, so resuming a Box search does not
turn it into a Card search. It is also remembered in `localStorage` with
the Game.

### Sealed Does Not Force `includes`

It seemed obvious it should: no two Stores name a Box the same way, so
broad Matching seemed the only useful option.

But this Form's `wide` mode also trims the List to its first Line, because
browsing a Card family happens one at a time. A quantity-based Box list is
as valid as a Card list; forcing that mode would discard line 2 onward.

It is also unnecessary. The API already broadens every sealed Query and
groups it by its Key regardless of Mode. The Match `fieldset` is therefore
hidden for sealed Products: it describes Printings and derivatives,
neither of which an unopened Box has.

## The Field Arrives Filled In

Every Game and Catalog combination has a Default. All six were measured
against Stores:

| | Cards | Sealed | Measured Offers |
| --- | --- | --- | --- |
| Magic | `Sol Ring` | `Play Booster` | 32 |
| Pokémon | `Pikachu` | `Prismatic Evolutions Booster Bundle` | 3 |
| Yu-Gi-Oh | `Dark Magician` | `Booster Box` | 6 |

The Measurements taught us that short Names find more than long ones.
`Maze of Millennia Booster Box` times out; `Booster Box` returns six Offers.
Chilean Stores put the Set first and spell it their own way, so the Title
printed on the factory Box finds nothing.

The Example follows the Game and Catalog until someone writes their own
Text. Their Text takes precedence: changing Games does not erase their List.

## Numbers Survive

The List parser trims a trailing standalone Number because a Card uses it
as a Collector number: `1 Sol Ring (LTC) 344 *F*` requests one Sol Ring.

For a Box, that Number is part of its Name.

```text
before:  'Set de Batalla 2024'  →  Order(1, 'Set de Batalla')
```

`strip_decorations` now splits its Patterns into two groups. Markers — foil,
`#!Commander`, brackets — are always removed. Printing information — the
Edition in parentheses and the Number — is removed only for single Cards.

## A Box Has No Card Catalog

`CardArt` requests Images from the Game's Catalog. For a Box, there is
nobody to ask: `/cards/metadata` knows Cards.

For sealed Products the Panel makes no Query and uses the Store's Photo.
Querying anyway would leave it at "Looking for the image…" forever.

Assisted Search is hidden for the same reason: `/cards/autocomplete` does
not accept `kind`, and a Search that never finds anything is worse than none.

## The Image Was Lost across Four Layers

This Finding was the hardest to locate and the most instructive.

The API does send each Offer's Photo. Measured: 31 of 32 in Magic, 3 of 3
in Pokémon. Yet it never reached the Browser, getting lost four times:

| Layer | What Happened |
| --- | --- |
| `muchi/api/client.py` | `build_offer` read twenty JSON fields, excluding `image` |
| `muchi/mtg/search.py` | `SearchOffer` had nowhere to store it |
| `server/presenter.py` | The Row sent to the Browser did not carry it either |
| `web/src/components/OfferList.vue` | The Frontend looked in `offer.metadata.image`, where the API never puts it |

None had been noticed because single Cards did not need that Field:
`CardArt` fetched their Images independently from the Catalog. The Gap
had always existed and appeared only when something had no Catalog.

`image` deliberately became the last Field in `SearchOffer`: some Tests
construct Offers by Position.

## Saying "There Are None" Claims Knowledge We Lack

A Screen showed these together:

```text
Bloomburrow Play Booster: could not query lacripta.cl; its offers are missing.
No offers to display.
```

The first line reports Failure; the second reports Absence. With unanswered
Sources, that Absence is actually missing Information.

The Contract already said a nonempty `faults` list means an incomplete
Response. The Interface displayed that alongside its contradiction.

Now, when Notices of level `warning` exist, the Text changes:

> No offers arrived, and some sources did not respond. Missing offers may
> still exist: try again in a while.

It uses the existing `notices`, without changing the Contract or Proxy.

## The Disappearing Field

Polling returned `502` every Cycle with "The response does not satisfy the
API contract" and no hint about which Field failed.

Two Fields declared required by the API were absent: `sequence` had
`omitempty`, disappearing at zero, and `offers` arrived as `null` instead
of an empty List when an Item found nothing. Both are normal States for
an Item still running or not found.

`build_results` now reads them tolerantly, and `parse_reply` names the Field:

```text
The response does not satisfy the API contract: missing field “sequence”.
```

That was the information needed to fix it, and the Message had hidden it.

> `omitempty` was also fixed on the side violating the Contract. That change
> is described in muchi-api's counterpart Document.

## What Remains Open

- No real sealed Search has been run from the Interface. Everything here
  was verified by calling the API directly. We still need to open the UI,
  submit a Box list and see the Photo in the side Panel.
- Three of six Yu-Gi-Oh Offers arrive without Photos. Those Stores do not
  publish them, so none can be supplied without inventing one. The Panel
  already handles it: without `image`, it renders nothing.
- The Cart has never been tested with Boxes. It should work the same way —
  Quantities are Quantities — but that remains an Assumption, not a Measurement.
