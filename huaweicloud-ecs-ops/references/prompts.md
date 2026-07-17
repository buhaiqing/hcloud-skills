# Prompts — ECS (Elastic Cloud Server)

> **Purpose**: Categorized AI prompts for ECS operations.
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

### 2.1 CPU Utilization High

```
You are analyzing CPU high alert on ECS instance {{resource_id}}.
Metric: cpu_util = {{value}}%, threshold = {{threshold}}%
Duration: {{duration}}
Region: {{region}}

Investigate:
1. Which process is consuming CPU? Run: hcloud ECS listInstances --os-type Linux && hcloud ECS describeInstances
2. Is this normal behavior or anomaly? Check historical baseline from CES.
3. Any recent changes (scaling, deployment) in the last 24h via CTS?

Provide:
- Root cause assessment (High/Medium/Low confidence)
- Recommended actions
- Whether scale-up or scale-out is recommended
```

### 2.2 Memory Utilization High

```
You are analyzing memory high alert on ECS instance {{resource_id}}.
Metric: memory_util = {{value}}%, threshold = {{threshold}}%
Duration: {{duration}}
Available memory: {{available}}MB

Investigate:
1. Is memory monotonically increasing (potential leak)?
2. Which process consuming most memory?
3. Swap usage patterns?

Provide:
- Memory leak probability (High/Medium/Low)
- Estimated exhaustion time if leak confirmed
- Recommended actions
```

### 2.3 Disk Full Warning

```
You are analyzing disk full warning on ECS instance {{resource_id}}.
Disk usage: {{usage}}%
Growth rate: {{rate}}%/hour
Time to full: {{hours}} hours
Disk type: {{disk_type}} (system/data)

Investigate:
1. Which directories consuming most space? Run: df -h, du -sh /var/log/*
2. Any large log files or core dumps?
3. Is this normal growth or abnormal?

Provide:
- Root cause assessment
- Cleanup recommendations (log rotation, old files)
- Resize recommendation if cleanup insufficient
```

### 2.4 ECS Instance Unreachable

```
You are diagnosing ECS instance {{resource_id}} unreachable.
Status: {{status}}
Last reachable: {{last_reachable}}
VPC: {{vpc_id}}, Subnet: {{subnet_id}}

Investigate:
1. Instance status via hcloud ECS describeInstances --instance-ids {{resource_id}}
2. VPC subnet ACL and security group rules
3. Recent operations via CTS (stop, reboot, delete)
4. Underlying host health (baremetal issues)

Provide:
- Reachability assessment
- Next troubleshooting steps
- Estimated recovery time
```

### 2.5 ECS Instance Performance Degradation

```
You are diagnosing performance degradation on ECS {{resource_id}}.
Symptoms: {{symptoms}} (slow response, high latency, connection failures)
Metrics:
- CPU: {{cpu}}%
- Memory: {{memory}}%
- Disk IO: {{disk_io}}%
- Network: {{network}}%

Investigate:
1. Is this compute-bound, memory-bound, IO-bound, or network-bound?
2. Check for noisy neighbor via hcloud CES getMetrics
3. Review internal processes and external traffic patterns

Provide:
- Bottleneck identification
- Remediation recommendations
- Whether migration to new host needed
```

### 2.6 Security Group Misconfiguration

```
You are diagnosing security group issue on ECS {{resource_id}}.
Symptom: {{symptom}} (cannot connect, connection reset, timeout)
Current security groups: {{sg_ids}}
Expected ports/protocols: {{expected}}

Investigate:
1. Current SG rules via hcloud VPC listSecurityGroupRules
2. Are required ports open?
3. Is source IP correctly restricted?
4. Any recent SG rule changes via CTS?

Provide:
- Misconfiguration details
- Required rule changes
- Risk assessment of proposed changes
```

---

## 3. Investigation Prompts

### 3.1 Log Investigation

```
Search LTS logs for pattern "{{pattern}}" on ECS {{resource_id}}.
Time range: {{start_time}} to {{end_time}}
Log group: {{log_group}}, Log stream: {{log_stream}}

Extract:
- Error frequency and distribution
- Correlated events across other instances
- Root error (not just symptom)

Provide findings in structured format with timestamps.
```

### 3.2 Change Correlation Investigation

```
Find CTS changes on ECS {{resource_id}} between {{start_time}} and {{end_time}}.
Filter: operation type = {{operation_types}}

Correlate with alarm at {{alarm_time}}.

Provide:
- Chronological list of changes
- Correlation score for each change
- Most likely cause with confidence
```

### 3.3 Performance Baseline Investigation

```
Compare current performance of ECS {{resource_id}} against baseline.

Current (last 1h):
- CPU: {{cpu}}%
- Memory: {{memory}}%
- Disk: {{disk}}%
- Network in: {{net_in}}Mbps, out: {{net_out}}Mbps

Baseline (weekly average):
- CPU: {{baseline_cpu}}%
- Memory: {{baseline_memory}}%
- Disk: {{baseline_disk}}%
- Network in: {{baseline_net_in}}Mbps, out: {{baseline_net_out}}Mbps

Identify anomalies and classify as:
- Normal variance
- Suspicious change requiring attention
- Critical issue needing immediate action
```

### 3.4 Dependency Health Investigation

```
Investigate dependency health for ECS {{resource_id}}.

Dependencies:
- RDS: {{rds_ids}}
- OBS: {{obs_bucket}}
- ELB: {{elb_id}}
- Redis: {{redis_ids}}

For each dependency:
1. Reachability via hcloud DCS listInstances / hcloud RDS listInstances
2. Latency from CES metrics
3. Recent failure events

Provide dependency health matrix with risk flags.
```

### 3.5 Instance Migration Investigation

```
Investigate ECS {{resource_id}} for live migration fitness.

Current state:
- Status: {{status}}
- CPU usage: {{cpu}}%
- Memory usage: {{memory}}%
- Disk type: {{disk_type}}
- Attached volumes: {{volume_ids}}

Check:
1. Is instance in correct AZ for migration?
2. Any volume type constraints (SSD vs SAS)?
3. Memory overcommit settings?

Provide migration eligibility and risk assessment.
```

---

## 4. Remediation Prompts

### 4.1 Scale Up Recommendation

```
Based on current ECS {{resource_id}} metrics:
- CPU: {{cpu}}%
- Memory: {{memory}}%
- Current flavor: {{flavor}}
- Trend: {{trend}} over past {{hours}}h

Recommend:
1. Scale up (larger flavor) or scale out (add instances)?
2. Target flavor specification?
3. Estimated cost impact (monthly)?

Provide recommendation with confidence level and risk factors.
```

### 4.2 Disk Resize Recommendation

```
Analyze disk resize need for ECS {{resource_id}}.

Current:
- System disk: {{system_disk}}GB, usage {{system_usage}}%
- Data disk: {{data_disk}}GB, usage {{data_usage}}%
- Growth rate: {{growth_rate}}GB/day

Predict: Days until disk full at current growth rate = {{days_to_full}}

Recommend:
1. Resize now or schedule?
2. Target size?
3. Whether to migrate to EVS disk type?

Provide resize plan with minimal downtime approach.
```

### 4.3 Security Group Remediation

```
Analyze security group remediation for ECS {{resource_id}}.

Current security group: {{sg_id}}
Issue: {{issue_description}}
Required access: {{required_access}}

Check:
1. Will proposed rules introduce security risk?
2. Is there a staging SG to test rules?
3. What's the rollback plan if access breaks?

Provide:
- Required SG rule changes
- Implementation sequence
- Validation steps
- Rollback procedure
```

### 4.4 Instance Restart Recommendation

```
Analyze if ECS {{resource_id}} restart would help.

Current state:
- Uptime: {{uptime}} days
- Memory leak probability: {{leak_probability}}%
- Last restart: {{last_restart}}
- Process issues: {{process_issues}}

Recommend:
1. Restart now or schedule maintenance window?
2. Is graceful shutdown possible?
3. What services need to be restarted after?

Provide restart plan with health check verification.
```

---

## 5. 巡检 Prompts

### 5.1 Daily ECS Health Check

```
Perform daily health check for ECS instance {{resource_id}}.

Checks:
1. CPU/Memory/Disk metrics within thresholds via CES
2. Instance status via hcloud ECS describeInstances
3. Security group rules unchanged via hcloud VPC listSecurityGroupRules
4. No recent critical alarms
5. Backup status if applicable

Provide:
- Health score (0-100)
- Issues found with severity
- Actions recommended
```

### 5.2 Weekly Capacity Review

```
Perform weekly capacity review for ECS {{resource_id}}.

Review:
1. Resource utilization trends (CPU, memory, disk) via CES
2. Capacity headroom for next 30 days
3. Cost optimization opportunities (downsize underutilized)
4. Upcoming traffic spikes (events, promotions)

Provide capacity forecast and recommendations.
```

### 5.3 Monthly Security Audit

```
Perform monthly security audit for ECS {{resource_id}}.

Audit:
1. All security group rules via hcloud VPC listSecurityGroupRules
2. Login history via CTS (filter: login events)
3. Unused credentials or IAM policies
4. Network exposure (public IPs, open ports)

Provide security posture report with remediation priorities.
```

---

## 6. 报告 Prompts

### 6.1 Incident Summary Report

```
Generate incident summary for ECS incident.

Incident ID: {{incident_id}}
Duration: {{start_time}} to {{end_time}}
Affected instance: {{resource_id}}
Impact: {{impact_description}}

Timeline:
{{timeline_entries}}

Root cause: {{root_cause}}

Actions taken:
{{actions_taken}}

Lessons learned:
{{lessons_learned}}

Provide formatted incident report.
```

### 6.2 Weekly Operations Summary

```
Generate weekly ECS operations report for {{resource_id}}.

Period: {{start_date}} to {{end_date}}

Metrics:
- Availability: {{availability}}%
- Avg CPU: {{avg_cpu}}%
- Avg Memory: {{avg_memory}}%
- Peak disk usage: {{peak_disk}}%

Incidents:
{{incident_list}}

Changes:
{{change_list}}

Provide executive summary and detailed metrics.
```

---

## 7. Compliance Checklist

- [x] 20 categorized prompts (6 diagnosis + 5 investigation + 4 remediation + 3 巡检 + 2 报告)
- [x] Each prompt includes context variables ({{variable}})
- [x] Each prompt specifies output format
- [x] Diagnosis prompts include confidence level
- [x] Commands reference hcloud CLI where applicable
