#!/usr/bin/env sh
# Deploys and Schedules the Queue Sweeper.
set -eu
cd -- "$(dirname -- "$0")"
. ./config/deploy.env
. ./deploy-common.sh
DEPLOY_COMPONENT="Barredor"
prepare_deploy "$@"
read_worker

cloud functions deploy "$SWEEPER_NAME" --gen2 --trigger-http --no-allow-unauthenticated \
	--runtime="$RUNTIME" --region="$REGION" --source=. --entry-point=SweepQueue --ignore-file=.gcloudignore \
	--service-account="$API_EMAIL" --build-service-account="projects/$PROJECT/serviceAccounts/$BUILD_EMAIL" \
	--memory="$MEMORY" --timeout=120s --min-instances=0 --max-instances=1 --concurrency=1 \
	--set-env-vars="GOOGLE_CLOUD_PROJECT=$PROJECT,MUCHI_STORES_CONFIG=serverless_function_source_code/config/stores.yaml,MUCHI_TASK_REGION=$REGION,MUCHI_TASK_QUEUE=$TASK_QUEUE,MUCHI_TASK_URL=$WORKER_URL,MUCHI_TASK_SERVICE_ACCOUNT=$TASK_EMAIL,MUCHI_SCRY_ENABLED=$MUCHI_SCRY_ENABLED" \
	--set-secrets="MUCHI_API_TOKEN=$MUCHI_API_TOKEN" --format=none
if "$DRY_RUN"; then
	SWEEPER_URL="https://$SWEEPER_NAME-PREVIEW.run.app"
	SWEEPER_SERVICE=$SWEEPER_NAME
else
	SWEEPER_URL=$(cloud functions describe "$SWEEPER_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.uri)')
	SWEEPER_SERVICE=$(cloud functions describe "$SWEEPER_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.service)')
fi
cloud run services add-iam-policy-binding "${SWEEPER_SERVICE##*/}" --region="$REGION" \
	--member="serviceAccount:$TASK_EMAIL" --role=roles/run.invoker --condition=None --format=none
cloud services enable cloudscheduler.googleapis.com --format=none
existing_job=$(cloud scheduler jobs list --location="$REGION" --filter="name:$SWEEPER_JOB" --format='value(name.basename())')
job_action=create
if [ "$existing_job" = "$SWEEPER_JOB" ]; then job_action=update; fi
cloud scheduler jobs "$job_action" http "$SWEEPER_JOB" --location="$REGION" \
	--schedule="$SWEEPER_SCHEDULE" --uri="$SWEEPER_URL" --http-method=POST \
	--oidc-service-account-email="$TASK_EMAIL" --oidc-token-audience="$SWEEPER_URL" \
	--attempt-deadline=120s --format=none
printf '%s\n' '✅ Barredor Desplegado y Programado'
