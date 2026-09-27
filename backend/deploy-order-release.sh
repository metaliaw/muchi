#!/usr/bin/env sh
# Deploys and Schedules the Order Releaser.
set -eu
cd -- "$(dirname -- "$0")"
. ./config/deploy.env
. ./deploy-common.sh
DEPLOY_COMPONENT="Liberador de Ordenes"
prepare_deploy "$@"
read_worker

cloud functions deploy "$ORDER_RELEASE_NAME" --gen2 --trigger-http --no-allow-unauthenticated \
	--runtime="$RUNTIME" --region="$REGION" --source=. --entry-point=ReleaseOrders --ignore-file=.gcloudignore \
	--service-account="$API_EMAIL" --build-service-account="projects/$PROJECT/serviceAccounts/$BUILD_EMAIL" \
	--memory="$MEMORY" --timeout=120s --min-instances=0 --max-instances=1 --concurrency=1 \
	--set-env-vars="GOOGLE_CLOUD_PROJECT=$PROJECT,MUCHI_STORES_CONFIG=serverless_function_source_code/config/stores.yaml,MUCHI_TASK_REGION=$REGION,MUCHI_TASK_QUEUE=$TASK_QUEUE,MUCHI_TASK_URL=$WORKER_URL,MUCHI_TASK_SERVICE_ACCOUNT=$TASK_EMAIL" \
	--set-secrets="MUCHI_API_TOKEN=$MUCHI_API_TOKEN" --format=none
if "$DRY_RUN"; then
	ORDER_RELEASE_URL="https://$ORDER_RELEASE_NAME-PREVIEW.run.app"
	ORDER_RELEASE_SERVICE=$ORDER_RELEASE_NAME
else
	ORDER_RELEASE_URL=$(cloud functions describe "$ORDER_RELEASE_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.uri)')
	ORDER_RELEASE_SERVICE=$(cloud functions describe "$ORDER_RELEASE_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.service)')
fi
cloud run services add-iam-policy-binding "${ORDER_RELEASE_SERVICE##*/}" --region="$REGION" \
	--member="serviceAccount:$TASK_EMAIL" --role=roles/run.invoker --condition=None --format=none
cloud services enable cloudscheduler.googleapis.com --format=none
existing_job=$(cloud scheduler jobs list --location="$REGION" --filter="name:$ORDER_RELEASE_JOB" --format='value(name.basename())')
job_action=create
if [ "$existing_job" = "$ORDER_RELEASE_JOB" ]; then job_action=update; fi
cloud scheduler jobs "$job_action" http "$ORDER_RELEASE_JOB" --location="$REGION" \
	--schedule="$ORDER_RELEASE_SCHEDULE" --uri="$ORDER_RELEASE_URL" --http-method=POST \
	--oidc-service-account-email="$TASK_EMAIL" --oidc-token-audience="$ORDER_RELEASE_URL" \
	--attempt-deadline=120s --format=none
printf '%s\n' '✅ Liberador de Ordenes Desplegado y Programado'
