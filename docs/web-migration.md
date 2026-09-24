[English](web-migration.md) · [Español](web-migration.es.md)

# The Frontend and Its Boundary

Muchi used to have only one possible Interface: the Frontend rendered,
decided and stored State in the same Run. Changing a visual Detail meant
repainting the entire Page, and any other Interface — an App, Widget or Bot —
would have had to implement the Rules again.

Now there are two Parts and a clear Boundary:

```text
Browser ──► web/  (Vue 3 + Vite)       renders, does not decide
                │  fetch /api/...
                ▼
           server/  (FastAPI, the BFF) decides and keeps the Token
                │  Bearer + /v1
                ▼
        Muchi API (Cloud Run)         searches and persists Searches
```

## Why a BFF Instead of Calling the API from the Browser

The API Security token is a Bearer token. Carrying it in the Browser would
publish it: anyone could open Developer tools and copy it. The BFF keeps it
on the Server and exposes only what the Frontend needs.

The Domain rules also stay in one place. Treatment, the Muchi Dollar,
Currency ordering, the cheapest Offer and Cart allocation live in
`server/presenter.py` and reuse `muchi/mtg/`, the same Modules used by `app.py`.
The Frontend receives presentation-ready JSON: Badges with their Text,
Prices with their Currency and the Offer marked as cheapest.

## What the BFF Exposes

| Route | Purpose |
| --- | --- |
| `GET /api/config` | Muchi Dollar, Polling interval and Limits. Never the Token. |
| `GET /api/muchi` | The Cat's phrases: Greetings, Petting, Help and Light. |
| `POST /api/decklist` | Parses the List without using a Search. |
| `POST /api/searches` | Creates a Search. Receives Text and an Idempotency key. |
| `GET /api/searches/{id}` | State and presented Offers in one Request. |
| `POST /api/searches/{id}/cancel` | Cancels the active Search. |
| `GET /api/searches/{id}/cart?shipping=` | The Cart in CLP, allocated by Store. |
| `GET /api/sources` | Source status. |

Errors preserve the existing Domain distinction: a `502` carries
`retriable: true`, and the Frontend retries while retaining received data;
a `409` carries `retriable: false` and stops automatic Polling, because that
Search no longer exists.

The BFF stores no State. Every Request asks the API, so a new Cloud Run
Instance serves the same way as the previous one, and the `?search=<id>` Link
continues working across Sessions and Machines.

## What the Frontend Does

`web/src/App.vue` orchestrates: it loads Configuration, polls every
`poll_seconds` while the Search is pending and stops on a terminal State.
Search history and Theme selection live in `localStorage`; previously they
lived in the Server session and were lost when it closed.

The Palette is the same one from `muchi/mtg/style.py`, now in
`web/src/styles.css`. Dark mode changes CSS variable Values without rewriting Rules.

## Development

```bash
cp .env.example .env      # fill in MUCHI_API_URL and MUCHI_API_TOKEN
./start-web.sh            # BFF on :8000, Frontend at http://127.0.0.1:5173
```

Vite forwards `/api` to the BFF, so the Browser sees a single Origin and no
CORS configuration is needed in Development or Production.

## Deployment on GCP

One Cloud Run Service serves the compiled Frontend and the BFF. The
`Dockerfile` builds `web/` with Node and copies `dist` into the Python Image.

```bash
gcloud builds submit --config cloudbuild.yaml \
  --substitutions=_SERVICE=muchi-web,_REGION=southamerica-east1
```

The Token is mounted from Secret Manager during deployment; it is not
written into the Image or Repository:

```bash
gcloud run services add-iam-policy-binding muchi-web --region=southamerica-east1 \
  --member=allUsers --role=roles/run.invoker
```

A Bucket with a CDN could serve the Frontend more cheaply, but the Token
would still require a second Service. A single Cloud Run service with
`min-instances=0` costs practically nothing while nobody visits, and avoids
both CORS and a second Deployment to keep synchronized.

## The Previous Frontend

It is gone. `app.py`, its Configuration, its Dependency and the Parts that
built HTML for it were removed when Vue replaced it. Its History lives in
the Commits; its Code is no longer in the current tree.
