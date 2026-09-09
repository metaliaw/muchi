#!/usr/bin/env bash
#
# Muchi a Cloud Run. Tres tiempos: Verifica la Cuenta, Verifica el Secreto,
# Sube el Servicio. Nada mas.
#
# Cloud Run escala a cero: sin visitas no hay Instancia encendida ni cobro, y
# la primera visita paga el cold start. Streamlit necesita WebSockets, que es
# la razon por la que esto no vive en App Engine.
#
#   ./deploy.sh                     -> deploy con la region por defecto
#   MUCHI_REGION=us-east1 ./deploy.sh
#
# El Token lo comparte muchi-api, que ya lo guardo en Secret Manager al
# desplegarse. La URL viaja como Variable porque el endpoint es publico:
# guardar en el Vault algo que cualquiera puede leer solo esconde donde mirar.
#
set -euo pipefail

SERVICE="muchi"
# La misma Region que la API: cada Busqueda cruza este salto, y entre
# continentes cuesta cientos de milisegundos que el Usuario si nota.
REGION="${MUCHI_REGION:-southamerica-east1}"
API_URL="${MUCHI_API_URL:-https://muchi-serve-api-c2ce6c7oga-rj.a.run.app}"
TOKEN_SECRET="muchi-api-token"


# --------------------------------------------------------- lo que dice Muchi
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
    PINK=$'\033[38;5;211m'; DIM=$'\033[2m'; RED=$'\033[31m'; OFF=$'\033[0m'
else
    PINK=""; DIM=""; RED=""; OFF=""
fi

say()  { printf '%s~nya~%s %s\n' "$PINK" "$OFF" "$1"; }
note() { printf '%s      %s%s\n' "$DIM" "$1" "$OFF"; }
die()  { printf '%s~nya!~%s %s\n' "$RED" "$OFF" "$1" >&2; exit 1; }


# ------------------------------------------------------- la cuenta y el sitio
# Un deploy sin Proyecto activo no falla en el comando: falla dos minutos
# despues, con un error de permisos que no menciona el Proyecto por ningun lado.
check_account() {
    command -v gcloud >/dev/null 2>&1 \
        || die "No Encuentro gcloud. Instala el SDK de Google Cloud"

    PROJECT="$(gcloud config get-value project 2>/dev/null)"
    case "$PROJECT" in
        ''|'(unset)') die "No Hay Proyecto Activo: gcloud config set project TU-PROYECTO" ;;
    esac

    say "Deployando $SERVICE a $PROJECT ($REGION)"
    note "API: $API_URL"
}


# ------------------------------------------------------- el secreto, primero
# Se comprueba antes de construir la Imagen. Un build dura minutos y termina en
# un Servicio que arranca sin Credenciales: mejor cortar en el segundo uno.
check_secret() {
    gcloud secrets describe "$TOKEN_SECRET" >/dev/null 2>&1 \
        || die "Falta el Secreto '$TOKEN_SECRET'. Lo Crea el deploy.sh de muchi-api"
}


# ------------------------------------------------------ el permiso de leerlo
# El Servicio corre como la Cuenta de Compute por defecto, que nace sin Acceso
# al Vault. Sin esta Concesion el Contenedor ni siquiera arranca.
grant_secret_access() {
    local number email
    number="$(gcloud projects describe "$PROJECT" --format='value(projectNumber)')"
    email="$number-compute@developer.gserviceaccount.com"

    gcloud secrets add-iam-policy-binding "$TOKEN_SECRET" \
        --member="serviceAccount:$email" \
        --role=roles/secretmanager.secretAccessor \
        --condition=None --format=none --quiet
}


# --------------------------------------------------------------- y a subirlo
# --source construye la Imagen con Cloud Build y la publica en Artifact
# Registry sin Dockerfile local que empujar. La Afinidad de Sesion manda cada
# Pestana a la misma Instancia: el estado de Streamlit vive en memoria, y una
# Busqueda que rebota de Instancia vuelve a empezar de cero.
push_service() {
    say "Construyendo y Subiendo -- esto tarda unos minutos"
    note "El primer deploy crea el Repositorio de Imagenes"
    echo

    gcloud run deploy "$SERVICE" \
        --source . \
        --region "$REGION" \
        --platform managed \
        --allow-unauthenticated \
        --port 8080 \
        --memory 1Gi \
        --cpu 1 \
        --min-instances 0 \
        --max-instances 4 \
        --timeout 3600 \
        --session-affinity \
        --set-env-vars "MUCHI_ENV=production,MUCHI_API_URL=$API_URL" \
        --set-secrets "MUCHI_API_TOKEN=$TOKEN_SECRET:latest"
}


check_account
check_secret
grant_secret_access
push_service
