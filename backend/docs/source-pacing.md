# Source Pacing

[English](source-pacing.md) | [Español](source-pacing.es.md)

## Summary

Muchi depends on sources it does not control. One store goes down, another is
slow, another asks for a pause, and another says it has nothing. Each response
forces a decision: ask again, wait, stop asking for a while, or tell the user.

Those decisions were scattered through the code—a counter here, a circuit
breaker there, one more retry elsewhere—and together they formed a policy nobody
had written down. This document makes that policy explicit.

This is not an implementation guide; see [game search](game-search-providers.md)
and [sealed products](sealed-products.md) for that. It explains the reasoning
behind the policy and is most useful when deciding how to handle a new case.

## The Rule

> **Punish failures. Do not punish slowness, honesty, or requests for a pause.**

Everything else follows from that.

## The Penalty Ladder

There are four levels, from mildest to most severe. A source moves up only when
it keeps failing.

### 1. Nothing — Throttling

A store returns **429**. Muchi gives it the thirty seconds it requested and does
not record a strike.

A 429 does not mean a store is sick; it means the store is alert enough to say
that requests are arriving too quickly. Punishing it for that would punish the
store doing the right thing and make `/health/sources` lie by showing it as down.

The delay is per domain and never moves a reserved turn earlier: two throttles
together cannot shorten the wait created by the first.

### 2. Retry — Uncertainty

A **5xx**, timeout, or dead connection is retried up to three times, with
increasing delays and some randomness so two searches do not retry at once.

If the source sends `Retry-After`, Muchi respects it up to a **30-second** cap.
Longer waits stop looking like a brief pause and start looking like “come back
later”; users should not have to watch a frozen screen for five minutes because a
store asked them to.

An already open circuit is not retried. That would mean continuing to ask a
source that is serving its penalty.

### 3. Record — Short Memory

The failure is recorded in `last_failure` with its latency, and the consecutive
failure count increases by one. The source continues receiving requests normally.

### 4. Open Circuit — Silence

After the **fifth consecutive failure**, the source stops receiving requests.
Anyone asking for it during that interval gets a failure that says when the
pause ends.

The wait **doubles with each further failure**: one minute, two, four, up to one
hour. A fixed one-minute pause helped with brief outages but failed during an
eight-day outage: the [store that was down for eight days](store-down-eight-days.md)
still received five requests per minute, and each request consumed its timeout
from the search that asked for it.

This is still a mild penalty and the strongest one Muchi applies: only to a
source that failed five times in a row, and one successful response erases it
completely.

## Forgiveness

**One successful response erases every strike.**

```go
if sourceErr == nil {
    record.LastSuccess, record.ConsecutiveFailures, record.CircuitOpenUntil = &now, 0, nil
    return
}
```

There is no probation, half-count, or lingering history. A store that failed
four times and succeeds on the fifth returns to zero, just like a store that
never failed.

This is deliberate and worth defending. Muchi's sources are small Chilean
stores: a deployment, a full hosting plan, or an overnight maintenance window
can cause an outage. Punishing a source after it has recovered does not improve
offers; it only makes Muchi slower to notice that the store is back.

## What Is Never Punished

Four responses may look like failures but are not. Confusing them is the most
expensive mistake a system like this can make because all four are **honest**.

| The source says | Muchi understands | And does not call it |
| --- | --- | --- |
| “I do not have that card” | `not_found` | a failure |
| “I do not know whether it is in stock” | `unknown` | “out of stock” |
| “You are going too fast” | a 30-second throttle | an outage |
| *(nobody marked it)* | keep asking | assume it is unusable |

The last case is the subtlest. Source filters—the game-by-set filter and
`sealed`—**fail open**: a source nobody marked is one nobody has reviewed yet,
not one that cannot help. That is why `sealed` is a `*bool`, not a `bool`: an
absent value means yes.

That is also why Muchi reports “no offers” only when **every** source answered.
If one source did not respond, the result is marked incomplete and names the
missing sources. Saying “none” when someone did not answer claims knowledge we
do not have.

## Who Is Actually Penalized

Muchi does make a strict judgment, but not about stores: **it judges offers**.

An offer **30% below the median** of its peers is marked `suspicious`. Peers are
offers for the same card in the same currency, never another card or currency—a
dollar and a peso are not comparable just because of their numeric values.

Two parts of this penalty matter:

- **It is a mark, not a deletion.** The offer remains in the list with its price
  and link. Muchi warns; the user decides.
- **It applies to the offer, not its publisher.** A store is not marked because
  one offer is unusual. The price is judged, not the person who posted it.

## Uncertainty Favors the Caller

The same policy applies to the cache, where the penalty is not being remembered:

| Response | Cache duration |
| --- | --- |
| Complete, with offers | 3 days |
| Complete, empty | 2 minutes |
| Incomplete (a source failed) | **none** |

An incomplete response is never cached: repeating it for days would turn one bad
minute at a store into a lasting absence. A complete empty response lasts two
minutes instead of three days because “nobody sells it today” grows stale much
faster than a price.

## Why the Asymmetry

All these choices lean in the same direction, and the reason should be clear.

**The two possible mistakes do not cost the same.** Showing an extra offer costs
someone a click and a moment of doubt. Hiding a store that has the card costs a
purchase—and the caller never learns what they missed.

Muchi is an intermediary between someone searching and stores that do not even
know it exists. An intermediary that is wrong by staying silent is worse than
useless: its mistake is invisible.

## What Remains Open

- **The constants have not been measured.** Five failures, a one-minute circuit
  and its one-hour cap, a 30-second throttle, and 30% below the median are
  reasonable numbers nobody has validated against data. `PaceSources` lets the
  pacing values change without recompiling; the others require an edit.
- **There is no long-term memory.** The counter tracks only consecutive failures,
  so a store that fails 40% of the time never reaches five and never rests, while
  a store that fails five times overnight and works perfectly the rest of the day
  is penalized. This is the cost of total forgiveness and likely the first place
  this policy will break.
- **429s are not measured in ordinary use.** The ones observed were caused by
  repeated diagnostic requests. Their frequency during real use is unknown.
