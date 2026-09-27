# Checkout Without Agents

**English** | [Español](agentless-checkout.es.md)

Exploration · September 25, 2026 · Complements the
[web agent purchases proposal](web-agent-purchases-proposal.md).

## The Idea

A browser agent is the most expensive and least predictable way to buy. Every
store platform already exposes deterministic ways to fill a cart and reach
checkout. If we use them, the agent becomes only a fallback.

Muchi provides the payment method. The question is how much of the path can run
on plain HTTP or a fixed script, with no model deciding what to do.

## Three Levels of Automation

| Level | What is automated | Who pays | Tool |
| --- | --- | --- | --- |
| 0. Link | Filled cart and open checkout | A person, with one click | URL built by Muchi |
| 1. Order | Cart, shipping details, and created order | Later bank transfer | Platform HTTP API |
| 2. Script | The whole checkout, including payment | Muchi's method | Playwright with fixed steps, no model |

The browser agent remains level 3: unusual stores or steps the script does not
recognize.

## What Each Platform Offers

Magic stores configured today: seven Shopify, two WooCommerce, and four paused
Jumpseller stores. No PrestaShop store sells Magic.

### Shopify: A Direct Link to Checkout

Shopify has **cart permalinks**: `https://store/cart/VARIANT:QTY,VARIANT:QTY`
creates a new cart and redirects straight to checkout. Muchi already stores the
variant ID in `offer.VariantID`, so the link needs no extra work.

- It accepts parameters to prefill email and shipping address, plus `discount`
  and `note`. Which ones the current checkout honors needs confirmation.
- The AJAX API (`/cart/add.js`, `/cart.js`) can validate cart price and stock
  before showing the total.
- **Payment cannot be completed through an API** by a third party: Shopify
  checkout is a page. But it is **the same page on every Shopify store**, so one
  Playwright script covers all seven.

Conclusion: level 0 for free, level 2 with one shared script.

### WooCommerce: The Only One That Reaches an Order Without a Browser

The **Store API** (`/wp-json/wc/store/v1/`) is public and powers the block
checkout itself:

1. `POST cart/add-item` with `id` and `quantity`. The response carries a
   `Cart-Token` header that identifies the cart in later calls.
2. `GET cart` returns totals, available shipping, and `payment_methods`.
3. `POST cart/select-shipping-rate` picks shipping.
4. `POST checkout` with an address and `payment_method` **creates the order**.

With bank transfer (`bacs`), the order stays "pending payment" and the store
sends its bank details. No browser involved. With Webpay, Flow, or Mercado Pago,
the response carries `payment_result.redirect_url`, and a payment page begins.

One caveat: some stores disable the Store API or use the classic checkout.
For a single simple product, `/?add-to-cart=ID&quantity=N` adds it by link —
`stores.CheckoutLink` uses it for a single-line WooCommerce cart, the one case
it can name without the Store API.

### Jumpseller: The MCP Hands Over the Link

The Storefront MCP returns, per variant, one URL to add to cart and another to
buy. That is level 0 with nothing new to build. The MCP payment tools do not
exist yet, so level 2 needs a script, as with Shopify. Jumpseller checkout is
also shared across its stores.

### PrestaShop

The `index.php?controller=cart&add=1&id_product=…&id_product_attribute=…`
controller adds products but needs the session's static `token`, read from the
HTML. Not urgent: no Magic store uses it.

## Payment Is the Real Limit

Filling the cart is solved on all three platforms. Paying is the hard part:

- **Card through Webpay or Mercado Pago.** It asks for card details and almost
  always a 3-D Secure approval in the bank's app. That phone approval cannot be
  automated, and that is as it should be.
- **Bank transfer.** Many Chilean singles stores accept it. The order is created
  unpaid and the transfer happens afterward. This path has the fewest moving
  parts.
- **Automating the transfer.** Still to research: Chilean payment-initiation or
  transfer APIs, and whether any can pay a third party without manual
  confirmation. Unverified.

Proposal: **prioritize stores that accept bank transfer**. Muchi creates the
order through an API or script, and payment is grouped into one transfer per
store. The operator only approves transfers; they never browse.

## Risks That Do Not Change

- Store terms may forbid automated purchases.
- Transferring before the store confirms stock can leave money waiting on a
  refund. Waiting for the confirmation email is safer.
- A fixed script breaks when the store changes its checkout. That is why Shopify
  and Jumpseller matter: a platform change is fixed once.
- The proposal's rules still apply: never retry a payment blindly, and stop if
  the total changed.

## What Still Needs Verification

The container where this was written cannot reach the store domains, so
**nothing here was tested live**. Before building:

1. Open a cart permalink on two Shopify stores and confirm prefill.
2. Walk the Store API on onplay.cl and lacripta.cl up to `GET cart`, without
   creating an order, and record their `payment_methods`.
3. Check which stores accept bank transfer.
4. Read each store's terms on automated purchases.

## Level 0 Implemented

`POST /v1/searches/{search_id}/checkout` takes `items` with `offer_id` and
`quantity` and returns one entry per store. Shopify stores answer `mode: cart`
with the permalink; a single-line WooCommerce cart answers `mode: cart` with
an `add-to-cart` link; everything else answers `mode: product_pages` with each
line's page. If a Shopify line lacks its variant (a scry.cl offer, for
example), or a WooCommerce cart carries more than one line, the whole store
falls back to product pages so no buyer lands on an incomplete cart.

## Level 1 on WooCommerce: Quotes

When the `/checkout` request carries `shipping` (`country` and `region` by name,
such as `Chile` and `Región Metropolitana`), each WooCommerce store fills a fresh cart
through the Store API: it takes a `Cart-Token`, adds the lines, and sets the
address. It answers `quote` with subtotal, shipping, total, shipping rates, and
payment methods. A line the store trims for lack of stock shows in `notices`.

No order is created and no stock is held: a WooCommerce cart does not reserve
units; only checkout (the draft order) holds them for a few minutes. The cart is
abandoned and expires on its own. Add-to-cart calls are never retried, since a
retry would double the quantity.

Only simple products sent by the store itself are quoted. Aggregator offers and
variable products get no `quote`.

## Regions and Shopify

The region is written by name, as in `stores.yaml`. A fixed table in
`internal/stores/regions.go` turns it into the region's ISO code: WooCommerce
receives `CL-RM` and Shopify `RM`. The code itself is accepted too.

Shopify quotes through its AJAX API: `POST /cart/add.js` builds a cart in a
cookie of its own, `GET /cart.js` gives the subtotal, and
`GET /cart/shipping_rates.json` gives the rates for the address. With no rate
selected, the total uses the cheapest one. Shopify reveals no payment methods
before checkout. Jumpseller still has no quote: its cart is a form, and its
Magic stores are paused.

## Level 1 Implemented: Orders

`POST /v1/searches/{search_id}/orders` places a real order at one store, for
offers the search already found. Every line must resolve to the same store
domain — an order is one store's transaction, unlike `/checkout`'s per-store
grouping — so a cart spanning more than one store is rejected before any call
is made. An `Idempotency-Key` guards a doubled tap the way `POST /searches`
does: a retried call with the same key and the same cart answers the order
already placed, never a second one.

Only WooCommerce supports this today (`422 order_not_supported` for every
other platform). `woocommerce.Client.PlaceOrder`
(`internal/stores/woocommerce/order.go`) fills the cart, reads the shipping
rate the cart already selected, and checks out by bank transfer (`bacs`) —
the one payment method the Store API confirms without a browser or a card.
The order lands `pending`; nothing here waits for the transfer to arrive.

The Store API checkout needs a buyer identity `QuoteCart` never did (name,
email) — a cart quote only needs to know *where*, an order needs to know
*who*. Until an operator names a real one, every order would check out as
Muchi itself: `MUCHI_ORDER_BUYER_NAME` and `MUCHI_ORDER_BUYER_EMAIL` set that
identity; left unset, the client falls back to a reserved, undeliverable
placeholder (`orders@muchi.invalid`, RFC 2606) and `PlaceOrder` refuses to run
rather than check out on a mailbox nobody reads. `config/deploy.env` now sets
`ORDER_BUYER_EMAIL=cangrejometralleta@gmail.com` as a provisional inbox —
someone reads every order, not a real Muchi mailbox yet.

`model.Order` carries a `Status` (`pending → confirmed | released`),
persisted one Firestore document per order (`db.Store.CreateOrder`,
`GetOrder`, `MoveOrderStatus`). A move is transactional and names the status
it expects to move *from*, so a stale caller — a webhook confirming an order
a sweep already released — loses instead of overwriting.

## Suggested Next Step

Nothing yet moves a `pending` order forward or back: no webhook confirms a
transfer, and nothing releases an order whose transfer never arrived. The
Sweeper that backs `SweepQueue` (`internal/sweep`) does not fit — it counts
waiting search items and replaces lost worker wake-ups, a different shape
from reading `orders` where `status == "pending"` and `updated_at` is past a
cutoff. A sibling entry point, `ReleaseOrders`, on its own schedule, is the
next piece.

Once that exists: try a live order on a pilot WooCommerce store, behind the
buyer's explicit confirmation, and confirm the store's own confirmation email
reaches the provisional inbox and matches what the Store API answered.
