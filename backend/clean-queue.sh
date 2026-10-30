#!/usr/bin/env sh
# Drena la Cola de Items con la Sesion gcloud Activa.
#
# Una Tarea no nombra su Item: dice "despertate y toma lo que haya". Un Turno
# que termina sin trabajar gasta un Despertar y nada lo repone, asi que un
# Item puede quedar esperando sin Tarea que lo llame. Este Script repone esos
# Despertares a mano.
#
# El Barredor hace lo mismo cada cinco Minutos. Esto es para antes de que el
# Barredor exista, o para no esperarlo.
#
# Drenar Busquedas ya vencidas fabrica Documentos envenenados mientras el
# Arreglo del Reclamo no este desplegado: al completarlas, el Item se
# estaciona en un Vencimiento que ya paso y vuelve elegible e inreclamable.
# Con el Arreglo puesto, se saltan; sin el, tapan la Cola de nuevo.
#
# Llama al Barredor, que dice cuantos Despertares repuso. Cero significa que
# no queda Trabajo esperando: es una Cuenta, no una Adivinanza. Medir por
# Tiempo no sirve, porque un Turno servido desde la Cache tarda lo mismo que
# uno vacio.
#
#   ./clean-queue.sh                 -> barre hasta que no quede Trabajo
#   ./clean-queue.sh --max 5         -> como mucho cinco Barridos
#   ./clean-queue.sh --dry-run       -> dice que haria
set -eu
cd -- "$(dirname -- "$0")"
. ./config/deploy.env
PROJECT=""
MAX=40
DRY_RUN=false
# Entre Barridos hay que dejar que la Cola se vacie: el Barredor encola, el
# Worker toma. Sin esta Espera, el Barrido siguiente cuenta lo mismo dos veces.
WAIT_SECONDS=30
STEP="Configuración"
trap 'code=$?; if [ "$code" -ne 0 ]; then printf "❌ %s Falló\n" "$STEP" >&2; fi' 0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --project|--region|--max)
      [ "$#" -ge 2 ] || { printf '%s\n' "Falta el Valor de $1" >&2; exit 1; }
      case "$1" in
        --project) PROJECT=$2 ;; --region) REGION=$2 ;; --max) MAX=$2 ;;
      esac
      shift 2 ;;
    --dry-run) DRY_RUN=true; shift ;;
    --help|-h)
      printf '%s\n' '🐱 clean-queue.sh [--project ID] [--region REGION]' \
        '  [--max BARRIDOS] [--dry-run]'
      exit 0 ;;
    *) printf '%s\n' "Argumento Desconocido: $1" >&2; exit 1 ;;
  esac
done
case "$MAX" in ''|*[!0-9]*) printf '%s\n' 'Usa --max con un Número.' >&2; exit 1 ;; esac
[ "$MAX" -gt 0 ] || { printf '%s\n' '--max debe ser mayor que cero.' >&2; exit 1; }
command -v gcloud >/dev/null 2>&1 || { printf '%s\n' 'Instala Google Cloud CLI.' >&2; exit 1; }
command -v curl >/dev/null 2>&1 || { printf '%s\n' 'Instala curl.' >&2; exit 1; }
[ -n "$PROJECT" ] || PROJECT=$(gcloud config get-value project 2>/dev/null)
case "$PROJECT" in ''|'(unset)'|*[!a-z0-9-]*) printf '%s\n' 'Selecciona un Proyecto con --project.' >&2; exit 1 ;; esac
case "$REGION" in ''|*[!a-z0-9-]*) printf '%s\n' 'Región Inválida.' >&2; exit 1 ;; esac

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
  PINK=$(printf '\033[38;5;211m'); DIM=$(printf '\033[2m'); OFF=$(printf '\033[0m')
else
  PINK=""; DIM=""; OFF=""
fi
say()  { printf '%s~nya~%s %s\n' "$PINK" "$OFF" "$1"; }
note() { printf '%s      %s%s\n' "$DIM" "$1" "$OFF"; }

STEP="Ubicar el Barredor"
SWEEPER_URL=$(gcloud functions describe "$SWEEPER_NAME" --gen2 --region="$REGION" \
  --project="$PROJECT" --format='value(serviceConfig.uri)' 2>/dev/null || true)
case "$SWEEPER_URL" in
  https://*) ;;
  *) printf '%s\n' "No encuentro el Barredor $SWEEPER_NAME en $REGION." >&2; exit 1 ;;
esac
say "Barriendo con $SWEEPER_NAME"
note "$SWEEPER_URL"
note "Hasta $MAX Barridos, esperando ${WAIT_SECONDS}s entre uno y otro"

if "$DRY_RUN"; then
  note "Ensayo: no se barre nada"
  printf '✅ %s\n' 'Vista Previa Completa'
  exit 0
fi

STEP="Barrer la Cola"
sweeps=0
total=0
TOKEN=$(gcloud auth print-identity-token)
while [ "$sweeps" -lt "$MAX" ]; do
  sweeps=$((sweeps + 1))
  if [ $((sweeps % 20)) -eq 0 ]; then TOKEN=$(gcloud auth print-identity-token); fi
  reply=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' -d '{}' "$SWEEPER_URL" || printf '')
  woken=$(printf '%s' "$reply" | sed -n 's/.*"woken":[[:space:]]*\([0-9][0-9]*\).*/\1/p')
  case "$woken" in
    ''|*[!0-9]*)
      printf '%s\n' "El Barredor no contestó una Cuenta: $reply" >&2
      exit 1 ;;
  esac
  if [ "$woken" -eq 0 ]; then
    say "No queda Trabajo esperando; repuse $total Despertares"
    printf '✅ %s\n' 'Cola Barrida'
    exit 0
  fi
  total=$((total + woken))
  note "Barrido $sweeps: repuso $woken Despertares"
  sleep "$WAIT_SECONDS"
done
say "Corté en el Tope de $MAX Barridos; repuse $total Despertares"
note "Vuelve a correrlo si queda Trabajo esperando"
printf '⚠️  %s\n' 'Tope Alcanzado'
