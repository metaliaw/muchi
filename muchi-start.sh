#!/usr/bin/env bash
#
# Muchi, de cero a corriendo. Tres tiempos: Prepara el entorno, Instala lo que
# falte, Levanta el server. Nada mas.
#
# Es idempotente: la primera vez crea el .venv y baja las dependencias, las
# siguientes arranca directo. Solo reinstala si requirements.txt cambio.
#
#   ./muchi-start.sh                            -> http://localhost:8501
#   ./muchi-start.sh --server.port 9123         -> otro puerto
#   ./muchi-start.sh --server.address 127.0.0.1 -> solo tu maquina
#
# Levanta dos procesos: la API (FastAPI, :8000) y la app (Streamlit, :8501).
# Si MUCHI_API_URL ya apunta a otra parte, la API local no se levanta.
#
# Todo lo que le pases viaja tal cual a `streamlit run app.py`.
#
set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
VENV="$ROOT/.venv"
VENV_PYTHON="$VENV/bin/python"
DEPS_STAMP="$VENV/.muchi-deps"
MIN_PYTHON="3.12"


# --------------------------------------------------------- lo que dice Muchi
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
    PINK=$'\033[38;5;211m'; DIM=$'\033[2m'; RED=$'\033[31m'; OFF=$'\033[0m'
else
    PINK=""; DIM=""; RED=""; OFF=""
fi

say()   { printf '%s~nya~%s %s\n' "$PINK" "$OFF" "$1"; }
note()  { printf '%s      %s%s\n' "$DIM" "$1" "$OFF"; }
die()   { printf '%s~nya!~%s %s\n' "$RED" "$OFF" "$1" >&2; exit 1; }


# ---------------------------------------------------- buscar un python usable
# El interprete del sistema cambia de nombre segun la distro, y algunos son
# demasiado viejos para streamlit. Probamos por orden y verificamos la version
# preguntandosela al propio interprete, no parseando su `--version`.
find_python() {
    local candidate
    for candidate in "${MUCHI_PYTHON:-}" python3 python; do
        [ -n "$candidate" ] || continue
        command -v "$candidate" >/dev/null 2>&1 || continue
        "$candidate" -c 'import sys; raise SystemExit(0 if sys.version_info >= (3, 12) else 1)' 2>/dev/null || continue
        printf '%s' "$candidate"
        return 0
    done
    return 1
}


# ------------------------------------------------------------ el entorno solo
# Si el .venv ya esta, no lo tocamos. Si no, lo creamos con el python que
# encontramos -- y si falta el modulo venv (Debian y Ubuntu lo separan en su
# propio paquete) lo decimos con nombre y apellido en vez de escupir un
# traceback.
prepare_venv() {
    if [ -x "$VENV_PYTHON" ]; then
        return 0
    fi

    local python
    python="$(find_python)" || die "No Encuentro Python $MIN_PYTHON o Mas Nuevo (el README lo Pide). Instalalo, o Apuntame al tuyo con MUCHI_PYTHON=/ruta/a/python"

    say "Creando el Entorno en .venv con $($python -c 'import sys; print("Python %d.%d.%d" % sys.version_info[:3])')"
    if ! "$python" -m venv "$VENV" 2>/dev/null; then
        die "No Pude Crear el Entorno. En Debian/Ubuntu Falta el Paquete: sudo apt install python3-venv"
    fi

    "$VENV_PYTHON" -m pip install --quiet --upgrade pip
}


# ------------------------------------------------- las dependencias, una vez
# La huella de requirements.txt queda guardada dentro del .venv. Mientras el
# archivo no cambie, arrancar cuesta cero: no hay pip que resuelva nada.
install_requirements() {
    local fingerprint
    fingerprint="$(cksum "$ROOT/requirements.txt")"

    if [ -f "$DEPS_STAMP" ] && [ "$(cat "$DEPS_STAMP")" = "$fingerprint" ]; then
        return 0
    fi

    say "Instalando Dependencias (esto pasa una sola vez)"
    "$VENV_PYTHON" -m pip install --quiet --requirement "$ROOT/requirements.txt" \
        || die "Fallo la Instalacion de Dependencias"

    printf '%s' "$fingerprint" > "$DEPS_STAMP"
}


# ------------------------------------------------------------ y a levantarlo
# Dos procesos: la API (FastAPI) y la app (Streamlit). La app le habla a la API
# por HTTP; aca se levantan las dos juntas para no andar abriendo terminales.
# Si MUCHI_API_URL ya apunta a otra parte, no se levanta la local.
wait_for_api() {
    local i
    for i in $(seq 1 60); do
        "$VENV_PYTHON" -c "import urllib.request; urllib.request.urlopen('$1/inventory', timeout=1)" 2>/dev/null \
            && return 0
        sleep 0.5
    done
    return 1
}

run_all() {
    say "Levantando Muchi -- Ctrl+C para Cortar"
    note "API en http://localhost:8000, app en http://localhost:8501"
    echo
    cd "$ROOT"

    if [ -z "${MUCHI_API_URL:-}" ]; then
        export MUCHI_API_URL="http://localhost:8000"
        "$VENV_PYTHON" -m uvicorn muchi.api.app:app --host 127.0.0.1 --port 8000 &
        API_PID=$!
        trap 'kill "$API_PID" 2>/dev/null' EXIT INT TERM
        note "Esperando a que la API este lista..."
        wait_for_api "$MUCHI_API_URL" || die "La API no Levanto. Corre: $VENV_PYTHON -m uvicorn muchi.api.app:app"
    fi

    "$VENV_PYTHON" -m streamlit run app.py "$@"
}


prepare_venv
install_requirements
run_all "$@"
