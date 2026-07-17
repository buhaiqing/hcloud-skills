# Prompts — CES (Cloud Eye Service)

> **Purpose**: Categorized AI prompts for CES monitoring operations.
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

### 2.1 Alarm Storm Detection

```
You are analyzing alarm storm on CES tenant {{tenant_id}}.
Alarm count: {{alarm_count}} in {{time_window}} minutes
Alarm types: {{alarm_types}}
Top affected services: {{affected_services}}

Investigate:
1. Is this a single root cause triggering cascade?
2. Which alarm is the root (not symptom)?
3. Any recent changes that could trigger this?

Prioritize:
- Identify root alarm
- Suppress redundant alarms
- Address root cause

Provide:
- Root cause assessment
- Alarm correlation analysis
- Recommended suppression strategy
```

### 2.2 Metric Anomaly Detection

```
You are analyzing metric anomaly on {{resource_type}} {{resource_id}}.
Metric: {{metric_name}}
Current value: {{current_value}}
Expected range: {{min_value}} - {{max_value}}
Deviation: {{deviation}}%
Timestamp: {{timestamp}}

Investigate:
1. Is this a true anomaly or expected behavior?
2. Correlate with other metrics on same resource
3. Check for recent scaling events or deployments

Provide:
- Anomaly classification (spike, drift, step change)
- Confidence level (High/Medium/Low)
- Recommended threshold adjustment
```

### 2.3 Monitoring Agent Health

```
You are diagnosing CES agent health on instance {{resource_id}}.
Agent status: {{agent_status}}
Last heartbeat: {{last_heartbeat}}
Metric gap: {{metric_gap}} minutes

Investigate:
1. Agent process running? Check via hcloud ECS describeInstances
2. Agent logs available? Check LTS for agent-{{resource_id}} logs
3. Network connectivity to CES endpoint?
4. Disk space for metric buffering?

Provide:
- Agent health assessment
- Recovery steps if unhealthy
- Data gap impact analysis
```

### 2.4 Threshold Tuning Analysis

```
You are analyzing alarm threshold for {{resource_id}}.

Current threshold: {{current_threshold}}
Metric: {{metric_name}} = {{current_value}}%
Alarm history:
- Alarms triggered: {{alarm_count}} in past {{period}}
- False positive rate: {{false_positive_rate}}%
- True positive rate: {{true_positive_rate}}%

Check:
1. Is current threshold too sensitive or too loose?
2. What's the business impact of missing an alarm?
3. Can we use dynamic threshold based on time series?

Provide:
- Threshold adjustment recommendation
- Expected alarm frequency change
- Implementation risk
```

### 2.5 Cross-Service Correlation

```
You are analyzing cross-service alarm correlation for {{resource_id}}.

Alarms triggered:
{{alarm_list}}

Time range: {{start_time}} to {{end_time}}

Investigate:
1. Which service is the root cause?
2. Which alarms are downstream effects?
3. Any dependency relationship between services?

Use CES alarm correlation rules and dependency graph.

Provide:
- Root cause service identification
- Downstream impact chain
- Recommended alarm grouping
```

### 2.6 SLA Breach Prediction

```
You are predicting SLA breach risk for {{resource_id}}.

Current metrics:
- Availability: {{availability}}%
- Latency P99: {{latency_p99}}ms
- Error rate: {{error_rate}}%

SLA targets:
- Availability: {{sla_availability}}%
- Latency P99: {{sla_latency}}ms
- Error rate: {{sla_error_rate}}%

Trend: {{trend}} over past {{days}} days

Predict:
1. Days until SLA breach at current trend
2. Which metric will breach first?
3. Required improvement to meet SLA?

Provide breach risk assessment and mitigation recommendations.
```

---

## 3. Investigation Prompts

### 3.1 Metric Data Gap Investigation

```
Investigate metric data gap for {{resource_id}}.

Gap details:
- Metric: {{metric_name}}
- Gap start: {{gap_start}}
- Gap end: {{gap_end}}
- Gap duration: {{gap_duration}} minutes

Check:
1. Was agent running during gap? Via hcloud CES listAgents
2. Network connectivity at gap time?
3. CES API availability status?
4. Any scheduled maintenance?

Provide:
- Gap cause analysis
- Data completeness percentage
- Recovery options for historical data
```

### 3.2 Alarm Notification Investigation

```
Investigate alarm notification delivery for alarm {{alarm_id}}.

Expected recipients: {{recipients}}
Notification channels: {{channels}} (SMS/Email/Webhook)
Delivery status: {{delivery_status}}

Check via CTS:
1. Was alarm triggered? Filter: alarm_id = {{alarm_id}}
2. Were notifications sent?
3. Any delivery failures logged?

Provide:
- Notification delivery status
- Failure reason if any
- Recommended fix for notification delivery
```

### 3.3 Historical Trend Analysis

```
Analyze historical trends for {{resource_id}} metric {{metric_name}}.

Time range: {{start_time}} to {{end_time}}
Granularity: {{granularity}} (5min/1h/1d)

Analyze:
1. Daily/weekly/monthly patterns (seasonality)
2. Anomaly history (when and why)
3. Growth trajectory
4. Correlation with other metrics

Provide:
- Trend summary with statistics
- Forecast for next {{forecast_period}}
- Anomaly explanation if significant
```

### 3.4 Resource Group Health Investigation

```
Investigate health of resource group {{group_id}}.

Resources in group: {{resource_count}}
Healthy: {{healthy_count}}
Unhealthy: {{unhealthy_count}}
Unknown: {{unknown_count}}

For unhealthy resources:
- List resources with active alarms
- Identify common alarm patterns
- Check for shared dependencies

Provide:
- Group health score
- Critical issues requiring immediate attention
- Recommended group-level actions
```

### 3.5 Dashboard Configuration Investigation

```
Investigate CES dashboard {{dashboard_id}} configuration.

Dashboard name: {{dashboard_name}}
Widgets: {{widget_count}}
Last modified: {{last_modified}}

Check:
1. Are all widgets showing data (no stale metrics)?
2. Are alarm widgets correctly configured?
3. Any widgets referencing deleted resources?
4. Are thresholds aligned with current SLA?

Provide:
- Dashboard health assessment
- Stale widget list
- Recommended configuration updates
```

---

## 4. Remediation Prompts

### 4.1 Alarm Suppression Recommendation

```
Analyze alarm suppression for CES alarm storm.

Situation:
- Alarms: {{alarm_count}} in {{time_window}} minutes
- Root cause: {{root_cause}}
- Estimated duration: {{estimated_duration}}

Recommend:
1. Suppress alarms temporarily (yes/no)?
2. Suppression scope (which alarms/-resources)?
3. Suppression duration?

Caution: Alarm suppression reduces visibility.

Provide suppression plan with:
- Scope definition
- Duration justification
- Communication plan
- Auto-resume trigger
```

### 4.2 Metric Retention Adjustment

```
Analyze metric retention settings for {{resource_id}}.

Current retention:
- 5min granularity: {{retention_5min}} days
- 1h granularity: {{retention_1h}} days
- 1d granularity: {{retention_1d}} days

Requirements:
- Historical analysis depth: {{required_depth}} days
- Storage cost concern: {{cost_constraint}}

Recommend:
1. Optimal retention per granularity
2. Cost vs. analysis need tradeoff
3. Archival strategy for long-term storage

Provide retention plan with cost estimate.
```

### 4.3 Alarm Template Remediation

```
Analyze alarm template remediation for {{template_id}}.

Template: {{template_name}}
Active alarms using template: {{alarm_count}}
Issue: {{issue_description}}

Check:
1. Current threshold values
2. Recent true positive vs false positive rate
3. Business impact of current settings

Recommend:
1. Threshold adjustments
2. Additional conditions (e.g., consecutive checks)
3. New alarm template design

Provide updated template with justification.
```

### 4.4 Monitoring Plan Optimization

```
Optimize monitoring plan for {{resource_id}}.

Current plan:
- Metrics monitored: {{metric_count}}
- Alarm rules: {{alarm_count}}
- Notification groups: {{notification_count}}

Efficiency analysis:
- Alarms with no action taken: {{no_action_count}}
- Duplicate alarms: {{duplicate_count}}
- Metrics never anomalous: {{unused_metric_count}}

Recommend:
1. Remove redundant alarms
2. Consolidate duplicate monitoring
3. Add monitoring gaps identified

Provide optimization plan with expected improvement.
```

---

## 5. 巡检 Prompts

### 5.1 Daily Monitoring Health Check

```
Perform daily monitoring health check via CES.

Checks:
1. All agents healthy? Via hcloud CES listAgents
2. No metric data gaps > 30 minutes?
3. All critical alarms acknowledged?
4. Dashboard availability?

Provide:
- Monitoring system health score
- Issues requiring attention
- Recommendations for next 24h
```

### 5.2 Weekly Alarm Review

```
Perform weekly alarm review for {{project_id}}.

Review period: {{start_date}} to {{end_date}}

Analyze:
1. Alarm frequency by service
2. Alarm true positive rate
3. Mean time to acknowledge (MTTA)
4. Mean time to resolve (MTTR)

Provide:
- Alarm efficiency metrics
- Problematic alarm patterns
- Recommended threshold adjustments
```

### 5.3 Monthly Monitoring Coverage Audit

```
Perform monthly monitoring coverage audit.

Audit scope: {{project_id}} / {{region}}

Check:
1. All critical resources have monitoring?
2. All alarms have valid notification targets?
3. Dashboard access for all stakeholders?
4. Monitoring costs vs. value?

Provide:
- Coverage percentage by resource type
- Gap list with prioritization
- Cost optimization opportunities
```

---

## 6. 报告 Prompts

### 6.1 Incident Monitoring Analysis

```
Generate monitoring analysis for incident {{incident_id}}.

Incident details:
- Duration: {{start_time}} to {{end_time}}
- Affected: {{affected_resources}}
- Impact: {{impact_description}}

Monitoring data:
- Alarms triggered: {{alarm_list}}
- Metric anomalies: {{anomaly_list}}
- Timeline correlation

Provide:
- What monitoring observed before/during/after incident
- Alarm response time analysis
- Recommendations to improve monitoring response
```

### 6.2 Monthly Monitoring Summary

```
Generate monthly monitoring summary for {{project_id}}.

Period: {{month}}

Summary:
- Total alarms: {{total_alarms}}
- Alarm by severity: {{severity_breakdown}}
- Top alerting resources: {{top_resources}}
- Monitoring costs: {{costs}}

Achievements:
- Improved coverage: {{coverage_improvement}}
- Reduced false positives: {{fp_reduction}}%

Provide detailed monthly report with trends and recommendations.
```

---

## 7. Compliance Checklist

- [x] 20 categorized prompts (6 diagnosis + 5 investigation + 4 remediation + 3 巡检 + 2 报告)
- [x] Each prompt includes context variables ({{variable}})
- [x] Each prompt specifies output format
- [x] Diagnosis prompts include confidence level
- [x] Commands reference hcloud CLI where applicable
