#!/usr/bin/env bash
#
# Imprime el Token de Muchi en stdout y lo sincroniza en .env.
#
#   ./get-secret.sh                      -> la Version vigente
#   ./get-secret.sh --version 3          -> una Version anterior
#   ./get-secret.sh --secret otro-token  -> otro Secreto
#
# El Token lo guarda muchi-api en Secret Manager; los Servicios lo montan
# desde ahi. Este script existe para el Desarrollo local, donde no hay
# Cloud Run que monte nada y el .env es la unica via.
#
# `set +x` primero: con xtrace heredado del shell, el Token aparece en la
# traza de cada linea que lo toca.
set +x
set -euo pipefail
umask 077

ROOT="$(cd -- "$(dirname -- "$0")" && pwd)"
SECRET_NAME="muchi-api-token"
VERSION="latest"
PROJECT=""
WORK_DIR=""


fail()  { printf '❌ %s\n' "$1" >&2; exit 1; }
cloud() { gcloud "$@" --project="$PROJECT" --quiet; }

usage() {
    printf '%s\n' '🐱 get-secret.sh [--project ID] [--secret NOMBRE] [--version NUMERO]' \
        'Imprime el Valor del Secreto en stdout y lo Escribe en .env.'
}

# El Directorio temporal guarda el Token un instante. Se borra pase lo que
# pase, incluso si gcloud muere a la mitad.
cleanup() { [[ -n "$WORK_DIR" ]] && rm -rf -- "$WORK_DIR"; }
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM


# ------------------------------------------------------------ lo que se pide
read_arguments() {
    while (( $# )); do
        case "$1" in
            --project|--secret|--version)
                (( $# >= 2 )) || fail "Falta el Valor de $1"
                case "$1" in
                    --project) PROJECT=$2 ;;
                    --secret)  SECRET_NAME=$2 ;;
                    --version) VERSION=$2 ;;
                esac
                shift 2 ;;
            --help|-h) usage; exit 0 ;;
            *) fail "Argumento Desconocido: $1" ;;
        esac
    done
}


# --------------------------------------------------------- lo que debe estar
# Los Nombres se validan antes de llegar a gcloud: viajan a una linea de
# comandos, y un Valor con formas raras deja de ser un Argumento.
check_arguments() {
    [[ -n "$PROJECT" ]] || PROJECT="$(gcloud config get-value project 2>/dev/null)"

    [[ "$PROJECT" =~ ^[a-z][a-z0-9-]+$ ]] || fail 'Proyecto Inválido.'
    [[ "$SECRET_NAME" =~ ^[a-zA-Z0-9_-]+$ ]] || fail 'Nombre de Secreto Inválido.'
    [[ "$VERSION" == latest || "$VERSION" =~ ^[1-9][0-9]*$ ]] \
        || fail 'Usa latest o una Versión Numérica Positiva.'

    command -v gcloud >/dev/null || fail 'Instala Google Cloud CLI.'
    command -v mktemp >/dev/null || fail 'Instala coreutils (mktemp).'
}


# ------------------------------------------------------ el .env, sin perderlo
# Se reescribe linea por linea para conservar el resto del Archivo: un
# `sed -i` sobre un Token con barras o ampersands lo corrompe en silencio.
sync_env_token() {
    local token=$1
    local env_file="$ROOT/.env"
    local tmp

    if [[ ! -f "$env_file" ]]; then
        printf 'MUCHI_API_TOKEN=%s\n' "$token" > "$env_file"
        return 0
    fi

    tmp="$(mktemp "$env_file.XXXXXXXX")"
    while IFS= read -r line || [[ -n "$line" ]]; do
        if [[ "$line" == MUCHI_API_TOKEN=* ]]; then
            printf 'MUCHI_API_TOKEN=%s\n' "$token"
        else
            printf '%s\n' "$line"
        fi
    done < "$env_file" > "$tmp"
    mv -f "$tmp" "$env_file"
}


# ------------------------------------------------------------- y a buscarlo
# --out-file en vez de capturar stdout: gcloud escribe el Token en un Archivo
# con umask 077 y nunca pasa por una Variable de entorno de un subproceso.
show_token() {
    WORK_DIR="$(mktemp -d "${TMPDIR:-/tmp}/muchi-secret.XXXXXXXX")"
    cloud secrets versions access "$VERSION" --secret="$SECRET_NAME" --out-file="$WORK_DIR/token"

    local token
    token="$(<"$WORK_DIR/token")"
    printf '%s\n' "$token"
    sync_env_token "$token"

    printf '✅ %s:%s Escrito en .env\n' "$SECRET_NAME" "$VERSION" >&2
}


read_arguments "$@"
check_arguments
show_token
