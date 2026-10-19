#!/usr/bin/env bash
# Levanta el BFF en Desarrollo, en 8000. Vite le reenvía /api desde 5173.
set -euo pipefail
cd -- "$(dirname -- "$0")"

if [ ! -f .env ]; then
  echo "Falta bff/.env. Copia bff/.env.example y completa MUCHI_API_URL y MUCHI_API_TOKEN." >&2
  exit 1
fi

PYTHON=python3
if [ -x .venv/bin/python ]; then PYTHON=.venv/bin/python; fi
"$PYTHON" -m pip install -q -r requirements-web.txt
"$PYTHON" -m pip install -q requests pyyaml python-dotenv

# El Recargador mira solo lo que el BFF sirve: sin .venv ni node_modules se
# despierta rapido, y el Catalogo recarga igual que el Codigo.
exec "$PYTHON" -m uvicorn server.main:app --port 8000 \
  --reload \
  --reload-dir server --reload-dir muchi --reload-dir constants \
  --reload-include '*.json'
