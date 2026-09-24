[English](share-store-stock.md) · [Español](share-store-stock.es.md)

# How to Share Your Store's Stock with Muchi

If you run a Card store and want its Offers to appear on Muchi, you can
share your Inventory through Moxfield lists or your website. This Guide
explains what information to prepare so the Integration can be evaluated.

Sources are integrated into the Muchi API. The Frontend displays their
Results; it has no Store registration form and does not import Inventories
directly. Confirm the availability of each Integration with the Service maintainers.

## Option 1: Moxfield Lists

Prepare Links to the Lists representing the Inventory you want to publish.
They must be readable without signing into your account.

Include this information:

- Store name and List links.
- Available quantity of each Card and, where relevant, Edition, Language,
  Condition and Finish.
- Currency and Pricing method: state whether you use a reference Price,
  an exchange Rate or another rule. Describe separate rules for regular
  and foil Cards when applicable.
- Store link or a channel where someone can inquire and buy.
- How often you update the Lists and how you record sold-out Products.

Clarify whether Quantities represent available Stock or simply the contents
of a reference List. A List alone neither sets a sale Price nor guarantees
that its Cards are available.

Share this information when requesting Integration. The team must confirm
how the API will read the Lists and represent Prices and Purchase links.

## Option 2: Your Website

Share your Store's address and some Product links that identify their
variants. If you have a Catalog or Stock API, include its documentation
and an example Response without Credentials.

Useful information for each Offer:

| Field | What It Should Represent |
| --- | --- |
| Card | Name and, if available, a stable Identifier. |
| Variant | Edition, Finish, Condition and Language. |
| Price | Sale amount and Currency. |
| Stock | Available quantity or availability Status for that variant. |
| Link | Page where the Offer can be viewed or purchased. |

If your site exposes an API, explain how to search for a Card, page through
the Catalog and recognize a sold-out Product. Include Request limits and
any Authentication requirements.

If you only have Product pages, share examples with and without Stock.
The team will evaluate whether reliable Queries are possible and which
Adapter the API needs. Compatibility with every platform is not assumed.

## Request Integration

Open a request in the [Project Issues](https://github.com/metaliaw/muchi/issues)
with your Store name, chosen option and the information above. If you
cannot access the Repository, coordinate through the contact channel where
Muchi was shared with you.

Do not include Passwords or Tokens in the request. If Integration requires
a Credential, arrange delivery through a private channel and restrict its
permissions to reading the Inventory you want to share.

Before considering Integration complete, check with the team that Prices,
variants, availability and Links match your Store. Agree on how to
communicate changes to your Lists or website.

## What People Searching for Your Cards Will See

Muchi displays Offers received from the API, including Store, Price,
Currency, Stock status and Link. Purchases are completed outside Muchi
through the Offer link. Keeping Inventory current helps prevent sold-out
Cards and outdated Prices from appearing.

[Back to the README](README.md).
