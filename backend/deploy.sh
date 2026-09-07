#!/usr/bin/env sh
# Deploys Muchi with the Active gcloud Session.
set -eu
cd -- "$(dirname -- "$0")"
. ./config/deploy.env
PROJECT=""
TOKEN_FILE=""
DRY_RUN=false
STEP="Configuración"
trap 'code=$?; if [ "$code" -ne 0 ]; then printf "❌ %s Falló\n" "$STEP" >&2; fi' 0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --project|--region|--token-secret|--token-file)
      [ "$#" -ge 2 ] || { printf '%s\n' "Falta el Valor de $1" >&2; exit 1; }
      case "$1" in
        --project) PROJECT=$2 ;; --region) REGION=$2 ;;
        --token-secret) TOKEN_SECRET=$2 ;; --token-file) TOKEN_FILE=$2 ;;
      esac
      shift 2 ;;
    --dry-run) DRY_RUN=true; shift ;;
    --help|-h)
      printf '%s\n' '🐱 deploy.sh [--project ID] [--region REGION]' \
        '  [--token-secret NOMBRE:VERSION] [--token-file ARCHIVO] [--dry-run]'
      exit 0 ;;
    *) printf '%s\n' "Argumento Desconocido: $1" >&2; exit 1 ;;
  esac
done
command -v gcloud >/dev/null 2>&1 || { printf '%s\n' 'Instala Google Cloud CLI.' >&2; exit 1; }
[ -n "$PROJECT" ] || PROJECT=$(gcloud config get-value project 2>/dev/null)
case "$PROJECT" in ''|'(unset)'|*[!a-z0-9-]*) printf '%s\n' 'Selecciona un Proyecto con --project.' >&2; exit 1 ;; esac
case "$REGION" in ''|*[!a-z0-9-]*) printf '%s\n' 'Región Inválida.' >&2; exit 1 ;; esac
case "$TOKEN_SECRET" in *:*) ;; *) printf '%s\n' 'Usa --token-secret NOMBRE:VERSION.' >&2; exit 1 ;; esac
SECRET_NAME=${TOKEN_SECRET%:*}
SECRET_VERSION=${TOKEN_SECRET##*:}
case "$SECRET_NAME" in ''|*[!a-zA-Z0-9_-]*) exit 1 ;; esac
case "$SECRET_VERSION" in latest) ;; ''|*[!0-9]*) exit 1 ;; esac
[ -z "$TOKEN_FILE" ] || [ -s "$TOKEN_FILE" ] || { printf '%s\n' 'Archivo de Token Ausente o Vacío.' >&2; exit 1; }
API_EMAIL="$API_ACCOUNT@$PROJECT.iam.gserviceaccount.com"
WORKER_EMAIL="$WORKER_ACCOUNT@$PROJECT.iam.gserviceaccount.com"
TASK_EMAIL="$TASK_ACCOUNT@$PROJECT.iam.gserviceaccount.com"
BUILD_EMAIL="$BUILD_ACCOUNT@$PROJECT.iam.gserviceaccount.com"

cloud() {
  if "$DRY_RUN"; then
    printf 'gcloud' >&2; printf ' "%s"' "$@" "--project=$PROJECT" --quiet >&2; printf '\n' >&2
  else
    gcloud "$@" "--project=$PROJECT" --quiet
  fi
}
step() { STEP=$1; printf '\n🐱 %s\n' "$STEP"; }
grant_project() {
  cloud projects add-iam-policy-binding "$PROJECT" --member="serviceAccount:$1" --role="$2" --condition=None --format=none
}

printf '🐱 Muchi en GCP\nProyecto: %s\nRegión: %s\n' "$PROJECT" "$REGION"
if "$DRY_RUN"; then printf '%s\n' '⚠️ Vista Previa: GCP sin Cambios'; else
  [ -n "$(gcloud auth list --filter=status:ACTIVE --format='value(account)')" ] || { printf '%s\n' 'Ejecuta gcloud auth login.' >&2; exit 1; }
fi

step 'Servicios y Secreto'
cloud services enable cloudfunctions.googleapis.com run.googleapis.com cloudbuild.googleapis.com \
  artifactregistry.googleapis.com firestore.googleapis.com cloudtasks.googleapis.com \
  secretmanager.googleapis.com iam.googleapis.com iamcredentials.googleapis.com
if [ -n "$TOKEN_FILE" ]; then
  existing=$(cloud secrets list --filter="name:$SECRET_NAME" --format='value(name.basename())')
  if [ "$existing" != "$SECRET_NAME" ]; then
    cloud secrets create "$SECRET_NAME" --replication-policy=automatic
  fi
  version=$(cloud secrets versions add "$SECRET_NAME" --data-file="$TOKEN_FILE" --format='value(name)')
  SECRET_VERSION=${version##*/}
fi
if "$DRY_RUN"; then SECRET_VERSION=1; fi
version=$(cloud secrets versions describe "$SECRET_VERSION" --secret="$SECRET_NAME" --format='value(name)')
state=$(cloud secrets versions describe "$SECRET_VERSION" --secret="$SECRET_NAME" --format='value(state)')
if ! "$DRY_RUN"; then
  [ "$state" = ENABLED ] || { printf '%s\n' 'Secreto no Habilitado.' >&2; exit 1; }
  SECRET_VERSION=${version##*/}
fi
TOKEN_SECRET="$SECRET_NAME:$SECRET_VERSION"

step 'Cuentas y Permisos'
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
  cloud secrets add-iam-policy-binding "$SECRET_NAME" --member="serviceAccount:$email" --role=roles/secretmanager.secretAccessor --condition=None --format=none
done
cloud iam service-accounts add-iam-policy-binding "$TASK_EMAIL" --member="serviceAccount:$API_EMAIL" --role=roles/iam.serviceAccountUser --condition=None --format=none
agent=$(cloud beta services identity create --service=cloudtasks.googleapis.com --format='value(email)')
if "$DRY_RUN"; then agent="service-PROJECT_NUMBER@gcp-sa-cloudtasks.iam.gserviceaccount.com"; fi
[ -n "$agent" ] || exit 1
grant_project "$agent" roles/cloudtasks.serviceAgent

step 'Firestore y Cola'
existing=$(cloud firestore databases list --filter='name:(default)' --format='value(name)')
if [ -z "$existing" ]; then
  cloud firestore databases create '--database=(default)' --location="$FIRESTORE_LOCATION" --type=firestore-native
fi
for group in searches items item_offers idempotency offer_cache; do
  cloud firestore fields ttls update expires_at --collection-group="$group" '--database=(default)' --enable-ttl --async
done
existing=$(cloud tasks queues list --location="$REGION" --filter="name:$TASK_QUEUE" --format='value(name.basename())')
action=create
if [ "$existing" = "$TASK_QUEUE" ]; then action=update; fi
cloud tasks queues "$action" "$TASK_QUEUE" --location="$REGION" \
  --max-dispatches-per-second="$DISPATCH_RATE" --max-concurrent-dispatches="$CONCURRENT_TASKS"

step 'Despliegue del Worker'
cloud functions deploy "$WORKER_NAME" --gen2 --trigger-http --no-allow-unauthenticated \
  --runtime="$RUNTIME" --region="$REGION" --source=. --entry-point=ProcessSearch --ignore-file=.gcloudignore \
  --service-account="$WORKER_EMAIL" --build-service-account="projects/$PROJECT/serviceAccounts/$BUILD_EMAIL" \
  --memory="$MEMORY" --timeout="$TIMEOUT" --min-instances=0 --max-instances="$MAX_INSTANCES" --concurrency=1 \
  --set-env-vars="GOOGLE_CLOUD_PROJECT=$PROJECT,MUCHI_STORES_CONFIG=serverless_function_source_code/config/stores.yaml" \
  --set-secrets="MUCHI_API_TOKEN=$TOKEN_SECRET" --format=none
WORKER_URL=$(cloud functions describe "$WORKER_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.uri)')
WORKER_SERVICE=$(cloud functions describe "$WORKER_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.service)')
if "$DRY_RUN"; then WORKER_URL="https://$WORKER_NAME-PREVIEW.run.app"; WORKER_SERVICE=$WORKER_NAME; fi
case "$WORKER_URL" in https://*) ;; *) printf '%s\n' 'URL del Worker Ausente.' >&2; exit 1 ;; esac
cloud run services add-iam-policy-binding "${WORKER_SERVICE##*/}" --region="$REGION" \
  --member="serviceAccount:$TASK_EMAIL" --role=roles/run.invoker --condition=None --format=none
printf '%s\n' '✅ Worker Privado Preparado'

step 'Despliegue de la API'
cloud functions deploy "$API_NAME" --gen2 --trigger-http --allow-unauthenticated \
  --runtime="$RUNTIME" --region="$REGION" --source=. --entry-point=ServeAPI --ignore-file=.gcloudignore \
  --service-account="$API_EMAIL" --build-service-account="projects/$PROJECT/serviceAccounts/$BUILD_EMAIL" \
  --memory="$MEMORY" --timeout="$TIMEOUT" --min-instances=0 --max-instances="$MAX_INSTANCES" --concurrency=1 \
  --set-env-vars="GOOGLE_CLOUD_PROJECT=$PROJECT,MUCHI_STORES_CONFIG=serverless_function_source_code/config/stores.yaml,MUCHI_TASK_REGION=$REGION,MUCHI_TASK_QUEUE=$TASK_QUEUE,MUCHI_TASK_URL=$WORKER_URL,MUCHI_TASK_SERVICE_ACCOUNT=$TASK_EMAIL" \
  --set-secrets="MUCHI_API_TOKEN=$TOKEN_SECRET" --format=none
API_URL=$(cloud functions describe "$API_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.uri)')
if "$DRY_RUN"; then printf '%s\n' '✅ Vista Previa Completa'; else
  case "$API_URL" in https://*) ;; *) exit 1 ;; esac
  printf '😺 API y Worker Desplegados\nAPI: %s\nSalud: %s/v1/health\n' "$API_URL" "$API_URL"
fi
