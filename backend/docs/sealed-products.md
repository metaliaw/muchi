# Sealed Products

[English](sealed-products.md) | [Español](sealed-products.es.md)

## Summary

A box of booster packs is not an expensive card. It is a different object, with
a different catalog, different sources, different naming, and different grouping.
The API learned to search for it with `kind=sealed`. Testing that search against
real stores exposed five defects that were not caused by sealed products: they
were already there, and sealed search made them visible.

They share the same seam as the [search findings](search-findings.md): **what the
code assumes about a source versus what the source actually does**. An aggregator
that indexes singles, a store that rate-limits, a printing catalog that knows
nothing about boxes. Each wrong assumption produced a different defect.

This document covers the API. Front defects live in `docs/sealed-products.es.md` in
[metaliaw/muchi](https://github.com/metaliaw/muchi), where the code they explain
lives.

## What Identifies a Box

Loose cards are grouped by their name after removing printing details: `Winged
Kuriboh`, `LDS3-EN100 "Winged Kuriboh" Common`, and `Winged Kuriboh (PUR)` are
the same card sold by three stores.

A box is different. `ReadSealedKey` in `internal/model/offer.go` preserves the
**entire title** and prefixes the set:

```
set|title            →  chaos origins|chaos origins booster box [1st edition]
```

The reason is price, not aesthetics. A Booster Pack and a Booster Display from
the same set share all but one word and differ in value by a factor of ten.
Trimming the tail groups them together, making a pack look suspiciously cheap
compared with an unrelated display.

`PriceGroupOf` puts sealed products in the same branch as `match=includes`: each
title competes against its actual peers, not against the requested name.

## Which Source Serves Which Product

Each game uses a different aggregator, and they do not all index the same things.

| Game | Aggregator | Returns boxes? |
| --- | --- | --- |
| Magic | scry.cl | **No.** It indexes singles |
| Pokémon | tcgmatch | Yes |
| Yu-Gi-Oh! | tcgmatch | Yes |

This was measured, not assumed. A sealed Magic query returned 32 offers, all 32
from direct stores; scry.cl contributed none. The same singles query returns more
than one hundred.

Querying scry.cl anyway is not free: it spends a request, waits for its full timeout,
and returns nothing. That wait is marked as an incomplete response, making the
caller think the box might exist somewhere that was not searched.

### The `sealed` Property

That is why an aggregator or store can declare it in `config/stores.yaml`:

```yaml
search_providers:
  scrycl:
    sealed: false   # indexes loose cards
  tcgmatch:
    sealed: true
```

It is a `*bool`, not a `bool`, and the difference matters. **Absent means yes.**
A plain `bool` would cause every unmarked source to stop receiving sealed queries
on deployment day. An unmarked source has not been reviewed yet; it does not mean
the source does not sell boxes.

`keepSealedSources` filters by that marker and **fails open**, like the set
filter: omitting offers does more harm to a caller than including an extra one.

The property also exists in `stores` with the same meaning and default. No store
uses it today: `gameofmagicsingles.cl` and `singles.collectorcenter.cl` suggest
their type in the domain, but a name is not evidence, and disabling them based on
a guess is exactly what this design avoids.

## The Game Filter

`Booster Box` does not name a game. A store selling several games returns all of
them, and a Yu-Gi-Oh! query once returned Cardfight!! Vanguard boxes.

`keepGameSealed` uses the game's set list to discard unrelated products: a sealed
title includes its set, and a set belongs to one game. A card name gives this
filter to singles searches for free; here, no such name is available.

It also fails open. A game without a set list, or with a list that fails to load,
returns what the sources provided.

## A Box's Image

`applyPrintImages` fills a missing image from the game's printing catalog. It
**must never run for a box**, but it did for a while.

The harm is not a missing image; it is a false one. `Bloomburrow` is both a set
and the name of cards printed in it, so a box without a photo could end up
showing an image of a card inside the box. A blank is noticeable; a wrong image
that looks right is not.

A box uses the image its store published, or none. Measured results: 31 of 32 in
Magic, 3 of 3 in Pokémon, and 3 of 6 in Yu-Gi-Oh!. Missing images belong to stores
that do not publish one; there is nowhere to get it without inventing it.

## When Every Source Fails, That Is an Answer

This was the most expensive defect and had nothing to do with sealed products.

`collectOffers` returns the last source error along with offers. When **no**
source answered, the handler had no offers and an error, so it returned **HTTP
500**—and discarded `faults` along the way.

The server knew exactly which store failed and why, yet returned a blank error.

It looked like pure intermittency. The same query returned 500 once and 32 offers
the next time, depending on whether any store managed to answer. Sealed Magic was
stacked against success because scry.cl contributed a guaranteed failure to every
query.

Today `findCardOffers` returns 500 only when there is an error **and** no
`faults` explain it. With the faults populated, it returns 200, empty offers, and
the named faults. The contract already said a non-empty `faults` list means the
response is incomplete and must not be cached; the code now follows it.

**The fix belongs in the handler, not `collectOffers`, by design.** The worker
calls `collectOffers` directly and uses that error to mark the item
`source_error`. Moving the behavior lower would have stored a card nobody could
query in Firestore as “searched with no offers”—a persistent lie, worse than a
500.

## HTTP 429: A Healthy Store Asking for a Pause

Once `faults` were visible, they showed what the 500 had hidden:

```
'Play Booster'  (1st attempt) →  32 offers
'Play Booster'  (2nd attempt) →  0 offers, oasisgames: HTTP 429
'Bloomburrow'                 →  0 offers, three stores with 429
```

Stores rate-limit requests. `updateSource` treated every error alike: five in a
row opened the circuit for a minute. A 429 counted like “connection refused.”

That punishes a store for behaving well. A 429 does not mean the store is sick;
it is alert enough to say requests are arriving too quickly. It was also
reported as down in `/health/sources`, which was false.

**Today a throttle is recorded in `last_failure` but does not advance the
counter.** `source.Throttled` recognizes only 429; a 500, 503, or dead connection
still opens the circuit as before.

### Spacing

`reserveTraffic` already rate-limited each domain with a Firestore lease: 250 ms
between calls to the same host. That floor was not increased. The decision is
deliberate: choosing a higher constant would punish every store for the behavior
of three and guess a number no store published.

Instead of guessing, the system listens. `delaySource` pushes that domain's next
turn back by 30 seconds after a 429. The store asked for space, and it gets it.

Two design details: the push **never moves a turn earlier**—two throttles cannot
shorten the wait earned by the first—and applies **per domain**, so one slow store
does not silence the others. `PaceSources(pacing, cooldown)` tunes both without
recompiling.

## `sequence` Travels Even When It Is Zero

The contract declares `sequence` required in `SearchItem`. The struct serialized
it with `omitempty`, and the field was assigned only when an item finished.

A running item has value zero, and `omitempty` **removed that zero from JSON**.
Every in-progress item traveled without a field promised by the spec, breaking a
client that followed the contract during each polling cycle.

The dishonest side was fixed: `omitempty` was removed and the spec's `minimum: 1`
was lowered to `0`, documenting that zero means “not finished yet.” Consumers
use this field to order completed results and distinguish the rest by `status`,
never by a missing field.

## What Remains Open

- **How disruptive 429 is in ordinary use.** The cases measured were caused by
  us sending repeated queries to the same domains. The limit exists and a list
  of twenty cards will hit it, but real usage has not been measured.
- **No store is marked `sealed: false`.** The property exists; the values await
  evidence.
- **No autocomplete or metadata for sealed products.** `/cards/autocomplete` and
  `/cards/metadata` know only about cards, so the Front hides its assisted search
  when a buyer asks for boxes.
