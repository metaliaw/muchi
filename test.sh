#!/usr/bin/env sh
# Runs every Test of the Repository: the Server in Python, the Front in JS.
set -eu
cd -- "$(dirname -- "$0")"

PYTHON=python3
if [ -x .venv/bin/python ]; then PYTHON=.venv/bin/python; fi

"$PYTHON" -m pip install -q -r bff/requirements-dev.txt

echo "== Python"
PYTHONPATH=bff "$PYTHON" -m pytest -q bff/tests "$@"

echo "== JavaScript"
# Sin Dependencias instaladas no hay Runner: `npm ci` las Trae una vez y el
# Script Sigue sirviendo en una Copia recién clonada.
if [ ! -d frontend/node_modules ]; then npm --prefix frontend ci; fi
npm --prefix frontend run test
