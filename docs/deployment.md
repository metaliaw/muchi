[English](deployment.md) · [Español](deployment.es.md)

# Deploying Muchi

One command publishes the whole platform, in the order its parts depend on each
other:

```sh
./deploy.sh          # the API, then the Web
./deploy.sh api      # the API only
./deploy.sh web      # the Web only
```

Every option after the target goes to the API scripts, so
`./deploy.sh api --dry-run` previews the API without touching Google Cloud.

## Two services, two deploys

| Target | Runs | Goes to |
| --- | --- | --- |
| `api` | `backend/deploy-worker.sh`, `-sweeper.sh`, `-order-release.sh`, `-api.sh` | Cloud Functions |
| `web` | `deploy-web.sh`: builds the image from `bff/` and `frontend/`, then Firebase Hosting | Cloud Run `muchi-web` and Firebase Hosting |

The API goes first because the Web consumes its routes and fields. Deploying
the Web against an older API is what turns a new field into a blank screen.

`api` deploys the worker, the sweeper and the order releaser together with the
API on purpose: they are built from the same source, and a new API next to an
old worker disagrees about what a search item looks like.

## What `api` leaves alone

Infrastructure (accounts, Firestore, the queue and its TTL policies) changes
rarely and asks for more permissions, so `./deploy.sh api` does not touch it.
The first time, or when it changes, run `backend/deploy.sh`, which does
everything. The API side is described in
[Deployment with Cats](../backend/docs/operations/deployment.md).

## The Token

The BFF is the only piece that holds the API token: Cloud Run mounts it from
the `muchi-api-token` secret, and the browser never sees it. The same secret is
mounted by the worker, the API and the BFF, so rotating it is a coordinated
deploy, not a single write. Rotate from `backend/` with `./rotate-secret.sh`;
[Deployment, Revisions, and Secret Rotation](../backend/docs/operations/deployment-revisions-secrets.md)
explains the incident that made the order matter.

`./get-secret.sh` downloads the current token into `bff/.env` for local work.

## Before the first deploy from a new checkout

- `gcloud auth login` and an active project (`gcloud config get-value project`).
- `firebase` and `npm` installed; `deploy-web.sh` checks both.
- `bff/.env` is only for local runs. A deploy reads nothing from it.

See the [architecture](architecture.md) for why the Web is one image and one
origin.
