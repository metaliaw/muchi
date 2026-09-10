#!/usr/bin/env sh
# Deploys the Private Search Worker.
set -eu
cd -- "$(dirname -- "$0")"
. ./config/deploy.env
. ./deploy-common.sh
DEPLOY_COMPONENT="Worker"
prepare_deploy "$@"

cloud functions deploy "$WORKER_NAME" --gen2 --trigger-http --no-allow-unauthenticated \
	--runtime="$RUNTIME" --region="$REGION" --source=. --entry-point=ProcessSearch --ignore-file=.gcloudignore \
	--service-account="$WORKER_EMAIL" --build-service-account="projects/$PROJECT/serviceAccounts/$BUILD_EMAIL" \
	--memory="$MEMORY" --timeout="$TIMEOUT" --min-instances=0 --max-instances="$MAX_INSTANCES" --concurrency=1 \
	--set-env-vars="GOOGLE_CLOUD_PROJECT=$PROJECT,MUCHI_STORES_CONFIG=serverless_function_source_code/config/stores.yaml,MUCHI_SCRY_ENABLED=$MUCHI_SCRY_ENABLED,MUCHI_SCRY_COMMUNITY_ENABLED=$MUCHI_SCRY_COMMUNITY_ENABLED" \
	--set-secrets="MUCHI_API_TOKEN=$MUCHI_API_TOKEN" --format=none
read_worker
cloud run services add-iam-policy-binding "${WORKER_SERVICE##*/}" --region="$REGION" \
	--member="serviceAccount:$TASK_EMAIL" --role=roles/run.invoker --condition=None --format=none
printf '%s\n' '✅ Worker Privado Desplegado'
