#!/usr/bin/env bash
#
# Estado de la Publicidad en AdSense: si ya hay un Slot que desplegar.
#
# MUCHI_ADSENSE_SLOT no se inventa: lo asigna Google al crear un Bloque, y un
# Bloque no se puede crear mientras la Cuenta revisa el Sitio. Este Script
# pregunta en que Paso de esa Espera estamos, sin abrir la Interfaz.
#
#   ./check-ads.sh
#
# La API solo Lee. Crear el Bloque sigue siendo Asunto de la Interfaz de
# AdSense: adunits.create existe, pero esta reservado a los Publishers de
# AdSense for Platforms y a una Cuenta normal le responde que no.
#
set -euo pipefail

# El mismo Identificador que sirve el BFF en /ads.txt. Es publico: viaja en
# cada Pagina y el Archivo lo declara a Internet entero.
CLIENT="${MUCHI_ADSENSE_CLIENT:-ca-pub-6368656861543000}"
ACCOUNT="${CLIENT#ca-}"
BASE="https://adsense.googleapis.com/v2"


# --------------------------------------------------------- lo que dice Muchi
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
    PINK=$'\033[38;5;211m'; DIM=$'\033[2m'; RED=$'\033[31m'; OFF=$'\033[0m'
else
    PINK=""; DIM=""; RED=""; OFF=""
fi

say()  { printf '%s~nya~%s %s\n' "$PINK" "$OFF" "$1"; }
note() { printf '%s      %s%s\n' "$DIM" "$1" "$OFF"; }
die()  { printf '%s~nya!~%s %s\n' "$RED" "$OFF" "$1" >&2; exit 1; }


# ------------------------------------------------------- con que Credenciales
# AdSense no es Cloud: no alcanza con la Cuenta activa de gcloud, hace falta
# que el ADC lleve el Scope de AdSense. Sin el, la Consulta muere en un 403
# que habla de Scopes y no de como arreglarlo.
check_access() {
    command -v gcloud >/dev/null 2>&1 \
        || die "No Encuentro gcloud. Instala el SDK de Google Cloud"
    command -v python3 >/dev/null 2>&1 \
        || die "No Encuentro python3"

    PROJECT="$(gcloud config get-value project 2>/dev/null)"
    case "$PROJECT" in
        ''|'(unset)') die "No Hay Proyecto Activo: gcloud config set project TU-PROYECTO" ;;
    esac

    TOKEN="$(gcloud auth application-default print-access-token 2>/dev/null)" \
        || die "El ADC no Responde. Corre: gcloud auth application-default login --scopes=https://www.googleapis.com/auth/adsense,https://www.googleapis.com/auth/cloud-platform"
    [ -n "$TOKEN" ] \
        || die "El ADC no Responde. Corre: gcloud auth application-default login --scopes=https://www.googleapis.com/auth/adsense,https://www.googleapis.com/auth/cloud-platform"
}


# ------------------------------------------------------------ preguntar y leer
# La Cabecera x-goog-user-project no es Adorno: sin ella la Consulta se cobra
# al Proyecto generico de gcloud, donde adsense.googleapis.com no esta
# habilitada, y el 403 culpa a un Proyecto que no es el tuyo.
# El Cuerpo sale por RESPONSE y no por la Salida estandar: dentro de $( ) el
# die correria en un Subshell, su exit mataria solo a ese Subshell y el Flujo
# seguiria con un Cuerpo vacio hasta reventar mas adelante, lejos de la Causa.
fetch() {
    local path="$1" body status
    body="$(curl -sS -w $'\n%{http_code}' \
        -H "Authorization: Bearer $TOKEN" \
        -H "x-goog-user-project: $PROJECT" \
        "$BASE/$path")"
    status="${body##*$'\n'}"
    body="${body%$'\n'*}"

    if [ "$status" != "200" ]; then
        printf '%s\n' "$body" >&2
        case "$status" in
            401|403) die "AdSense respondio $status en $path. Puede ser el Scope del ADC, la API sin habilitar (gcloud services enable adsense.googleapis.com), o que $CLIENT no sea tu Identificador de Editor: el Cuerpo de arriba lo distingue" ;;
            404) die "AdSense respondio 404. Revisa que $CLIENT sea tu Identificador de Editor" ;;
            *)   die "AdSense respondio $status" ;;
        esac
    fi
    RESPONSE="$body"
}


# Cada Recurso trae un state propio. Se muestran los tres porque la Espera
# avanza de a uno: la Cuenta lista primero, el Cliente despues, el Sitio al
# final, y el Bloque recien cuando los tres estan READY.
show() {
    local json="$1" campos="$2" titulo="$3" vacio="$4"
    printf '%s' "$json" | python3 -c '
import json, sys
campos, titulo, vacio = sys.argv[1].split(","), sys.argv[2], sys.argv[3]
datos = json.load(sys.stdin) or {}
filas = next((v for v in datos.values() if isinstance(v, list)), [])
print(f"  {titulo}")
if not filas:
    print(f"    {vacio}")
for fila in filas:
    print("    " + "  ".join(str(fila.get(c, "-")) for c in campos))
' "$campos" "$titulo" "$vacio"
}


check_access
say "Publicidad de $CLIENT"
note "Proyecto $PROJECT"
echo

fetch "accounts"
show "$RESPONSE" "name,state" "Cuenta" "ninguna: la Cuenta de AdSense no existe todavia"

fetch "accounts/$ACCOUNT/adclients"
show "$RESPONSE" "reportingDimensionId,productCode,state" "Cliente" "ninguno"

fetch "accounts/$ACCOUNT/sites"
show "$RESPONSE" "domain,state,autoAdsEnabled" "Sitios" "ninguno: falta registrar el Dominio"

fetch "accounts/$ACCOUNT/adclients/$CLIENT/adunits"
UNITS="$RESPONSE"
show "$UNITS" "reportingDimensionId,displayName,state" \
     "Bloques" "ninguno: no hay Slot que desplegar"
echo

# El reportingDimensionId del Bloque es, literal, el data-ad-slot que espera
# GoogleAd.vue. Si hay uno Activo, el Despliegue ya puede encenderlo.
SLOT="$(printf '%s' "$UNITS" | python3 -c '
import json, sys
unidades = (json.load(sys.stdin) or {}).get("adUnits", [])
activos = [u for u in unidades if u.get("state") == "ACTIVE"]
print(activos[0]["reportingDimensionId"] if activos else "")
')"

if [ -n "$SLOT" ]; then
    say "Hay Slot: $SLOT"
    note "MUCHI_ADSENSE_SLOT=$SLOT ./deploy.sh"
else
    say "Todavia no hay Slot"
    note "GETTING_READY es Google revisando el Sitio; no hay nada que hacer"
    note "salvo esperar. READY habilita crear el Bloque en la Interfaz de"
    note "AdSense: Anuncios -> Por bloque de anuncios -> Display -> Adaptable"
fi
