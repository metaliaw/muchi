#!/usr/bin/env bash
# Imprime el Token de Muchi en stdout y lo sincroniza en .env.
set +x
set -euo pipefail
umask 077
ROOT=$(cd -- "$(dirname -- "$0")" && pwd)
# shellcheck source=config/deploy.env
source "$ROOT/config/deploy.env"
PROJECT=""
SECRET_NAME=${MUCHI_API_TOKEN%:*}
VERSION="latest"
STREAMLIT_URL="https://muchitgc.streamlit.app/"
STREAMLIT_KEY="MUCHI_API_TOKEN"
STREAMLIT_DOCS="https://docs.streamlit.io/deploy/streamlit-community-cloud/deploy-your-app/secrets-management"
WORK_DIR=""

usage() {
  printf '%s\n' '🐱 get-secret.sh --project ID [--secret NOMBRE] [--version NUMERO]' \
    'Imprime el Valor del Secreto en stdout y lo Escribe en .env.'
}

cleanup() {
  if [[ -n "$WORK_DIR" ]]; then rm -rf -- "$WORK_DIR"; fi
}

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
  printf '\nStreamlit: %s\nPanel: https://share.streamlit.io/\nGuía: %s\n' "$STREAMLIT_URL" "$STREAMLIT_DOCS"
  printf '%s\n' \
    '1. Entra al Panel con la Cuenta Dueña de la App.' \
    '2. Abre la App: menú ⋮ → Settings → Secrets.' \
    "3. Comprueba que $STREAMLIT_KEY Coincide con el Token impreso arriba." \
    '4. Si lo Cambias, pulsa Save y luego Reboot app.'
  streamlit_fallback
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

fail() { printf '❌ %s\n' "$1" >&2; exit 1; }
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
    --project|--secret|--version)
      (( $# >= 2 )) || fail "Falta el Valor de $1"
      case "$1" in
        --project) PROJECT=$2 ;; --secret) SECRET_NAME=$2 ;; --version) VERSION=$2 ;;
      esac
      shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) fail "Argumento Desconocido: $1" ;;
  esac
done
if [[ -z "$PROJECT" ]]; then PROJECT=$(gcloud config get-value project 2>/dev/null); fi
[[ "$PROJECT" =~ ^[a-z][a-z0-9-]+$ ]] || fail 'Proyecto Inválido.'
[[ "$SECRET_NAME" =~ ^[a-zA-Z0-9_-]+$ ]] || fail 'Nombre de Secreto Inválido.'
[[ "$VERSION" == latest || "$VERSION" =~ ^[1-9][0-9]*$ ]] || fail 'Usa latest o una Versión Numérica Positiva.'
command -v gcloud >/dev/null || fail 'Instala Google Cloud CLI.'
command -v mktemp >/dev/null || fail 'Instala coreutils (mktemp).'

WORK_DIR=$(mktemp -d "${TMPDIR:-/tmp}/muchi-secret.XXXXXXXX")
cloud secrets versions access "$VERSION" --secret="$SECRET_NAME" --out-file="$WORK_DIR/token"
TOKEN=$(<"$WORK_DIR/token")
printf '%s\n' "$TOKEN"
sync_env_token "$TOKEN"
instructions
