# CCE Monitoring — Huawei Cloud Cloud Container Engine

## Cluster Metrics

| Metric | CES Namespace | Type | Unit | Recommended Alarm |
|--------|--------------|------|------|------------------|
| Cluster CPU utilization | SYS.CCE | Gauge | % | > 80% for 5min |
| Cluster Memory utilization | SYS.CCE | Gauge | % | > 85% for 5min |
| Node count | SYS.CCE | Gauge | count | < min_node_count |
| Pod count | SYS.CCE | Gauge | count | > 90% of node capacity |
| API server latency | SYS.CCE | Gauge | ms | P99 > 1000ms |
| API server error rate | SYS.CCE | Gauge | % | > 1% for 5min |

## Node Metrics

| Metric | CES Namespace | Type | Unit | Recommended Alarm |
|--------|--------------|------|------|------------------|
| CPU utilization | SYS.ECS | Gauge | % | > 80% for 5min |
| Memory utilization | SYS.ECS | Gauge | % | > 85% for 5min |
| Disk usage (root) | SYS.ECS | Gauge | % | > 90% for 5min |
| Disk usage (data) | SYS.ECS | Gauge | % | > 90% for 5min |
| Network inbound | SYS.ECS | Rate | B/s | Sudden drops (connectivity) |
| Network outbound | SYS.ECS | Rate | B/s | Sudden drops (connectivity) |
| OOM kills | AGT.CCE | Counter | count | Any increase |
| kubelet status | AGT.CCE | Gauge | 0/1 | = 0 (Node not reporting) |
| Pod restart count | AGT.CCE | Counter | count | > 5 in 10min |
| Container CPU | AGT.CCE | Gauge | % | Per-container monitoring |
| Container Memory | AGT.CCE | Gauge | MB | Per-container monitoring |

## Node Pool Metrics

| Metric | Source | Type | Unit | Description |
|--------|--------|------|------|-------------|
| Current node count | CCE API | Gauge | count | Actual nodes in pool |
| Desired node count | CCE API | Gauge | count | Target nodes |
| Autoscaling events | CCE API | Counter | count | Scale-up/scale-down count |
| Pending pods | Kubernetes | Gauge | count | Pods that can't schedule to pool |

## Dashboard Templates

### Cluster Health Dashboard

- Cluster phase status (Available/Error/Upgrading)
- Node count and node health ratio (Active/NotReady)
- API server latency and error rate
- Pod scheduling success rate
- Resource utilization (CPU/memory/disk) across all nodes

### Node Pool Dashboard

- Node pool status and node count
- Autoscaling history (timeline of scale events)
- Node resource utilization per pool
- Pending pod count per pool

### Cost Dashboard

- Node-hour consumption daily trend
- Cost per node pool
- Idle node detection (CPU < 10% for 7+ days)
- Spot vs pay-per-use vs subscription cost breakdown

## Alarm Rule Patterns

### Node NotReady

```
Namespace:    SYS.CCE
Metric:       cluster_node_status
Dimension:    cluster_id=<cluster_id>, instance_id=<node_id>
Condition:    value = 0 (NotReady)
Period:       60s
Evaluation:   3 consecutive periods
Level:        Critical
Action:       Trigger runbook: check node, restart kubelet, replace node
```

### High CPU Usage

```
Namespace:    SYS.ECS
Metric:       cpu_util
Dimension:    instance_id=<node_id>
Condition:    average > 80
Period:       300s
Evaluation:   3 consecutive periods
Level:        Warning
Action:       Trigger scale-up, check workloads
```

### Disk Space Low

```
Namespace:    SYS.ECS
Metric:       disk_util
Dimension:    instance_id=<node_id>, disk_type=data
Condition:    average > 90
Period:       300s
Evaluation:   3 consecutive periods
Level:        Critical
Action:       Clean logs/images, expand volume, replace node
```

### OOM Kill Spike

```
Namespace:    AGT.CCE
Metric:       oom_killed
Dimension:    cluster_id=<cluster_id>
Condition:    count increase > 0
Period:       300s
Evaluation:   1 consecutive period
Level:        Warning
Action:       Check pod memory limits, right-size workloads
```

## Monitoring Agent Integration

| Agent | Namespace | Installation | Purpose |
|-------|-----------|-------------|---------|
| Cloud Eye Agent | SYS.* | Auto-installed on nodes | OS-level metrics (CPU, memory, disk, network) |
| CCE Metrics Agent | AGT.* | Installed as CCE addon | Container-level metrics, OOM, pod stats |
| AOM Agent | AGT.* | Installed as CCE addon | Application performance monitoring |

## Cost Metrics

| Metric | Source | Description |
|--------|--------|-------------|
| Node-hour billing | ECS billing | Hourly cost per node flavor |
| Volume storage cost | EVS billing | Cost of system and data volumes |
| LoadBalancer cost | ELB billing | Cost of LB services in cluster |
| EIP cost | EIP billing | Cost of public IP addresses |
| Addon resource cost | CCE billing | Compute cost of addon pods (coredns, metrics-server, etc.) |

## Security Metrics

| Metric | Source | Description |
|--------|--------|-------------|
| Failed auth attempts | ECS/SSH | SSH login failures on nodes |
| Security group violations | VPC | Denied network connections |
| Pod privilege escalation | Kubernetes | Pods running as root |
| Image pull errors | Kubernetes | Unauthenticated registry pulls |
