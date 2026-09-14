#!/usr/bin/env bash
#
# Onboarding de la Publicidad: deja listo todo lo que se puede dejar listo.
#
# La Cuenta, el Sitio y el Bloque los crea la Interfaz de AdSense y no hay API
# que los reemplace: sites y adunits solo Leen, y adunits.create esta reservado
# a los Publishers de AdSense for Platforms. Lo que si se automatiza es la otra
# mitad, la que decide la Aprobacion: que la API responda, que el Dominio este
# en el aire, que el ads.txt lo declare y que el Identificador sea el mismo en
# los tres Lugares donde aparece. Google revisa eso; equivocarlo cuesta Dias.
#
#   ./setup-ads.sh
#
# Es idempotente: repetirlo no cambia nada que ya este bien. Termina diciendo
# que Paso sigue y quien lo tiene que dar.
#
set -euo pipefail

CLIENT="${MUCHI_ADSENSE_CLIENT:-ca-pub-6368656861543000}"
ACCOUNT="${CLIENT#ca-}"
BASE="https://adsense.googleapis.com/v2"
SCOPES="https://www.googleapis.com/auth/adsense,https://www.googleapis.com/auth/cloud-platform"
FALTA=0


# --------------------------------------------------------- lo que dice Muchi
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
    PINK=$'\033[38;5;211m'; DIM=$'\033[2m'; RED=$'\033[31m'
    GREEN=$'\033[32m'; YELLOW=$'\033[33m'; OFF=$'\033[0m'
else
    PINK=""; DIM=""; RED=""; GREEN=""; YELLOW=""; OFF=""
fi

say()  { printf '\n%s~nya~%s %s\n' "$PINK" "$OFF" "$1"; }
note() { printf '%s      %s%s\n' "$DIM" "$1" "$OFF"; }
ok()   { printf '  %s✓%s %s\n' "$GREEN" "$OFF" "$1"; }
# Una Falta no corta: el Onboarding se revisa entero de una Pasada, porque
# arreglar una Cosa y volver a esperar el Script es la Forma lenta.
bad()  { printf '  %s✗%s %s\n' "$RED" "$OFF" "$1"; FALTA=$((FALTA + 1)); }
warn() { printf '  %s!%s %s\n' "$YELLOW" "$OFF" "$1"; }
die()  { printf '%s~nya!~%s %s\n' "$RED" "$OFF" "$1" >&2; exit 1; }


# ------------------------------------------------------------ las Herramientas
check_tools() {
    say "Herramientas"
    local tool
    for tool in gcloud python3 curl; do
        command -v "$tool" >/dev/null 2>&1 || die "No Encuentro $tool"
    done
    ok "gcloud, python3 y curl estan"
}


# --------------------------------------------------------- el Proyecto y la API
# AdSense no es Cloud, pero su API se factura a un Proyecto de Cloud igual que
# cualquier otra: sin habilitarla, la Consulta muere en un 403 que habla de
# Permisos y no de Habilitacion.
enable_api() {
    say "Proyecto y API"
    PROJECT="$(gcloud config get-value project 2>/dev/null)"
    case "$PROJECT" in
        ''|'(unset)') die "No Hay Proyecto Activo: gcloud config set project TU-PROYECTO" ;;
    esac
    ok "Proyecto $PROJECT"

    if gcloud services list --enabled --format='value(config.name)' 2>/dev/null \
        | grep -qx adsense.googleapis.com; then
        ok "adsense.googleapis.com ya estaba habilitada"
    else
        note "Habilitando adsense.googleapis.com"
        gcloud services enable adsense.googleapis.com --quiet
        ok "adsense.googleapis.com habilitada"
    fi
}


# -------------------------------------------------------------- las Credenciales
# El ADC por Defecto no lleva el Scope de AdSense: hay que pedirlo aparte, y
# eso abre el Navegador. Se pregunta antes porque reescribe las Credenciales
# por Defecto de la Maquina, y algo mas podria estar apoyandose en ellas.
check_credentials() {
    say "Credenciales"
    if probe_api; then
        ok "El ADC ya alcanza para AdSense"
        return 0
    fi

    warn "El ADC no tiene el Scope de AdSense"
    note "Hace falta re-autorizar, y eso reescribe el ADC de esta Maquina"
    note "gcloud auth application-default login --scopes=$SCOPES"
    printf '  ¿Lo corro ahora? [s/N] '
    local respuesta
    read -r respuesta || respuesta=""
    case "$respuesta" in
        s|S|si|Si|y|Y) ;;
        *) die "Sin el Scope no hay nada que consultar. Corre el Comando de arriba cuando puedas" ;;
    esac

    gcloud auth application-default login --scopes="$SCOPES"
    probe_api || die "Sigue sin alcanzar. Revisa que la Cuenta elegida sea la Dueña de AdSense"
    ok "El ADC ya alcanza para AdSense"
}

probe_api() {
    local status
    status="$(curl -s -o /dev/null -w '%{http_code}' \
        -H "Authorization: Bearer $(gcloud auth application-default print-access-token 2>/dev/null)" \
        -H "x-goog-user-project: $PROJECT" "$BASE/accounts" || echo 000)"
    [ "$status" = "200" ]
}


# ------------------------------------------------------------ preguntar y leer
# El Cuerpo sale por RESPONSE y no por la Salida estandar: dentro de $( ) el
# die correria en un Subshell y su exit mataria solo a ese Subshell.
fetch() {
    local path="$1" body status
    body="$(curl -sS -w $'\n%{http_code}' \
        -H "Authorization: Bearer $(gcloud auth application-default print-access-token 2>/dev/null)" \
        -H "x-goog-user-project: $PROJECT" "$BASE/$path")"
    status="${body##*$'\n'}"
    RESPONSE="${body%$'\n'*}"
    [ "$status" = "200" ] || { printf '%s\n' "$RESPONSE" >&2; die "AdSense respondio $status en $path"; }
}

pluck() { printf '%s' "$1" | python3 -c 'import json,sys
datos = json.load(sys.stdin) or {}
filas = next((v for v in datos.values() if isinstance(v, list)), [])
for fila in filas:
    print("\t".join(str(fila.get(c, "")) for c in sys.argv[1].split(",")))' "$2"; }


# ----------------------------------------------------------------- el Dominio
# El Dominio no se adivina ni se escribe a mano: se pregunta a AdSense cual
# registraste, porque es ese y no otro el que la Revision va a visitar.
find_domain() {
    say "Sitio registrado"
    fetch "accounts/$ACCOUNT/sites"
    local sitios
    sitios="$(pluck "$RESPONSE" "domain,state")"

    if [ -z "$sitios" ]; then
        bad "AdSense no tiene ningun Sitio registrado"
        note "Agregalo en https://adsense.google.com -> Sitios -> Agregar sitio"
        DOMAIN=""
        return 0
    fi

    # Redireccion y no Tuberia: detras de un | el while corre en un Subshell y
    # cada ✗ que contara ahi se perderia al cerrarlo, dejando el Total mintiendo.
    while IFS=$'\t' read -r dominio estado; do
        case "$estado" in
            READY) ok "$dominio esta READY" ;;
            *)     warn "$dominio esta $estado (Google lo esta revisando)" ;;
        esac
    done < <(printf '%s\n' "$sitios")

    DOMAIN="${MUCHI_SITE_DOMAIN:-$(printf '%s\n' "$sitios" | head -1 | cut -f1)}"
    note "Verificando https://$DOMAIN"
}


# -------------------------------------------------- lo que la Revision va a ver
# Estas son las Comprobaciones que valen: son exactamente las que fallan solas
# y en silencio, y cada Falla cuesta otra Ronda de Revision.
check_site() {
    [ -n "$DOMAIN" ] || return 0
    say "Lo que la Revision va a ver en $DOMAIN"

    local status
    status="$(curl -s -o /dev/null -w '%{http_code}' -L --max-time 20 "https://$DOMAIN/" || echo 000)"
    if [ "$status" = "200" ]; then
        ok "La Portada responde 200 por https"
    else
        bad "La Portada responde $status: la Revision no va a poder entrar"
    fi

    # El ads.txt tiene que vivir en la Raiz del Dominio aprobado, no en el
    # .web.app de Hosting: un Dominio sin ads.txt propio no declara nada.
    local ads
    ads="$(curl -s -L --max-time 20 "https://$DOMAIN/ads.txt" || echo "")"
    if printf '%s' "$ads" | grep -q "${CLIENT#ca-}"; then
        ok "ads.txt declara ${CLIENT#ca-}"
    else
        bad "ads.txt no declara ${CLIENT#ca-} en la Raiz del Dominio"
        note "Lo sirve el BFF en /ads.txt; revisa que el Dominio derive ahi"
    fi

    local html
    html="$(curl -s -L --max-time 20 "https://$DOMAIN/" || echo "")"
    if printf '%s' "$html" | grep -q "google-adsense-account.*$CLIENT"; then
        ok "La Etiqueta google-adsense-account lleva $CLIENT"
    else
        bad "Falta la Etiqueta google-adsense-account con $CLIENT en el <head>"
        note "Vive en web/index.html"
    fi

    if printf '%s' "$html" | grep -q "adsbygoogle.js?client=$CLIENT"; then
        ok "El Cargador adsbygoogle.js apunta a $CLIENT"
    else
        bad "El Cargador adsbygoogle.js falta o apunta a otro Cliente"
        note "Vive en web/index.html"
    fi

    # El Identificador aparece en tres Lugares y ninguno lee a los otros: el
    # <head> compilado, el /ads.txt del BFF y el /api/config que mira el Front.
    # Si se separan, la Publicidad se pide a un Editor y se declara a otro.
    # Las tres Comprobaciones que siguen salen de /api/config, asi que primero
    # hay que saber que contesto algo. Buscar una Cadena dentro de una Respuesta
    # vacia da Ausencia, y la Ausencia se leeria como Exito.
    local config
    config="$(curl -s -L --max-time 20 "https://$DOMAIN/api/config" || echo "")"
    if ! printf '%s' "$config" | grep -q '"adsense_client"'; then
        bad "/api/config no contesta un Config: el BFF no esta detras de $DOMAIN"
        return 0
    fi

    if printf '%s' "$config" | grep -q "\"adsense_client\":\"$CLIENT\""; then
        ok "/api/config reporta el mismo Cliente"
    else
        bad "/api/config reporta otro Cliente que el <head> y el ads.txt"
    fi

    if printf '%s' "$config" | grep -q '"adsense_slot":""'; then
        warn "/api/config trae adsense_slot vacio: la Publicidad esta apagada"
    else
        ok "/api/config ya trae un Slot"
    fi

    if printf '%s' "$config" | grep -q '"environment":"production"'; then
        ok "El Servicio corre en production"
    else
        bad "El Servicio no dice production: AdSpot.vue no pide Anuncios fuera de ahi"
    fi
}


# --------------------------------------------------------------- lo que avisa
# Un Bloqueo de Politica no aparece en ningun state: vive en las Alertas, y sin
# leerlas la Espera parece Tramite cuando en realidad hay algo que arreglar.
check_alerts() {
    say "Alertas de la Cuenta"
    fetch "accounts/$ACCOUNT/alerts"
    local alertas
    alertas="$(pluck "$RESPONSE" "severity,message")"
    if [ -z "$alertas" ]; then
        ok "Ninguna"
        return 0
    fi
    while IFS=$'\t' read -r severidad mensaje; do
        case "$severidad" in
            SEVERE) bad "$severidad: ${mensaje:0:150}" ;;
            *)      warn "$severidad: ${mensaje:0:150}" ;;
        esac
    done < <(printf '%s\n' "$alertas")
}


# -------------------------------------------------------- y que sigue, y de quien
cierre() {
    say "Estado en AdSense"
    ./check-ads.sh | tail -n +3

    say "Lo que falta"
    if [ "$FALTA" -gt 0 ]; then
        note "$FALTA Cosa(s) marcadas con ✗ arriba: esas son tuyas y se arreglan"
        note "antes de esperar la Revision, porque la Revision las va a mirar."
    else
        note "Del lado del Codigo y del Despliegue, nada."
    fi
    echo
    note "A mano, porque la API solo Lee:"
    note "  1. El Sitio en READY -> lo decide Google revisando. Solo esperar."
    note "  2. El Bloque -> https://adsense.google.com"
    note "     Anuncios -> Por bloque de anuncios -> Display -> Adaptable"
    note "  3. El Slot al Despliegue -> MUCHI_ADSENSE_SLOT=<slot> ./deploy.sh"
    echo
    note "./check-ads.sh repite el Estado sin repetir estas Comprobaciones."
}


cd "$(dirname "$0")"
check_tools
enable_api
check_credentials
find_domain
check_site
check_alerts
cierre
