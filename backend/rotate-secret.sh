#!/usr/bin/env bash
# Rotates Muchi Credentials with the Active gcloud Session and syncs .env.
set +x
set -euo pipefail
umask 077
ROOT=$(cd -- "$(dirname -- "$0")" && pwd)
# shellcheck source=config/deploy.env
source "$ROOT/config/deploy.env"
PROJECT=""
SECRET_NAME=${MUCHI_API_TOKEN%:*}
VERSION=""
STREAMLIT_URL="https://muchitgc.streamlit.app/"
STREAMLIT_KEY="MUCHI_API_TOKEN"
STREAMLIT_DOCS="https://docs.streamlit.io/deploy/streamlit-community-cloud/deploy-your-app/secrets-management"
DRY_RUN=false
WORK_DIR=""
STEP="Configuración"

usage() {
  printf '%s\n' '🐱 rotate-secret.sh --project ID [--region REGION]' \
    '  [--secret NOMBRE] [--version NUMERO] [--streamlit-url HTTPS_URL]' \
    '  [--streamlit-key CLAVE] [--dry-run]' \
    'Sin --version: Genera un Token nuevo. Con --version: Reanuda o Revierte.'
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
      if [[ "$line" == MUCHI_API_TOKEN=* ]]; then
        printf 'MUCHI_API_TOKEN=%s\n' "$token"
      else
        printf '%s\n' "$line"
      fi
    done < "$env_file" > "$tmp"
    mv -f "$tmp" "$env_file"
  else
    printf 'MUCHI_API_TOKEN=%s\n' "$token" > "$env_file"
  fi
}

while (( $# )); do
  case "$1" in
    --project|--region|--secret|--version|--streamlit-url|--streamlit-key)
      (( $# >= 2 )) || fail "Falta el Valor de $1"
      case "$1" in
        --project) PROJECT=$2 ;; --region) REGION=$2 ;; --secret) SECRET_NAME=$2 ;;
        --version) VERSION=$2 ;; --streamlit-url) STREAMLIT_URL=$2 ;; --streamlit-key) STREAMLIT_KEY=$2 ;;
      esac
      shift 2 ;;
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
[[ "$STREAMLIT_KEY" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]] || fail 'La Clave debe ser un Nombre TOML simple.'
[[ "$STREAMLIT_URL" =~ ^https://[a-zA-Z0-9.-]+/?$ ]] || fail 'Usa la URL HTTPS de la App, sin Ruta ni Parámetros.'

streamlit_fallback() {
  printf '\n⚠️ %s\n' 'Si no Aparece el menú Settings ni la Sección Secrets:'
  printf '%s\n' \
    '· Entra con la Cuenta Dueña de la App; el Panel solo Muestra sus Apps.' \
    '· Dentro de la App, abajo a la Derecha: Manage app → ⋮ → Settings.' \
    '· Si sigue Ausente, Redespliega la App; el menú Aparece tras el Redespliegue.' \
    '· Al Redesplegar, el Token va en Advanced settings… → Secrets, antes de Deploy.' \
    '· Streamlit no Lee .env ni secrets.toml del Repo; el Secreto vive solo en ese Panel.'
  printf '· Guía Oficial: %s\n' "$STREAMLIT_DOCS"
}

instructions() {
  printf '\n✅ Versión: %s:%s\n' "$SECRET_NAME" "$VERSION"
  printf 'Secret Manager: https://console.cloud.google.com/security/secret-manager/secret/%s/versions?project=%s\n' "$SECRET_NAME" "$PROJECT"
  printf 'Streamlit: %s\nPanel: https://share.streamlit.io/\nGuía: %s\n' "$STREAMLIT_URL" "$STREAMLIT_DOCS"
  printf '%s\n' \
    '1. Entra al Panel con la Cuenta Dueña de la App.' \
    '2. Abre la App: menú ⋮ → Settings → Secrets.' \
    "3. Pega la Línea $STREAMLIT_KEY = \"…\" impresa arriba; Conserva las otras Claves." \
    '4. Pulsa Save. Si la App mantiene el Cliente en Caché, usa Reboot app.' \
    '5. Crea una Búsqueda desde Streamlit y Confirma que no devuelve 401.' \
    'Comparte estos Enlaces e Instrucciones; el Token ya está Impreso Arriba.'
  streamlit_fallback
}
if "$DRY_RUN"; then
  printf '🐱 Vista Previa: %s / %s\n' "$PROJECT" "$REGION"
  printf '%s\n' "API: $API_NAME · Worker: $WORKER_NAME" \
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
cloud secrets describe "$SECRET_NAME" --format='value(name)' >/dev/null
API_SERVICE=$(cloud functions describe "$API_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.service)')
WORKER_SERVICE=$(cloud functions describe "$WORKER_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.service)')
API_SERVICE=${API_SERVICE##*/}
WORKER_SERVICE=${WORKER_SERVICE##*/}
[[ "$API_SERVICE" =~ ^[a-z0-9-]+$ && "$WORKER_SERVICE" =~ ^[a-z0-9-]+$ ]] || fail 'Servicios Cloud Run Ausentes.'
for service in "$API_SERVICE" "$WORKER_SERVICE"; do
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
step 'Token'
TOKEN=$(<"$WORK_DIR/token")
printf '%s\n' "$TOKEN"
printf '%s = "%s"\n' "$STREAMLIT_KEY" "$TOKEN"
sync_env_token "$TOKEN"
instructions
printf '\n⚠️ Actualiza Streamlit al Terminar: su Token Anterior dejará de Funcionar.\n'

for service in "$WORKER_SERVICE" "$API_SERVICE"; do
  step "Actualizando $service"
  revision=$(cloud run services update "$service" --region="$REGION" \
    --update-secrets="MUCHI_API_TOKEN=$SECRET_NAME:latest" --format='value(status.latestReadyRevisionName)')
  [[ "$revision" =~ ^[a-z0-9-]+$ ]] || fail 'Revisión Lista Ausente.'
  cloud run services update-traffic "$service" --region="$REGION" --to-latest --format=none
done

step 'Autenticación de la API'
API_URL=$(cloud run services describe "$API_SERVICE" --region="$REGION" --format='value(status.url)')
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
printf 'API: %s\n✅ API y Worker Actualizados. Falta Guardar el Token en Streamlit.\n' "$API_URL"
