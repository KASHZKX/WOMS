#!/usr/bin/env sh
set -eu

if [ "${WOMS_INTEGRATION_TESTS:-}" != "1" ]; then
  echo "Skipping manual integration tests: set WOMS_INTEGRATION_TESTS=1 to run them."
  echo "This command never starts Docker Compose; provide DATABASE_URL and/or REDIS_ADDR for existing services."
  exit 0
fi

ran=0

if [ -n "${DATABASE_URL:-}" ]; then
  ran=1
  echo "Running PostgreSQL integration test packages with developer-provided DATABASE_URL."
  go test ./internal/api ./cmd/scheduler-worker
else
  echo "Skipping PostgreSQL integration test packages: DATABASE_URL is not set."
fi

if [ -n "${REDIS_ADDR:-}" ]; then
  ran=1
  echo "Running Redis integration test packages with developer-provided REDIS_ADDR."
  go test ./internal/api ./internal/lock
else
  echo "Skipping Redis integration test packages: REDIS_ADDR is not set."
fi

if [ "$ran" -eq 0 ]; then
  echo "No integration services configured. Set DATABASE_URL and/or REDIS_ADDR to run manual integration tests."
fi
