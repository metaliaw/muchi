# Google Cloud Startup Credits

[English](google-cloud-startup-credits.md) | [Español](creditos-google-cloud-startups.md)

This document records what was learned on September 20, 2026, about the Google
for Startups Cloud Program and why Muchi cannot apply yet. It describes no work
that has been done or committed to; it is a note about when to revisit the topic.

## Why It Came Up

The original question was whether Muchi could benefit from the collaboration
between Midnight and Google Cloud ([announcement](https://midnight.network/blog/google-cloud-midnight-ecosystem-collaboration)).
The answer is no. Midnight is a privacy blockchain using zero-knowledge proofs,
designed for financial institutions, governments, and healthcare systems that
need to share sensitive data without revealing it. Muchi compares public prices
from public stores: there is no private data to protect, no counterparty
requiring confidentiality, and no on-chain payment.

The only useful part of that announcement is Google Cloud credits, which are
also available without blockchain. The rest of this document covers the general
program.

## The Three Tiers

| Tier | Benefits | Requirements |
| --- | --- | --- |
| Start | Up to US$2,000 in Google Cloud and Firebase credits | Company founded less than 5 years ago, no prior credits beyond the free trial, **no institutional investor** |
| Scale | Year 1: 100% of spend up to US$100,000; Year 2: 20% up to another US$100,000 | Pre-seed through Series A equity funding from an institutional investor; SAFE agreements qualify |
| Scale AI | Year 1: up to US$250,000, on top of Scale benefits | Scale requirements, plus AI is core to the primary product, company founded less than 10 years ago, and no more than US$5,000 in prior credits |

Sources: [eligibility and benefits](https://cloud.google.com/startup/benefits) and
[early-stage funding](https://cloud.google.com/startup/early-stage). Verify
amounts before applying; they change.

## What Applies to Muchi

**Nothing today.** The program requires a company, and Muchi is not incorporated
yet. That is the actual blocker, not credit or funding requirements.

Once incorporated, the relevant tier is **Start**: US$2,000. This sounds small
next to Scale, but Muchi's spending is limited. Current infrastructure includes
Firestore, Cloud Functions, Cloud Tasks, Cloud Scheduler, Secret Manager,
Artifact Registry, and Storage. At today's scale, US$2,000 could cover an entire
year.

One detail could disqualify Muchi: the requirement says applicants must not have
received previous credits beyond the free trial. The US$300 trial does not
count against it; credits from another Google program do.

## When to Revisit Scale

Scale and Scale AI depend on funding, not the product: without an institutional
investor, access is unavailable regardless of the use case.

If Muchi ever raises a round, the Scale AI case is not the price comparison—that
is not an AI startup—but the [web agent purchases proposal](web-agent-purchases-proposal.md).
Purchases executed by autonomous agents fit the profile the program seeks.
Remember this if both the proposal and funding move forward.

## How to Apply

The Start tier application is at
[cloud.google.com/startup](https://cloud.google.com/startup). Scale applicants
usually enter through a referral from an investor, accelerator, or program
partner.
