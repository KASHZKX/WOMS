#!/usr/bin/env sh
set -eu

RELEASE="${RELEASE:-woms}"
NAMESPACE="${NAMESPACE:-woms}"
CHART="${CHART:-./deploy/helm/woms}"
RENDERED_MANIFEST="${RENDERED_MANIFEST:-}"
VALUES_FILE="${VALUES_FILE:-}"
values_args=""
if [ -n "$VALUES_FILE" ]; then
  values_args="-f $VALUES_FILE"
fi
cleanup_files=""

if [ -n "$RENDERED_MANIFEST" ]; then
  rendered="$RENDERED_MANIFEST"
else
  rendered="$(mktemp)"
  cleanup_files="$rendered"

  # shellcheck disable=SC2086
  helm template "$RELEASE" "$CHART" --dependency-update --namespace "$NAMESPACE" $values_args >"$rendered"
fi

grep -q "kind: ScaledObject" "$rendered"
grep -q "name: ${RELEASE}-woms-web-hpa" "$rendered"
grep -q "horizontalPodAutoscalerConfig:" "$rendered"
grep -q "scaleTargetRef:" "$rendered"
grep -q "name: ${RELEASE}-woms-web" "$rendered"
grep -q "replicas: 2" "$rendered"
grep -q "minReplicaCount: 2" "$rendered"
grep -q "maxReplicaCount: 10" "$rendered"
grep -q "type: prometheus" "$rendered"
grep -q 'metricName: "woms_web_nginx_requests_per_second_per_pod"' "$rendered"
grep -q 'nginx_http_requests_total' "$rendered"
grep -q 'woms-web-nginx' "$rendered"
grep -q 'threshold: "20"' "$rendered"
grep -q "kind: Service" "$rendered"
grep -q "type: ClusterIP" "$rendered"
grep -q "name: nginx-exporter" "$rendered"
grep -q "job_name: woms-web-nginx" "$rendered"
grep -q "scaleUp:" "$rendered"
grep -q "stabilizationWindowSeconds: 0" "$rendered"
grep -q "scaleDown:" "$rendered"
grep -q "stabilizationWindowSeconds: 120" "$rendered"
grep -q "kind: PodDisruptionBudget" "$rendered"
grep -q "name: ${RELEASE}-woms-api" "$rendered"
grep -q "name: ${RELEASE}-woms-web" "$rendered"
grep -q "minAvailable: 1" "$rendered"

if grep -q "type: kafka" "$rendered"; then
  echo "unexpected Kafka KEDA trigger in active chart" >&2
  exit 1
fi
if grep -q "metricType: Utilization" "$rendered"; then
  echo "unexpected CPU KEDA trigger in active chart" >&2
  exit 1
fi
if grep -qi "gthulhu" "$rendered"; then
  echo "unexpected Gthulhu resources in active chart render" >&2
  exit 1
fi

metrics_disabled="$(mktemp)"
cleanup_files="$cleanup_files $metrics_disabled"
trap 'rm -f $cleanup_files' EXIT
# shellcheck disable=SC2086
helm template "$RELEASE" "$CHART" --dependency-update --namespace "$NAMESPACE" $values_args --set web.metrics.enabled=false >"$metrics_disabled"
if grep -q "kind: ScaledObject" "$metrics_disabled"; then
  echo "unexpected web ScaledObject when web.metrics.enabled=false" >&2
  exit 1
fi

echo "web HPA/KEDA render verification passed"
