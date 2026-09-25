# A Store Down for Eight Days

**English** | [Español](store-down-eight-days.es.md)

September 2026 · the `v3.netdecker.cl` case and what it taught us about the circuit breaker

## The Symptom

Yu-Gi-Oh! searches were slow and always returned the same warning:

```
Dark Magician: could not query www.deckscards.cl, v3.netdecker.cl;
offers are missing.
```

The search still worked—37 offers from `tcgmatch.cl` and `konohastore.cl`—but
each request paid the wait for a store that would not answer.

## Measurements

`GET /v1/health/sources`, on September 21, 2026:

| Source | Consecutive failures | Last success | Circuit |
| --- | ---: | --- | --- |
| `v3.netdecker.cl` | **39** | September 13 | Open, 1 minute |
| `www.deckscards.cl` | 0 | Earlier that morning | Closed |

And from outside, using `curl`:

- `v3.netdecker.cl` returns **503** in under a second. It is truly down and has
  been for eight days.
- `www.deckscards.cl` returns **200** in 1.8 seconds. It is not down.

These are two different problems that the same warning combined.

## Why the Circuit Was Not Enough

The [penalty ladder](source-pacing.md) opened the circuit after five consecutive
failures for **one fixed minute**. That works for a brief outage. For an outage
of eight days, the sequence is:

```
circuit open for 1 minute → closes → 5 searches query and fail
                             → circuit open for 1 minute → …
```

A store down for eight days still received **five queries per minute**. Each
cost the requesting search its `timeout_seconds`—ten seconds—and gave the down
store one more hit. The 39 consecutive failures in the record were not an
anomaly; they were the expected result of a penalty that did not grow.

## The Change

The circuit wait now **doubles with each failure** starting at the fifth, up to
one hour:

| Consecutive failures | Wait |
| ---: | --- |
| 5 | 1 minute |
| 6 | 2 minutes |
| 8 | 8 minutes |
| 11 | 1 hour |
| 39 | 1 hour (cap) |

```go
func circuitWait(failures int) time.Duration {
    wait := circuitCooldown << min(failures-circuitThreshold, 16)
    return min(wait, circuitCeiling)
}
```

It lives in `internal/db/storage.go` next to `updateSource`, where the
decision to open the circuit already lived.

**Forgiveness did not change.** One successful response still clears the whole
counter and the ladder: a store that comes back is treated like one that never
failed. That is what makes a growing penalty tolerable.

**The cap exists for the same reason.** One hour is the longest a recovered
store waits before we notice it is back. Without a cap, an eight-day outage could
leave a weeks-long penalty on a store that has recovered.

## What This Change Does Not Fix

- **`www.deckscards.cl` is slow, not down.** Its configuration declares
  `estimated_response_seconds: 145`. In a normal search it does not respond
  before the timeout and appears in the warning as though it failed. The circuit
  does not affect it—its counter is zero, correctly. Options include giving it
  more time and making the whole search wait, treating it as a slow source through
  a separate path, or accepting that it may arrive late. This is a product
  decision and has not been made.
- **The warning combines both cases.** “Could not query X, Y” is true of both,
  but one is down and the other was late. Separating them would tell the caller
  and maintainer something different.
- **Nobody measures how long a circuit stayed open.** The record stores when it
  opens until, not how many times it opened or for how long. Without that, the
  one-hour cap remains a judgment rather than a measurement, like the other
  unvalidated constants in [Source Pacing](source-pacing.md).

## Status

The change is committed in this repository and **has not been deployed yet**.
Deploying it requires `./deploy.sh`, which uploads infrastructure, worker,
sweeper, and API.

Until deployment, `v3.netdecker.cl` continues receiving five queries per minute,
and Yu-Gi-Oh! searches continue paying its ten-second wait.
