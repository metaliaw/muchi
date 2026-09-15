#!/usr/bin/env sh
# Deploys every Muchi Component in Dependency Order.
set -eu
cd -- "$(dirname -- "$0")"

case "${1:-}" in
--help | -h)
	printf '%s\n' '🐱 deploy.sh [Opciones]' \
		'  Despliega: Infraestructura, Worker, Barredor y API.' \
		'  Cada deploy-COMPONENTE.sh acepta las mismas Opciones.'
	exit 0
	;;
esac

./build.sh
export MUCHI_BUILD_VERIFIED=1

./deploy-infra.sh "$@"
./deploy-worker.sh "$@"
./deploy-sweeper.sh "$@"
./deploy-api.sh "$@"
