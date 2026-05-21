# WOMS GKE Baseline Deployment

This guide deploys WOMS to an existing GKE Standard cluster as a pod-complete baseline. It intentionally keeps public Ingress disabled for the first GKE milestone; use `kubectl port-forward` for browser and API validation.

## Deployment Defaults

- Namespace and Helm release: `woms`
- Application images: `docker.io/d11nn/woms-api:v0.1.41`, `docker.io/d11nn/woms-scheduler-worker:v0.1.41`, and `docker.io/d11nn/woms-web:v0.1.41`
- Ingress: disabled with `ingress.enabled=false`
- Gthulhu: disabled with `gthulhu.enabled=false` and `keda.gthulhu.enabled=false`
- Autoscaling: KEDA enabled for scheduler-worker Kafka lag and CPU HPA
- Storage: GKE default `standard-rwo` StorageClass
- Data services: chart-managed PostgreSQL, Redis, and Kafka

The expected workload surface includes `postgres-0`, `redis-master-0`, `kafka-controller-0`, API and web deployments with two replicas each, at least one worker replica managed by KEDA, Prometheus, Grafana, and the completed Kafka topic hook job.

## Preflight

Confirm that `kubectl` points at the target GKE cluster and that no old WOMS release conflicts with this baseline:

```bash
kubectl config current-context
kubectl get nodes -o wide
kubectl get storageclass
helm list -A
helm dependency list ./deploy/helm/woms
```

The baseline assumes the GKE cluster already has healthy nodes, metrics-server, and the `standard-rwo` StorageClass. On GKE, `standard-rwo` uses the Compute Engine persistent disk CSI driver and waits to provision the disk until the first consuming pod is scheduled.

## Install KEDA

WOMS creates a `ScaledObject` for `woms-woms-worker`, so KEDA must exist before installing the chart:

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

## Render And Deploy

Render first to catch chart errors before writing resources:

```bash
helm template woms ./deploy/helm/woms \
  --namespace woms \
  --set ingress.enabled=false \
  --set gthulhu.enabled=false \
  --set keda.gthulhu.enabled=false \
  --set api.jwtSecret=render-only-dummy
```

Deploy the baseline:

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

## Verification

Check the Kubernetes resources and run the project verifier:

```bash
kubectl get pod,deploy,statefulset,job,pvc,scaledobject,hpa,pdb -n woms
NAMESPACE=woms ./scripts/verify-k8s.sh
```

Verify the Kafka topic hook:

```bash
kubectl logs job/woms-woms-kafka-topic -n woms
kubectl exec -n woms kafka-controller-0 -- \
  kafka-topics.sh --bootstrap-server kafka.woms.svc.cluster.local:9092 \
  --describe --topic woms.schedule.jobs
```

Validate the browser UI and API through local forwards:

```bash
kubectl port-forward svc/woms-woms-web 8081:8080 -n woms
```

Open `http://localhost:8081`.

In another shell:

```bash
kubectl port-forward svc/woms-woms-api 8080:8080 -n woms
BASE_URL=http://localhost:8080 ./scripts/smoke-api.sh
```

## Proof That The Deployment Is On GKE

Use these commands when you need to prove that the WOMS pods are deployed on GKE rather than a local Kubernetes cluster:

```bash
kubectl config current-context
kubectl get nodes -o wide
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.providerID}{"\n"}{end}'
kubectl get pods -n woms -o wide
kubectl get storageclass
kubectl get pvc -n woms -o wide
```

Expected GKE evidence:

- The active context uses the GKE context format, for example `gke_project-f3371832-9c1b-464b-a50_asia-northeast1-a_woms-gke-cluster`.
- Node names start with `gke-`, for example `gke-woms-gke-cluster-woms-default-poo-ddd9256d-0v79`.
- Node `providerID` values start with `gce://`, for example `gce://project-f3371832-9c1b-464b-a50/asia-northeast1-a/...`.
- `kubectl get pods -n woms -o wide` shows WOMS pods scheduled onto the `gke-woms-gke-cluster...` nodes.
- StorageClass `standard-rwo` uses provisioner `pd.csi.storage.gke.io`, and WOMS PVCs bind through that StorageClass.
- GKE-managed namespaces such as `gke-managed-system`, `gmp-system`, and `gmp-public` are present in `kubectl get ns`.

Evidence captured from the baseline deployment on 2026-05-20:

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

## Teardown To Avoid GKE Cost

When the baseline is only needed for verification, remove it promptly. The namespace deletion is important because StatefulSet PVCs can otherwise keep GKE Persistent Disk resources allocated after a Helm uninstall.

```bash
helm uninstall woms -n woms
kubectl delete namespace woms --wait=true --timeout=10m

helm uninstall keda -n keda
kubectl delete namespace keda --wait=true --timeout=10m
```

Verify that workload, release, PVC, and PV resources are gone:

```bash
helm list -A
kubectl get ns
kubectl get pods -n woms
kubectl get pvc -n woms
kubectl get pv
kubectl get crd scaledobjects.keda.sh
```

Teardown result captured on 2026-05-20:

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
# woms and keda are absent; only GKE/system namespaces remain.

kubectl get pods -n woms
No resources found in woms namespace.

kubectl get pvc -n woms
No resources found in woms namespace.

kubectl get pv
No resources found

kubectl get crd scaledobjects.keda.sh
Error from server (NotFound): customresourcedefinitions.apiextensions.k8s.io "scaledobjects.keda.sh" not found
```

## Follow-Up Work

- Public GKE HTTP(S) Load Balancer or Gateway exposure is intentionally deferred.
- Community `ingress-nginx` is not recommended for this baseline because it reached retirement after March 2026; future public exposure should use a maintained GKE-native or Gateway API path.
- Gthulhu monitor-only on GKE is deferred because it needs explicit planning for privileged pods, eBPF, hostPID, hostPath, and a suitable Linux node pool.
- Production hardening should replace demo credentials, set `api.jwtSecret`, review stateful service HA, and decide whether PostgreSQL/Redis/Kafka remain chart-managed or move to managed services.

References:

- KEDA deployment: https://keda.sh/docs/2.19/deploy/
- KEDA Kafka scaler: https://keda.sh/docs/2.19/scalers/apache-kafka/
- GKE persistent disk CSI driver: https://docs.cloud.google.com/kubernetes-engine/docs/how-to/persistent-volumes/gce-pd-csi-driver
- GKE Ingress concepts: https://docs.cloud.google.com/kubernetes-engine/docs/concepts/ingress
- Ingress NGINX retirement: https://www.kubernetes.dev/blog/2025/11/12/ingress-nginx-retirement/
