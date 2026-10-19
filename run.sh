#!/usr/bin/env bash
# Levanta el BFF en 8000 y el Front en 5173. La API es la que apunte
# bff/.env; levantarla en local es backend/start.sh, y Pide Firestore.
set -euo pipefail
cd -- "$(dirname -- "$0")"

start_services() {
	(cd bff && ./run.sh) &
	bff_pid=$!
	(cd frontend && ./run.sh) &
	frontend_pid=$!
	trap stop_services INT TERM EXIT
	printf '%s\n' 'Front: http://localhost:5173' 'BFF:   http://localhost:8000/api/health' 'Ctrl+C detiene ambos procesos.'
	wait -n "$bff_pid" "$frontend_pid"
}

stop_services() {
	trap - INT TERM EXIT
	kill "$bff_pid" "$frontend_pid" 2>/dev/null || true
	wait "$bff_pid" "$frontend_pid" 2>/dev/null || true
}

start_services
