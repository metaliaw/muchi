[English](when-muchi-buys.md) · [Español](when-muchi-buys.es.md)

# When Muchi Buys

September 2026 · a Plan spoken aloud, not a date

Today Muchi sells nothing. It compares Offers, builds a Cart in Pesos and leaves
you at each Store's door; you make the Purchase yourself, once per Store,
with your Details and your Card. That works, and it is not broken.

But it is a strange ending to a good Search. Muchi has just read twelve Stores
for you, and the reward is a List of twelve Tabs. **We want Muchi to Buy**:
choose the Cart once, pay once, and receive the Cards together. We are working
on it, and this Document explains how far we have got — less far than it
sounds — and what remains unresolved.

## The Muchi Dollar Already Said It

The [Muchi Dollar](../README.md#the-muchi-dollar) is not the market exchange
rate. It is a commercial Rate, and its components have been written into
[`config/rates.defaults.yaml`](../config/rates.defaults.yaml) from the start:

> the Cost of bringing in the Card, and the Margin for the Intermediaries who
> will make the Purchase **when Muchi buys**.

That "when" was not decoration. The Number is sized for a Muchi that buys,
rather than one that only compares. That is why it does not follow the day's
Dollar rate: it pays for an Operation, beyond the currency Conversion.

Today those Intermediaries are People. Someone enters the Store, fills out
the Form, pays, waits and forwards the purchase. That makes the Number what it is.

## The Plan: Agents

The idea is to automate that part with Agents: Programs that complete each
Store's Checkout — navigate the Form, confirm the Variant, pay and retrieve
the Receipt — instead of a Person doing it twelve times.

This is not an exotic idea; it is the same Work, done by something that does
not get tired at the fifth Store. It is also the Cost component with the most
room to change, because it is the only one that currently grows with the
Number of Stores.

**We are still examining the Implementation and Costs.** That is literally
the current state. No Agent is running, no Store purchase has been made this
way, and we do not yet know how much it costs.

## What Remains Unresolved

- The Cost of an Agent per Purchase. Navigating a Checkout is not free; if an
  Agent costs more than the Person it replaces, there is nothing to automate.
  That Number decides the whole Project.
- A partially completed Purchase. Twelve Stores, eleven payments succeed and
  one fails. A mistaken Comparator costs you a Visit; a mistaken Buyer leaves
  you out of pocket with half a List. This answer matters more than Speed.
- Payments and Credentials. Buying requires Payment methods and Shipping
  details. Where they live, who sees them and what is stored are Security
  decisions, and cannot be rushed.
- The Stores. Buying for you differs from linking to a Store. A Store may
  want either one without the other, and we should ask.
- Where the Cost belongs. If an Agent is cheaper than a Person, does the Muchi
  Dollar go down, or do the Savings pay for something else? If charged
  separately, is it a visible Service fee in the Cart, outside the Rate?

That last question deserves a clear explanation. It is why this Document exists.

## Costing, Now and Later

Today the Cart Total has three visible components:

| Component | Source |
| --- | --- |
| Card Price | What the Store publishes, in its Currency. |
| Shipping | One charge per Store, not per Card. |
| Exchange Rate | The Muchi Dollar, when an Offer arrives in Dollars without its own Rate. |

A Muchi that buys adds another component: **the Cost of making the Purchase**.
The open question is whether that Cost stays inside the Rate, where it is
hidden today, or becomes visible under its own Name.

We lean toward the latter. A Number covering two different things is hard to
discuss: if the Muchi Dollar rises, nobody knows whether bringing the Card or
buying it became more expensive. Separating them makes the Price longer to
read and much easier to defend.

That is a preference, not a Decision. Once decided, it will be written here
and the README will change.

## What We Do Not Promise

There is no Date or Waiting list. None of this is half built and waiting for
a Button. If tomorrow we decide the Costs do not work, the Plan falls through,
and this Document will explain why. That is the other half of its purpose.

Meanwhile, Muchi compares and you still make the Purchase. What you see in
the Cart is what you pay at the Stores.

[Back to the README](../README.md).
