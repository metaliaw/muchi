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
#   ./clean_queue.sh                 -> drena hasta que la Cola se calle
#   ./clean_queue.sh --max 20        -> como mucho veinte Turnos
#   ./clean_queue.sh --dry-run       -> dice que haria
set -eu
cd -- "$(dirname -- "$0")"
. ./config/deploy.env
PROJECT=""
MAX=500
DRY_RUN=false
# Un Turno que trabaja tarda Segundos; uno que no encuentra nada vuelve al
# instante. Ese Corte separa "hay Trabajo" de "la Cola esta vacia".
IDLE_SECONDS=2
# Un solo Turno rapido puede ser una Carrera con otro Worker. Tres seguidos
# son una Cola vacia.
IDLE_STREAK=3
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
      printf '%s\n' '🐱 clean_queue.sh [--project ID] [--region REGION]' \
        '  [--max TURNOS] [--dry-run]'
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

STEP="Ubicar el Worker"
WORKER_URL=$(gcloud functions describe "$WORKER_NAME" --gen2 --region="$REGION" \
  --project="$PROJECT" --format='value(serviceConfig.uri)' 2>/dev/null || true)
case "$WORKER_URL" in
  https://*) ;;
  *) printf '%s\n' "No encuentro el Worker $WORKER_NAME en $REGION." >&2; exit 1 ;;
esac
say "Drenando $WORKER_NAME"
note "$WORKER_URL"
note "Hasta $MAX Turnos, o hasta $IDLE_STREAK Turnos vacíos seguidos"

if "$DRY_RUN"; then
  note "Ensayo: no se invoca a nadie"
  printf '✅ %s\n' 'Vista Previa Completa'
  exit 0
fi

STEP="Drenar la Cola"
turns=0
worked=0
idle=0
# El Token dura una Hora; se renueva cada cincuenta Turnos por si el Drenaje
# es largo.
TOKEN=$(gcloud auth print-identity-token)
while [ "$turns" -lt "$MAX" ]; do
  turns=$((turns + 1))
  if [ $((turns % 50)) -eq 0 ]; then TOKEN=$(gcloud auth print-identity-token); fi
  seconds=$(curl -s -o /dev/null -w '%{time_total}' -X POST \
    -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
    -d '{}' "$WORKER_URL" || printf '0')
  # Sin Decimales: dash no compara Fracciones, y el Corte no las necesita.
  whole=${seconds%%.*}
  case "$whole" in ''|*[!0-9]*) whole=0 ;; esac
  if [ "$whole" -ge "$IDLE_SECONDS" ]; then
    worked=$((worked + 1))
    idle=0
    note "Turno $turns: trabajó ${seconds}s"
  else
    idle=$((idle + 1))
    if [ "$idle" -ge "$IDLE_STREAK" ]; then
      say "La Cola quedó vacía tras $worked Turnos con Trabajo"
      printf '✅ %s\n' 'Cola Drenada'
      exit 0
    fi
  fi
done
say "Corté en el Tope de $MAX Turnos; $worked hicieron Trabajo"
note "Vuelve a correrlo si quedan Ítems esperando"
printf '⚠️  %s\n' 'Tope Alcanzado'
