# WOMS GKE Baseline Deployment

本文件說明如何把 WOMS 部署到既有 GKE Standard 叢集，目標是先完成所有核心 pods 在 GKE 上 `Running` / `Ready`。第一版刻意不開 public Ingress；瀏覽器與 API 驗證先使用 `kubectl port-forward`。

## 部署預設值

- Namespace 與 Helm release：`woms`
- Application images：`docker.io/d11nn/woms-api:v0.1.41`、`docker.io/d11nn/woms-scheduler-worker:v0.1.41`、`docker.io/d11nn/woms-web:v0.1.41`
- Ingress：使用 `ingress.enabled=false` 關閉
- Gthulhu：使用 `gthulhu.enabled=false` 與 `keda.gthulhu.enabled=false` 關閉
- Autoscaling：啟用 KEDA，worker 由 Kafka lag 與 CPU HPA 控制
- Storage：GKE 預設 `standard-rwo` StorageClass
- Data services：由 chart 管理 PostgreSQL、Redis 與 Kafka

預期 workload 會包含 `postgres-0`、`redis-master-0`、`kafka-controller-0`、各兩個 replicas 的 API 與 web、至少一個由 KEDA 管理的 worker replica、Prometheus、Grafana，以及完成狀態的 Kafka topic hook job pod。

## Preflight

先確認 `kubectl` 已指向目標 GKE 叢集，且沒有舊 WOMS release 衝突：

```bash
kubectl config current-context
kubectl get nodes -o wide
kubectl get storageclass
helm list -A
helm dependency list ./deploy/helm/woms
```

此 baseline 假設 GKE 叢集已有健康 nodes、metrics-server 與 `standard-rwo` StorageClass。在 GKE 中，`standard-rwo` 使用 Compute Engine persistent disk CSI driver，並會等到第一個使用 PVC 的 pod 被排程後才 provision disk。

## 安裝 KEDA

WOMS 會為 `woms-woms-worker` 建立 `ScaledObject`，因此安裝 chart 前必須先有 KEDA：

```bash
helm repo add kedacore https://kedacore.github.io/charts
helm repo update
helm upgrade --install keda kedacore/keda \
  --namespace keda --create-namespace \
  --wait --timeout 10m

kubectl get pods -n keda
kubectl get crd scaledobjects.keda.sh
kubectl get apiservice v1beta1.external.metrics.k8s.io
```

## Render 與部署

先 render，避免 chart 錯誤直接寫入叢集：

```bash
helm template woms ./deploy/helm/woms \
  --namespace woms \
  --set ingress.enabled=false \
  --set gthulhu.enabled=false \
  --set keda.gthulhu.enabled=false \
  --set api.jwtSecret=render-only-dummy
```

部署 baseline：

```bash
helm upgrade --install woms ./deploy/helm/woms \
  --namespace woms --create-namespace \
  --dependency-update \
  --wait --timeout 20m \
  --set ingress.enabled=false \
  --set gthulhu.enabled=false \
  --set keda.gthulhu.enabled=false \
  --set imageRegistry=docker.io/d11nn \
  --set api.image.tag=v0.1.41 \
  --set worker.image.tag=v0.1.41 \
  --set web.image.tag=v0.1.41
```

## 驗證

檢查 Kubernetes resources，並執行專案驗證腳本：

```bash
kubectl get pod,deploy,statefulset,job,pvc,scaledobject,hpa,pdb -n woms
NAMESPACE=woms ./scripts/verify-k8s.sh
```

驗證 Kafka topic hook：

```bash
kubectl logs job/woms-woms-kafka-topic -n woms
kubectl exec -n woms kafka-controller-0 -- \
  kafka-topics.sh --bootstrap-server kafka.woms.svc.cluster.local:9092 \
  --describe --topic woms.schedule.jobs
```

透過 local forward 驗證瀏覽器 UI 與 API：

```bash
kubectl port-forward svc/woms-woms-web 8081:8080 -n woms
```

開啟 `http://localhost:8081`。

另一個 shell 執行：

```bash
kubectl port-forward svc/woms-woms-api 8080:8080 -n woms
BASE_URL=http://localhost:8080 ./scripts/smoke-api.sh
```

## 證明部署位置是 GKE

若需要證明 WOMS pods 是部署在 GKE，而不是本地 Kubernetes，可使用下列指令：

```bash
kubectl config current-context
kubectl get nodes -o wide
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.providerID}{"\n"}{end}'
kubectl get pods -n woms -o wide
kubectl get storageclass
kubectl get pvc -n woms -o wide
```

預期會看到的 GKE 證據：

- Active context 使用 GKE context 格式，例如 `gke_project-f3371832-9c1b-464b-a50_asia-northeast1-a_woms-gke-cluster`。
- Node 名稱以 `gke-` 開頭，例如 `gke-woms-gke-cluster-woms-default-poo-ddd9256d-0v79`。
- Node `providerID` 以 `gce://` 開頭，例如 `gce://project-f3371832-9c1b-464b-a50/asia-northeast1-a/...`。
- `kubectl get pods -n woms -o wide` 會顯示 WOMS pods 被排程到 `gke-woms-gke-cluster...` nodes。
- StorageClass `standard-rwo` 的 provisioner 是 `pd.csi.storage.gke.io`，WOMS PVCs 也透過此 StorageClass 綁定。
- `kubectl get ns` 會看到 `gke-managed-system`、`gmp-system`、`gmp-public` 等 GKE-managed namespaces。

2026-05-20 baseline 部署時抓到的證據：

```text
current-context:
gke_project-f3371832-9c1b-464b-a50_asia-northeast1-a_woms-gke-cluster

node providerIDs:
gke-woms-gke-cluster-woms-default-poo-ddd9256d-0v79  gce://project-f3371832-9c1b-464b-a50/asia-northeast1-a/gke-woms-gke-cluster-woms-default-poo-ddd9256d-0v79
gke-woms-gke-cluster-woms-default-poo-ddd9256d-4qtr  gce://project-f3371832-9c1b-464b-a50/asia-northeast1-a/gke-woms-gke-cluster-woms-default-poo-ddd9256d-4qtr
gke-woms-gke-cluster-woms-default-poo-ddd9256d-wbvv  gce://project-f3371832-9c1b-464b-a50/asia-northeast1-a/gke-woms-gke-cluster-woms-default-poo-ddd9256d-wbvv

WOMS pod placement:
kafka-controller-0      Running  gke-woms-gke-cluster-woms-default-poo-ddd9256d-4qtr
postgres-0              Running  gke-woms-gke-cluster-woms-default-poo-ddd9256d-0v79
redis-master-0          Running  gke-woms-gke-cluster-woms-default-poo-ddd9256d-wbvv
woms-woms-api-*         Running  gke-woms-gke-cluster-woms-default-poo-ddd9256d-0v79 / 4qtr
woms-woms-web-*         Running  gke-woms-gke-cluster-woms-default-poo-ddd9256d-0v79 / 4qtr
woms-woms-worker-*      Running  gke-woms-gke-cluster-woms-default-poo-ddd9256d-4qtr

WOMS PVCs:
data-kafka-controller-0      Bound  standard-rwo
data-postgres-0              Bound  standard-rwo
redis-data-redis-master-0    Bound  standard-rwo
```

## 解除部署以避免 GKE 費用

若 baseline 只用於驗證，完成後應盡快移除。刪除 namespace 很重要，因為只執行 Helm uninstall 時，StatefulSet 建立的 PVC 可能仍保留，導致 GKE Persistent Disk 繼續存在。

```bash
helm uninstall woms -n woms
kubectl delete namespace woms --wait=true --timeout=10m

helm uninstall keda -n keda
kubectl delete namespace keda --wait=true --timeout=10m
```

確認 workload、release、PVC 與 PV 都已移除：

```bash
helm list -A
kubectl get ns
kubectl get pods -n woms
kubectl get pvc -n woms
kubectl get pv
kubectl get crd scaledobjects.keda.sh
```

2026-05-20 解除部署結果：

```text
helm uninstall woms -n woms
release "woms" uninstalled

kubectl delete namespace woms --wait=true --timeout=10m
namespace "woms" deleted

helm uninstall keda -n keda
release "keda" uninstalled

kubectl delete namespace keda --wait=true --timeout=10m
namespace "keda" deleted

helm list -A
NAME  NAMESPACE  REVISION  UPDATED  STATUS  CHART  APP VERSION

kubectl get ns
# woms 與 keda 已不存在，只剩 GKE/system namespaces。

kubectl get pods -n woms
No resources found in woms namespace.

kubectl get pvc -n woms
No resources found in woms namespace.

kubectl get pv
No resources found

kubectl get crd scaledobjects.keda.sh
Error from server (NotFound): customresourcedefinitions.apiextensions.k8s.io "scaledobjects.keda.sh" not found
```

## 後續工作

- Public GKE HTTP(S) Load Balancer 或 Gateway exposure 刻意延後。
- 不建議在此 baseline 使用 community `ingress-nginx`，因為它已在 2026 年 3 月後退休；後續對外公開應使用仍維護的 GKE-native 或 Gateway API 路線。
- Gthulhu monitor-only on GKE 延後處理，因為需要明確規劃 privileged pods、eBPF、hostPID、hostPath 與合適的 Linux node pool。
- 正式化時應替換 demo credentials、設定 `api.jwtSecret`、檢查 stateful service HA，並決定 PostgreSQL/Redis/Kafka 要維持 chart-managed 或改用 managed services。

參考資料：

- KEDA deployment: https://keda.sh/docs/2.19/deploy/
- KEDA Kafka scaler: https://keda.sh/docs/2.19/scalers/apache-kafka/
- GKE persistent disk CSI driver: https://docs.cloud.google.com/kubernetes-engine/docs/how-to/persistent-volumes/gce-pd-csi-driver
- GKE Ingress concepts: https://docs.cloud.google.com/kubernetes-engine/docs/concepts/ingress
- Ingress NGINX retirement: https://www.kubernetes.dev/blog/2025/11/12/ingress-nginx-retirement/
