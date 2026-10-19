#!/usr/bin/env bash
# Despliega Muchi: la API a Cloud Functions y la Web —BFF y Front— a Cloud Run
# y Firebase Hosting. La API va primero: la Web Consume sus Rutas.
#
#   ./deploy.sh        -> la API y luego la Web
#   ./deploy.sh api    -> solo la API (sus Opciones: backend/deploy.sh --help)
#   ./deploy.sh web    -> solo la Web
set -euo pipefail
cd -- "$(dirname -- "$0")"

target="${1:-all}"
[ "$#" -gt 0 ] && shift
case "$target" in
	api) ./backend/deploy-api.sh "$@" ;;
	web) ./deploy-web.sh "$@" ;;
	all) ./backend/deploy-api.sh "$@" && ./deploy-web.sh ;;
	*) echo "Uso: ./deploy.sh [api|web|all]" >&2; exit 1 ;;
esac
