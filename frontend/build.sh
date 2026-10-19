#!/usr/bin/env sh
# Verifica el Front y Compila el Artefacto que el BFF Sirve.
set -eu
cd -- "$(dirname -- "$0")"
# Sin Dependencias instaladas no hay Runner: `npm ci` las Trae una vez.
if [ ! -d node_modules ]; then npm ci; fi
npm run test
npm run build
