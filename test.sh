#!/usr/bin/env sh
# Runs every Test of the Repository: the Server in Python, the Front in JS.
set -eu
cd -- "$(dirname -- "$0")"

PYTHON=python3
if [ -x .venv/bin/python ]; then PYTHON=.venv/bin/python; fi

echo "== Python"
PYTHONPATH=. "$PYTHON" -m pytest -q "$@"

echo "== JavaScript"
# Sin Dependencias instaladas no hay Runner: `npm ci` las Trae una vez y el
# Script Sigue sirviendo en una Copia recién clonada.
if [ ! -d web/node_modules ]; then npm --prefix web ci; fi
npm --prefix web run test
