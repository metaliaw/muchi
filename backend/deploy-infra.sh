#!/usr/bin/env sh
# Prepares Shared Production Infrastructure.
set -eu
cd -- "$(dirname -- "$0")"
. ./config/deploy.env
. ./deploy-common.sh
DEPLOY_COMPONENT="Infraestructura"
prepare_deploy "$@"

cloud services enable cloudfunctions.googleapis.com run.googleapis.com cloudbuild.googleapis.com \
	artifactregistry.googleapis.com firestore.googleapis.com cloudtasks.googleapis.com \
	secretmanager.googleapis.com iam.googleapis.com iamcredentials.googleapis.com
if [ -n "$TOKEN_FILE" ]; then
	existing=$(cloud secrets list --filter="name:$SECRET_NAME" --format='value(name.basename())')
	if [ "$existing" != "$SECRET_NAME" ]; then cloud secrets create "$SECRET_NAME" --replication-policy=automatic; fi
	cloud secrets versions add "$SECRET_NAME" --data-file="$TOKEN_FILE" --format='value(name)'
fi
if ! "$DRY_RUN"; then
	state=$(cloud secrets versions describe latest --secret="$SECRET_NAME" --format='value(state)')
	[ "$state" = ENABLED ] || fail_deploy 'Secreto no Habilitado.'
fi

for account in "$API_ACCOUNT" "$WORKER_ACCOUNT" "$TASK_ACCOUNT" "$BUILD_ACCOUNT"; do
	email="$account@$PROJECT.iam.gserviceaccount.com"
	existing=$(cloud iam service-accounts list --filter="email=$email" --format='value(email)')
	if [ -z "$existing" ]; then cloud iam service-accounts create "$account"; fi
done
grant_project "$API_EMAIL" roles/datastore.user
grant_project "$API_EMAIL" roles/cloudtasks.enqueuer
grant_project "$WORKER_EMAIL" roles/datastore.user
for role in roles/logging.logWriter roles/artifactregistry.writer roles/storage.objectViewer; do
	grant_project "$BUILD_EMAIL" "$role"
done
for email in "$API_EMAIL" "$WORKER_EMAIL"; do
	cloud secrets add-iam-policy-binding "$SECRET_NAME" --member="serviceAccount:$email" \
		--role=roles/secretmanager.secretAccessor --condition=None --format=none
done
cloud iam service-accounts add-iam-policy-binding "$TASK_EMAIL" --member="serviceAccount:$API_EMAIL" \
	--role=roles/iam.serviceAccountUser --condition=None --format=none
agent=$(cloud beta services identity create --service=cloudtasks.googleapis.com --format='value(email)')
if "$DRY_RUN"; then agent="service-PROJECT_NUMBER@gcp-sa-cloudtasks.iam.gserviceaccount.com"; fi
[ -n "$agent" ] || fail_deploy 'Agente de Cloud Tasks Ausente.'
grant_project "$agent" roles/cloudtasks.serviceAgent

existing=$(cloud firestore databases list --filter='name:(default)' --format='value(name)')
if [ -z "$existing" ]; then
	cloud firestore databases create '--database=(default)' --location="$FIRESTORE_LOCATION" --type=firestore-native
fi
for group in searches items item_offers idempotency offer_cache; do
	cloud firestore fields ttls update expires_at --collection-group="$group" '--database=(default)' --enable-ttl --async
done
result_index=$(cloud firestore indexes composite list \
	--filter='queryScope=COLLECTION AND fields.fieldPath:search_id AND fields.fieldPath:completion_sequence' \
	--format='value(name)')
if [ -z "$result_index" ]; then
	cloud firestore indexes composite create --collection-group=items --query-scope=collection \
		--field-config=field-path=search_id,order=ascending \
		--field-config=field-path=completion_sequence,order=ascending --async
fi
existing=$(cloud tasks queues list --location="$REGION" --filter="name:$TASK_QUEUE" --format='value(name.basename())')
action=create
if [ "$existing" = "$TASK_QUEUE" ]; then action=update; fi
cloud tasks queues "$action" "$TASK_QUEUE" --location="$REGION" \
	--max-dispatches-per-second="$DISPATCH_RATE" --max-concurrent-dispatches="$CONCURRENT_TASKS"
printf '%s\n' '✅ Infraestructura Preparada'
