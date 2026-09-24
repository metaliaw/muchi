[English](README.md) · [Español](README.es.md)

# 🐱 MUCHI.cl

MUCHI helps you find Magic Cards and compare Store offers. Search for one
Card or paste a complete List to calculate a proposed Purchase in CLP,
including Shipping costs.

The MUCHI API performs Searches and retains their Results. This Vue 3
Frontend displays Progress, Offers and the Cart.

To look behind the Screen, explore
[MUCHI's Architecture](docs/architecture.md), its public Decisions and
ways to contribute.

> 📚 **This Repository is meant to be read as well as run.** Every MUCHI
> Decision has a Document explaining its Reasons, including Decisions that
> went wrong. If you came to learn, start with the
> [Development Diaries](#development-diaries) and return to the Code afterward.

## Search for Cards

1. Enter a Card name or paste a List with Quantities.
2. Press **Buscar** (Search). Offers appear as the Search progresses.
3. Review Results and open **Carrito en CLP** (Cart in CLP) to compare the Purchase.

The Cart does not buy yet: it allocates your List across Stores and gives
you their Links. A Notice at the top says purchasing is still under
construction and links to the Repository so you can tell us what you saw.
`MUCHI_CART_READY=1` removes that Notice when purchasing is ready: one
Environment line, without an Interface deployment.

Each Search accepts 1–100 Entries, with 1–99 copies per Entry. These are
the Limits of a Commander deck rather than a configurable Preference:

```text
1 Sol Ring
4 Lightning Bolt
2 Counterspell
```

You can cancel an active Search or resume it through its Link or Identifier.

The **Tratamiento** (Treatment) column summarizes each Offer's Finish,
Language and Condition. The API usually returns null for those Fields,
so they are also read from the Variant text and Store title. With no Clue,
the Column stays empty; no Label is invented.

The **Stock** column says `No confirmado` (Unconfirmed) when the Source
publishes no Inventory: Aggregators index Prices, not Stock. It does not
mean sold out, and those Offers still enter the Cart.

The Table opens sorted from cheapest to most expensive, mixing all Cards
in the List. Each Currency has its own sorted Block. Click any Column
heading to reorder. Prices use Chilean formatting, `1.791`, with Decimal
places only if an Offer includes them.

The **Sospechoso** (Suspicious) column explains why MUCHI distrusts a Price,
using words instead of the API's code. It stays empty without an Alert.
The Cart excludes those Offers.

**Búsquedas Recientes** (Recent Searches) reopens Searches visited during
the Session without submitting them again. Save their Links to recover
them in another Session; the API must still retain those Results.

While a Search is pending, State and Offers are polled every 3 seconds.
Each Request waits at most 10 seconds before timing out. Both Numbers live in
[`config/api.defaults.yaml`](config/api.defaults.yaml), adjustable through
`MUCHI_API_POLL_SECONDS` and `MUCHI_API_TIMEOUT_SECONDS`. The Frontend reads
them from `/api/config` at startup, so the Interval displayed matches the
one used by its Timer.

Automatic Polling stops after final Results. If fetching Results fails,
the received State is retained and the Request retried. An expired or
rejected Search stops Retries and lets you create another.

The Polling interval does not limit Search duration. The local API queries
Stores sequentially and saves Offers after each Card; Store pagination
can leave Progress at `0 of 1` for several minutes. The last received State's
Time lets you check that the connection remains active. If Submission fails,
**Reintentar Envío** (Retry Submission) retains the Request and its
Idempotency key to avoid creating another Search for the same attempt.

Offers display their original Currency and API-reported Stock and Price
warnings. The Cart uses Offers with no Price alert and no sold-out Stock,
converted to Pesos when needed. Unknown Stock is not confirmed availability;
check the Store's Offer before buying.

## Getting Listed on MUCHI

MUCHI has no registration Form. Everything shown here arrived because
someone requested it and someone connected it. The path begins with a
Conversation:

- [Project Issues](https://github.com/metaliaw/muchi/issues) are preferred:
  the discussion stays written and anyone can read it later.
- [Instagram](https://www.instagram.com/muchi_tgc) or
  [TikTok](https://www.tiktok.com/@muchi_tgc), if you prefer those channels.

Never send Passwords or Tokens in a Request. If Integration requires a
Credential, coordinate through a private Channel with read-only access
to the Inventory you want to share.

### If You Have a Store

You can propose adding Inventory through:

- Moxfield lists: share List links, available Quantities and your Pricing method.
- Your website: share a Catalog or Stock API address, including Prices,
  availability and Purchase links.

See [How to Share Your Store's Stock](share-store-stock.md) for what to
prepare and how to request Integration. Connection happens in the MUCHI
API; publishing a Link does not automatically add a Store.

### If You Sell Cards Individually

You do not need a Store. If you sell spare Cards and keep them in a public
Moxfield list, Option 1 of the Guide works for you too. MUCHI needs a List
readable without your Session, with Quantities, Prices and a way to contact
you to buy.

The same expectations apply: say whether Quantities are actual Stock or
reference-list contents, keep them current and mark sold-out Cards. An
Offer that no longer exists costs the Person who follows it. If you sell
only occasionally and cannot update regularly, say so beforehand.

### If You Want MUCHI to Support Another Game

MUCHI currently searches Magic Cards, but the Frontend never hardcoded
that Game list: it requests `GET /api/supported-games` and renders the
response. Adding a Game is API-side Work — Sources, Names and Editions —
rather than a change in this Repository.

Open an Issue describing the Game and, especially, where people buy it
in Chile: Stores or Lists already selling those Cards. A Game without
Sources returns empty Searches, so that information matters more than
the request itself. If you also sell that Game, mention it: a new Game
arriving with its first Source already has something to show.

## Advertising and Support

The Search panel keeps one advertising Space during and after a Search.
One in four Searches shows the promoted Store; the other three show a
responsive Google AdSense unit. Selection depends on the Search ID and
stays stable across Polls.

Set `MUCHI_ADSENSE_CLIENT` and `MUCHI_ADSENSE_SLOT` to AdSense's public
Identifiers. If missing, MUCHI shows an internal Promotion. The promoted
Store uses `MUCHI_SPONSOR_NAME`, `MUCHI_SPONSOR_TEXT` and `MUCHI_SPONSOR_URL`.

## The MUCHI Dollar

Some Stores publish their own exchange Rate, and the API converts with it
before returning the Offer. When an Offer arrives in Dollars without that
reference, MUCHI uses the MUCHI Dollar: one public, visible Value.

```yaml
# config/rates.defaults.yaml
muchi_dolar: 1000
```

**It is a commercial exchange Rate, not the market Dollar rate.** It covers
the Cost of bringing in the Card and the Margin of the Intermediaries who
will make the Purchase when MUCHI buys. It moves when those Costs change,
and is visible because Buyers deserve to know the Number used to convert
the Price.

It no longer appears in the Cart's interface: shown beside Shipping and
the Total, it read like part of Shipping instead of a currency Conversion,
so the display came out until it can sit next to a Dollar Price instead.
The Conversion still applies the same. It can be changed in that File or
with `MUCHI_RATES_MUCHI_DOLAR`, and the Cart's API response still reports
the Value it used, even while the Interface stays quiet about it. Offers
in other Currencies keep their original Value and are excluded from the
Cart: without a declared Rate, MUCHI invents none.

> 🚧 We are still deciding how to operate it. What it is has been settled:
> a visible commercial Rate including Cost and Margin. What remains open
> is how often to review it, who changes it and on what Signal, whether
> one Value covers all Stores, and what happens to a saved Search when
> the Number changes later. Today it is one fixed Value, edited manually:
> the simplest working Option while we decide. If you are learning from
> this Pattern, that is its actual State. The Value used is always the
> one the Cart's response reports for that Cart.
> Part of this uncertainty depends on something that has yet to happen:
> [When MUCHI Buys](docs/when-muchi-buys.md).

## We Are Working toward Purchasing

Today MUCHI leaves you at the Store's door: you compare, build a Cart and
make the Purchase yourself, once per Store. We want MUCHI to buy for you:
choose once, pay once and receive the Cards together.

The Plan is to automate purchasing with Agents that complete each Store's
Checkout instead of a Person repeating it twelve times. We are still
examining Implementation and Costs. No Agent is running and there is no
Date to promise.

This directly relates to the [MUCHI Dollar](#the-muchi-dollar), which
already includes the Margin of Intermediaries who will buy when MUCHI does.
Today they are People, and their part of the Cost could change with an
Agent. If it changes, we must decide whether the MUCHI Dollar falls or
the purchasing Cost becomes its own visible charge, separate from the Rate.

[When MUCHI Buys](docs/when-muchi-buys.md) explains the unresolved Cost per
Purchase, partial Purchases, Payments and Store relationships, how today's
Total is built and which component would be added.

## Run the Project

You need Python with `venv` and access to a MUCHI API instance. Copy
[.env.example](.env.example) to `.env` and configure:

```dotenv
MUCHI_ENV=development
MUCHI_API_URL=http://127.0.0.1:8081
MUCHI_API_TOKEN=your-security-token
```

`MUCHI_API_URL` accepts the base URL with or without `/v1`. The Security
token is required and sent as `Authorization: Bearer <MUCHI_API_TOKEN>`,
following the [API Specification](https://github.com/cangrejometralleta/muchi-api/blob/main/openapi.yaml).
Configure it on the Frontend Server; never publish it in the Repository or Links.

On Linux or macOS:

```bash
./start-web.sh   # BFF on :8000, Frontend at http://127.0.0.1:5173
```

The local API needs two Processes. From the
[muchi-api Repository](https://github.com/cangrejometralleta/muchi-api),
run `./run.sh serve` and `./run.sh work`: the former receives Requests,
the latter processes them. The Frontend Token must match the API's Token.
A Search stuck in `queued` needs an available Worker.

For Production, choose `MUCHI_ENV=production` and configure the URL and
Token in the Deployment environment.

### The Token

`./get-secret.sh` downloads the current Token from Secret Manager and
writes it to `.env`. It is for local Development; Cloud Run mounts it
automatically in the Cloud.

Rotate it using `./rotate-secret.sh` from the
[muchi-api Repository](https://github.com/cangrejometralleta/muchi-api).
The Secret is created and versioned there, and Rotation reaches all
three Consumers: Worker, API and this Frontend. A second Rotator here
would silently become stale.

## The Vue Frontend

The Frontend is Vue 3, served by a FastAPI BFF that keeps the Security
token and makes decisions on its behalf.

In Production, Cloud Run serves the BFF and retains a Frontend copy.
Firebase Hosting publishes static Files and forwards dynamic Routes to
the same Service. One Script deploys both in that Order:

```bash
./deploy.sh
```

The [Migration note](docs/web-migration.md) explains the Boundary
between `web/` and `server/` and the BFF Routes.

## Configuration and Architecture

The Frontend loads Configuration in this Order:

1. [Connection defaults](config/api.defaults.yaml).
2. `config/api.development.yaml` or `config/api.production.yaml`, according
   to `MUCHI_ENV`.
3. `MUCHI_API_TIMEOUT_SECONDS` and `MUCHI_API_POLL_SECONDS`, if defined.

Times must be positive and finite; defaults are a 3-second Interval and
10-second Timeout. Search limits — a hundred Entries, ninety-nine Copies —
come from the Commander format rather than Configuration. The Frontend
reads them from `/api/config` and displays them beside the Form, keeping
the Number in one place. Configuration files use YAML; Credentials are
injected separately.

MUCHI's Phrases, including Greetings and Help, live in
[constants/phrases.yaml](constants/phrases.yaml). `muchi/mtg/phrases.py`
loads that Content and generates Bubbles and Hearts; `messaging.py`
only decides Message priority.

The API receives the List through `POST /v1/searches`. The Frontend polls
State and Results, retaining the visible Search and received Offers in
the Session. Search persistence belongs to the API.

The Frontend no longer uses SQLite or indexes Stores directly. The current
Specification does not include Price history, manual indexing, Card
search by rules text or Commander recommendations. Legacy Adapters and
Defaults remaining in this Repository are not connected to the current
Flow. Existing SQLite files are neither deleted nor imported.

## Tests

With your virtual Environment active:

```bash
pip install -r requirements-dev.txt
python -m pytest -q
```

The regular Suite tests the HTTP client and Interface with mocked Services,
without connecting to the API. For real Integration, start the API,
configure `.env` and run:

```bash
MUCHI_API_INTEGRATION=1 python -m pytest tests/test_muchi_api_integration.py
```

To test Sol Ring from the Frontend through real Offers, with the Worker
running — this may take several minutes:

```bash
MUCHI_API_INTEGRATION=1 MUCHI_API_SEARCH_INTEGRATION=1 python -m pytest -q tests/test_muchi_api_integration.py
```

Add `MUCHI_API_SEARCH_ID` to verify an existing Search.

## Development Diaries

MUCHI is written for others to read. Every Document explains a Decision,
its Reasons and, where relevant, the Assumption that failed. The following
index links every Document in `docs/` in English. For the matching Spanish
index, see [Diarios de desarrollo](README.es.md#diarios-de-desarrollo).

### Start Here

| Document | What You Learn |
| --- | --- |
| [MUCHI Architecture](docs/architecture.md) | Frontend/API boundaries, Security, Data, Operations and a Provider-neutral guide to reproducing the Pattern. |
| [The Frontend and Its Boundary](docs/web-migration.md) | What the Frontend renders, what the BFF decides, its Routes and Cloud Run deployment. |
| [How Muchi Speaks](docs/how-muchi-speaks.md) | Phrase Catalog, Bubble precedence, three Notice channels, Sprite sheet and reduced Motion. |
| [Sprite Lab](docs/muchi-sprite-lab.html) | Interactive Sprite previews, States, Sheets and animation examples. Open the HTML in a Browser. |
| [Sealed Products](docs/sealed-product.md) | Catalog selection, measured Defaults, Box photos and misleading empty Results. |
| [MUCHI API Contract](https://github.com/cangrejometralleta/muchi-api/blob/main/openapi.yaml) | Consumed Routes and their Input/Output shapes. Owned by muchi-api. |

### The API Has Its Own Repository

MUCHI spans two public Repositories. This one renders the Experience and
holds the BFF. [muchi-api](https://github.com/cangrejometralleta/muchi-api)
queries Stores, orders Offers and retains Results.

Each subject is documented in its owning location:

- The [OpenAPI Contract](https://github.com/cangrejometralleta/muchi-api/blob/main/openapi.yaml)
  lives there. A Copy here aged silently while Readers thought it was current.
- The [Bruno collections](https://github.com/cangrejometralleta/muchi-api/tree/main/bruno)
  live there. They call the API without the BFF, helping distinguish
  Frontend failures from Service failures.
- [Backend Architecture](https://github.com/cangrejometralleta/muchi-api/blob/main/docs/architecture.md)
  covers Queue, Worker, Persistence and Expiration there. The Frontend
  and its Boundary are described [here](docs/architecture.md).
- [Source Pacing](https://github.com/cangrejometralleta/muchi-api/blob/main/docs/source-pacing.md)
  explains how the API treats Stores that fail, run slowly or request
  a pause. That Policy informs MUCHI's Notices: a Source that did not
  answer must never be mistaken for a nonexistent Card.

External Repository documents keep their upstream language; this
Repository's translations cover the Documents maintained here.

### Decisions Still Open

- [The MUCHI Dollar](#the-muchi-dollar): a public commercial Rate whose
  definition is settled but whose operation remains open. A Number
  converting other people's Prices deserves an explanation even while
  Decisions are unfinished.
- [When MUCHI Buys](docs/when-muchi-buys.md): the Plan for automated Agent
  purchases, Implementation and Costs under consideration, and their
  relationship to the MUCHI Dollar and Cart costing. A Plan without a Date.

### Decisions in Detail

- [The Stock Request](docs/api/stock-order.md): rounds containing each
  Card type's cheapest Candidate in one Request, so Cost grows with
  Uncertainty rather than List length. The BFF retains the Route and
  `stock_check_limit` (currently 3). The Frontend calls it at the end,
  including what the Browser has already Confirmed.
- [Why Stock Reverification Is Optional](docs/optional-reverification.md):
  the Load of revisiting Stores for every cheap Offer, why that Pass
  cannot be mandatory, and how the Buyer's Browser covers one in five.
- [The Traffic We Do Not Pay For](docs/browser-traffic.md): who
  pays for each Request, when it moves to the Browser, and the Boundary
  between using open access and treating Visitors as a scraping fleet.
- [Search Findings](docs/search-findings.md): Assumptions broken
  when Searches returned different Cards rather than Printings, their
  Fixes and remaining Questions. API-side Findings live in its Repository.
- [Advertising, End to End](docs/advertising.md): Identifiers, the path
  to the first Ad, Google's Review requirements and troubleshooting.

### Getting Your Offers onto MUCHI

- [Share a Store's Stock](share-store-stock.md): what to prepare and how
  to request Integration, including sellers without a Store.
- [Getting Listed on MUCHI](#getting-listed-on-muchi): Stores, individual
  sellers and new Games, and where to request each.

## License

MUCHI is Free Software under the
[GNU Affero General Public License v3.0 or later](LICENSE).

You may use, read, modify and redistribute it. Affero adds the condition
relevant here: anyone operating MUCHI — or a modified Version — as a
Network service must offer its Source code to the People using it.
Improved Forks are welcome; closed hosted Forks are not.

That is why MUCHI displays **Ver el código** (View the code) on its own
Screen: the Source offer required by Section 13 points to this Repository.

```text
Copyright (C) 2026 MUCHI

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
```

[muchi-api](https://github.com/cangrejometralleta/muchi-api) uses the same
License. Two Repositories, one Rule.

## The Harness and Manifesto

Both Repositories—the [MUCHI Front](https://github.com/metaliaw/muchi/blob/main/README.es.md)
and [muchi-api](https://github.com/cangrejometralleta/muchi-api)—use
[OneTwoThree](https://github.com/cangrejometralleta/OneTwoThree), my shared
Development Harness and Manifesto.
