# Deployment with Cats

[English](deployment.md) | [Español](deployment.es.md)

## Summary

`deploy.sh` and `deploy.cmd` run `gcloud` directly using your `gcloud auth login`
session. They use the active Google Cloud CLI project or accept `--project`. Go
does not need to be installed locally to deploy; GCP builds the functions from
source.

```sh
./deploy.sh
```

```bat
deploy.cmd
```

There is no automatic deployment from GitHub. Each deployment runs from a
machine with the operator's session and [permissions](#permissions).

## The Four Components

On Unix, each component can also be deployed separately. `deploy.sh` runs them
in this order:

```sh
./deploy-infra.sh
./deploy-worker.sh
./deploy-sweeper.sh
./deploy-api.sh
```

The order matters. The API and sweeper discover the URL of the deployed worker:
they need it to exist but do not deploy it again.

A code change only requires the final three scripts. Infrastructure rarely
changes, and separating it also reduces permissions needed for everyday
deployments.

## What Each Script Does

`deploy-infra.sh` enables services, reuses or creates accounts, Firestore, and
the queue, and requests TTL policies for the five collections. An existing
Firestore instance keeps its location.

`deploy-worker.sh` deploys the private `ProcessSearch` worker and grants Cloud
Tasks permission to invoke it through OIDC.

`deploy-sweeper.sh` deploys the sweeper and its Cloud Scheduler wakeup every five
minutes.

`deploy-api.sh` deploys `ServeAPI`, publicly reachable at the IAM layer and
protected by the application token, with the actual worker URL. GCP does not run
a resident worker.

## Permissions

The deployer needs these roles in the project:

```text
roles/serviceusage.serviceUsageAdmin    enables APIs
roles/iam.serviceAccountAdmin           creates runtime accounts
roles/resourcemanager.projectIamAdmin   grants their roles
roles/iam.serviceAccountUser            deploys as those accounts
roles/cloudfunctions.admin              deploys functions
roles/run.admin                         manages revisions behind them
roles/storage.admin                     uploads the build source
roles/cloudtasks.admin                  manages the search queue
roles/cloudscheduler.admin              manages the sweeper wakeup
roles/secretmanager.admin               reads and versions the token
roles/datastore.owner                   manages Firestore, indexes, and TTL
```

Grant a role like this, repeating the command for each role:

```sh
gcloud projects add-iam-policy-binding my-project \
  --member='user:operator@example.cl' --role=roles/cloudfunctions.admin
```

Only `deploy-infra.sh` needs the first three roles. Someone deploying only the
worker, sweeper, and API can omit them, preventing everyday deployments from
rewriting project permissions by accident.

The service accounts created by `deploy-infra.sh` are different: they govern
runtime access in GCP, not deployment. The [architecture diagram](architecture.md)
shows where each identity is used.

This list was inferred from the scripts, not tested in a real session with
minimum permissions. If a new account is missing or has an extra role, update
this list.

The project must exist and have billing enabled.

## Options

Shared values live in `config/deploy.env`: the `muchi-serve-api` and
`muchi-process-search` functions, `southamerica-east1` region, `muchi-searches`
queue, service accounts, and limits. Select another project, region, or Secret
Manager reference with:

```sh
./deploy.sh --project my-project --region southamerica-east1 --token-secret muchi-api-token
```

Add `--dry-run` to review commands without changing GCP.

## The Token

Both services receive the live reference `muchi-api-token:latest`: each new
instance reads the newest enabled version, so rotating the secret does not
require a deployment. The deployment checks that the latest version exists and
is enabled, and warns if you pass a different version.

The secret must exist before the first deployment. Otherwise scripts fail while
checking that its latest version is enabled, after deployment has begun.

To create the secret or add a version, use `--token-file` with the absolute path
to a private file outside the repository, with no trailing newline. The token is
not printed or passed as a command-line value. The scripts do not load `.env`.

Rotating the secret alone is not enough. The
[deployment, revisions, and secrets incident](deployment-revisions-secrets.md)
explains why a correct rotation must also move traffic and update every consumer.

## What Gets Deployed and Its Limits

Stores from `config/stores.yaml` are included in the deployed source; the Go
runtime reads them from `serverless_function_source_code/config/stores.yaml`.

Default limits are 512 MiB, a 1800-second timeout, concurrency 1, and up to 5
instances per function. The queue allows 1 dispatch per second and 2 concurrent
tasks. HTTP tasks have a deadline of 1800 seconds
(`MUCHI_TASK_DEADLINE_SECONDS`) to finish searches through long Jumpseller
pagination. Use the asynchronous flow locally for these searches.

## Output

The scripts include cats and status lines, together with native `gcloud` progress
and errors. An error stops the script; resources already created remain so the
operator can retry. At the end, the API URL and its `/v1/health` route are printed.
