# GKE Sales Order Creation Incident And Fix Note

Date: 2026-05-20

## Summary

After deploying WOMS to GKE, the sales user flow appeared to accept a new order successfully, but the order did not visibly appear in the order list. At the same time, GKE event logs showed API probe warnings and one KEDA certificate mount warning.

The investigation found that the deployed API and PostgreSQL write path were working. A direct deployed-API reproduction successfully created and listed an order through the same sales preview-confirm flow. The most likely user-visible issue was frontend state: after confirming a draft order, the UI could remain on a different selected production line or keep filters that hide the newly created order.

The API probe warnings were real but startup-related: the API waits for Kafka before listening on `:8080`, while Kubernetes liveness/readiness probes were already checking `/healthz` and `/readyz`. Adding a startup probe prevents Kubernetes from restarting the API during dependency warm-up.

## Reported Symptoms

- Sales order creation showed a success message.
- The order did not appear in the visible order list.
- GKE events included API readiness/liveness warnings:

```text
Readiness probe failed: Get "http://10.82.0.10:8080/readyz": dial tcp 10.82.0.10:8080: connect: connection refused
Liveness probe failed: Get "http://10.82.1.12:8080/healthz": dial tcp 10.82.1.12:8080: connect: connection refused
```

- GKE events also included a KEDA startup mount warning:

```text
MountVolume.SetUp failed for volume "certificates" : secret "kedaorg-certs" not found
```

## Evidence From Investigation

Current GKE state during investigation:

- Helm releases `keda` and `woms` were deployed.
- WOMS pods were `Running` and deployments were available.
- PostgreSQL, Redis, Kafka, API, web, worker, Prometheus, and Grafana pods were all ready.
- `ScaledObject/woms-woms-worker` was `Ready=True` and `Active=True`.
- KEDA pods were `Running`.
- Secret `keda/kedaorg-certs` existed after startup.

API logs showed startup dependency retry, then successful API startup:

```text
kafka broker not ready attempt=1 error=no configured addresses are reachable: kafka:9092: dial tcp ... connect: connection refused
kafka broker not ready attempt=2 error=no configured addresses are reachable: kafka:9092: dial tcp ... connect: connection refused
kafka broker not ready attempt=3 error=no configured addresses are reachable: kafka:9092: dial tcp ... connect: connection refused
kafka broker ready after 4 attempts
woms api listening on :8080
```

The deployed API was tested through port-forward using the same sales order path:

1. Login as `sales`.
2. Create schedule preview with `draftOrder`.
3. Confirm the preview through `/api/orders/preview-confirm`.
4. List `/api/orders`.

Result:

```text
preview=PREVIEW-1779292914209719276
created order id=ORD-4772292
customer=GKE Debug Customer
status=待排程
lineId=A
```

The order was returned by `/api/orders`, proving the GKE API and PostgreSQL write path were functional. The temporary debug order was deleted afterward:

```text
{"deletedOrderIds":["ORD-4772292"]}
```

## Root Cause Assessment

Two separate issues were identified:

- User-visible order list issue: the frontend did not focus the newly created order after preview confirmation. If the active production line or filters did not match the created order, the success message could be shown while the order was hidden from the current view.
- Startup probe issue: API liveness/readiness probes started before the API finished waiting for Kafka/PostgreSQL readiness. This caused startup warnings and temporary API restarts.

The KEDA `kedaorg-certs` event was interpreted as a startup race during KEDA installation. The secret existed later, KEDA pods were running, and the WOMS `ScaledObject` was ready.

## Fixes Applied In Code

Frontend:

- `web/app.js` now uses the order returned by `/api/orders/preview-confirm`.
- After successful preview confirmation, the UI calls `focusCreatedOrder(order)`.
- The UI switches to the created order's production line.
- Customer and priority filters are cleared.
- Status filter is focused on the created order status, normally `待排程`.
- Selected order IDs are cleared to avoid stale selection state.

Shared UI helper:

- `web/ui.js` adds `filtersForCreatedOrder(order, fallbackStatus)`.

Helm API deployment:

- `deploy/helm/woms/templates/api-deployment.yaml` adds an API `startupProbe` on `/healthz`.
- Readiness and liveness probe periods are explicit.
- The startup probe allows up to 36 failures at 5-second intervals, giving the API up to 180 seconds to finish dependency startup before liveness restarts apply.

Tests:

- `web/ui.test.mjs` covers `filtersForCreatedOrder`.
- `deploy/helm/woms/chart-static.test.mjs` asserts that the API deployment template includes the startup probe.

## Verification

Commands run after the fix:

```bash
go test ./...
npx -p node@22 node --test web/*.test.mjs deploy/helm/woms/*.test.mjs
helm template woms ./deploy/helm/woms \
  --namespace woms \
  --set ingress.enabled=false \
  --set gthulhu.enabled=false \
  --set keda.gthulhu.enabled=false \
  --set api.jwtSecret=render-only-dummy
```

Results:

- Go tests passed.
- Node 22 web and Helm static tests passed: 39 tests.
- Helm template render passed.

## Deployment Note

The live GKE deployment was still using the published `v0.1.41` images during this investigation. The frontend fix will affect users only after building and publishing a new `woms-web` image and upgrading the Helm release to that image tag. The startup probe change will affect pods only after applying the updated Helm chart through `helm upgrade`.

