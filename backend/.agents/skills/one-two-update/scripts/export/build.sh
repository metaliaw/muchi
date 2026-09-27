#!/bin/sh
set -eu
cd -- "$(dirname -- "$0")"

verify_format() {
    test -z "$(gofmt -l .)" || { echo '❌ Format Failed'; return 1; }
}
verify_code() {
    go vet ./...
    go test ./...
}
build_exporter() {
    go build -buildvcs=false -trimpath -o bin/export .
    echo '✅ Exporter Built'
}

verify_format
verify_code
build_exporter
