# Capacity Forecasting — Huawei Cloud CCE

> Predictive capacity planning for CCE clusters: node pool exhaustion,
> pod scheduling failure prediction, storage growth, and cost forecasting.
> **Version:** 1.0.0

## Forecast Types

| Forecast Type | Method | Prediction Window | Input Data | Accuracy Target |
|---------------|--------|-------------------|------------|-----------------|
| Node pool exhaustion | Linear regression on node allocatable | 24–72h before 95% | 7d node allocatable ratio | ±10% |
| Pod scheduling failure | Logistic regression on pending pods | 1–4h before spike | 1h pending pod rate | ±20% |
| Storage (PVC) growth | Linear regression on persistent volume usage | 7d before 90% | 14d PVC usage | ±15% |
| Namespace quota breach | Trend on resource requests vs limits | 48h before 90% | 7d quota utilization | ±10% |

## Data Acquisition

### Node Pool Metrics

```bash
# Node allocatable ratio (per node pool)
hcloud ces list-metric-data \
  --namespace SYS.CCE \
  --metric_name node_allocatable_cpu \
  --dimension "cluster_id={{user.cluster_id}},node_pool={{user.node_pool}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 3600 \
  -o json

# Node count in Ready state
hcloud cce list-nodes \
  --cluster_id "{{user.cluster_id}}" \
  --output json | jq '[.items[] | select(.status == "Ready")] | length'
```

### Pod Scheduling Metrics

```bash
# Pending pods (via kubectl + CES custom metric)
kubectl get pods --all-namespaces \
  --field-selector status.phase=Pending \
  -o json | jq '[.items[]] | length'

# Custom metric via CES (push viaICAgent)
hcloud ces list-metric-data \
  --namespace SYS.CCE \
  --metric_name pod_scheduling_pending \
  --dimension "cluster_id={{user.cluster_id}},namespace=default" \
  --from "$(date -d '1 hour ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 60
```

### PVC Usage

```bash
# PVC usage per persistent volume claim
kubectl get pvc --all-namespaces -o json | jq '
  .items[] | {
    namespace: .metadata.namespace,
    name: .metadata.name,
    capacity: .status.capacity.storage,
    used: .status.used,
    percent: (.status.used / .status.capacity.storage * 100)
  }
'
```

## Forecast Algorithms

### Node Pool Exhaustion

```python
def forecast_node_pool(cluster_id, node_pool, days_ahead=7):
    """
    Linear regression on node allocatable CPU/memory to predict
    when the node pool will reach 95% utilization.
    """
    history = query_ces(
        namespace="SYS.CCE",
        metric="node_allocatable_cpu_ratio",
        dimensions={"cluster_id": cluster_id, "node_pool": node_pool},
        window="7d",
        period=3600,
    )

    values = [p["value"] for p in history]
    n = len(values)
    x = list(range(n))
    x_mean, y_mean = sum(x) / n, sum(values) / n

    slope = sum((xi - x_mean) * (yi - y_mean) for xi, yi in zip(x, values)) / \
            sum((xi - x_mean) ** 2 for xi in x)
    intercept = y_mean - slope * x_mean

    projected = slope * (n - 1 + days_ahead) + intercept
    days_to_95 = (95 - values[-1]) / slope if slope > 0 else float("inf")

    return {
        "current_utilization": values[-1],
        "slope_per_hour": slope,
        "projected_in_7d": projected,
        "days_to_95pct": days_to_95,
        "recommendation": "scale_node_pool" if days_to_95 <= 7 else "monitor",
    }
```

### Pod Scheduling Failure Prediction

```python
def forecast_scheduling_failure(cluster_id, namespace, hours_ahead=4):
    """
    Logistic regression on pending pod rate.
    Returns probability of scheduling failure within hours_ahead.
    """
    history = query_ces(
        namespace="SYS.CCE",
        metric="pod_scheduling_pending_rate",
        dimensions={"cluster_id": cluster_id, "namespace": namespace},
        window="1h",
        period=60,
    )

    values = [p["value"] for p in history]
    if not values:
        return {"error": "no data"}

    # Simple threshold-based heuristic (replace with trained model in production)
    current_rate = values[-1]
    avg_rate = sum(values) / len(values)
    trend = values[-1] - values[0]

    # Probability estimate
    prob_failure = min(1.0, max(0.0, (current_rate + trend * 2) / (avg_rate * 3)))

    return {
        "current_pending_rate": current_rate,
        "average_pending_rate": avg_rate,
        "trend": trend,
        "probability_failure_within_4h": prob_failure,
        "recommendation": "pre_scale" if prob_failure > 0.7 else "monitor",
    }
```

## Capacity Planning Tables

### Node Pool Scaling Recommendations

| Forecast Result | Action | Command |
|-----------------|--------|---------|
| days_to_95 < 3 | Immediate scale-up | `hcloud cce resize-node-pool --node_pool {{user.node_pool}} --count +3` |
| days_to_95 3–7 | Plan scale-up | Create scaling task for next maintenance window |
| PVC usage > 85% | Expand PVC | `kubectl patch pvc {{user.pvc_name}} -p '{"spec":{"resources":{"requests":{"storage":"50Gi"}}}}'` |
| Quota utilization > 90% | Request quota increase | `hcloud cce update-cluster-quota` |

### Namespace Quota Management

| Quota Type | Warning Threshold | Critical Threshold | Auto-Action |
|------------|-------------------|--------------------|--------------|
| CPU requests | 80% | 90% | Block new pods via LimitRange |
| Memory requests | 80% | 90% | Block new pods via LimitRange |
| Pod count | 85% | 95% | GC completed pods |

## Predictive Alert Rules

```bash
# Node pool exhaustion forecast
hcloud ces create-alarm-rule \
  --name "CCE-NodePool-Exhaust-Forecast" \
  --metric node_allocatable_cpu_ratio \
  --namespace SYS.CCE \
  --dimension "cluster_id={{user.cluster_id}},node_pool={{user.node_pool}}" \
  --condition "forecast_linear(72h) > 95%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"

# Pod scheduling failure预警
hcloud ces create-alarm-rule \
  --name "CCE-Pod-Schedule-Failure-Forecast" \
  --metric pod_scheduling_pending_rate \
  --namespace SYS.CCE \
  --dimension "cluster_id={{user.cluster_id}},namespace=default" \
  --condition "forecast_logistic(4h) > 0.7" \
  --alarm-level critical \
  --notifications "topic_arn:{{output.topic_arn}}"
```

## Cross-Skill Delegation

| Capacity Issue | Delegate To | Purpose |
|----------------|-------------|---------|
| Node pool scale-up | CCE skill (resize-node-pool) | Add nodes |
| PVC expansion | CCE skill (resize-pvc) | Expand storage |
| Namespace quota increase | IAM + CCE | Update quota |
| Cluster-level HPA | CCE skill (hpa) | Horizontal pod autoscaling |
| Cost from over-provisioning | Billing skill | Cost anomaly analysis |

## Knowledge Base Anchors

- CCE ↔ CES: [`references/monitoring.md`](../../huaweicloud-ces-ops/references/monitoring.md) — custom metric emission
- CCE node pool management: [`references/observability.md`](./observability.md) — LTS log linkage
- Pod scheduling: [`references/troubleshooting.md`](../../huaweicloud-cce-ops/references/troubleshooting.md)
