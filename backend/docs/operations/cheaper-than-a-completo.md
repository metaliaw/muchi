# Cheaper Than a Completo

**English** | [Español](cheaper-than-a-completo.es.md)

## A simple way to sustain the tests

Muchi is testing whether it can help people find collectible cards at good prices. Infrastructure costs money even when a service is small, and every contribution helps keep the tests running while we learn what people find useful.

The idea behind **“cheaper than a completo”** puts the cost on a familiar scale. We are not claiming that this trimester cost less than a completo. First we need the actual total and a local reference price we can explain.

## Period and available evidence

This document uses the most recently completed calendar quarter: **July 1 through September 30, 2026**. The query was made on October 5, 2026. The billing report may receive late-reported usage or adjustments.

`gcloud` confirmed that Muchi’s project has billing enabled and allowed a read-only review of deployed resources. It did not return the period’s charges: the budgets command found the Cloud Billing Budgets API disabled, and this machine does not have `bq` installed. I did not enable APIs or change resources to obtain the amounts.

| Quarterly figure | Status |
| --- | --- |
| Google Cloud charges | Pending lookup in Cloud Billing Reports or a billing export |
| Credits and adjustments | Pending lookup in billing details |
| Net total in CLP | Pending; not estimated |
| Completo equivalent | Pending a published local reference price |

Cloud Billing Reports can show costs by project, service, and period; the cost table report provides invoice details. A BigQuery export supports analysis of charges, credits, and services, but BigQuery usage may itself incur charges. [Billing reports](https://docs.cloud.google.com/billing/docs/reports) · [BigQuery export](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery)

## Infrastructure found

The read-only Google Cloud query found four second-generation functions, one Firestore database, one Cloud Tasks queue, and two Cloud Scheduler jobs in `southamerica-east1`. The code configures a five-minute queue sweep and hourly order release. This describes resources; **it does not establish the bill amount**.

Charges may come from Firestore storage and operations, function invocations and builds, tasks, scheduled jobs, logs, and other project services. The billing report must confirm which generated net charges and which were covered by free allowances or credits.

## Converting the total into “completos”

When we have the quarter’s net cost, we will compare Chilean pesos with an observable completo price from a source and date we disclose. The calculation is **quarterly cost in CLP ÷ reference price in CLP**. Prices vary by shop and city, so we will publish the reference value beside the result instead of presenting it as universal.

## How the community can help

**I would start with small voluntary contributions and direct shop sponsorships.** They are easy to explain and do not require filling Muchi with ads before we know its audience.

- **One-time contribution:** a support page can accept voluntary one-off amounts. We can describe each contribution using the selected reference — for example, “less than a completo” — and show which cost it helps cover.
- **Shop sponsorship:** offer card shops a fixed amount for a short trial. Mark sponsorship clearly as advertising and keep it separate from search ranking, compared prices, and recommendations.
- **Automatic advertising:** wait until audience and user experience are validated. AdSense reviews sites before serving ads and expects original, useful content and clear navigation. Do not ask visitors to click ads. [Prepare your site for AdSense](https://support.google.com/adsense/answer/7299563?hl=en) · [AdSense policies](https://support.google.com/adsense/answer/48182?hl=en)

Ko-fi is one tip-page option. Its published fee is 0% on one-time tips on the free plan; PayPal or Stripe may add their own fees, which vary by country, currency, and account. Confirm availability and the effective cost for the recipient before choosing it. [Ko-fi fees](https://help.ko-fi.com/hc/en-us/articles/360002506494-Does-Ko-fi-take-a-fee)

Before receiving money, we will define who receives it and how it will be recorded. We will not advertise tax benefits or describe a contribution as tax-deductible without checking the applicable case.

## A short, transparent trial

1. Download the closed quarter’s Cloud Billing report and separate usage, credits, taxes, and adjustments.
2. Publish the net cost in CLP, the lookup date, and the reference price used for the completo comparison.
3. Try a contribution button for a short period and talk with a few shops about clearly labelled sponsorship.
4. Report the amount received, payment fees, and how much of the infrastructure it covered.

The first goal is not to finance growth at any cost. It is to learn whether users and shops value Muchi enough to help pay for the tests.
