#!/usr/bin/env bash
# Despliega Muchi: la API a Cloud Functions y la Web —BFF y Front— a Cloud Run
# y Firebase Hosting. La API va primero: la Web Consume sus Rutas.
#
#   ./deploy.sh        -> la API y luego la Web
#   ./deploy.sh api    -> la API: Worker, Barredor, Liberador y API
#   ./deploy.sh web    -> solo la Web
#
# `api` no toca la Infraestructura —cuentas, Firestore, cola—: cambia poco y
# pide más permisos. La primera vez, o cuando cambie, corre backend/deploy.sh.
# Las Opciones de la API (--project, --dry-run…) van después del destino.
set -euo pipefail
cd -- "$(dirname -- "$0")"

deploy_api() {
	(
		cd backend
		./build.sh
		# Worker, Barredor y API salen del mismo Código: desplegar solo la API
		# la deja hablando con un Worker viejo.
		export MUCHI_BUILD_VERIFIED=1
		./deploy-worker.sh "$@"
		./deploy-sweeper.sh "$@"
		./deploy-order-release.sh "$@"
		./deploy-api.sh "$@"
	)
}

target="${1:-all}"
[ "$#" -gt 0 ] && shift
case "$target" in
	api) deploy_api "$@" ;;
	web) ./deploy-web.sh "$@" ;;
	all) deploy_api "$@" && ./deploy-web.sh ;;
	*) echo "Uso: ./deploy.sh [api|web|all]" >&2; exit 1 ;;
esac
