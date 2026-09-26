# Agentic Commerce Protocols

**English** | [Español](agentic-commerce-protocols.es.md)

## The Problem They All Solve

An agent can browse a catalog and fill a cart, but a purchase transfers money and
creates obligations. The buyer needs evidence of what was authorized, what was
ordered, and whether the seller accepted the payment. The protocols below place
trust boundaries between the buyer, agent, merchant, and payment provider.

## AP2: The One That Reached an Industry Standard

The Agent Payments Protocol (AP2) records authorization as signed evidence for
the buyer's intent, the accepted cart, and the payment instrument. Google
announced it in September 2025 and donated it to the FIDO Alliance in April
2026. Mastercard, PayPal, and American Express are among its supporting
organizations.

AP2 is implemented by the commerce platform and payment processor, not by an
agent or merchant independently. Shopify enabled agentic purchases by default
for eligible United States merchants in March 2026. The Chilean stores in
Muchi's list do not currently accept AP2.

## UCP: The Layer Above

The Universal Commerce Protocol (UCP) describes a common commerce layer for
discovery, checkout, and post-purchase operations. It is intended to let agents
and commerce platforms exchange capabilities without a custom integration for
every pair.

UCP can describe a shopping flow, while payment still depends on a supported
checkout and authorization mechanism. A shared protocol does not make every
store or payment provider compatible automatically.

## ACP: OpenAI and Stripe's Proposal

The Agentic Commerce Protocol (ACP) connects an agent-mediated purchase to a
merchant and a payment provider. It defines how product discovery, checkout, and
payment authorization can be coordinated while the merchant remains the seller
of record.

Availability depends on the participating platform and payment integration.
ACP does not by itself give an agent a way to buy from a store that has not
implemented it.

## MCP: Not a Commerce Protocol, but Available Here

The Model Context Protocol (MCP) connects a model to tools and data. It does not
define commerce authorization or settle payments, but a store can expose catalog
and cart operations through an MCP server.

Jumpseller provides a public Storefront MCP endpoint across its stores. It can
return structured product and variant data, stock, and direct cart or purchase
URLs. Its current search quality is limited, and the checkout tools are still
under development. It helps with discovery and cart creation but does not yet
complete a paid order.

## What Reaches Muchi's Stores

| Protocol | What it provides | Current reach in Muchi's store list |
| --- | --- | --- |
| AP2 | Signed authorization for intent, cart, and payment | Not currently accepted by listed Chilean stores |
| UCP | Shared commerce capabilities across discovery and checkout | No verified implementation among listed stores |
| ACP | Coordination among agent, merchant, and payment provider | No verified implementation among listed stores |
| MCP | Tool access to catalog and commerce operations | Jumpseller exposes catalog, stock, and cart links; checkout is pending |

## What We Propose to Do

Use the Jumpseller endpoint to validate structured catalog lookup and cart
creation with the stores already on the platform. Speak with Jumpseller about
its checkout tools before expanding the supervised purchase pilot.

For other stores, begin with a supervised browser flow that stops before
payment. Do not build a permanent solution around blind retries or checkout
automation when the platform may provide an explicit protocol.

The [web agent purchases proposal](web-agent-purchases-proposal.md) compares the
pilot, costs, safeguards, and store-specific integration options.

## Sources

- [AP2 Protocol](https://ap2-protocol.org/).
- [Jumpseller Storefront MCP](https://jumpseller.com/support/storefront-mcp/).
- [Web agent purchases proposal](web-agent-purchases-proposal.md), which records
  the source set and review date for its protocol and pricing claims.
