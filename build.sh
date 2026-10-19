#!/usr/bin/env sh
# Verifica cada Componente: la API, el BFF y el Front.
set -eu
cd -- "$(dirname -- "$0")"

case "${1:-all}" in
	api) ./backend/build.sh ;;
	web) ./bff/build.sh && ./frontend/build.sh ;;
	all) ./backend/build.sh && ./bff/build.sh && ./frontend/build.sh ;;
	*) echo "Uso: ./build.sh [api|web|all]" >&2; exit 1 ;;
esac
