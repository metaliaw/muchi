# Web Agent Purchases 🐈

**English** | [Español](web-agent-purchases-proposal.es.md)

Proposal for discussion · September 20, 2026 · Amounts in USD

## One Cart, More Possibilities

Today Muchi tells you where the cheapest card is and leaves you there. Buyers
finish with six tabs open, six half-filled carts, and six separate shipping fees
to pay. We propose handling that last stretch: the buyer builds a cart in Muchi,
and we manage the purchase at each store for a small fee per external order.

**We recommend starting with two stores and a supervised pilot.** Feasibility
depends on one number: how many purchases complete successfully without human
intervention.

### How It Would Work 🐾

Consider a concrete case. Valentina needs four cards for a deck. Muchi finds two
at a store in Providencia and two at one in Concepción, and shows that buying
from both is cheaper than using any single store.

1. **Select and quote.** Valentina selects the four cards and asks Muchi to
   manage the purchase. An agent opens each store, confirms the cards are
   available and that set and condition match what Muchi showed, then fills both
   carts up to checkout without paying. It returns two real totals, including
   shipping.
2. **Confirm and purchase.** Valentina sees both totals, Muchi's fee, and the
   final total. She accepts. Only then does the agent return and pay each store
   with a payment method we own.
3. **Coordinate delivery.** Muchi records both order numbers and tracks shipping.
   Valentina sees one status even though two stores and two carriers are
   involved.

What Valentina cannot see is what matters: minutes can pass between steps 1 and
2, and a card may sell during that time. The agent must notice and stop, not buy
something similar.

The commercial assumption is that the customer pays Muchi and we purchase from
the source store using our own payment method. Before launch, we must agree who
is responsible for cancellations, returns, and warranties.

## Why This Is Harder Than It Looks 🐈

A browser-operating agent seems like magic until it reaches checkout. Buying is
not searching: a search can be retried a hundred times without consequence, but
a payment happens once.

These are the four underlying problems, ordered by difficulty:

**Payment cannot be rolled back.** A database transaction that fails halfway
can be undone. A purchase cannot. If the agent clicks “pay” and the connection
drops before confirmation appears, we do not know whether an order exists.
Retrying may buy twice; not retrying may leave the buyer with nothing. Our
proposed rule is that the agent never retries a payment blindly: first it checks
the store's order history, and retries only if the order is absent.

**The correct card is hard to identify.** “Sol Ring” refers to dozens of cards:
different sets, languages, conditions, and foil status. Muchi already deals with
this when comparing prices, but a mistake there means showing a bad offer. In a
purchase, it means buying someone a card they did not request with real money.
The variant is where trust is most easily lost.

**The price changes while we look.** Stock, price, and shipping can change
between quote and payment. The agent must compare the total with what the buyer
accepted and stop if they differ, rather than adapting on its own.

**Websites defend themselves.** Stores use bot protection, captchas, and expiring
sessions. That is why traffic uses a residential proxy and why each attempt has
a meaningful cost. The clean alternative is to talk to stores and get access,
which is the competing hypothesis at the end of this proposal.

None of this requires technology that does not exist. It requires giving the
agent permission to stop. An agent that always completes the task is dangerous
here; we want it to pause and ask for help when something does not match.

## There Is a Standard, and a Shorter Path 🐈

These problems are not unique to us; the industry has named them.

### AP2: The Right Answer That Is Not Available to Us Yet

Google announced the **Agent Payments Protocol** in September 2025 and donated
it to the FIDO Alliance in April 2026, making it an industry standard backed by
Mastercard, PayPal, and American Express. It links three signed credentials:
what the person authorized, which cart they accepted, and which payment method
was used. It addresses the irreversible-payment problem at its root by creating
evidence that neither the store nor buyer can deny.

The catch is that **AP2 is implemented by the platform and payment processor,
not by us or individual stores**. Shopify enabled agentic purchases by default
in March 2026, but only for eligible merchants in the United States. None of the
Chilean stores on our list accepts it today, and they cannot enable it by
themselves.

This is not a path we can take within the pilot timeline. It is a reason not to
build something that depends on fighting checkout forever.

### Jumpseller MCP, Available Today

Jumpseller—a Chilean platform—**already exposes a public endpoint at all its
stores**, without authentication, where an agent can query the catalog in a
structured way. We tested it against a real store on our list and it responds.

For each product and variant, it returns price, SKU, currency, categories,
**variant-level stock availability**, and, most usefully, **a direct URL to add
the item to a cart and another to purchase it**.

This changes step one of Valentina's example. At a Jumpseller store, the agent
does not need to browse to build a cart: it queries, builds the link, and goes
straight to checkout. This removes HTML parsing, much of the browser time, and
some proxy traffic.

Two honest caveats. First, **the endpoint's search is weak**: searching for
“Lightning Bolt” returns anything containing “Lightning,” while “Sol Ring” did
not find a single Sol Ring among one hundred results. Our own card identity logic
is still required; the gain is structured data, not a search engine. Second,
**checkout is not ready yet**: Jumpseller says it is working on extending this
endpoint with payment tools. Today it reaches the cart, not a paid order.

The scope is limited: among configured stores, one active and four disabled
stores use Jumpseller. It does not solve the entire catalog. It solves one part
well and opens a conversation with a Chilean provider that has already built
half the path.

The four protocols, their status, and reach across platforms are in
[Agentic Commerce Protocols](agentic-commerce-protocols.md).

## Approximate Costs 🐾

Browser Use lets an agent operate a browser from a task. Its pricing combines
model cost plus 20%, browser time at US$0.02 per hour, and a residential proxy at
US$5 per GB, enabled by default.
[Official pricing](https://browser-use.com/pricing).

The following numbers are **assumptions for sizing the discussion, not
measurements**. They show whether the order of magnitude is cents or dollars,
which is the question that matters now:

| Per attempt | Simple | Base | Complex |
| --- | ---: | ---: | ---: |
| Automation cost | US$0.20 | US$0.45 | US$1.30 |
| Completes a successful order | 95% | 90% | 75% |
| Cost per successful order, with human help | US$0.26 | US$0.61 | US$2.07 |

Each column assumes model cost of US$0.10–US$0.75 before the markup, 3–10
minutes of browser time, 15–80 MB through the proxy, and five minutes of human
help per case at US$12 per hour. The calculation spreads the cost of all
attempts—including failures—across successful orders.

Direct cost ranges from twenty cents to two dollars per order, and **success
rate matters more than model price**. Moving from the simple to complex scenario
multiplies cost by eight, mostly because of the human help it requires. That is
why the pilot measures reliability before anything else.

This excludes development, maintenance, payment fees, taxes, returns, losses
from mistakes, products, and shipping.

### Proposed Fee to Test

A fee of **US$2 to US$4 for each confirmed external order at one store**, shown
before the buyer accepts. A cart split across three stores means three purchase
operations and may incur three shipping fees.

The final fee depends on observed costs and how much buyers value saving the
time. A fixed fee may be unattractive for small purchases.

## Test First, Then Scale 🐈

### Proposed Pilot

1. Choose two stores and products with clearly identifiable variants.
2. Run 100 tests that stop before payment.
3. Make up to 20 supervised real purchases, within an approved budget.
4. Measure accuracy, purchase success, cost per order, time, and human
   intervention.
5. Decide whether to expand coverage and what fee can sustain the service.

We propose an **indicative technical reserve of US$200 for automation**. Up to
120 attempts in the complex scenario cost about US$156. The reserve excludes
products, shipping, development, and human work.

Unpaid tests validate cart preparation but do not show whether the complete
purchase works. Supervised purchases check payment, confirmation, and shipping.

Initial goals are at least 95% correct carts and zero duplicate payments or
incorrect products. Any critical error pauses the pilot. A small sample does not
prove reliability at scale.

### Safeguards for Each Purchase 🐾

- Verify product, variant, quantity, seller, address, and total before payment.
- Use per-run secrets and restrict the domains where they can be entered.
- Apply amount limits and human review during the pilot.
- Check the external order before retrying a payment with an uncertain result.
- Stop the flow when accepted terms change.
- Review each store's terms and define after-sales responsibilities.

Storing a credential as a secret does not guarantee the agent can never see it:
after entry, it is available to the page and an agent with browser access.
[Secrets documentation](https://docs.browser-use.com/cloud/guides/secrets).

Browser Use allows a person to take control to review, authenticate, or complete
a payment and then continue the session.
[Human supervision](https://docs.browser-use.com/cloud/agent/human-in-the-loop).

## Where We Start: We Store No Personal Data Today 🐈

Before discussing how to protect buyer data, we should know what we currently
store. We reviewed the API and found less than expected:

- **There is no identity.** Authentication uses one shared token. There are no
  accounts or users, and no email, phone, address, or RUT field exists in the
  code.
- **Stored records expire automatically.** Each record has an expiration date; a
  search lives for 24 hours, and Firestore deletes it by policy while the code
  also discards expired records on read.
- **Logs do not reveal content.** Logs record IDs and counts, never what someone
  searched for. Metrics count only method and status.

The only quasi-personal data is the list of cards someone searches for. Today it
is harmless because it is **unlinkable**: nothing connects a search to a person.
That is an accident of having no accounts, not a written decision. The day login
exists, search history becomes personal data immediately.

This matters to the proposal: **buying on someone's behalf does not add personal
data to a system that already handles it; it introduces personal data into a
system that currently has none.** That is a larger change than it looks—we
become responsible for personal data—and a cleaner one because there is no
existing data debt to fix.

### Buyer Data 🐾

The purchase flow touches payment method, name, phone, delivery address, and, at
some stores, the buyer's account. We propose deciding four things in advance:

- **The agent does not see the buyer's card.** The customer pays Muchi through a
  payment processor; the agent pays with a method we own, ideally a virtual card
  per order, capped at the approved total. A mistake or abuse is bounded to one
  purchase.
- **The address is provided only when needed.** The agent receives it to complete
  shipping, not at session start.
- **Browser recordings contain personal data.** Screenshots and session traces
  show addresses and payment information. We need short retention, restricted
  access, and verifiable deletion, especially because a person reviews those
  sessions during the pilot.
- **Someone is accountable for the data.** Before the first real purchase, we
  must define who is responsible to the buyer and the role of each provider—the
  payment processor, Browser Use, and carrier.

[Law 21.719](https://www.bcn.cl/leychile/navegar?idNorma=1209272) on personal
data protection takes effect on December 1, 2026, with an agency and fines. The
pilot falls near that transition, so compliance should be reviewed with the
appropriate people before launch.

This does not require exotic technology. It calls for payment tokenization, data
minimization, and short retention.

## Competing Hypothesis: Integrate Recurring Stores 🐈

A store-specific integration may become more economical as order volume grows.

| Approach | Advantage | Cost that may dominate |
| --- | --- | --- |
| Agent for each purchase | Add stores with less initial development | Retries and supervision |
| Store integration | More control and repeatability | Development and maintenance |
| Hybrid model | Start quickly and optimize high-volume stores | Coordinating both paths |

We propose using agents to start and resolve exceptions, while evaluating APIs,
commercial agreements, or stable automation for recurring stores. An agent is a
way to start without asking anyone's permission; an integration is how to stay.

Jumpseller's endpoint shifts this comparison. For those stores, integration no
longer requires store-specific development: it is a public contract maintained
by the platform. We propose **speaking with Jumpseller before expanding the
pilot**, because its agentic checkout could determine whether Muchi buys through
an API or browser for much of the Chilean catalog.

The decision we seek is approval for a limited pilot to validate demand and cost
before committing to a final fee or promising universal coverage.

Many cats. No blind purchases. 🐈

---

**cangrejo metralleta 🦀**

### Sources

- [Browser Use: Agent quickstart](https://docs.browser-use.com/cloud/agent/quickstart).
- [Browser Use: Official pricing](https://browser-use.com/pricing).
- [Browser Use: Secrets and their limits](https://docs.browser-use.com/cloud/guides/secrets).
- [Browser Use: Human supervision](https://docs.browser-use.com/cloud/agent/human-in-the-loop).
- [Law 21.719 on personal data protection](https://www.bcn.cl/leychile/navegar?idNorma=1209272).
- [AP2: Protocol documentation](https://ap2-protocol.org/).
- [Jumpseller: Storefront MCP](https://jumpseller.com/support/storefront-mcp/).
- [Agentic Commerce Protocols](agentic-commerce-protocols.md), supporting
  material for this proposal.

Prices are from the review conducted for this proposal on September 20, 2026;
verify them before purchasing or setting a commercial fee.
