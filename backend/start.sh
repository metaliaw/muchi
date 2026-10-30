#!/usr/bin/env sh
# Runs Muchi with Docker Compose. Pass serve or work; serve is the Default.
set -eu

cd -- "$(dirname -- "$0")"
COMMAND="${1:-serve}"

verify_command() {
	if [ "$COMMAND" != "serve" ] && [ "$COMMAND" != "work" ]; then
		echo "❌ Unknown Command: $COMMAND"
		echo "Usage: ./run.sh [serve|work]"
		return 1
	fi
}

select_service() {
	case "$COMMAND" in
		serve) SERVICE="api" ;;
		work) SERVICE="worker" ;;
	esac
}

start_service() {
	echo "✅ Starting Muchi $COMMAND"
	docker compose up --build -d "$SERVICE"
	exec docker compose logs --follow --tail 50 "$SERVICE" firestore
}

verify_command
select_service
start_service
