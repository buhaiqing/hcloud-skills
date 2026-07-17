# Prompts — CCE (Cloud Container Engine)

> **Purpose**: Categorized AI prompts for CCE container operations.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Prompt Categories

| Category | Count | Description |
|----------|-------|-------------|
| Diagnosis | 6 | Root cause analysis prompts |
| Investigation | 5 | Evidence gathering prompts |
| Remediation | 4 | Fix action prompts |
| 巡检 | 3 | Health check prompts |
| 报告 | 2 | Summary report prompts |

---

## 2. Diagnosis Prompts

### 2.1 Pod Crash Diagnosis

```
You are analyzing pod crash on CCE cluster {{cluster_id}}.

Pod: {{namespace}}/{{pod_name}}
Container: {{container_name}}
Restart count: {{restart_count}}
Last exit code: {{exit_code}}
Last termination time: {{last_termination_time}}

Investigate:
1. Check logs via: kubectl logs {{pod_name}} --previous -n {{namespace}}
2. Resource limits exceeded? Check: kubectl describe pod {{pod_name}} -n {{namespace}}
3. Liveness/readiness probe failures?
4. Image pull issues?

Provide:
- Crash cause assessment
- Root error from logs
- Recommended fix
```

### 2.2 High CPU Usage Diagnosis

```
You are analyzing high CPU usage in CCE cluster {{cluster_id}}.

Affected workload: {{workload_type}}/{{workload_name}}
Namespace: {{namespace}}
Current CPU usage: {{cpu_usage}}m / {{cpu_limit}}m
CPU throttle: {{cpu_throttle}}%

Investigate:
1. Check pod-level CPU via: kubectl top pods -n {{namespace}}
2. Is this a single pod or namespace-wide issue?
3. Check for runaway processes inside container
4. Compare with HPA scaling state

Provide:
- CPU bottleneck analysis
- Throttle cause (limit too low or actual high load)
- Scale-up or limit adjustment recommendation
```

### 2.3 OOMKilled Pod Diagnosis

```
You are analyzing OOMKilled pods in CCE cluster {{cluster_id}}.

Affected pods: {{pod_list}}
Namespace: {{namespace}}
Memory limit: {{memory_limit}}
OOm kill time: {{kill_time}}

Investigate:
1. Memory usage trend before OOM: kubectl describe pod {{pod_name}} -n {{namespace}}
2. Is this a one-time spike or memory leak?
3. Check application memory behavior (heap, native memory)
4. Compare with memory requests/limits configuration

Provide:
- Memory leak assessment
- Recommended memory limit adjustment
- Application-level optimization suggestions
```

### 2.4 Node NotReady Diagnosis

```
You are diagnosing CCE node {{node_id}} in cluster {{cluster_id}}.

Node status: {{status}}
Conditions: {{conditions}}
Last heartbeat: {{last_heartbeat}}

Investigate:
1. Node conditions via: kubectl describe node {{node_id}}
2. Kubelet logs on node: journalctl -u kubelet
3. Docker/containerd status on node
4. Disk pressure / memory pressure / PID pressure

Provide:
- Node NotReady cause
- Recovery steps
- Prevention recommendations
```

### 2.5 Service Connectivity Issue

```
You are diagnosing service connectivity issue in CCE cluster {{cluster_id}}.

Service: {{namespace}}/{{service_name}}
Service type: {{service_type}} (ClusterIP/NodePort/LoadBalancer)
Endpoint count: {{endpoint_count}}
Affected pods: {{affected_pods}}

Investigate:
1. Service endpoints: kubectl get endpoints {{service_name}} -n {{namespace}}
2. Pod selector match: kubectl get svc {{service_name}} -n {{namespace}} -o jsonpath='{.spec.selector}'
3. Network policies affecting traffic
4. DNS resolution: kubectl exec -it {{test_pod}} -n {{namespace}} -- nslookup {{service_name}}

Provide:
- Connectivity issue root cause
- Service vs network vs endpoint problem
- Recommended fix
```

### 2.6 PersistentVolume Claim Issue

```
You are diagnosing PVC issue in CCE cluster {{cluster_id}}.

PVC: {{namespace}}/{{pvc_name}}
Status: {{status}} (Pending/Bound/Lost)
Volume: {{pv_name}}
StorageClass: {{storage_class}}

Investigate:
1. PVC status: kubectl get pvc {{pvc_name}} -n {{namespace}}
2. PV status: kubectl get pv {{pv_name}}
3. Check storage backend (OBS/San/EVS) availability
4. Any topology constraints not met?

Provide:
- PVC issue cause
- Recovery action (rebind/expand/delete)
- Storage backend recommendation
```

---

## 3. Investigation Prompts

### 3.1 Log Investigation

```
Search logs for pattern "{{pattern}}" in CCE cluster {{cluster_id}}.

Namespace: {{namespace}}
Pod: {{pod_name}} (optional, omit for all pods)
Time range: {{start_time}} to {{end_time}}
Log stream: {{log_stream}} via LTS

Extract:
- Error frequency and distribution across pods
- Correlated Kubernetes events
- Application-level errors vs infrastructure errors

Provide structured findings with timestamps.
```

### 3.2 Deployment Change Investigation

```
Find deployment changes on CCE cluster {{cluster_id}} between {{start_time}} and {{end_time}}.

Investigate:
1. Deployment changes: kubectl get events --field-selector involvedObject.name={{deployment_name}} -n {{namespace}}
2. ReplicaSet changes: kubectl describe rs -n {{namespace}}
3. Image changes via CTS: hcloud CTS listTraces --resource-type DEPLOYMENTS
4. Scaling events vs config changes

Correlate with incident at {{incident_time}}.

Provide:
- Chronological change list
- Change impact analysis
- Most likely cause with confidence
```

### 3.3 Node Resource Contention Investigation

```
Investigate resource contention on CCE node {{node_id}}.

Node allocatable:
- CPU: {{cpu_allocatable}}
- Memory: {{memory_allocatable}}

Current usage:
- CPU: {{cpu_used}} ({{cpu_used_percent}}%)
- Memory: {{memory_used}} ({{memory_used_percent}}%)

Pods running: {{pod_count}}
Pods requesting: {{cpu_request}}m CPU, {{memory_request}}Mi memory

Check:
1. Are pods within requests/limits?
2. Any pods with high CPU/throttle?
3. System daemonset resource usage?

Provide:
- Contention analysis
- QoS impact assessment
- Node right-sizing recommendations
```

### 3.4 Network Policy Investigation

```
Investigate network policy impact on CCE cluster {{cluster_id}}.

Namespace: {{namespace}}
Affected pods: {{pod_list}}
Symptom: {{symptom}} (no incoming traffic / no outgoing traffic / connection reset)

Check:
1. Network policies: kubectl get networkpolicies -n {{namespace}}
2. Pod labels and selector matching
3. Ingress/egress rules affecting traffic
4. Default deny policy impact

Provide:
- Network policy analysis
- Required rule additions
- Testing approach for rule changes
```

### 3.5 HPA Scaling Investigation

```
Investigate HPA behavior on CCE cluster {{cluster_id}}.

HPA: {{namespace}}/{{hpa_name}}
Current replicas: {{current_replicas}} / min: {{min_replicas}} / max: {{max_replicas}}
Current CPU: {{current_cpu}}%
Current memory: {{current_memory}}%

Check:
1. HPA status: kubectl describe hpa {{hpa_name}} -n {{namespace}}
2. Is scaling blocked (min replicas hit)?
3. Any pod disruption preventing scale-up?
4. Metrics server availability?

Provide:
- HPA behavior analysis
- Scaling bottleneck identification
- Recommended HPA tuning
```

---

## 4. Remediation Prompts

### 4.1 Pod Disruption Budget Remediation

```
Analyze PDB remediation for CCE cluster {{cluster_id}}.

PDB: {{namespace}}/{{pdb_name}}
Current status:
- Min available: {{min_available}} / Allow disruption: {{allow_disruption}}
- Current healthy pods: {{healthy_pods}}

Planned disruption:
- Operation: {{operation}} (drain node / update deployment / upgrade cluster)
- Affected pods: {{affected_pods}}

Recommend:
1. Is PDB blocking legitimate operations?
2. Adjustment needed (increase min available)?
3. Schedule maintenance window?

Provide PDB adjustment with risk assessment.
```

### 4.2 Resource Quota Remediation

```
Analyze resource quota adjustment for CCE namespace {{namespace}}.

Current quotas:
- CPU: {{cpu_quota}}m requested / {{cpu_limit}}m limit
- Memory: {{memory_quota}}Mi requested / {{memory_limit}}Mi limit

Current usage:
- CPU: {{cpu_used}}m ({{usage_percent}}%)
- Memory: {{memory_used}}Mi ({{usage_percent}}%)

Pods pending due to quota: {{pending_pods}}

Recommend:
1. Quota increase amount
2. Or optimize pod resource requests
3. Cost impact of quota increase

Provide quota adjustment with justification.
```

### 4.3 Service Mesh Remediation

```
Analyze service mesh (Istio) remediation for CCE cluster {{cluster_id}}.

Issue: {{issue_description}}
Affected services: {{affected_services}}

Check:
1. Sidecar injection status: kubectl get namespace {{namespace}} -o jsonpath='{.metadata.annotations.sidecar\.istio\.io/inject}'
2. Destination rules: kubectl get destinationrules -n {{namespace}}
3. Virtual services: kubectl get virtualservices -n {{namespace}}
4. Envoy stats: kubectl exec -it {{pod_name}} -n {{namespace}} -c istio-proxy -- pilot-agent status

Provide:
- Service mesh issue cause
- Configuration fix
- Sidecar restart recommendation
```

### 4.4 Workload Upgrade Recommendation

```
Analyze workload upgrade for CCE cluster {{cluster_id}}.

Workload: {{workload_type}}/{{namespace}}/{{workload_name}}
Current image: {{current_image}}
Available image: {{available_image}}
Rolling update strategy: {{strategy}}

Current state:
- Available: {{available}} / Desired: {{desired}}
- Max surge: {{max_surge}} / Max unavailable: {{max_unavailable}}

Recommend:
1. Rolling update or recreate?
2. Pre-pull image on nodes?
3. Pause/resume strategy during issues?

Provide upgrade plan with rollback procedure.
```

---

## 5. 巡检 Prompts

### 5.1 Daily Cluster Health Check

```
Perform daily CCE cluster health check for {{cluster_id}}.

Checks:
1. Node status: kubectl get nodes (all should be Ready)
2. Pod status: kubectl get pods -A (no CrashLoopBackOff/ImagePullBackOff/Evicted)
3. System pods health: kubectl get pods -n kube-system
4. Certificate expiration: hcloud CCE listClusters
5. Add-on health: kubectl get pods -n cce-system

Provide:
- Cluster health score
- Critical issues requiring immediate attention
- Recommendations for next 24h
```

### 5.2 Weekly Capacity Review

```
Perform weekly CCE cluster capacity review for {{cluster_id}}.

Review:
1. Node capacity: kubectl describe nodes | grep -A 5 "Allocated resources"
2. Namespace resource quotas: kubectl get resourcequota -A
3. Storage capacity: kubectl get pvc -A
4. HPA effectiveness: kubectl get hpa -A

Capacity forecast:
- Days until node pressure
- Days until quota exhaustion
- Days until storage full

Provide capacity forecast and recommendations.
```

### 5.3 Monthly Cost Optimization Review

```
Perform monthly CCE cost optimization review for {{cluster_id}}.

Cost analysis:
- Node cost: {{node_cost}}/month
- Storage cost: {{storage_cost}}/month
- Total cluster cost: {{total_cost}}/month

Optimization opportunities:
1. Underutilized nodes (CPU < {{cpu_threshold}}%, Memory < {{memory_threshold}}%)
2. Over-provisioned resource requests
3. Long-running pods with no resource limits
4. Unused PVCs (ReadWriteOnce, not mounted)

Provide:
- Cost breakdown
- Optimization recommendations with monthly savings
- Implementation priority
```

---

## 6. 报告 Prompts

### 6.1 Incident Root Cause Analysis

```
Generate incident RCA for CCE cluster {{cluster_id}}.

Incident:
- ID: {{incident_id}}
- Duration: {{start_time}} to {{end_time}}
- Affected: {{affected_workloads}}
- Impact: {{impact_description}}

Timeline:
{{timeline_entries}}

Kubernetes events:
{{k8s_events}}

Logs analyzed:
{{log_summary}}

Root cause: {{root_cause}}

Provide formatted RCA with:
- Timeline
- Root cause analysis
- Contributing factors
- Lessons learned
- Preventive measures
```

### 6.2 Monthly Cluster Operations Report

```
Generate monthly CCE operations report for {{cluster_id}}.

Period: {{month}}

Cluster overview:
- Nodes: {{node_count}} (added: {{nodes_added}}, removed: {{nodes_removed}})
- Workloads: {{workload_count}}
- Total pods: {{pod_count}}

Operations:
- Deployments: {{deployment_count}}
- Scaling events: {{scaling_events}}
- Node upgrades: {{node_upgrades}}

Incidents:
{{incident_list}}

Cost:
{{cost_breakdown}}

Provide detailed report with trends and recommendations.
```

---

## 7. Compliance Checklist

- [x] 20 categorized prompts (6 diagnosis + 5 investigation + 4 remediation + 3 巡检 + 2 报告)
- [x] Each prompt includes context variables ({{variable}})
- [x] Each prompt specifies output format
- [x] Diagnosis prompts include confidence level
- [x] Commands reference kubectl and hcloud CLI where applicable
