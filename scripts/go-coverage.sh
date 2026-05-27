#!/usr/bin/env sh
set -eu

if [ -z "${GOCACHE:-}" ]; then
  export GOCACHE="${TMPDIR:-/tmp}/woms-go-cache"
fi

go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out
