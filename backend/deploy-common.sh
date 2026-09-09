#!/usr/bin/env sh
# Shares Deployment Configuration without Owning a Component.

prepare_deploy() {
	PROJECT=""
	TOKEN_FILE=""
	DRY_RUN=false
	while [ "$#" -gt 0 ]; do
		case "$1" in
		--project | --region | --token-secret | --token-file | --scry)
			[ "$#" -ge 2 ] || fail_deploy "Falta el Valor de $1"
			case "$1" in
			--project) PROJECT=$2 ;;
			--region) REGION=$2 ;;
			--token-secret) MUCHI_API_TOKEN=$2 ;;
			--token-file) TOKEN_FILE=$2 ;;
			--scry) MUCHI_SCRY_ENABLED=$2 ;;
			esac
			shift 2
			;;
		--dry-run)
			DRY_RUN=true
			shift
			;;
		--help | -h)
			show_deploy_help
			exit 0
			;;
		*) fail_deploy "Argumento Desconocido: $1" ;;
		esac
	done
	command -v gcloud >/dev/null 2>&1 || fail_deploy 'Instala Google Cloud CLI.'
	[ -n "$PROJECT" ] || PROJECT=$(gcloud config get-value project 2>/dev/null)
	case "$PROJECT" in '' | '(unset)' | *[!a-z0-9-]*) fail_deploy 'Selecciona un Proyecto con --project.' ;; esac
	case "$REGION" in '' | *[!a-z0-9-]*) fail_deploy 'Región Inválida.' ;; esac
	case "$MUCHI_SCRY_ENABLED" in true | false) ;; *) fail_deploy 'Usa --scry true o --scry false.' ;; esac
	case "$MUCHI_API_TOKEN" in *:*) ;; *) fail_deploy 'Usa --token-secret NOMBRE:VERSION.' ;; esac
	SECRET_NAME=${MUCHI_API_TOKEN%:*}
	case "$SECRET_NAME" in '' | *[!a-zA-Z0-9_-]*) fail_deploy 'Nombre de Secreto Inválido.' ;; esac
	[ -z "$TOKEN_FILE" ] || [ -s "$TOKEN_FILE" ] || fail_deploy 'Archivo de Token Ausente o Vacío.'
	MUCHI_API_TOKEN="$SECRET_NAME:latest"
	API_EMAIL="$API_ACCOUNT@$PROJECT.iam.gserviceaccount.com"
	WORKER_EMAIL="$WORKER_ACCOUNT@$PROJECT.iam.gserviceaccount.com"
	TASK_EMAIL="$TASK_ACCOUNT@$PROJECT.iam.gserviceaccount.com"
	BUILD_EMAIL="$BUILD_ACCOUNT@$PROJECT.iam.gserviceaccount.com"
	if ! "$DRY_RUN"; then
		[ -n "$(gcloud auth list --filter=status:ACTIVE --format='value(account)')" ] || fail_deploy 'Ejecuta gcloud auth login.'
	fi
	printf '🐱 %s\nProyecto: %s\nRegión: %s\n' "$DEPLOY_COMPONENT" "$PROJECT" "$REGION"
	if "$DRY_RUN"; then printf '%s\n' '⚠️ Vista Previa: GCP sin Cambios'; fi
}

cloud() {
	if "$DRY_RUN"; then
		printf 'gcloud' >&2
		printf ' "%s"' "$@" "--project=$PROJECT" --quiet >&2
		printf '\n' >&2
	else
		gcloud "$@" "--project=$PROJECT" --quiet
	fi
}

read_worker() {
	if "$DRY_RUN"; then
		WORKER_URL="https://$WORKER_NAME-PREVIEW.run.app"
		WORKER_SERVICE=$WORKER_NAME
		return
	fi
	WORKER_URL=$(cloud functions describe "$WORKER_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.uri)')
	WORKER_SERVICE=$(cloud functions describe "$WORKER_NAME" --gen2 --region="$REGION" --format='value(serviceConfig.service)')
	case "$WORKER_URL" in https://*) ;; *) fail_deploy 'URL del Worker Ausente.' ;; esac
}

grant_project() {
	cloud projects add-iam-policy-binding "$PROJECT" --member="serviceAccount:$1" --role="$2" --condition=None --format=none
}

fail_deploy() {
	printf '❌ %s\n' "$1" >&2
	exit 1
}

show_deploy_help() {
	printf '%s\n' "🐱 $0 [--project ID] [--region REGION]" \
		'  [--token-secret NOMBRE:VERSION] [--token-file ARCHIVO]' \
		'  [--scry true|false] [--dry-run]'
}
