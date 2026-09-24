[English](advertising-en.md) · [Español](advertising.md)

# Advertising, End to End

Muchi displays Google AdSense Ads among Offers. This Document explains
what is required, in what Order, which parts are automated and which parts
Google decides by reviewing the Site.

## The Three Identifiers

They are often confused, and confusion costs days.

| Identifier | Example | Purpose |
| --- | --- | --- |
| Publisher ID (Client) | `ca-pub-6368656861543000` | Identifies the Account. One permanent ID. |
| Ad unit ID (Slot) | `1234567890` | Identifies a Space. One for each placement. |
| Customer ID | `3286538783` | Billing and Support. Not used in Muchi. |

None is a Secret. The Publisher ID travels with every Page, and `ads.txt`
declares it publicly. They therefore use Environment variables rather than
Secret Manager, which holds only `MUCHI_API_TOKEN`.

## The Flow

Four Steps; only the last is entirely yours.

```text
1. Account       create the Account and register the Domain  -> interface
2. Review        Google visits and approves the Site         -> wait
3. Ad unit       create the unit; Google assigns the Slot    -> interface
4. Deployment    MUCHI_ADSENSE_SLOT=<slot> ./deploy.sh        -> yours
```

The AdSense API is used for reading. `sites` and `adunits` provide `list`
and `get`; `adunits.create` exists but is reserved for *AdSense for Platforms*
publishers and rejects ordinary Accounts. No Script skips Steps 1 and 3;
pretending otherwise only hides the Wait.

The other half, which determines Step 2, can be automated.

## What the Review Checks

Google visits the registered Domain, rather than Hosting's `.web.app`
address, and checks four things. Each can fail silently and cost another
Review round.

- The Homepage responds with 200 over HTTPS. Review cannot approve a Site
  it cannot enter.
- `ads.txt` at the Domain root declares the Publisher. The BFF serves it in
  [`server/main.py`](../server/main.py); the Domain must route there.
- The `google-adsense-account` Tag appears in `<head>`, in
  [`web/index.html`](../web/index.html).
- The `adsbygoogle.js` Loader points to the same Publisher in that File.

The Identifier appears in three places that do not read one another:
the compiled `<head>`, the BFF's `/ads.txt` and the Frontend's `/api/config`.
If they diverge, Muchi requests Ads for one Publisher while declaring
another. It produces no Error; approval simply does not arrive, and you
learn about it days later.

## The Scripts

```bash
./setup-ads.sh    # full onboarding: prepares, checks and reports what is missing
./check-ads.sh    # status only; repeat while waiting
```

`setup-ads.sh` enables `adsense.googleapis.com`, obtains the ADC Scope —
asking first, because it rewrites the Machine's default Credentials —
and asks AdSense which Domain you registered, then verifies that Domain.
It checks the four requirements, compares the three Identifiers, reads
Account alerts and reports what is missing and who owns the next action.
It is idempotent: rerunning changes nothing already correct.

On Windows, `setup-ads.cmd` and `check-ads.cmd` do the same.

### Two API Frictions

Neither diagnoses itself; both are already handled by the Scripts.

The Scope. Default ADC does not include AdSense. Without it, a `403`
mentions Scopes without explaining the fix:

```bash
gcloud auth application-default login \
  --scopes=https://www.googleapis.com/auth/adsense,https://www.googleapis.com/auth/cloud-platform
```

The Quota project. AdSense is not Cloud, but its API is charged to a Cloud
Project. Without the Header, the Request is attributed to gcloud's generic
Project, and the `403` blames a Project that is not yours:

```text
-H "x-goog-user-project: YOUR-PROJECT"
```

## States and Their Meaning

`setup-ads.sh` and `check-ads.sh` display the Account, Client and Site `state`.
Progress happens one step at a time:

- `GETTING_READY`: Google is reviewing. Wait; it usually takes days.
- `READY`: unlocks the next Step. Once Client and Site are both `READY`,
  the Interface allows creating an Ad unit.
- `Bloques: ninguno` (no Ad units): there is no Slot to deploy yet.

A Policy block does not appear in `state`; it lives in Alerts, which the
Scripts read separately. Ignoring them makes a fixable problem look like
routine Waiting.

## Creating the Ad Unit

Once the Site is `READY`, choose:

**Ads → By ad unit → Display → Responsive.**

Google's Code contains `data-ad-slot="1234567890"`. Those ten Digits are
the Value. Do not copy the `<script>` or `<ins>`: those are built by
[`GoogleAd.vue`](../web/src/components/GoogleAd.vue).

The same Number is available through the API as the unit's
`reportingDimensionId`. `check-ads.sh` prints it and ends with a ready-to-copy Command.

## How the Frontend Chooses What to Render

Three conditions live in [`App.vue`](../web/src/App.vue) and
[`AdSpot.vue`](../web/src/components/AdSpot.vue):

1. `environment === 'production'`. Outside Production, the same Algorithm
   chooses a placement but renders a Placeholder, letting you test the
   Selection without affecting Google's Metrics.
2. Client and Slot are present. If either is missing, Muchi falls back to
   an internal Promotion instead of requesting an unavailable Ad.
3. Allocation. With a Sponsor configured, one in four Searches shows the
   Sponsor and three show AdSense. Selection depends on the Search ID's
   Hash and stays stable across Polls of the same Search.

If Google's Script fails to load because of an Ad blocker or Network
failure, `GoogleAd.vue` leaves no Hole: it says Muchi is still looking for Offers.

## The Variables

All are supplied as Environment variables, empty by default. Unexported
settings are omitted by the Frontend.

| Variable | Behavior When Missing |
| --- | --- |
| `MUCHI_ADSENSE_CLIENT` | Falls back to the fixed Publisher in `server/main.py`. |
| `MUCHI_ADSENSE_SLOT` | Ads stay off; the internal Promotion appears. |
| `MUCHI_SPONSOR_NAME` / `_TEXT` / `_URL` | No Sponsor: AdSense gets all four Searches. |

```bash
MUCHI_ADSENSE_SLOT=1234567890 ./deploy.sh
```

Export a whole `.env` with `set -a; . ./.env; set +a` before invoking it.

`deploy.sh` warns before uploading anything, and never stops Deployment:

```text
~nya?~ Este Deploy va sin Anuncios: MUCHI_ADSENSE_SLOT esta vacio
       ./check-ads.sh dice si Google ya Asigno un Bloque
```

This says the Deployment has no Ads because the Slot is empty and points
to `check-ads.sh` to see whether Google has assigned one. It is deliberately
a Notice: while Google reviews the Account, no Slot exists. Blocking
Deployment would leave Muchi unpublished over something outside our control.

## When Something Goes Wrong

| Symptom | Likely Cause |
| --- | --- |
| `403` with `ACCESS_TOKEN_SCOPE_INSUFFICIENT` | ADC lacks the Scope. |
| `403` naming someone else's Project | Missing `x-goog-user-project`. |
| `403 The caller does not have permission` | The Publisher ID is not yours. |
| `404` for `ads.txt` | The BFF serves it only with a `ca-pub-` Client. |
| Site stays `GETTING_READY` for weeks | Check Alerts and the four Review requirements. |
| Slot exists but no Ad appears | Service is not `production`, or the Slot never reached Deployment; inspect `/api/config`. |
| Slot and `production` are set, but the Space is empty | Normal during a new unit's first hours, and with an Ad blocker enabled. |
