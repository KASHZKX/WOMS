#!/usr/bin/env sh
set -eu

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is not installed or not on PATH" >&2
  exit 1
fi

: "${SONAR_SCANNER_IMAGE:=sonarsource/sonar-scanner-cli:latest}"
: "${SONAR_HOST_URL:=http://host.docker.internal:9000}"
: "${SONAR_TOKEN:?Set SONAR_TOKEN before running Docker-based local SonarScanner analysis}"

PROJECT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)

set -- run --rm \
  --user "$(id -u):$(id -g)" \
  -e SONAR_HOST_URL="$SONAR_HOST_URL" \
  -e SONAR_TOKEN="$SONAR_TOKEN" \
  -e SONAR_USER_HOME=/tmp/.sonar \
  -v "$PROJECT_DIR:/usr/src" \
  -w /usr/src

if [ -n "${SONAR_DOCKER_NETWORK:-}" ]; then
  set -- "$@" --network "$SONAR_DOCKER_NETWORK"
else
  set -- "$@" --add-host=host.docker.internal:host-gateway
fi

docker "$@" "$SONAR_SCANNER_IMAGE" \
  -Dsonar.host.url="$SONAR_HOST_URL" \
  -Dsonar.token="$SONAR_TOKEN"
