# GKE Sales Order Creation Incident And Fix Note

日期：2026-05-20

## 摘要

WOMS 部署到 GKE 後，Sales 使用者新增訂單時畫面顯示成功，但訂單沒有出現在目前可見的訂單列表中。同時，GKE event logs 出現 API probe warnings，以及一筆 KEDA certificate mount warning。

調查結果顯示，已部署的 API 與 PostgreSQL 寫入路徑本身是正常的。透過 deployed API 直接重現相同的 sales preview-confirm 流程，可以成功建立訂單並從 `/api/orders` 查回。最可能造成使用者感覺「成功但沒出現」的原因是前端狀態：確認草稿訂單後，畫面可能仍停在不同產線，或保留會隱藏新訂單的篩選條件。

API probe warnings 則是真實但屬於啟動期問題：API 會先等待 Kafka，再開始 listen `:8080`；但 Kubernetes liveness/readiness probes 已經開始檢查 `/healthz` 與 `/readyz`。加入 startup probe 後，可以避免 Kubernetes 在 dependency warm-up 階段重啟 API。

## 回報現象

- Sales 新增訂單後顯示成功。
- 訂單未出現在可見訂單列表。
- GKE events 出現 API readiness/liveness warnings：

```text
Readiness probe failed: Get "http://10.82.0.10:8080/readyz": dial tcp 10.82.0.10:8080: connect: connection refused
Liveness probe failed: Get "http://10.82.1.12:8080/healthz": dial tcp 10.82.1.12:8080: connect: connection refused
```

- GKE events 也出現 KEDA startup mount warning：

```text
MountVolume.SetUp failed for volume "certificates" : secret "kedaorg-certs" not found
```

## 調查證據

調查時的 GKE 狀態：

- Helm releases `keda` 與 `woms` 都是 deployed。
- WOMS pods 都是 `Running`，deployments 可用。
- PostgreSQL、Redis、Kafka、API、web、worker、Prometheus、Grafana pods 都已 ready。
- `ScaledObject/woms-woms-worker` 為 `Ready=True` 與 `Active=True`。
- KEDA pods 為 `Running`。
- 啟動後 `keda/kedaorg-certs` secret 已存在。

API logs 顯示啟動時曾等待 Kafka，之後成功啟動 API：

```text
kafka broker not ready attempt=1 error=no configured addresses are reachable: kafka:9092: dial tcp ... connect: connection refused
kafka broker not ready attempt=2 error=no configured addresses are reachable: kafka:9092: dial tcp ... connect: connection refused
kafka broker not ready attempt=3 error=no configured addresses are reachable: kafka:9092: dial tcp ... connect: connection refused
kafka broker ready after 4 attempts
woms api listening on :8080
```

透過 port-forward 對 deployed API 測試相同 sales 訂單流程：

1. 使用 `sales` 登入。
2. 使用 `draftOrder` 建立 schedule preview。
3. 透過 `/api/orders/preview-confirm` 確認 preview。
4. 查詢 `/api/orders`。

結果：

```text
preview=PREVIEW-1779292914209719276
created order id=ORD-4772292
customer=GKE Debug Customer
status=待排程
lineId=A
```

該訂單可從 `/api/orders` 查回，證明 GKE API 與 PostgreSQL 寫入路徑正常。臨時 debug 訂單已在測試後刪除：

```text
{"deletedOrderIds":["ORD-4772292"]}
```

## 根因判定

這次分成兩個獨立問題：

- 使用者可見訂單列表問題：前端在 preview confirm 後沒有聚焦到新建立訂單。如果目前 active production line 或 filters 不符合該訂單，即使 API 成功寫入，使用者仍會看到目前列表沒有新資料。
- 啟動期 probe 問題：API liveness/readiness probes 在 API 完成 Kafka/PostgreSQL readiness 等待前就開始檢查，造成啟動期 warnings 與短暫 API restart。

KEDA `kedaorg-certs` event 判定為 KEDA 安裝啟動期 race。後續 secret 已存在、KEDA pods Running，且 WOMS `ScaledObject` Ready。

## 已套用的程式修正

Frontend：

- `web/app.js` 現在會使用 `/api/orders/preview-confirm` 回傳的新訂單。
- preview confirm 成功後，UI 會呼叫 `focusCreatedOrder(order)`。
- UI 會切到新訂單的 production line。
- 清除 customer 與 priority filters。
- status filter 聚焦到新訂單狀態，通常是 `待排程`。
- 清除 selected order IDs，避免殘留舊選取狀態。

Shared UI helper：

- `web/ui.js` 新增 `filtersForCreatedOrder(order, fallbackStatus)`。

Helm API deployment：

- `deploy/helm/woms/templates/api-deployment.yaml` 新增 API `/healthz` startup probe。
- readiness 與 liveness probe period 明確化。
- startup probe 允許最多 36 次失敗、每 5 秒一次，也就是 API 最多有 180 秒可完成 dependency startup，之後才開始套用 liveness restart 行為。

Tests：

- `web/ui.test.mjs` 覆蓋 `filtersForCreatedOrder`。
- `deploy/helm/woms/chart-static.test.mjs` 確認 API deployment template 包含 startup probe。

## 驗證

修正後執行的指令：

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

結果：

- Go tests passed。
- Node 22 web 與 Helm static tests passed，共 39 tests。
- Helm template render passed。

## 部署注意事項

調查時 live GKE deployment 仍使用已發布的 `v0.1.41` images。Frontend 修正需要 build 並發布新的 `woms-web` image，再透過 Helm upgrade 指向新 image tag，使用者才會看到修正後行為。Startup probe 修正也需要透過 `helm upgrade` 套用更新後的 chart，才會影響 GKE pods。

