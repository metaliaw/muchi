#!/usr/bin/env bash
# Imprime el Token de Muchi en stdout y lo sincroniza en .env.
#
# Es para el Desarrollo local: en la Nube los Servicios montan el Token desde
# Secret Manager y nadie lo copia a mano a ningun panel.
set +x
set -euo pipefail
umask 077
ROOT=$(cd -- "$(dirname -- "$0")" && pwd)
# shellcheck source=config/deploy.env
source "$ROOT/config/deploy.env"
PROJECT=""
SECRET_NAME=${MUCHI_API_TOKEN%:*}
VERSION="latest"
WORK_DIR=""

usage() {
  printf '%s\n' '🐱 get-secret.sh --project ID [--secret NOMBRE] [--version NUMERO]' \
    'Imprime el Valor del Secreto en stdout y lo Escribe en .env.'
}

cleanup() {
  if [[ -n "$WORK_DIR" ]]; then rm -rf -- "$WORK_DIR"; fi
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
printf '✅ %s:%s Escrito en .env\n' "$SECRET_NAME" "$VERSION" >&2
