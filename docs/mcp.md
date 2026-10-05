**English** · [Español](mcp.es.md)

# Guide to MUCHI's Public MCP

MUCHI exposes its catalog and store offers through the Model Context Protocol
(MCP). A compatible agent can search for cards from a chatbot; it does not
need to open the website or know the MUCHI API token.

## Connect a Chatbot

Add a remote server in the chatbot's MCP settings with these values. Field
names vary by client:

| Field | Value |
| --- | --- |
| Name | MUCHI |
| Transport | Streamable HTTP |
| URL | `https://muchitcg.cl/mcp/` |
| Authentication | None |

The endpoint is public and requires no account, API key or token. Configure it
once, then request searches in the conversation. The chatbot must support
remote MCP servers over Streamable HTTP.
The BFF keeps `MUCHI_API_TOKEN` on the server; never give it to the chatbot.

After connecting, confirm that the `search_cards` and `get_search_results`
tools appear.

For local development, connect to `http://127.0.0.1:8000/mcp/` after starting
the BFF as described in the [README](../README.md).

## Request a Search

Name MUCHI and the cards you want. If you omit options, the defaults are Magic,
exact matching and single cards.

```text
Search MUCHI for these Magic cards:
1 Sol Ring
4 Lightning Bolt
2 Counterspell
Wait until the search is complete, collect every page and sort by ascending
price. Return the store, price, stock and link.
```

For sealed products, say so explicitly:

```text
Search MUCHI for a sealed box of [product name].
Wait for the final results and sort by ascending price.
```

You do not need to repeat the URL with each request. If the chatbot has other
servers connected, naming MUCHI helps it choose this one.

## How Search Continues

`search_cards` starts a search in the MUCHI API and returns a `search_id`. The
server keeps processing the work even before the chatbot checks the result.

The agent calls `get_search_results` with that ID. While `state.done` is
`false`, it waits briefly and checks again. When `has_more` is `true`, it sends
the returned `cursor` as `after` to collect the next page. It finishes when
the search is done and no pages remain.

Ask the chatbot to wait for completion. If the conversation is interrupted,
save the `search_id` and ask it to resume while the API still retains the
results.

## Offer Ordering

Request the order you want. `get_search_results` accepts:

- `price_asc`: lowest to highest CLP price; this is the default.
- `price_desc`: highest to lowest CLP price.

Sorting is within each card type, and offers without a CLP conversion appear
last. The response includes store, price, currency, stock, link and suspicious
price signals when the source provides them.

MCP does not receive the requester's location. The website can sort by distance
with browser permission; that location is not sent to MUCHI. MCP can sort by
price, but not by "near me."

## Tools

### `search_cards`

Starts a search and returns its `search_id` and initial state. The search runs
in the background and is retained by the MUCHI API.

| Argument | Type | Default | Use |
| --- | --- | --- | --- |
| `decklist` | text | — | One card per line; accepts `4 Sol Ring` or `Sol Ring`. |
| `game` | text | `magic` | A game supported by the MUCHI API. |
| `match` | `exact` or `includes` | `exact` | How to match the card name. |
| `kind` | `single` or `sealed` | `single` | Single cards or sealed products. |

Lists can contain up to 100 entries, with quantities from 1 to 99. Lines that
cannot be parsed cause the tool to return an error; they are not silently
discarded.

Example `decklist`:

```text
1 Sol Ring
4 Lightning Bolt
2 Counterspell
```

The response includes the search ID, state, progress, entries found and entries
that failed. Save `search_id` to read the offers.

### `get_search_results`

Reads the current state, up to 50 list entries and a flat list of their offers.
`item_position` links each offer to its entry.

| Argument | Type | Default | Use |
| --- | --- | --- | --- |
| `search_id` | text | — | ID returned by `search_cards`. |
| `after` | integer | `0` | Cursor from the previous page. |
| `match` | `exact` or `includes` | `exact` | Use the mode sent when creating the search. |
| `sort_by` | `price_asc` or `price_desc` | `price_asc` | Sort by CLP price within each card type. |

The page contains up to 50 list entries and their offers. `item_position` links
each offer to its entry. Use the cursor to request the next page.

## Limits and Security

- MCP only starts searches and reads their results; it does not create orders
  or make purchases.
- Searches use the existing validation, presentation and API.
- Any agent that can reach the URL can start searches. The internal API and its
  token stay on the server.
- The transport accepts the website hosts and local development addresses to
  reject requests with an unrelated Host header.

The API retains results and applies its own expiration. An agent should keep
the returned ID while those results remain available.
