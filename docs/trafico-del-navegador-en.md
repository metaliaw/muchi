[English](trafico-del-navegador-en.md) · [Español](trafico-del-navegador.md)

# The Traffic We Do Not Pay For

September 2026 · the Pattern, not the trick

Muchi Queries other people's Stores to exist. Every Query costs Queue time,
Bandwidth, a running Instance and, above all, another Visit a Store receives
from us. The Cost is the entire chain, beyond Google's bill — Cloud Run
scales to zero.

Some Queries can go directly from the Buyer's Browser without passing
through us. This Document explains when moving them there is worthwhile,
what makes it possible and where we draw the line.

## The Three Wallets

Someone pays for every Request in Muchi:

| Who | Cost | Example |
| --- | --- | --- |
| Us | Instance, Queue, our IP facing the Store | Looking up Prices across twenty Sources |
| The Browser | Nothing on our side; an open Tab and the visitor's Connection | Fetching a Card image |
| Nobody | The Data was already there | Rereading a saved Search |

The third is best, which is why the latest Search is saved in `localStorage`:
reloading does not send anyone back to the Stores. When Data does not yet
exist, the choice is between the first two.

## Why the Architecture Allows It

The Frontend and BFF travel in one Image under one Origin. That was done
for the Token — carrying it in the Frontend would publish it — and also
means the Browser need not ask us for something it can fetch itself.

The other side decides what the Browser can reach. A Store sending
`Access-Control-Allow-Origin` declares its Data public and readable from
other pages. It is a Door the Store opened.

Where that Door is closed, we do not enter. A fetch made in `no-cors` mode
returns an opaque Response: no readable Body, Status or Headers. A web page
cannot work around that. The Stock confirmation Document lists the four
Platforms where attempts hit that boundary.

## What Already Travels through the Browser

This Pattern was already present in the Repository before it had a Name:

- Offer images. `offer.image` is a Store CDN URL placed directly in an
  `<img>`. It never passes through us. A hundred-Card Search loads a hundred
  Images without costing us a byte.
- Card art. The BFF requests only the Scryfall record; the Browser fetches
  the Image from Scryfall's CDN.
- AdSense. Google serves the Script and Ad to the Browser. We send only an
  Identifier.
- Stock confirmation. The latest addition, and the only one deliberately
  chosen for this reason: the Browser reads Shopify Stores'
  `/products/<handle>.js` and sends us only the Answer. The full explanation
  is in [Why Stock Reverification Is Optional](reverificacion-opcional-en.md).

## The Rule

Move a Request to the Browser when all four conditions hold:

1. The Store opened access. There is CORS, an `<img>`, or a `<script>` its
   Owner publishes for that purpose. No Proxy or workaround.
2. Someone requested it. A Tap, rather than a Loop. An automatic Request
   turns the Buyer's Tab into a Robot they never switched on.
3. It has a Limit. `browser_check_limit` per Card, stopping at the first yes.
   Without a cap, a hundred-Card List sends five hundred Requests from
   someone's home.
4. It serves the Person making it. The Query answers that Person's Question
   at that Moment. We do not collect Data for ourselves.

The fourth condition separates this from using Visitors as a scraping
fleet. Someone pressing "Confirm stock" wants to know whether the Card is
available; saving us a Visit is a consequence. If we ever ask the Browser
to fetch something useful only to us, it has become a different Pattern.

## What We Measured

A real Sol Ring Search returned 201 Offers from 18 Hosts:

- The Browser can reach one in five Offers: the Shopify ones.
- Stopping at the first yes took one Query and half a second to move the
  Crown from a sold-out Offer to a confirmed one.
- Without stopping, it would take 82. The Limit matters more than the Reach.

## What We Do Not Do

- Bypass Bot challenges. `cartasmagicsur.cl` returns `429` to every Client
  except a JavaScript-enabled Browser. A real Browser could pass it. We do
  not use it for that: the Store's Door applies to us too. Its Offer stays
  `unknown`.
- Query Community profiles. Sellers in scry's Marketplace are Platform
  users, not Stores. Their Offers are already excluded at the Source;
  checking Stock would visit a Person's page when they never opened a
  storefront for us.
- Store aging Data without its age. Confirmed Stock returns with a
  Timestamp and expires automatically. Saving a bare yes would turn a
  saving into a lie.
- Invent a no. Closed CORS, a failed Network or invalid JSON returns null.
  Reporting sold out when nobody said so is worse than not knowing.

## Where Else It Could Apply

- An Offer's live Price. Shopify's same JSON includes the Price. Confirming
  it has not changed costs the same as checking Stock; an outdated Price
  is another way for a Purchase to fail.
- WooCommerce, if Stores open access. Its Store API already serves
  `is_in_stock` as JSON; only a Header is missing. That covers about 2,400
  Catalog offers. We must request that change from the Stores.

## What Remains Open

- Measure how often Confirmation changes anything. If almost none move the
  Crown, the Button costs Attention without providing Certainty.
- Decide whether the Pattern needs a visible Notice. Today the Person
  pressing the Button does not know their Browser will contact five
  Stores. It is little Traffic and serves them, but explaining it costs
  only a line.
