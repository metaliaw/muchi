[English](pedido-stock-en.md) · [Español](pedido-stock.md)

# Check Stock before Awarding the Crown

September 2026 · the Conversation between the BFF and the API

## Why We Ask

The Frontend crowns the cheapest Offer for each Card. A low Price for a
sold-out Card sends someone on a wasted Visit to the Store. Before awarding
the crown, Muchi asks again — from cheapest to most expensive — until a
Store Confirms that it has the Card.

Store requests belong on the API side: the Worker already knows how to talk
to each Provider, with its Adapter and Timeout. Repeating that Logic in the
BFF would publish it twice and duplicate the Traffic Stores receive from us.
The BFF therefore visits nothing: it Decides whom to ask and in what Order.

## The Contract

```text
POST /v1/searches/{search_id}/stock
{"offers": ["offer-1", "offer-2"]}
```

```text
200
{"offers": [
  {"id": "offer-1", "stock_status": "unavailable", "stock_quantity": 0},
  {"id": "offer-2", "stock_status": "available",   "stock_quantity": 3}
]}
```

- `id` is the same `id` the Offer carries in `/searches/{id}/results`. A
  caller-selected URL is Rejected with 404: this is not a Proxy.
- `stock_quantity` is optional. Missing means the Store does not disclose it;
  zero means we asked and none remain. Muchi displays these States
  differently. Shopify says whether a Variant is sold, not how many remain;
  Jumpseller does Count.
- A Store that does not Respond returns `unknown`, rather than an Error: the
  caller moves to the next Offer instead of losing the entire Search.

## The Three Answers and the Crown

| The Store Says | Muchi Understands | The Crown |
| --- | --- | --- |
| `available` | It has the Card | Keeps it, and the Round ends |
| `unavailable` or zero Units | It does not have the Card | Goes to the next Offer |
| `unknown` | It does not disclose availability | Remains a fallback; asking continues |

Uncertainty does not close the Round. Between a cheap unknown and an
expensive yes, the yes wins: the Recommendation exists so someone can Buy.
A Card that never arrives is not a cheap Purchase. An unknown wins only
when nobody Confirms; a Card whose Stores all said no receives no Crown.

## The Limit

`/api/searches/{id}/stock` asks in Rounds: the first Candidate for each Card
type travels in one Request, and only types without Confirmation proceed to
the next Round. Cost grows with Uncertainty, rather than List length.
`stock_check_limit` — currently 3, in
[`config/offers.defaults.yaml`](../../config/offers.defaults.yaml) — caps it.

We do not ask about an already sold-out Offer, one without conversion to
Pesos — it cannot compete for the Crown — or one the API did not give an
`id`. Nor do we ask about one the Browser already Confirmed: `POST` on the
same Route accepts `{"checks": [{"offer_id": "...", "available": true}]}`
and starts the Rounds there. A Store that answered the Buyer need not answer
us too. `GET` follows the same path with no prior Knowledge.

A Tap checks one Offer only. Selecting an Offer in the List confirms that
Store alone. When the Browser can reach it — Shopify, with open CORS — the
Question never reaches us: it travels as `checks` with `ask=false`. When it
cannot — a Store read from Moxfield lists, a Catalog that serves no JSON —
the same `POST` accepts `{"checks": [], "asking": ["<offer_id>"]}` and we
ask, with `plan` narrowed to that Offer. Without that restriction, the Tap
received no answer or cost an entire Round.

The Limit governs the Question, not the Crown. There are two Lists:
`ranking` — all viable Offers for each Card, cheapest first — decides who
competes, and `plan` — its first `stock_check_limit` entries — decides whom
we visit. A Browser-confirmed Offer competes even in ninth place, because
confirming it cost us no Visit. Confusing those Lists awards the Crown to
the cheapest unknown despite a confirmed yes further down, exactly what
happened with `cartasmagicsur.cl`.

## What Remains beyond Reach

A Store behind a Bot challenge cannot be checked. `cartasmagicsur.cl`
returns `429` with `x-vercel-mitigated: challenge` to any Client other than
a JavaScript-enabled Browser, and scry.cl — which indexes it — publishes no
Stock. That Offer remains `unknown`, which is why the rule above exists:
otherwise the cheapest unknown would keep the Crown without Confirmation.

This differs from [`verify_stock`](../reverificacion-opcional-en.md), which
rechecks every candidate Offer during a Search and defaults to off because
of its Load. Here we ask at the end, one at a time, only while no Offer has
Confirmed. That Document also explains which Offers the Browser can reach
itself and how long its Confirmations last.
