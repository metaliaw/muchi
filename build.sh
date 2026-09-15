#!/usr/bin/env sh
# Verifies every Component and Builds the Front Artifact.
set -eu
cd -- "$(dirname -- "$0")"

PYTHON=python3
if [ -x .venv/bin/python ]; then PYTHON=.venv/bin/python; fi

PYTHONPATH=. "$PYTHON" -m pytest -q
npm --prefix web ci
npm --prefix web run build
