#!/usr/bin/env sh
set -eu

if [ -z "${GOCACHE:-}" ]; then
  export GOCACHE="${TMPDIR:-/tmp}/woms-go-cache"
fi

go test -coverprofile=coverage.out -covermode=atomic ./...
coverage_report="$(go tool cover -func=coverage.out)"
printf '%s\n' "$coverage_report"

case "${WOMS_COVERAGE_POLICY:-short-term}" in
  short-term)
    min_coverage="${WOMS_GO_COVERAGE_MIN:-40.0}"
    ;;
  medium-term)
    min_coverage="${WOMS_GO_COVERAGE_MIN:-70.0}"
    ;;
  long-term)
    min_coverage="${WOMS_GO_COVERAGE_MIN:-80.0}"
    ;;
  *)
    echo "Unknown WOMS_COVERAGE_POLICY=${WOMS_COVERAGE_POLICY}. Use short-term, medium-term, or long-term." >&2
    exit 1
    ;;
esac

actual_coverage="$(printf '%s\n' "$coverage_report" | awk '/^total:/ {gsub("%", "", $3); print $3}')"
if [ -z "$actual_coverage" ]; then
  echo "Unable to determine Go total coverage from coverage.out." >&2
  exit 1
fi

awk -v actual="$actual_coverage" -v required="$min_coverage" 'BEGIN {
  if ((actual + 0) < (required + 0)) {
    printf "Go coverage %.1f%% is below required %.1f%%.\n", actual, required > "/dev/stderr"
    exit 1
  }
  printf "Go coverage %.1f%% meets required %.1f%%.\n", actual, required
}'
