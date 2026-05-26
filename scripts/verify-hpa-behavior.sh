#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-woms}"
RELEASE="${RELEASE:-woms}"
KUBECTL="${KUBECTL:-kubectl}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-600}"
CONCURRENCY="${CONCURRENCY:-80}"
DURATION="${DURATION:-5m}"
MAX_REPLICAS="${MAX_REPLICAS:-10}"
LOAD_PATH="${LOAD_PATH:-/}"
LOAD_URL="${LOAD_URL:-}"
WEB_DEPLOY="${RELEASE}-woms-web"
WEB_HPA="${RELEASE}-woms-web-hpa"
WEB_SERVICE="${RELEASE}-woms-web"
PUBLIC_INGRESS="${RELEASE}-woms-public"

current_ready_replicas() {
  replicas="$("$KUBECTL" get deploy "$WEB_DEPLOY" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || true)"
  echo "${replicas:-0}"
}

current_hpa_desired_replicas() {
  replicas="$("$KUBECTL" get hpa "$WEB_HPA" -n "$NAMESPACE" -o jsonpath='{.status.desiredReplicas}' 2>/dev/null || true)"
  echo "${replicas:-0}"
}

current_hpa_max_replicas() {
  replicas="$("$KUBECTL" get hpa "$WEB_HPA" -n "$NAMESPACE" -o jsonpath='{.spec.maxReplicas}' 2>/dev/null || true)"
  echo "${replicas:-$MAX_REPLICAS}"
}

wait_replicas() {
  want="$1"
  op="$2"
  deadline=$((SECONDS + TIMEOUT_SECONDS))
  while [ "$SECONDS" -lt "$deadline" ]; do
    replicas="$(current_ready_replicas)"
    if [ "$op" = "ge" ] && [ "$replicas" -ge "$want" ]; then
      return 0
    fi
    if [ "$op" = "le" ] && [ "$replicas" -le "$want" ]; then
      return 0
    fi
    sleep 10
  done
  echo "Timed out waiting for ${WEB_DEPLOY} replicas ${op} ${want}" >&2
  "$KUBECTL" get deploy,hpa,scaledobject,svc "$WEB_DEPLOY" "$WEB_HPA" "$WEB_SERVICE" -n "$NAMESPACE" || true
  return 1
}

wait_hpa_desired_replicas() {
  want="$1"
  deadline=$((SECONDS + TIMEOUT_SECONDS))
  while [ "$SECONDS" -lt "$deadline" ]; do
    replicas="$(current_hpa_desired_replicas)"
    if [ "$replicas" -ge "$want" ]; then
      return 0
    fi
    sleep 10
  done
  echo "Timed out waiting for ${WEB_HPA} desiredReplicas >= ${want}" >&2
  "$KUBECTL" describe hpa "$WEB_HPA" -n "$NAMESPACE" >&2 || true
  return 1
}

duration_seconds() {
  case "$1" in
    *m) echo "$((${1%m} * 60))" ;;
    *s) echo "${1%s}" ;;
    *[!0-9]*) echo "DURATION must be a number of seconds, or end with s/m: $1" >&2; exit 2 ;;
    *) echo "$1" ;;
  esac
}

target_url="$LOAD_URL"
if [ -z "$target_url" ]; then
  ingress_host="$("$KUBECTL" get ingress "$PUBLIC_INGRESS" -n "$NAMESPACE" -o jsonpath='{.spec.rules[0].host}' 2>/dev/null || true)"
  if [ -n "$ingress_host" ]; then
    tls_count="$("$KUBECTL" get ingress "$PUBLIC_INGRESS" -n "$NAMESPACE" -o jsonpath='{.spec.tls[*].hosts}' 2>/dev/null | wc -w | tr -d ' ')"
    scheme="http"
    if [ "${tls_count:-0}" -gt 0 ]; then
      scheme="https"
    fi
    target_url="${scheme}://${ingress_host}${LOAD_PATH}"
  fi
fi
if [ -z "$target_url" ]; then
  load_balancer_host="$("$KUBECTL" get svc "$WEB_SERVICE" -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || true)"
  if [ -z "$load_balancer_host" ]; then
    load_balancer_host="$("$KUBECTL" get svc "$WEB_SERVICE" -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null || true)"
  fi
  if [ -n "$load_balancer_host" ]; then
    target_url="http://${load_balancer_host}:8080${LOAD_PATH}"
  fi
fi
if [ -z "$target_url" ]; then
  echo "Set LOAD_URL, enable ${PUBLIC_INGRESS}, or expose ${WEB_SERVICE} as LoadBalancer before running web traffic verification" >&2
  "$KUBECTL" get ingress,svc "$PUBLIC_INGRESS" "$WEB_SERVICE" -n "$NAMESPACE" -o wide >&2 || true
  exit 1
fi

echo "Target: ${target_url}"
echo "Prometheus query: nginx_http_requests_total per-pod rate via woms_web_nginx_requests_per_second_per_pod"

"$KUBECTL" get scaledobject "$WEB_DEPLOY" -n "$NAMESPACE" -o yaml
"$KUBECTL" get hpa "$WEB_HPA" -n "$NAMESPACE"

initial_ready="$(current_ready_replicas)"
initial_desired="$(current_hpa_desired_replicas)"
max_replicas="$(current_hpa_max_replicas)"
baseline="$initial_ready"
if [ "$initial_desired" -gt "$baseline" ]; then
  baseline="$initial_desired"
fi
target_replicas=$((baseline + 1))
if [ "$target_replicas" -gt "$max_replicas" ]; then
  echo "Cannot prove scale-up: baseline replicas ${baseline} already reaches maxReplicas=${max_replicas}" >&2
  exit 1
fi
echo "Initial web replicas: ready=${initial_ready}, hpaDesired=${initial_desired}, hpaMax=${max_replicas}; expecting scale-up to at least ${target_replicas}"

if command -v hey >/dev/null 2>&1; then
  hey -z "$DURATION" -c "$CONCURRENCY" "$target_url"
elif command -v ab >/dev/null 2>&1; then
  ab -t "$(duration_seconds "$DURATION")" -c "$CONCURRENCY" "$target_url"
else
  echo "Install hey or ab locally, then run: hey -z ${DURATION} -c ${CONCURRENCY} ${target_url}" >&2
  exit 2
fi

wait_hpa_desired_replicas "$target_replicas"
wait_replicas "$target_replicas" ge
"$KUBECTL" get hpa,deploy,pod -n "$NAMESPACE" -l app.kubernetes.io/component=web

echo "web HPA behavior verification passed: replicas increased above baseline ${baseline}"
