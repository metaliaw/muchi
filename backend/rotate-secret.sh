#!/usr/bin/env bash
# Rotates Muchi Credentials with the Active gcloud Session and syncs .env.
#
# Alcanza a todos los Consumidores del Token: el Worker, la API y los Fronts.
# Los Fronts se despliegan desde otro Repositorio, pero montan este mismo
# Secreto, y una Rotacion que no los incluye los deja autenticando con un Token
# muerto. Uno que todavia no existe se saltea con un Aviso, para que la
# Migracion pueda nombrar al nuevo antes de desplegarlo.
#
# Con --webhook Rota el Secreto que Firma los Webhooks de Pedidos WooCommerce.
# Solo la API lo Monta; la Primera Rotacion lo Crea. Cada Tienda lo Guarda a
# mano en su propio Admin, asi que una Rotacion Obliga a Pegarlo de nuevo alli.
set +x
set -euo pipefail
umask 077
ROOT=$(cd -- "$(dirname -- "$0")" && pwd)
# shellcheck source=config/deploy.env
source "$ROOT/config/deploy.env"
PROJECT=""
SECRET_NAME=${MUCHI_API_TOKEN%:*}
ENV_KEY=MUCHI_API_TOKEN
WEBHOOK=false
VERSION=""
DRY_RUN=false
WORK_DIR=""
STEP="Configuración"

usage() {
  printf '%s\n' '🐱 rotate-secret.sh --project ID [--region REGION] [--webhook]' \
    '  [--secret NOMBRE] [--version NUMERO] [--dry-run]' \
    'Sin --version: Genera un Token nuevo. Con --version: Reanuda o Revierte.' \
    '--webhook: Rota MUCHI_ORDER_WEBHOOK_SECRET; solo la API; lo Crea si Falta.'
}
cleanup() {
  local code=$?
  if [[ -n "$WORK_DIR" ]]; then rm -rf -- "$WORK_DIR"; fi
  if (( code != 0 )); then
    printf '❌ %s Falló\n' "$STEP" >&2
    if [[ -n "$VERSION" ]]; then
      printf 'Versión Conservada: %s:%s\nReintenta con --project %s --secret %s --version %s\n' \
        "$SECRET_NAME" "$VERSION" "$PROJECT" "$SECRET_NAME" "$VERSION" >&2
    fi
  fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
fail() { printf '❌ %s\n' "$1" >&2; exit 1; }
step() { STEP=$1; printf '\n🐱 %s\n' "$STEP"; }
cloud() { gcloud "$@" --project="$PROJECT" --quiet; }
sync_env_token() {
  local token=$1
  local env_file="$ROOT/.env"
  local tmp
  if [[ -f "$env_file" ]]; then
    tmp=$(mktemp "$env_file.XXXXXXXX")
    while IFS= read -r line || [[ -n "$line" ]]; do
      if [[ "$line" == "$ENV_KEY"=* ]]; then
        printf '%s=%s\n' "$ENV_KEY" "$token"
      else
        printf '%s\n' "$line"
      fi
    done < "$env_file" > "$tmp"
    mv -f "$tmp" "$env_file"
    grep -q "^$ENV_KEY=" "$env_file" || printf '%s=%s\n' "$ENV_KEY" "$token" >> "$env_file"
  else
    printf '%s=%s\n' "$ENV_KEY" "$token" > "$env_file"
  fi
}

while (( $# )); do
  case "$1" in
    --project|--region|--secret|--version)
      (( $# >= 2 )) || fail "Falta el Valor de $1"
      case "$1" in
        --project) PROJECT=$2 ;; --region) REGION=$2 ;; --secret) SECRET_NAME=$2 ;;
        --version) VERSION=$2 ;;
      esac
      shift 2 ;;
    --webhook) WEBHOOK=true; SECRET_NAME=${ORDER_WEBHOOK_SECRET%:*}; ENV_KEY=MUCHI_ORDER_WEBHOOK_SECRET; shift ;;
    --dry-run) DRY_RUN=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) fail "Argumento Desconocido: $1" ;;
  esac
done
if [[ -z "$PROJECT" ]]; then PROJECT=$(gcloud config get-value project 2>/dev/null); fi
[[ "$PROJECT" =~ ^[a-z][a-z0-9-]+$ ]] || fail 'Proyecto Inválido.'
[[ "$REGION" =~ ^[a-z0-9-]+$ ]] || fail 'Región Inválida.'
[[ "$SECRET_NAME" =~ ^[a-zA-Z0-9_-]+$ ]] || fail 'Nombre de Secreto Inválido.'
[[ -z "$VERSION" || "$VERSION" =~ ^[1-9][0-9]*$ ]] || fail 'Usa una Versión Numérica Positiva.'

instructions() {
  printf '\n✅ Versión: %s:%s\n' "$SECRET_NAME" "$VERSION"
  printf 'Secret Manager: https://console.cloud.google.com/security/secret-manager/secret/%s/versions?project=%s\n' "$SECRET_NAME" "$PROJECT"
}

if "$DRY_RUN" && "$WEBHOOK"; then
  printf '🐱 Vista Previa: %s / %s\n' "$PROJECT" "$REGION"
  printf '%s\n' "API: $API_NAME" \
    "Se Creará $SECRET_NAME si Falta, con Acceso para $API_ACCOUNT." \
    'Se Actualizará MUCHI_ORDER_WEBHOOK_SECRET y se Enviará el Tráfico a la nueva Revisión.' \
    'No se Generan Secretos ni se Modifica GCP.'
  VERSION=${VERSION:-NUEVA}
  instructions
  exit 0
fi
if "$DRY_RUN"; then
  printf '🐱 Vista Previa: %s / %s\n' "$PROJECT" "$REGION"
  printf '%s\n' "API: $API_NAME · Worker: $WORKER_NAME · Fronts: $FRONT_NAMES" \
    'Se Actualizará MUCHI_API_TOKEN y se Enviará el Tráfico a las nuevas Revisiones.' \
    'No se Generan Tokens ni se Modifica GCP.'
  VERSION=${VERSION:-NUEVA}
  instructions
  exit 0
fi
command -v gcloud >/dev/null || fail 'Instala Google Cloud CLI.'
command -v python3 >/dev/null || fail 'Instala Python 3.'
WORK_DIR=$(mktemp -d "${TMPDIR:-/tmp}/muchi-rotation.XXXXXXXX")

step 'Servicios y Secreto Verificados'
if "$WEBHOOK" && ! cloud secrets describe "$SECRET_NAME" --format=none 2>/dev/null; then
  [[ -z "$VERSION" ]] || fail "$SECRET_NAME no Existe; Rota sin --version para Crearlo."
  cloud secrets create "$SECRET_NAME" --replication-policy=automatic --format=none
  printf '✅ %s Creado\n' "$SECRET_NAME"
fi
cloud secrets describe "$SECRET_NAME" --format='value(name)' >/dev/null
if "$WEBHOOK"; then
  cloud secrets add-iam-policy-binding "$SECRET_NAME" \
    --member="serviceAccount:$API_ACCOUNT@$PROJECT.iam.gserviceaccount.com" \
    --role=roles/secretmanager.secretAccessor --condition=None --format=none
fi
API_SERVICE=$(cloud functions describe "$API_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.service)')
WORKER_SERVICE=$(cloud functions describe "$WORKER_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.service)')
API_SERVICE=${API_SERVICE##*/}
WORKER_SERVICE=${WORKER_SERVICE##*/}
[[ "$API_SERVICE" =~ ^[a-z0-9-]+$ && "$WORKER_SERVICE" =~ ^[a-z0-9-]+$ ]] || fail 'Servicios Cloud Run Ausentes.'

# Los Fronts se resuelven antes de tocar nada: los que existen entran a la
# Lista y los que no avisan, para que la Rotacion no muera a mitad de camino
# por un Servicio que todavia nadie desplego.
FRONTS=""
CONSUMERS=$FRONT_NAMES
if "$WEBHOOK"; then CONSUMERS=""; fi
for front in $CONSUMERS; do
  [[ "$front" =~ ^[a-z0-9-]+$ ]] || fail "Nombre de Front Invalido: $front"
  if cloud run services describe "$front" --region="$REGION" --format=none 2>/dev/null; then
    FRONTS="$FRONTS $front"
  else
    printf '⚠️ %s\n' "Front '$front' Ausente en $REGION: se Saltea."
  fi
done
"$WEBHOOK" || [[ -n "$FRONTS" ]] || fail 'Ningun Front Desplegado; revisa --region.'

# El Token lo Montan Worker, API y Fronts; el Secreto del Webhook, solo la API.
SERVICES="$WORKER_SERVICE $API_SERVICE $FRONTS"
if "$WEBHOOK"; then SERVICES=$API_SERVICE; fi

for service in $SERVICES; do
  "$WEBHOOK" && continue
  cloud run services describe "$service" --region="$REGION" --format=json > "$WORK_DIR/$service.json"
  python3 - "$WORK_DIR/$service.json" "$service" <<'PY'
import json, sys
service = json.load(open(sys.argv[1]))
containers = service['spec']['template']['spec']['containers']
if len(containers) != 1:
    raise SystemExit('Se requiere un Servicio con un Contenedor.')
reference = next((v.get('valueFrom', {}).get('secretKeyRef') for v in containers[0].get('env', []) if v['name'] == 'MUCHI_API_TOKEN'), None)
if not reference:
    raise SystemExit('MUCHI_API_TOKEN no está Vinculado a Secret Manager.')
print(f"Referencia Anterior de {sys.argv[2]}: {reference['name']}:{reference['key']}")
PY
done

step 'Versión del Secreto'
if [[ -z "$VERSION" ]]; then
  python3 -c 'import secrets,sys; sys.stdout.write(secrets.token_hex(32))' > "$WORK_DIR/token"
  reference=$(cloud secrets versions add "$SECRET_NAME" --data-file="$WORK_DIR/token" --format='value(name)')
  VERSION=${reference##*/}
  [[ "$VERSION" =~ ^[1-9][0-9]*$ ]] || fail 'Respuesta de Versión Inválida.'
else
  state=$(cloud secrets versions describe "$VERSION" --secret="$SECRET_NAME" --format='value(state)')
  [[ "$state" == ENABLED ]] || fail 'Versión no Habilitada.'
  cloud secrets versions access "$VERSION" --secret="$SECRET_NAME" --out-file="$WORK_DIR/token"
fi
latest=$(cloud secrets versions describe latest --secret="$SECRET_NAME" --format='value(name)')
LATEST_VERSION=${latest##*/}
if [[ "$VERSION" != "$LATEST_VERSION" ]]; then
  printf '\n⚠️ %s\n' "Los Servicios Apuntan a $SECRET_NAME:latest, hoy la Versión $LATEST_VERSION." \
    "Para Volver a la Versión $VERSION, Deshabilita las Posteriores en Secret Manager."
fi
step 'Valor'
TOKEN=$(<"$WORK_DIR/token")
printf '%s\n' "$TOKEN"
sync_env_token "$TOKEN"
instructions

# El Orden sigue al Trafico, de adentro hacia afuera: el Worker no atiende a
# nadie, la API atiende al Front, el Front atiende al Usuario.
for service in $SERVICES; do
  step "Actualizando $service"
  revision=$(cloud run services update "$service" --region="$REGION" \
    --update-secrets="$ENV_KEY=$SECRET_NAME:latest" --format='value(status.latestReadyRevisionName)')
  [[ "$revision" =~ ^[a-z0-9-]+$ ]] || fail 'Revisión Lista Ausente.'
  cloud run services update-traffic "$service" --region="$REGION" --to-latest --format=none
done

API_URL=$(cloud run services describe "$API_SERVICE" --region="$REGION" --format='value(status.url)')
if "$WEBHOOK"; then
  step 'Firma del Webhook'
  # Una Entrega Firmada en `on-hold` no Mueve ningun Pedido: 200 Prueba que la
  # API ya Verifica con el Valor nuevo, 401 que sigue con el anterior.
  python3 - "$WORK_DIR/token" "$API_URL" <<'PY'
import base64, hashlib, hmac, pathlib, sys, urllib.error, urllib.parse, urllib.request
url = urllib.parse.urlsplit(sys.argv[2])
if url.scheme != 'https' or not url.hostname or not url.hostname.endswith('.run.app'):
    raise SystemExit('URL de Cloud Run Inválida.')
secret = pathlib.Path(sys.argv[1]).read_bytes()
body = b'{"id":1,"status":"on-hold"}'
signature = base64.b64encode(hmac.new(secret, body, hashlib.sha256).digest()).decode()
request = urllib.request.Request(sys.argv[2].rstrip('/') + '/v1/webhooks/woocommerce/rotation.invalid/orders',
    data=body, headers={'Content-Type': 'application/json', 'X-WC-Webhook-Signature': signature})
try:
    with urllib.request.urlopen(request, timeout=30) as response:
        if response.status != 200:
            raise SystemExit(f'Validación Fallida: HTTP {response.status}')
except urllib.error.HTTPError as error:
    raise SystemExit(f'Validación Fallida: HTTP {error.code}') from None
except urllib.error.URLError:
    raise SystemExit('No se pudo Conectar con la API.') from None
print('✅ API Verifica Firmas con la Versión Seleccionada')
PY
  printf 'API: %s\n⚠️ Pega el Valor nuevo en el Webhook de cada Tienda WooCommerce.\n' "$API_URL"
  exit 0
fi
step 'Autenticación de la API'
python3 - "$WORK_DIR/token" "$API_URL" <<'PY'
import pathlib, sys, urllib.error, urllib.parse, urllib.request
class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None
url = urllib.parse.urlsplit(sys.argv[2])
if url.scheme != 'https' or not url.hostname or not url.hostname.endswith('.run.app') or url.username or url.password:
    raise SystemExit('URL de Cloud Run Inválida.')
token = pathlib.Path(sys.argv[1]).read_text()
if not token or any(ord(c) < 33 or ord(c) > 126 for c in token):
    raise SystemExit('Formato del Token Inválido.')
request = urllib.request.Request(sys.argv[2].rstrip('/') + '/v1/health/sources', headers={'Authorization': 'Bearer ' + token})
try:
    with urllib.request.build_opener(NoRedirect).open(request, timeout=30) as response:
        if response.status != 200:
            raise SystemExit(f'Validación Fallida: HTTP {response.status}')
except urllib.error.HTTPError as error:
    raise SystemExit(f'Validación Fallida: HTTP {error.code}') from None
except urllib.error.URLError:
    raise SystemExit('No se pudo Conectar con la API.') from None
print('✅ API Autenticada con la Versión Seleccionada')
PY
printf 'API: %s\n✅ Worker, API y Front Actualizados.\n' "$API_URL"
