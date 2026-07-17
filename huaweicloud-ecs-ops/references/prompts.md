# Prompts — Huawei Cloud ECS

> **Purpose:** Structured prompts for ECS AIOps operations. Derived from `prompt-handbook-template.md`.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Instance Health Check
```
Analyze ECS instance {{resource_id}} health status:
- Current metric values: CPU {{cpu_usage}}%, Memory {{mem_usage}}%, Disk {{disk_usage}}%
- Recent alert history: {{alert_count}} alerts in past {{time_window}}
- Related events: {{event_summary}}
Determine if instance is healthy and recommend actions.

Applicable CES metrics: SYS.ECS.cpu_usage, SYS.ECS.mem_usedPercent, SYS.ECS.diskUsage_percent
```

### 1.2 Root Cause Analysis
```
Given ECS instance {{resource_id}} shows:
- Symptom: {{symptom_description}}
- First observed: {{first_observed_time}}
- Metric anomaly: CPU {{cpu_anomaly}}%, Memory {{mem_anomaly}}%, Network {{net_anomaly}}%
- Correlated CTS events: {{cts_events}}
Perform root cause analysis and provide ranked hypothesis list with confidence scores.

Common ECS failure modes: agent disconnection, OS unresponsiveness, network partition, resource exhaustion
```

### 1.3 Performance Degradation Diagnosis
```
ECS instance {{resource_id}} performance degraded:
- Latency increased from {{baseline_latency}}ms to {{current_latency}}ms
- Error rate changed from {{baseline_error}}% to {{current_error}}%
- Traffic {{traffic_change_direction}} by {{traffic_change_percent}}%
- Resource utilization: CPU {{cpu_util}}%, Memory {{mem_util}}%, Disk IO {{io_util}}%
Diagnose root cause and suggest remediation steps.

Possible causes: CPU throttling, memory leak, disk I/O bottleneck, network congestion, application bug
```

### 1.4 Availability Incident Response
```
ECS instance {{resource_id}} availability issue detected:
- Availability: {{current_availability}}% (SLO target: 99.95%)
- Impact duration: {{duration_minutes}} minutes
- Affected scope: {{affected_scope}}
- Instance state: {{instance_state}}
- AZ: {{az}}
Generate incident response playbook with next steps.

ECS-specific: Check system status, agent connectivity, underlying hypervisor health, and scheduling capacity
```

---

## 2. Inspection Prompts

### 2.1 Routine Health Inspection
```
Perform routine inspection on ECS instances:
- List all ECS instances in {{scope}}
- Check for instances with CPU > 80% OR Memory > 85% sustained for 30 minutes
- Identify instances with no monitoring data for > 2 hours
- Flag any instances with active CES alerts
- Check for instances in error state or pending status
Report findings in structured format.

Scope options: region, AZ, VPC, tag-based resource group
```

### 2.2 Cost Optimization Scan
```
Scan ECS for cost optimization opportunities:
- Identify idle instances (CPU < 5%, Network < 1MB/s for 14 days)
- Find oversized instances (utilization < 30% over 30 days)
- Check for instances without proper sizing (night/weekend waste)
- Verify reserved capacity coverage vs on-demand usage
- Check for overdue EIP associations (unused public IPs)
Provide prioritized action list with estimated monthly savings.

Right-sizing targets: avg CPU < 40% → downsize; avg CPU > 80% → upsize
```

### 2.3 Security Compliance Check
```
Audit ECS security compliance:
- Verify security groups follow least-privilege (no 0.0.0.0/0 unless explicitly required)
- Check for exposed management ports (22, 3389 without IP restriction)
- Validate VPC mode (no instances in default VPC with public access)
- Confirm EVS encryption at rest is enabled
- Check for instances missing security patches (OS level)
- Verify IAM agent is running on all instances
Report compliance status and remediation priorities.

Severity: Critical = exposed management port, High = default VPC with EIP, Medium = missing encryption
```

### 2.4 Capacity Planning Review
```
Review ECS capacity for {{scope}}:
- Current utilization: CPU {{cpu_util}}%, Memory {{mem_util}}%, EVS {{storage_util}}%
- Instance type distribution: {{instance_type_distribution}}
- Growth trend: {{growth_rate}}% weekly average
- Projected capacity exhaustion: {{exhaustion_date}}
- Available AZ capacity: {{az_capacity}}
Provide scaling recommendations with timeline.

Capacity dimensions: vCPU quotas, EVS volume limits, EIP limits, security group rule limits
```

---

## 3. Anomaly Detection Prompts

### 3.1 Metric Anomaly Analysis
```
Analyze metric anomaly for ECS instance {{resource_id}}:
- Metric: {{metric_name}} (CES namespace: SYS.ECS)
- Current value: {{current_value}} (baseline: {{baseline_value}})
- Deviation: {{deviation_percent}}% from baseline
- Duration: {{anomaly_duration}} minutes
- Similar historical patterns: {{historical_patterns}}
Determine if this is a true anomaly or false positive, and assess severity.

ECS anomaly patterns: CPU spike (往往是瞬时负载), memory gradual increase (可能是 leak), disk I/O sudden drop (可能是路径变化)
```

### 3.2 Alarm Storm Triage
```
Triage alarm storm affecting ECS:
- Total alarms: {{alarm_count}} in past 15 minutes
- Alarms by type: CPU {{cpu_alarms}}, Memory {{mem_alarms}}, Disk {{disk_alarms}}
- Primary affected instances: {{affected_instances}}
- Probable root cause: {{probable_root_cause}}
- Recommended triage actions: suppress non-critical, investigate primary resource
Prioritize alarms and suggest suppression strategy.

ECS alarm storm common causes: AZ failure, hypervisor issue, underlying network partition
```

### 3.3 Trend Analysis
```
Analyze metric trend for ECS instance {{resource_id}}:
- Metric: {{metric_name}} (SYS.ECS.{{metric_name}})
- Time range: {{start_time}} to {{end_time}}
- Trend direction: {{trend_direction}} (slope: {{trend_slope}}%/day)
- Seasonal pattern: {{seasonal_pattern_detected}} (work hours vs off-hours)
- Forecasted value at {{forecast_date}}: {{forecasted_value}}
Assess if trend indicates impending issue.

Memory leak pattern: monotonic increase over days, never returns to baseline
```

### 3.4 Cross-Metric Correlation
```
Correlate metrics for ECS instance {{resource_id}}:
- Primary symptom: {{primary_metric}} {{primary_direction}} to {{primary_value}}
- Candidate causes:
  - CPU: {{cpu_value}}% (normal: 20-70%)
  - Memory: {{mem_value}}% (normal: 30-80%)
  - Disk Read: {{disk_read}}KB/s (normal: varies)
  - Disk Write: {{disk_write}}KB/s (normal: varies)
  - Network In: {{net_in}}KB/s (normal: varies)
Identify most likely correlation and suggest investigation path.

ECS correlation rules: CPU+Memory both high → application issue; Disk I/O high + CPU low → I/O bound workload
```

---

## 4. Operations Prompts

### 4.1 Backup Verification
```
Verify backup status for ECS instance {{resource_id}}:
- CBS backups: {{cbs_backup_count}} snapshots, last: {{last_cbs_backup}}
- Consistency group backup: {{cg_backup_status}}
- Backup retention: {{retention_days}} days
- Restoration test status: {{restoration_test_status}}
- Related CBR vault: {{vault_id}}
Validate backup completeness and recommend verification steps.

ECS backup types: CBS snapshot (volume-level), CBR backup (application-consistent)
```

### 4.2 Scaling Decision
```
Evaluate scaling needs for ECS auto scaling group {{scaling_group_id}}:
- Current instances: {{current_instances}}
- Current average CPU: {{avg_cpu}}%
- Scaling policy: {{scaling_policy}} (target {{target_util}}%)
- Cooldown period: {{cooldown_minutes}} minutes
- Recent scaling history: {{scaling_history}}
- Available AZ capacity: {{az_capacity}}
Recommend whether to scale out, scale in, or take no action.

Scale-out triggers: CPU > 70% for 5min, Network > 80% for 5min
Scale-in triggers: CPU < 30% for 15min, no scaling in last 15min
```

### 4.3 Configuration Drift Detection
```
Detect configuration drift for ECS instance {{resource_id}}:
- Expected security groups: {{expected_sg}}
- Actual security groups: {{actual_sg}}
- Expected tags: {{expected_tags}}
- Actual tags: {{actual_tags}}
- Instance metadata: {{instance_metadata}}
- Related monitoring alerts: {{related_alerts}}
Recommend reconciliation actions.

ECS drift sources: manual security group changes, tag modifications, instance attribute updates
```

### 4.4 Disaster Recovery Drill
```
Execute DR drill for ECS deployment:
- Primary region: {{primary_region}}, AZ: {{primary_az}}
- DR region: {{dr_region}}, AZ: {{dr_az}}
- RTO target: {{rto_target}} minutes
- RPO target: {{rpo_target}} minutes
- Test type: {{test_type}} (table-top/functional/full)
-_failover group: {{failover_group_id}}
Validate DR procedures and document gaps.

ECS DR mechanisms: AZ redundancy, multi-AZ ASG, volume replication via CBR
```

---

## 5. Optimization Prompts

### 5.1 Performance Tuning Recommendation
```
Recommend performance tuning for ECS instance {{resource_id}}:
- Current performance: latency {{latency}}ms, throughput {{throughput}}req/s
- Target performance: latency {{target_latency}}ms, throughput {{target_throughput}}req/s
- Resource constraints: vCPU {{vcpu}}, Memory {{memory}}GB
- Recent changes: {{recent_changes}}
Propose tuning actions with expected impact and risk assessment.

ECS tuning options: kernel parameters (sysctl), application-level optimizations, instance type change
```

### 5.2 Architecture Review
```
Review architecture of ECS deployment:
- Current architecture: {{architecture_description}}
- Workload characteristics: {{workload_type}} ({{peak_rps}} peak RPS)
- Performance requirements: latency < {{latency_target}}ms, availability {{availability_target}}%
- Scalability requirements: {{scale_requirement}}
- High availability setup: {{ha_setup}}
Identify architectural improvements and trade-offs.

ECS architecture patterns: standalone, ASG multi-AZ, mixed with containers (CCE), bare metal (BMS)
```

### 5.3 Resource Right-Sizing
```
Right-size ECS instance {{resource_id}}:
- Current instance type: {{current_type}} ({{vcpu}}vCPU, {{memory}}GB Memory)
- Current average utilization: CPU {{avg_cpu}}%, Memory {{avg_mem}}%
- Peak utilization: CPU {{peak_cpu}}%, Memory {{peak_mem}}%
- Application requirements: {{app_requirements}}
- Cost sensitivity: {{cost_sensitivity}}
Recommend optimal instance type with justification.

Right-sizing workflow: analyze 30-day metrics → identify underutilization → select next smaller type → validate compatibility
```

### 5.4 Cost-Benefit Analysis
```
Perform cost-benefit analysis for ECS instance {{resource_id}}:
- Current monthly cost: {{monthly_cost}} (on-demand/reserved)
- Current performance: CPU {{avg_cpu}}%, Memory {{avg_mem}}%
- Alternative instance types:
  - {{alt_type_1}}: {{alt1_cost}}/mo, suitable for {{alt1_suitable}}
  - {{alt_type_2}}: {{alt2_cost}}/mo, suitable for {{alt2_suitable}}
- Reserved instance savings potential: {{ri_savings}}%
Provide recommendation with ROI calculation.

RI recommendation: utilization > 60% → consider RI; < 30% → consider downsizing
```

---

## 6. Knowledge Base Prompts

### 6.1 Fault Pattern Matching
```
Match current ECS situation to known fault patterns:
- Current symptoms: {{symptoms}}
- Affected instance: {{resource_id}} (AZ: {{az}})
- Time of occurrence: {{occurrence_time}}
- Known ECS fault patterns:
  1. Agent disconnection: instance reachable but CES agent unresponsive
  2. Hypervisor issues: instance CPU steal high, performance degradation
  3. Network partition: instance isolated from specific CIDR ranges
  4. Resource exhaustion: memory/CPU at 100% causing unresponsiveness
  5. Disk I/O hang: EVS latency spike causing application timeout
Identify most similar pattern and suggest resolution path.
```

### 6.2 Resolution Guidance Retrieval
```
Retrieve resolution guidance for ECS issue:
- Issue type: {{issue_type}}
- Error code/message: {{error_code}} (e.g., "InstanceFault", "AgentHeartbeatTimeout")
- Instance state: {{instance_state}}
- Recent actions taken: {{recent_actions}}
Return relevant knowledge base entries with success rates.

ECS common issues: instance stuck in "starting", agent installation failures, password reset failures
```

### 6.3 Similar Incident Search
```
Search for similar past ECS incidents:
- Current incident: {{incident_description}}
- Instance type: {{instance_type}}
- AZ: {{az}}
- Time window: past 90 days
- Similarity criteria: same symptom, same instance type, same AZ
Return past incidents with resolution approaches and outcomes.
```

### 6.4 Best Practice Recommendation
```
Recommend best practices for ECS deployment:
- Current configuration: {{current_config}}
- Workload type: {{workload_type}}
- Industry best practices: use ASG for stateless workloads, multi-AZ for availability
- Huawei Cloud Well-Architected Framework: use dedicated host for stateful, ASG for scaling
- Common pitfalls: single AZ deployment, no monitoring agent, oversized instances
Provide prioritized recommendation list.

ECS WAF alignment: Security (security groups + VPC), Reliability (multi-AZ + ASG), Cost (right-sizing + RI)
```

---

## 7. Change Management Prompts

### 7.1 Change Impact Assessment
```
Assess impact of planned change on ECS:
- Change type: {{change_type}} (e.g., instance resize, security group update)
- Target instances: {{target_instances}}
- Change window: {{change_window}}
- Rollback plan: {{rollback_plan}}
- Affected services: {{affected_services}} (ELB backends, RDS connections)
Evaluate risk and provide approval recommendation.

ECS change risks: resize causes brief outage (5-10min), security group changes immediate effect
```

### 7.2 Change Correlation Analysis
```
Correlate recent changes with ECS issues:
- Issue observed: {{issue_description}}
- Time of observation: {{issue_time}}
- Changes in past 2 hours:
  - {{change_1}} at {{time_1}}: {{change_details_1}}
  - {{change_2}} at {{time_2}}: {{change_details_2}}
Determine if any change likely caused the issue.

Common ECS change triggers: scaling activities, security group updates, ASG policy changes
```

### 7.3 Pre-Change Validation
```
Validate readiness for ECS change:
- Change details: {{change_details}}
- Prerequisites met: {{prerequisites_status}} (backup, capacity check)
- Instance health: {{instance_health_status}}
- Backup status: {{backup_status}}
- Monitoring coverage: {{monitoring_coverage}} (CES agent running)
- Target AZ capacity: {{az_capacity_available}}
Confirm change can proceed or list blocking issues.
```

### 7.4 Post-Change Verification
```
Verify ECS change completed successfully:
- Change ID: {{change_id}}
- Expected changes: {{expected_changes}}
- Instance post-change state: {{instance_state}}
- Monitoring metrics: CPU {{cpu}}, Memory {{mem}}, Network {{net}}
- Alerts triggered: {{alerts_triggered}}
- Application health check: {{app_health_check}}
Confirm success or flag issues requiring attention.
```

---

## 8. Reporting Prompts

### 8.1 Daily Operations Report
```
Generate daily ECS operations report:
- Total instances: {{total_instances}}
- Instances with active alerts: {{alert_count}}
- Major incidents: {{incident_count}} (resolved: {{resolved}}, ongoing: {{ongoing}})
- Changes deployed: {{change_count}}
- Performance summary:
  - Avg CPU: {{avg_cpu}}%
  - Avg Memory: {{avg_mem}}%
  - Instances > 80% CPU: {{high_cpu_count}}
- Action items: {{action_items}}
```

### 8.2 Weekly Trend Report
```
Generate weekly ECS trend report:
- Alert volume trend: {{alert_trend}}% vs previous week ({{previous_week_count}} → {{this_week_count}})
- Top alert types: {{top_alert_types}}
- Incident summary: {{incident_count}} incidents, {{availability}}% uptime
- Capacity utilization: {{capacity_trend}}% (CPU {{avg_cpu}}%, Memory {{avg_mem}}%)
- Cost trend: {{cost_trend}}% (current: {{monthly_cost}}, previous: {{prev_monthly_cost}})
- Recommendations: {{recommendations}}
```

### 8.3 Monthly SLA Report
```
Generate monthly SLA report for ECS:
- SLO targets: Availability 99.95%, Latency P99 < 200ms
- Actual performance:
  - Availability: {{availability}}% (violations: {{availability_violations}})
  - Latency P99: {{latency_p99}}ms
  - Error rate: {{error_rate}}%
- Error budget consumed: {{error_budget_consumed}}% of {{monthly_budget}}min
- SLA violations: {{sla_violations}} incidents
- Root cause of violations: {{violation_root_causes}}
- Improvement actions: {{improvement_actions}}
```

### 8.4 Executive Summary
```
Generate executive summary for ECS operations:
- Key metrics: {{total_instances}} instances, {{monthly_cost}} cost, {{availability}}% availability
- Incidents: {{incident_count}} total, {{major_incidents}} major
- Service level: {{sla_achievement}}% SLA compliance
- Cost performance: {{cost_per_instance}}/instance/month
- Strategic recommendations: {{strategic_recommendations}}
```

---

## Appendix: ECS-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{resource_id}}` | ECS instance ID | `01b52a47-e23f-403a-9be8-4a5d2e1c3f67` |
| `{{az}}` | Availability zone | `cn-north-4a` |
| `{{instance_type}}` | ECS flavor | `c6.large.2` |
| `{{scaling_group_id}}` | Auto scaling group ID | `01b52a47-e23f-403a-9be8-4a5d2e1c3f67` |
| `{{vault_id}}` | CBR vault ID | `vault-12345` |
| `{{vcpu}}` | vCPU count | `2` |
| `{{memory}}` | Memory in GB | `4` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance (Prompt Handbook P1-3)*
