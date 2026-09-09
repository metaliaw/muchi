#!/usr/bin/env bash
# Levanta el Front nuevo en Desarrollo: el BFF en 8000 y Vite en 5173.
# Vite reenvía /api al BFF, así el Navegador ve un solo Origen, igual que en
# Producción, donde el mismo Servicio sirve el Front compilado.
set -euo pipefail
cd "$(dirname "$0")"

if [ ! -f .env ]; then
  echo "Falta .env. Copia .env.example y completa MUCHI_API_URL y MUCHI_API_TOKEN." >&2
  exit 1
fi

python3 -m pip install -q -r requirements-web.txt
python3 -m pip install -q requests pyyaml python-dotenv

if [ ! -d web/node_modules ]; then
  (cd web && npm install)
fi

# El Recargador mira solo lo que el BFF sirve: sin .venv ni node_modules se
# despierta rapido, y el Catalogo recarga igual que el Codigo.
python3 -m uvicorn server.main:app --port 8000 \
  --reload \
  --reload-dir server --reload-dir muchi --reload-dir constants \
  --reload-include '*.json' &
BFF=$!
trap 'kill "$BFF" 2>/dev/null || true' EXIT INT TERM

cd web && npm run dev
