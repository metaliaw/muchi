#!/usr/bin/env sh
# Verifica el BFF: sus Pruebas en Python.
set -eu
cd -- "$(dirname -- "$0")"

PYTHON=python3
if [ -x .venv/bin/python ]; then PYTHON=.venv/bin/python; fi

"$PYTHON" -m pip install -q -r requirements-dev.txt
PYTHONPATH=. "$PYTHON" -m pytest -q "$@"
