#!/usr/bin/env sh
# Verifies every Component and Builds the Front Artifact.
set -eu
cd -- "$(dirname -- "$0")"

./test.sh
npm --prefix web ci
npm --prefix web run build
