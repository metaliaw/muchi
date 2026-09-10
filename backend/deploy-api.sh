#!/usr/bin/env sh
# Deploys the Public API.
set -eu
cd -- "$(dirname -- "$0")"
. ./config/deploy.env
. ./deploy-common.sh
DEPLOY_COMPONENT="API"
prepare_deploy "$@"
read_worker

cloud functions deploy "$API_NAME" --gen2 --trigger-http --allow-unauthenticated \
	--runtime="$RUNTIME" --region="$REGION" --source=. --entry-point=ServeAPI --ignore-file=.gcloudignore \
	--service-account="$API_EMAIL" --build-service-account="projects/$PROJECT/serviceAccounts/$BUILD_EMAIL" \
	--memory="$MEMORY" --timeout="$TIMEOUT" --min-instances=0 --max-instances="$MAX_INSTANCES" --concurrency=1 \
	--set-env-vars="GOOGLE_CLOUD_PROJECT=$PROJECT,MUCHI_STORES_CONFIG=serverless_function_source_code/config/stores.yaml,MUCHI_TASK_REGION=$REGION,MUCHI_TASK_QUEUE=$TASK_QUEUE,MUCHI_TASK_URL=$WORKER_URL,MUCHI_TASK_SERVICE_ACCOUNT=$TASK_EMAIL,MUCHI_SCRY_ENABLED=$MUCHI_SCRY_ENABLED,MUCHI_SCRY_COMMUNITY_ENABLED=$MUCHI_SCRY_COMMUNITY_ENABLED" \
	--set-secrets="MUCHI_API_TOKEN=$MUCHI_API_TOKEN" --format=none
if "$DRY_RUN"; then
	printf '%s\n' '✅ API Lista para Desplegar'
else
	API_URL=$(cloud functions describe "$API_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.uri)')
	case "$API_URL" in https://*) ;; *) fail_deploy 'URL de API Ausente.' ;; esac
	printf '✅ API Desplegada\nAPI: %s\nSalud: %s/v1/health\n' "$API_URL" "$API_URL"
fi
