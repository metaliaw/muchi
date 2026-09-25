# AP2 and ACP: Fit Analysis for Muchi

[English](ap2-acp-fit-analysis.md) | [Español](ap2-acp-fit-analysis.es.md)

Analysis for discussion · September 24, 2026

## Why This Document Exists

[Agentic Commerce Protocols](agentic-commerce-protocols.md) surveys AP2, UCP,
ACP, and MCP in general. The
[web agent purchases proposal](web-agent-purchases-proposal.md) already
concludes AP2 is not a path within the pilot timeline. This document goes one
level deeper on the two protocols that carry the actual payment step, AP2 and
ACP, and asks a narrower question: does either one apply to Muchi's stated
model — Muchi purchases from a store using Muchi's own payment method, then
charges the buyer — today, or on a foreseeable horizon.

## AP2 in One Paragraph

The Agent Payments Protocol, announced by Google in September 2025 and donated
to the FIDO Alliance in April 2026, links three signed credentials: what the
buyer authorized, which cart they accepted, and which payment method was used.
It creates non-repudiable evidence between buyer, agent, and merchant. It is
implemented by the commerce platform and payment processor — Shopify enabled it
by default for eligible United States merchants in March 2026 — not by an
individual store or by us.

## ACP in One Paragraph

The Agentic Commerce Protocol, proposed by OpenAI and Stripe, connects an
agent-mediated purchase to a merchant and a payment provider while the merchant
stays the seller of record. Like AP2, it depends on the merchant's platform and
payment processor supporting it; an agent cannot add ACP to a store that has
not implemented it.

## Why Neither Fits Muchi's Model

Both protocols solve the same underlying problem: giving the **buyer** signed,
non-repudiable evidence that they authorized a specific purchase, so buyer and
merchant cannot later disagree about what was ordered or paid. That problem is
real for a platform connecting a buyer's payment method directly to a merchant.

It is not Muchi's problem today. In the proposal's model, the buyer never pays
the store — Muchi does, with a payment method Muchi owns, and Muchi separately
charges the buyer for the total plus a fee. The authorization chain AP2 and ACP
formalize (buyer → merchant) does not exist between Valentina and the store; it
exists between Valentina and Muchi, and that leg is a normal, single-party
charge with no agentic ambiguity to prove.

There is also a hard availability gate, independent of the model question:
neither protocol is implemented by any store on Muchi's current list. AP2
requires the merchant's platform and processor to support it; none do. ACP has
the same requirement and, as of this analysis, no confirmed deployment among
Chilean card and TCG stores.

## When This Could Change

The fit question and the availability question are separate, and either can
change on its own:

- **If Muchi's model changes** to the buyer paying the store directly through
  Muchi as an intermediary (rather than Muchi purchasing on the buyer's
  behalf), the AP2/ACP authorization chain becomes directly relevant, because
  Muchi would then need exactly the evidence they provide.
- **If a Chilean store's platform adds AP2 or ACP** independent of any change
  on Muchi's side — for example, if Jumpseller extended its
  [Storefront MCP](https://jumpseller.com/support/storefront-mcp/) checkout
  work to include one of these protocols — it would be worth revisiting for
  that store specifically, but the model mismatch above would still apply
  unless Muchi's purchasing model also changed.

Neither condition currently holds, so this stays a watch item, not a build
item.

## What Actually Blocks the Proposal Today

Per the [web agent purchases proposal](web-agent-purchases-proposal.md), the
near-term path is the supervised browser pilot and the Jumpseller MCP endpoint
for the stores that support it — not a payment authorization protocol. AP2 and
ACP would reduce risk in the buyer-to-merchant leg if Muchi ever became that
kind of intermediary; they do not reduce risk in the leg Muchi actually
operates in today.

## Sources

- [AP2 Protocol](https://ap2-protocol.org/).
- [Agentic Commerce Protocols](agentic-commerce-protocols.md), the general
  survey this document narrows.
- [Web agent purchases proposal](web-agent-purchases-proposal.md), which
  defines Muchi's proposed purchasing model.
- [Jumpseller Storefront MCP](https://jumpseller.com/support/storefront-mcp/).
