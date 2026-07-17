# Prompts — Huawei Cloud CCE

> **Purpose:** Structured prompts for CCE (Cloud Container Engine) AIOps operations. Derived from `prompt-handbook-template.md`.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Cluster Health Check
```
Analyze CCE cluster {{cluster_id}} health status:
- Cluster status: {{cluster_status}} (Available/Abnormal)
- Node count: {{node_count}} (Ready: {{ready_nodes}}, NotReady: {{not_ready_nodes}})
- Pod count: {{pod_count}} (Running: {{running_pods}}, Pending: {{pending_pods}}, Failed: {{failed_pods}})
- CPU request: {{cpu_request}}% / {{cpu_limit}}%
- Memory request: {{mem_request}}% / {{mem_limit}}%
- Active events: {{event_count}} in past {{time_window}}
Determine cluster health and recommend actions.

CCE health indicators: API server availability, etcd latency, node condition status, addon health
```

### 1.2 Node Health Diagnosis
```
Diagnose CCE node {{node_id}} issues:
- Node status: {{node_status}} (Ready/NotReady/Unknown)
- Node condition: {{node_condition}} (MemoryPressure/DiskPressure/PIDPressure/NetworkUnavailable)
- CPU usage: {{cpu_usage}}%, Memory usage: {{mem_usage}}%
- Pod count on node: {{pod_count}} ({{system_pods}} system, {{user_pods}} user)
- Kubelet status: {{kubelet_status}}
- Recent events: {{recent_events}}
Determine root cause and remediation steps.

CCE node failure modes: kubelet down, kernel issues, container runtime failure, network plugin issues
```

### 1.3 Pod Failure Analysis
```
Analyze CCE pod {{pod_name}} in namespace {{namespace}} failure:
- Pod status: {{pod_status}} (Pending/Running/Succeeded/Failed/Unknown)
- Container states: {{container_states}}
- Restart count: {{restart_count}} (last 24h)
- Exit code: {{exit_code}}
- Last state reason: {{last_state_reason}}
- Resource limits: CPU {{cpu_limit}}, Memory {{mem_limit}}
- Node affinity: {{node_affinity}}
Identify failure cause and suggest remediation.

CCE pod failure patterns: OOMKilled (memory limit), ImagePullBackOff, CrashLoopBackOff, Evicted
```

### 1.4 Workload Performance Diagnosis
```
Diagnose CCE workload {{workload_type}} {{workload_name}} performance:
- Current replicas: {{current_replicas}} / {{desired_replicas}}
- CPU utilization: {{cpu_util}}% (request: {{cpu_request}}%)
- Memory utilization: {{mem_util}}% (request: {{mem_request}}%)
- Pod restart count: {{restart_count}}
- Service latency: P50 {{latency_p50}}ms, P99 {{latency_p99}}ms
- Request rate: {{qps}} req/s
- Error rate: {{error_rate}}%
Identify performance bottleneck and recommend actions.

CCE performance issues: CPU throttling ( CFS quota), memory pressure, pod density, network policy
```

---

## 2. Inspection Prompts

### 2.1 Cluster Capacity Inspection
```
Inspect CCE cluster {{cluster_id}} capacity:
- Node pools: {{node_pool_count}} pools
  {{#each node_pools}}
  - {{name}}: {{current}}/{{max}} nodes, {{cpu_total}} CPU / {{mem_total}}GB RAM
  {{/each}}
- Namespace resource quotas:
  {{#each namespaces}}
  - {{name}}: CPU {{cpu_quota}}, Memory {{mem_quota}}
  {{/each}}
- CPU allocation: {{cpu_allocated}}% (requested {{cpu_requested}}/{{cpu_total}})
- Memory allocation: {{mem_allocated}}% (requested {{mem_requested}}/{{mem_total}})
- Pod count: {{pod_count}}/{{pod_limit}}
Identify capacity constraints and recommend scaling.

CCE capacity dimensions: node count per AZ, VPC subnet CIDR, security group rules, load balancer quotas
```

### 2.2 Cost Optimization Scan
```
Scan CCE for cost optimization opportunities:
- Idle node pools (utilization < 20% for 14 days): {{idle_pools}}
- Underutilized nodes (CPU < 30%, Memory < 40%): {{underutilized_nodes}}
- Over-provisioned workloads: {{over_provisioned}}
- Unused storage (empty PVCs): {{unused_pvc}} GB
- Idle addons: {{idle_addons}}
- Right-size recommendations:
  {{#each recommendations}}
  - {{workload}}: current {{current}}, recommended {{recommended}} (save {{savings}}/mo)
  {{/each}}
Provide prioritized action list with estimated savings.

CCE cost optimization: node pool autoscaling, spot instances, right-sized resource requests, addon cleanup
```

### 2.3 Security Compliance Check
```
Audit CCE security compliance:
- RBAC configuration: {{rbac_status}} (compliant/non-compliant)
- Privileged pods: {{privileged_pods}} found
- Allow privilege escalation: {{privilege_escalation}} pods
- Host network access: {{host_network_pods}} pods
- Host path access: {{host_path_pods}} pods
- Container security context: {{security_context_compliance}}%
- Network policy enforcement: {{network_policy_status}}
- Secret encryption: {{secret_encryption_status}}
- CIS benchmark compliance: {{cis_compliance}}%
Report compliance status and remediation priorities.

CCE security severity: Critical = privileged container, High = host network, Medium = missing PSP
```

### 2.4 Addon Health Inspection
```
Inspect CCE addon health:
- Cluster {{cluster_id}} addons:
  {{#each addons}}
  - {{name}}: {{status}} (Expected: {{expected_version}}, Actual: {{actual_version}})
  {{/each}}
- Core DNS: {{coredns_status}} (pod count: {{coredns_pods}})
- Metrics Server: {{metrics_server_status}}
- Storage addon (everest): {{everest_status}}
- Network addon (CNI): {{cni_status}}
- Ingress addon: {{ingress_status}}
Identify addon issues affecting cluster operations.

CCE addon issues: version mismatch, resource constraints, upgrade failures, dependency conflicts
```

---

## 3. Anomaly Detection Prompts

### 3.1 Metric Anomaly Analysis
```
Analyze CCE metric anomaly:
- Cluster/Namespace: {{cluster_id}}/{{namespace}}
- Workload: {{workload_name}} ({{workload_type}})
- Metric: {{metric_name}}
- Current value: {{current_value}} (baseline: {{baseline_value}})
- Deviation: {{deviation_percent}}%
- Duration: {{duration}} minutes
- Pod-level breakdown: {{pod_breakdown}}
Determine if anomaly is significant and recommend investigation.

CCE anomaly patterns: sudden CPU spike (flash crowd), gradual memory increase (memory leak), pod evictions (resource pressure)
```

### 3.2 Pod Eviction Analysis
```
Analyze CCE pod eviction pattern:
- Cluster: {{cluster_id}}
- Namespace: {{namespace}}
- Evicted pods (past {{time_window}}): {{evicted_count}}
- Eviction reasons:
  {{#each eviction_reasons}}
  - {{reason}}: {{count}} pods
  {{/each}}
- Affected workloads: {{affected_workloads}}
- Node conditions at eviction time:
  - MemoryPressure: {{memory_pressure}}
  - DiskPressure: {{disk_pressure}}
  - PIDPressure: {{pid_pressure}}
Identify root cause and prevention strategy.

CCE eviction reasons: Evicted due to DiskPressure, MemoryPressure, PodPIDPressure, or node failure
```

### 3.3 OOMKilled Container Analysis
```
Analyze OOMKilled containers in CCE:
- Cluster: {{cluster_id}}
- Namespace: {{namespace}}
- OOMKilled count (past 24h): {{oomkill_count}}
- Top affected pods:
  {{#each top_oomkill}}
  - {{pod_name}}: {{oomkill_count}} times, limit {{memory_limit}}MB, usage {{memory_usage}}MB
  {{/each}}
- Memory request vs limit: {{memory_request}}/{{memory_limit}}
- Node memory available: {{node_available_memory}}MB
Determine if limits are too low or if memory leak exists.

CCE OOMKilled patterns: limit set too close to request, gradual memory growth, sudden memory spike
```

### 3.4 Ingress Traffic Anomaly
```
Analyze CCE ingress traffic anomaly:
- Cluster: {{cluster_id}}
- Ingress: {{ingress_name}}
- Current QPS: {{current_qps}} (baseline: {{baseline_qps}})
- Error rate: {{error_rate}}% (baseline: {{baseline_error_rate}}%)
- Latency P99: {{latency_p99}}ms (baseline: {{baseline_latency}}ms)
- Upstream response time: {{upstream_time}}ms
- Backend pod health: {{backend_health}}/{{backend_total}}
Detect if anomaly is in ingress controller, network, or backend.

CCE ingress anomaly detection: ELB backend errors, upstream timeout, certificate expiration, DNS issues
```

---

## 4. Operations Prompts

### 4.1 Node Drain Operation
```
Plan CCE node {{node_id}} drain operation:
- Current pod count: {{pod_count}}
- System pods: {{system_pods}} (do not evict)
- User pods: {{user_pods}}
- Pod disruption budget: {{pdb_status}}
- Drain timeout: {{drain_timeout}} minutes
- Eviction API response time: {{eviction_time}}ms
- Target: safely evict pods to {{target_node}} or scale down
Plan graceful drain with PDB protection.

CCE drain considerations: PDB minAvailable, terminationGracePeriodSeconds, preStop hook, service disruption
```

### 4.2 Cluster Upgrade Planning
```
Plan CCE cluster upgrade from {{from_version}} to {{to_version}}:
- Current cluster version: {{from_version}}
- Target version: {{to_version}}
- Cluster status: {{cluster_status}}
- Node pools: {{node_pool_count}}
  {{#each node_pools}}
  - {{name}}: {{node_count}} nodes, OS {{os_version}}
  {{/each}}
- Addons to upgrade:
  {{#each addons}}
  - {{name}}: {{current_version}} → {{target_version}}
  {{/each}}
- Backup status: {{backup_status}}
- Estimated upgrade time: {{upgrade_time}} minutes
- Rollback plan: {{rollback_plan}}
Validate upgrade path and identify risks.

CCE upgrade order: control plane first (no downtime), then node pools with rolling update
```

### 4.3 Workload Scaling Decision
```
Evaluate CCE workload scaling:
- Workload: {{workload_type}} {{workload_name}} in {{namespace}}
- Current replicas: {{current_replicas}}
- CPU utilization: {{cpu_util}}% (target: {{target_util}}%)
- Memory utilization: {{mem_util}}%
- Current HPA status: {{hpa_status}} (min {{hpa_min}}/max {{hpa_max}})
- Scaling metric: {{scaling_metric}} = {{metric_value}}
- Pod ready count: {{ready_pods}}/{{desired_pods}}
Recommend scale action (up/down/none) with justification.

CCE scaling triggers: CPU > 80%, Memory > 80%, custom metric from Prometheus
```

### 4.4 PVC Expansion Assessment
```
Assess CCE PVC expansion for {{pvc_name}}:
- PVC namespace: {{namespace}}
- Current size: {{current_size}}Gi
- Used size: {{used_size}}Gi ({{utilization}}%)
- StorageClass: {{storage_class}} (allowExpansion: {{allow_expansion}})
- Bound pod: {{pod_name}} (tolerations: {{tolerations}})
- Snapshot backup: {{snapshot_status}}
- Related PVCs in namespace: {{related_pvcs}}
Recommend expansion size and validate constraints.

CCE PVC expansion constraints: StorageClass must have allowExpansion=true, pod must tolerate disruption
```

---

## 5. Optimization Prompts

### 5.1 Resource Request Optimization
```
Optimize CCE workload {{workload_name}} resource requests:
- Current requests: CPU {{cpu_request}}, Memory {{mem_request}}
- Current limits: CPU {{cpu_limit}}, Memory {{mem_limit}}
- Actual usage (7-day avg): CPU {{avg_cpu}}, Memory {{avg_mem}}
- Actual usage (P95): CPU {{p95_cpu}}, Memory {{p95_mem}}
- QoS class: {{qos_class}} (Guaranteed/Burstable/BestEffort)
- Recommended requests: CPU {{rec_cpu}}, Memory {{rec_mem}}
- Recommended limits: CPU {{rec_cpu_limit}}, Memory {{rec_mem_limit}}
Validate recommendations don't risk OOM or throttling.

CCE right-sizing approach: use VPA recommendations, P95 usage + 20% headroom, match requests to typical load
```

### 5.2 HPA Configuration Tuning
```
Tune CCE HPA for {{workload_name}}:
- Current HPA: min {{hpa_min}}, max {{hpa_max}}, target CPU {{target_cpu}}%
- Current replicas: {{current_replicas}}
- Metric history (7 days):
  - CPU avg: {{avg_cpu}}%, max: {{max_cpu}}%
  - Requests avg: {{avg_qps}}, max: {{max_qps}}
- Stabilization window: {{stabilization_window}} seconds
- Recommended target: CPU {{rec_target}}% (from {{rec_target_util}}% utilization)
- Recommended min/max: {{rec_min}}/{{rec_max}}
- Estimated monthly cost change: {{cost_change}}
Validate HPA settings won't cause oscillation.

CCE HPA tuning: target 70-80% utilization, use custom metrics for latency-sensitive workloads
```

### 5.3 Node Pool Architecture Review
```
Review CCE node pool architecture:
- Cluster: {{cluster_id}}
- Current node pools:
  {{#each node_pools}}
  - {{name}}: {{node_count}} nodes, {{flavor}}, {{os}}, {{autoscaling}} (min {{min}}/max {{max}})
  {{/each}}
- Workload distribution:
  {{#each workloads}}
  - {{name}}: {{replicas}} pods, tolerates {{taints}}, prefers {{preferred}}
  {{/each}}
- Issues identified:
  {{#each issues}}
  - {{issue}} (impact: {{impact}}, effort: {{effort}})
  {{/each}}
Provide optimization recommendations.

CCE node pool optimization: use multiple node pools for workload isolation, spot for stateless, dedicated for stateful
```

### 5.4 Cost-Benefit Analysis
```
Perform CCE cost-benefit analysis:
- Cluster: {{cluster_id}}
- Monthly cost: {{monthly_cost}} CNY
  - Node cost: {{node_cost}} ({{node_count}} nodes × {{cost_per_node}})
  - Storage cost: {{storage_cost}} ({{storage_gb}}GB)
  - Network cost: {{network_cost}}
- Workload efficiency:
  - CPU efficiency: {{cpu_efficiency}}% (used/requested)
  - Memory efficiency: {{mem_efficiency}}%
- Optimization scenarios:
  {{#each scenarios}}
  - {{name}}: investment {{investment}}, saving {{saving}}/mo, ROI {{roi}} months
  {{/each}}
Provide recommendation with justification.

CCE cost scenarios: spot instances (30-70% savings), right-sizing (20-40% savings), reserved capacity (up to 60% savings)
```

---

## 6. Knowledge Base Prompts

### 6.1 Fault Pattern Matching
```
Match CCE issue to known fault patterns:
- Issue: {{issue_description}}
- Cluster: {{cluster_id}}
- Affected resources: {{affected_resources}}
- Time of occurrence: {{occurrence_time}}
- Known CCE fault patterns:
  1. API server unavailable: etcd leader election issue, OOM, disk I/O
  2. Node NotReady: kubelet OOM, container runtime failure, network plugin issue
  3. Pod Pending: resource quota exceeded, no matching nodes, PDB violation
  4. Pod Evicted: node resource pressure, preemption, delete grace period
  5. Service connectivity failure: CoreDNS issue, network policy, endpointslice desync
Identify most similar pattern and resolution path.
```

### 6.2 Resolution Guidance Retrieval
```
Retrieve resolution guidance for CCE issue:
- Issue type: {{issue_type}}
- Error message: {{error_message}}
- Resource state: {{resource_state}}
- Cluster version: {{cluster_version}}
- Recent changes: {{recent_changes}}
Return applicable runbook with success metrics.

CCE common issues: CrashLoopBackOff (check logs), ImagePullBackOff (check secret), Evicted (check resources)
```

### 6.3 Similar Incident Search
```
Search for similar past CCE incidents:
- Current incident: {{incident_description}}
- Cluster: {{cluster_id}}
- Time window: past 90 days
- Similarity criteria: same symptom, same cluster version, same workload type
Return past incidents with resolution approaches and outcomes.
```

### 6.4 Best Practice Recommendation
```
Recommend CCE best practices:
- Current configuration: {{current_config}}
- Workload type: {{workload_type}}
- Industry best practices: use HPA for stateless, PDB for stateful, priority classes for critical
- Huawei Cloud Well-Architected alignment: Reliability (multi-AZ, PDB), Security (RBAC, PSP), Cost (VPA, spot)
- Common pitfalls: no resource limits, single AZ, missing HPA, over-provisioned
Provide prioritized recommendation list.

CCE WAF alignment: use dedicated node pools for system workloads, enable pod disruption budgets
```

---

## 7. Change Management Prompts

### 7.1 Change Impact Assessment
```
Assess impact of CCE change:
- Change type: {{change_type}} (workload update / node pool resize / addon upgrade)
- Target resources: {{target_resources}}
- Change window: {{change_window}}
- Affected services: {{affected_services}}
- Service disruption risk: {{disruption_risk}}%
- Rollback capability: {{rollback_capability}}
Evaluate if change can proceed safely.

CCE change risks: workload update causes brief unavailability, node pool resize triggers pod rescheduling
```

### 7.2 Change Correlation Analysis
```
Correlate CCE changes with issues:
- Issue observed: {{issue_description}}
- Time of observation: {{issue_time}}
- Changes in past {{lookback_window}}:
  {{#each changes}}
  - {{change_type}} at {{change_time}}: {{change_details}}
  {{/each}}
- Downstream impact: {{downstream_impact}}
Determine if CCE changes caused or contributed to issue.

CCE change triggers: workload image update, HPA changes, node pool scaling, RBAC policy updates
```

### 7.3 Pre-Change Validation
```
Validate CCE change readiness:
- Change details: {{change_details}}
- Resource existence: {{resource_exists}}
- Permission validation: {{permission_ok}}
- Backup status: {{backup_status}}
- PDB status: {{pdb_status}} (allows disruption)
- Node capacity: {{node_capacity_available}}
Confirm change can proceed or list blocking issues.
```

### 7.4 Post-Change Verification
```
Verify CCE change completed successfully:
- Change ID: {{change_id}}
- Expected outcome: {{expected_outcome}}
- Workload status: {{workload_status}}
- Pod health: {{pod_health}}
- Service connectivity: {{service_connectivity}}
- Monitoring metrics: {{monitoring_metrics}}
- Alerts triggered: {{alerts_triggered}}
Confirm success or flag issues requiring attention.
```

---

## 8. Reporting Prompts

### 8.1 Daily Cluster Operations Report
```
Generate daily CCE operations report:
- Clusters: {{cluster_count}}
- Total nodes: {{total_nodes}} (Ready: {{ready_nodes}}, NotReady: {{not_ready_nodes}})
- Total pods: {{total_pods}} (Running: {{running}}, Pending: {{pending}}, Failed: {{failed}})
- Active alerts: {{alert_count}}
- Node pool utilization: avg CPU {{avg_cpu}}%, Memory {{avg_mem}}%
- Action items: {{action_items}}
```

### 8.2 Weekly Cost Optimization Report
```
Generate weekly CCE cost report:
- Total cost: {{monthly_cost}} CNY (vs {{prev_cost}} last week)
- By cluster:
  {{#each by_cluster}}
  - {{name}}: {{cost}} ({{node_count}} nodes)
  {{/each}}
- Optimization savings this week: {{savings_this_week}}
- Cumulative monthly savings: {{cumulative_savings}}
- Recommendations: {{recommendations}}
```

### 8.3 Monthly SLA Report
```
Generate monthly CCE SLA report:
- Cluster availability: {{cluster_availability}}% (target: 99.95%)
- Node availability: {{node_availability}}%
- Pod availability: {{pod_availability}}% (running/total)
- Mean time to recovery: {{mttr}} minutes
- SLA violations: {{sla_violations}}
- Root causes: {{violation_root_causes}}
- Improvement actions: {{improvement_actions}}
```

### 8.4 Executive Summary
```
Generate executive summary for CCE operations:
- Total clusters: {{cluster_count}}, nodes: {{node_count}}, pods: {{pod_count}}
- Cost performance: {{monthly_cost}} CNY, {{cost_per_node}}/node/month
- Availability: {{availability}}% SLA compliance
- Strategic recommendations: {{strategic_recommendations}}
```

---

## Appendix: CCE-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{cluster_id}}` | CCE cluster ID | `cce-12345` |
| `{{namespace}}` | Kubernetes namespace | `default`, `production` |
| `{{workload_type}}` | K8s workload type | `Deployment`, `StatefulSet`, `DaemonSet` |
| `{{workload_name}}` | Workload name | `nginx-deployment` |
| `{{node_id}}` | CCE node ID | `node-uuid` |
| `{{pvc_name}}` | PersistentVolumeClaim name | `data-pvc` |
| `{{addon_name}}` | CCE addon name | `coredns`, `metrics-server` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance (Prompt Handbook P1-3)*
