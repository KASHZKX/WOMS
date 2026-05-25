# WOMS Verification Guide

This workstation was used only for static, unit, Go, and Helm render checks. Local UI visual verification was not performed on this machine. Browser and GKE checks below must be run in an environment with a browser and a reachable Kubernetes LoadBalancer.

## 1. Local Non-UI Verification

```bash
npm run test:web
go test ./...
helm template woms ./deploy/helm/woms --namespace woms
./scripts/verify-hpa-render.sh
test -z "$(gofmt -l .)"
```

Expected:

- Frontend and Helm static tests pass.
- Go tests pass.
- Helm renders `Deployment/api`, `Deployment/worker`, `Deployment/web`, Prometheus/Grafana, the web `ScaledObject`, and PDBs.
- `ScaledObject` targets `Deployment/woms-woms-web` and creates HPA `woms-woms-web-hpa`.
- Active KEDA triggers contain only the Prometheus trigger for `woms_web_nginx_requests_per_second_per_pod`.
- Rendered manifests do not include worker Kafka lag, worker CPU, Gthulhu Prometheus triggers, `PodSchedulingMetrics`, or the Gthulhu child chart.

## 2. Manual Browser UI Verification

Run the app in a browser-capable environment, then verify:

1. Scheduler pending badge:
   - Log in as `scheduler-a` / `demo`.
   - Use a normal desktop browser width.
   - Confirm pending order cards show the `待排程` status badge on one line; `程` must not wrap.
   - Log in as `sales` / `demo` and confirm pending cards use the same badge shape and spacing.

2. Sales pending order editing:
   - Log in as `sales` / `demo`.
   - Create or locate a `待排程` order created by the same sales user.
   - Confirm the old triangle-only button is now the text button `訂單修改`.
   - Click it once: the existing due-date/quantity edit form expands.
   - Click it again: the form collapses.
   - Re-expand, change due date or quantity, submit, and confirm the order remains in the normal pending order workflow.

3. Sales draft preview calendar switch:
   - Log in as `sales` / `demo`.
   - Create a draft order with a future due date and open the schedule preview.
   - Click `待排程`: the preview calendar shows the current sales draft preview allocations.
   - Click `已排程`: the preview calendar switches to formal persisted calendar allocations.
   - Switch back to `待排程` and confirm the draft preview allocations return.
   - Confirm the final "放到待排程訂單" flow still creates a pending order.

## 3. GKE LoadBalancer Web HPA Verification

Deploy to GKE or an equivalent LoadBalancer-capable cluster:

```bash
helm upgrade --install woms ./deploy/helm/woms \
  --namespace woms --create-namespace
kubectl get svc woms-woms-web -n woms -w
```

Confirm active resources:

```bash
kubectl get scaledobject,hpa,deploy,pod,svc -n woms
kubectl describe hpa woms-woms-web-hpa -n woms
kubectl get scaledobject woms-woms-web -n woms -o yaml
```

Expected:

- `woms-woms-web` Service is `LoadBalancer`.
- `woms-woms-web` ScaledObject targets `Deployment/woms-woms-web`.
- HPA name is `woms-woms-web-hpa`.
- Trigger metric is `woms_web_nginx_requests_per_second_per_pod`.

Send multi-user traffic:

```bash
LB_IP="$(kubectl get svc woms-woms-web -n woms -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
hey -z 5m -c 80 "http://${LB_IP}:8080/"
```

Observe:

```bash
kubectl get hpa,deploy,pod -n woms -l app.kubernetes.io/component=web -w
```

Grafana:

- Open `http://<LOAD_BALANCER_IP>:8080/grafana/`.
- Open dashboard `WOMS web autoscaling`.
- Confirm `Per-pod NGINX req/s` rises during load.
- Confirm `NGINX req/s by web pod` shows traffic distributed across pods after scale-out.

Expected:

- KEDA/HPA increases web replicas above `minReplicaCount`.
- New web pods become Ready.
- Traffic spreads across multiple web pods.
- After traffic stops and cooldown passes, replicas scale down.

## 4. API, RBAC, And Calendar API Checks

```bash
JWT_SECRET=local-dev-secret go run ./cmd/api
curl -i http://localhost:8080/internal/auth/verify
```

Expected: missing token returns `401`.

Check role boundaries:

- Sales calling scheduler-only schedule job APIs returns `403`.
- Scheduler A cannot read or mutate Scheduler B line data.
- `GET /api/schedules/calendar?lineId=A&month=2026-05` returns persisted allocations for the authorized line.

## 5. Docker And Web Proxy Checks

```bash
docker build -f Dockerfile.api -t woms-api:local .
docker build -f Dockerfile.worker -t woms-scheduler-worker:local .
docker build -f Dockerfile.web -t woms-web:local .
docker compose up --build
```

Expected:

- API health: `curl http://localhost:8080/healthz`
- Web: `http://localhost:8081`
- Grafana through web proxy: `http://localhost:8081/grafana`
- Unauthenticated Grafana users see the Grafana login page.

## 6. Completion Checklist

- Local non-UI tests pass.
- Browser UI checks above are completed in a browser environment.
- GKE LoadBalancer/HPA checks above are completed in a cluster environment.
- README and both verification docs are updated in English and zh-TW.
- Generated files, secrets, local volumes, and build output remain uncommitted.
