#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-woms}"
RELEASE="${RELEASE:-woms}"
KUBECTL="${KUBECTL:-kubectl}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-600}"
CONCURRENCY="${CONCURRENCY:-80}"
DURATION="${DURATION:-5m}"
LOAD_PATH="${LOAD_PATH:-/}"
WEB_DEPLOY="${RELEASE}-woms-web"
WEB_HPA="${RELEASE}-woms-web-hpa"
WEB_SERVICE="${RELEASE}-woms-web"

wait_replicas() {
  want="$1"
  op="$2"
  deadline=$((SECONDS + TIMEOUT_SECONDS))
  while [ "$SECONDS" -lt "$deadline" ]; do
    replicas="$("$KUBECTL" get deploy "$WEB_DEPLOY" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || true)"
    replicas="${replicas:-0}"
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

load_balancer_host="$("$KUBECTL" get svc "$WEB_SERVICE" -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
if [ -z "$load_balancer_host" ]; then
  load_balancer_host="$("$KUBECTL" get svc "$WEB_SERVICE" -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')"
fi
if [ -z "$load_balancer_host" ]; then
  echo "Service ${WEB_SERVICE} has no LoadBalancer ingress yet" >&2
  "$KUBECTL" get svc "$WEB_SERVICE" -n "$NAMESPACE" -o wide >&2
  exit 1
fi

target_url="http://${load_balancer_host}:8080${LOAD_PATH}"
echo "Target: ${target_url}"
echo "Prometheus query: nginx_http_requests_total per-pod rate via woms_web_nginx_requests_per_second_per_pod"

"$KUBECTL" get scaledobject "$WEB_DEPLOY" -n "$NAMESPACE" -o yaml
"$KUBECTL" get hpa "$WEB_HPA" -n "$NAMESPACE"

if command -v hey >/dev/null 2>&1; then
  hey -z "$DURATION" -c "$CONCURRENCY" "$target_url"
elif command -v ab >/dev/null 2>&1; then
  ab -t "${DURATION%m}" -c "$CONCURRENCY" "$target_url"
else
  echo "Install hey or ab locally, then run: hey -z ${DURATION} -c ${CONCURRENCY} ${target_url}" >&2
  exit 2
fi

wait_replicas 2 ge
"$KUBECTL" get hpa,deploy,pod -n "$NAMESPACE" -l app.kubernetes.io/component=web

echo "web HPA LoadBalancer behavior verification passed"
