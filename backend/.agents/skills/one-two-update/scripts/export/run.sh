#!/bin/sh
set -eu
cd -- "$(dirname -- "$0")"

verify_runtime() {
    command -v go >/dev/null || { echo '❌ Go Required'; return 1; }
    command -v git >/dev/null || { echo '❌ Git Required'; return 1; }
}
export_snapshot() {
    exec go run . "$@"
}

verify_runtime
export_snapshot "$@"
