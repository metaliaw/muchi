#!/usr/bin/env bash
#
# Muchi a Cloud Run y Firebase Hosting. Verifica, Construye y Publica ambos.
#
# El Front y el BFF viajan en una sola Imagen y un solo Servicio: un Origen,
# ningun CORS, y el Codigo de Seguridad nunca cruza al Navegador.
#
# Cloud Run escala a cero: sin visitas no hay Instancia encendida ni cobro, y
# la primera visita paga el cold start.
#
#   ./deploy.sh                      -> despliega el Commit actual
#   MUCHI_REGION=us-east1 ./deploy.sh
#
# El Token lo comparte muchi-api, que lo guarda en Secret Manager. Rotarlo es
# Asunto suyo: ./rotate-secret.sh de ese Repositorio alcanza a este Servicio.
#
set -euo pipefail

SERVICE="muchi-web"
# La misma Region que la API: el BFF la consulta en cada Busqueda, y ese salto
# lo paga el Usuario esperando.
REGION="${MUCHI_REGION:-southamerica-east1}"
REPOSITORY="muchi"
TOKEN_SECRET="muchi-api-token"
CONFIG="cloudbuild.yaml"


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
# Un deploy sin Proyecto activo no falla en el comando: falla minutos despues,
# con un error de permisos que no menciona el Proyecto por ningun lado.
check_account() {
    command -v gcloud >/dev/null 2>&1 \
        || die "No Encuentro gcloud. Instala el SDK de Google Cloud"
    command -v npm >/dev/null 2>&1 \
        || die "No Encuentro npm. Instala Node.js"
    command -v firebase >/dev/null 2>&1 \
        || die "No Encuentro firebase. Instala Firebase CLI"
    [ -f "$CONFIG" ] || die "Falta $CONFIG. Corre esto desde la Raiz del Repositorio"
    [ -f firebase.json ] || die "Falta firebase.json. Ejecuta firebase init hosting"

    PROJECT="$(gcloud config get-value project 2>/dev/null)"
    case "$PROJECT" in
        ''|'(unset)') die "No Hay Proyecto Activo: gcloud config set project TU-PROYECTO" ;;
    esac

    # SHORT_SHA solo lo llena un Trigger. En un submit manual queda vacio y la
    # Imagen sale con la Etiqueta rota, asi que el SHA se pasa a mano.
    TAG="$(git rev-parse --short HEAD 2>/dev/null || echo manual)"
    if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
        TAG="$TAG-sucio"
    fi

    say "Deployando $SERVICE a $PROJECT ($REGION)"
    note "Imagen: $REPOSITORY/$SERVICE:$TAG"
}


# ------------------------------------------------------- el secreto, primero
# Se comprueba antes de construir. Un build dura minutos y termina en un
# Servicio que arranca sin Credenciales: mejor cortar en el segundo uno.
check_secret() {
    gcloud secrets describe "$TOKEN_SECRET" >/dev/null 2>&1 \
        || die "Falta el Secreto '$TOKEN_SECRET'. Lo Crea el deploy.sh de muchi-api"
}


# ------------------------------------------------- donde vive la Imagen
# El Repositorio no se crea solo: sin el, el push falla recien al final del
# build, despues de haber compilado todo.
prepare_repository() {
    gcloud artifacts repositories describe "$REPOSITORY" \
        --location="$REGION" >/dev/null 2>&1 && return 0

    say "Creando el Repositorio de Imagenes '$REPOSITORY'"
    gcloud artifacts repositories create "$REPOSITORY" \
        --repository-format=docker --location="$REGION" \
        --description="Imagenes de Muchi" --quiet
}


# ------------------------------------------------------ quien puede que cosa
# Cloud Build compila, pero ademas despliega: necesita mandar en Cloud Run y
# actuar como la Cuenta del Servicio. El Servicio, a su vez, necesita leer el
# Token. Las Concesiones son idempotentes: repetirlas no cambia nada.
grant_access() {
    local number builder runtime
    number="$(gcloud projects describe "$PROJECT" --format='value(projectNumber)')"
    builder="$number-compute@developer.gserviceaccount.com"
    # Sin --service-account, Cloud Run corre el Servicio como esa misma Cuenta.
    runtime="$builder"

    local role
    for role in roles/run.admin roles/artifactregistry.writer roles/logging.logWriter; do
        gcloud projects add-iam-policy-binding "$PROJECT" \
            --member="serviceAccount:$builder" --role="$role" \
            --condition=None --format=none --quiet
    done
    gcloud iam service-accounts add-iam-policy-binding "$runtime" \
        --member="serviceAccount:$builder" --role=roles/iam.serviceAccountUser \
        --condition=None --format=none --quiet

    gcloud secrets add-iam-policy-binding "$TOKEN_SECRET" \
        --member="serviceAccount:$runtime" \
        --role=roles/secretmanager.secretAccessor \
        --condition=None --format=none --quiet
}


# --------------------------------------------------------------- y a subirlo
# Cloud Build compila el Front con Node y lo mete en la Imagen del BFF, asi
# que node_modules nunca sube: .gcloudignore lo deja afuera y npm lo rearma
# adentro, igual en cualquier Maquina.
push_service() {
    say "Construyendo y Subiendo -- esto tarda unos minutos"
    note "El Front se compila dentro de la Imagen, no aca"
    echo

    local build_options
    build_options="_SERVICE=$SERVICE,_REGION=$REGION,_TOKEN_SECRET=$TOKEN_SECRET,_TAG=$TAG"
    build_options+=",_ADSENSE_CLIENT=${MUCHI_ADSENSE_CLIENT:-},_ADSENSE_SLOT=${MUCHI_ADSENSE_SLOT:-}"
    gcloud builds submit --config "$CONFIG" --substitutions="$build_options"

    local url
    url="$(gcloud run services describe "$SERVICE" --region="$REGION" \
        --format='value(status.url)')"
    say "Muchi esta en el aire"
    note "$url"
}


# Firebase sirve los Archivos estáticos y deriva el BFF al Servicio recién
# publicado. Compilar después de Cloud Run evita adelantar el Front al Servidor.
build_hosting() {
    say "Compilando el Front para Firebase Hosting"
    npm --prefix web ci
    npm --prefix web run build
}


push_hosting() {
    say "Publicando Firebase Hosting"
    firebase deploy --only hosting --project "$PROJECT"

    say "Muchi está publicado"
    note "https://$PROJECT.web.app"
}


check_account
check_secret
prepare_repository
grant_access
push_service
build_hosting
push_hosting
