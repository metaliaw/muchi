#!/usr/bin/env bash
set -euo pipefail
cd -- "$(dirname -- "$0")"
if [ ! -d node_modules ]; then npm ci; fi
exec npm run dev
