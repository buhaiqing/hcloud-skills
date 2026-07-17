# Prompts — Huawei Cloud CES

> **Purpose:** Structured prompts for CES (Cloud Eye Service) AIOps operations. Derived from `prompt-handbook-template.md`.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Monitoring Health Check
```
Analyze CES monitoring health status:
- Monitored resources: {{resource_count}} across {{namespace_count}} namespaces
- Active alarm rules: {{alarm_rule_count}}
- Alarms triggered: {{alarm_count}} in past {{time_window}}
- Data collection delay: {{collection_delay}} seconds
- Notification delivery status: {{notification_status}}
Determine if monitoring system is healthy and recommend actions.

CES health indicators: alarm rule evaluation latency, data ingestion delay, notification failure rate
```

### 1.2 Alarm Analysis
```
Analyze CES alarm {{alarm_id}}:
- Alarm name: {{alarm_name}}
- Namespace: {{namespace}} (e.g., SYS.ECS)
- Metric: {{metric_name}}
- Current value: {{current_value}} (threshold: {{threshold}})
- Duration: {{alarm_duration}} minutes
- Affected resources: {{affected_resources}}
- Related alarms: {{related_alarms}}
Perform root cause analysis and recommend actions.

CES alarm namespaces: SYS.ECS, SYS.RDS, SYS.ELB, SYS.VPC, SYS.DCS, SERVICE.*
```

### 1.3 Metric Anomaly Diagnosis
```
CES metric anomaly detected for {{namespace}}.{{metric_name}}:
- Resource: {{resource_id}}
- Current value: {{current_value}} (normal range: {{normal_range}})
- Deviation: {{deviation_percent}}% from baseline
- Historical pattern: {{historical_pattern}}
- Aggregation method: {{aggregation_method}} (AVG/MAX/MIN)
Determine if anomaly is significant and assess severity.

CES aggregation: AVG smooths spikes, MAX captures peaks, MIN captures valleys
```

### 1.4 Notification Delivery Analysis
```
Analyze CES notification delivery for alarm {{alarm_id}}:
- Notification type: {{notification_type}} (SMS/Email/SMS+Email)
- Topic URN: {{topic_urn}}
- Subscribers: {{subscriber_count}}
- Delivery status: {{delivery_status}}
- Delivery latency: {{delivery_latency}} seconds
- SMN quota usage: {{smn_quota_usage}}%
Identify delivery issues and recommend fixes.

CES notification channels: SMN topic, FunctionGraph, webhook,钉钉/企业微信
```

---

## 2. Inspection Prompts

### 2.1 Monitoring Coverage Inspection
```
Inspect CES monitoring coverage:
- Total cloud resources: {{total_resources}}
- Monitored resources: {{monitored_resources}} ({{coverage_percent}}%)
- Unmonitored resources: {{unmonitored_resources}}
- By namespace:
  {{#each namespaces}}
  - {{name}}: {{monitored}}/{{total}} ({{coverage}}%)
  {{/each}}
- Missing monitoring: {{missing_monitoring_details}}
Identify resources without monitoring and prioritize enabling coverage.

CES monitoring agents: CES agent (AGT.* namespace), direct API push, service native push
```

### 2.2 Alarm Rule Optimization Scan
```
Scan CES alarm rules for optimization:
- Total alarm rules: {{alarm_rule_count}}
- Alarm rules with no triggers in 30 days: {{unused_rules}}
- Duplicate alarm rules: {{duplicate_rules}}
- Overlapping thresholds: {{overlapping_rules}}
- Alarm storm risk: {{alarm_storm_risk}} (rules that could fire together)
Provide consolidation recommendations with estimated reduction.

CES alarm optimization: merge similar rules, increase evaluation periods, add suppressions
```

### 2.3 Data Quality Check
```
Check CES metric data quality:
- Namespace: {{namespace}}
- Metric: {{metric_name}}
- Data points (past 24h): {{data_points}} (expected: {{expected_points}})
- Data gap ratio: {{gap_ratio}}%
- Stale data (delay > 5min): {{stale_data_count}}
- Unusual values (outlier detection): {{outlier_count}}
Validate data integrity and flag quality issues.

CES data quality issues: agent down, API rate limits, service outage, dimension mismatch
```

### 2.4 Quota Usage Review
```
Review CES and related service quotas:
- CES alarm rules: {{alarm_rules_used}}/{{alarm_rules_limit}}
- CES alarm history retention: {{retention_days}} days
- SMN topics: {{smn_topics_used}}/{{smn_topics_limit}}
- SMN subscriptions: {{smn_subs_used}}/{{smn_subs_limit}}
- API rate limits: {{api_rate_used}}/{{api_rate_limit}}
Identify quota risks and recommend cleanup or quota increase.

CES quota management: alarm rules per namespace, dimensions per rule, notification frequency
```

---

## 3. Anomaly Detection Prompts

### 3.1 Metric Anomaly Analysis
```
Analyze metric anomaly for {{namespace}}.{{metric_name}} on {{resource_id}}:
- Current value: {{current_value}}
- Baseline (7-day avg): {{baseline_value}}
- Standard deviation: {{std_dev}}
- Anomaly score: {{anomaly_score}}/100
- Time range: {{start_time}} to {{end_time}}
- CES aggregation: {{aggregation_method}}
Determine if true anomaly, assess severity, and recommend investigation.

CES anomaly detection: uses sliding window + statistical threshold, may have false positives on volatile metrics
```

### 3.2 Alarm Storm Detection
```
Detect alarm storm in CES:
- Total alarms in past {{time_window}}: {{alarm_count}}
- Unique resources: {{unique_resources}}
- Unique namespaces: {{unique_namespaces}}
- Top alarm types:
  {{#each top_alarms}}
  - {{type}}: {{count}} ({{percentage}}%)
  {{/each}}
- Alarm burst start time: {{burst_start_time}}
- Probable root cause: {{probable_cause}}
Recommend alarm suppression and investigation strategy.

CES alarm storm mitigation: CES alarm templates with dependencies, suppression rules, severity escalation
```

### 3.3 Cross-Namespace Correlation
```
Correlate metrics across CES namespaces:
- Primary symptom: {{namespace}}.{{metric_name}} on {{resource_id}} = {{value}}
- Correlated metrics:
  - {{namespace2}}.{{metric2}}: {{value2}} (correlation: {{corr2}}%)
  - {{namespace3}}.{{metric3}}: {{value3}} (correlation: {{corr3}}%)
- Cross-service correlation: {{cross_service_correlation}}
- Lag analysis: {{metric2}} {{lags_behind}} seconds behind {{metric1}}
Identify root cause metric and propagation path.

CES cross-namespace correlation: ECS CPU → RDS connections, ELB traffic → ECS metrics
```

### 3.4 Trend Deviation Detection
```
Detect metric trend deviation in CES:
- Metric: {{namespace}}.{{metric_name}}
- Resource: {{resource_id}}
- Expected trend: {{expected_trend}}% daily
- Actual trend: {{actual_trend}}% daily
- Forecast deviation: {{forecast_deviation}}%
- Seasonal pattern: {{seasonal_pattern}}
- Anomaly confidence: {{confidence}}%
Assess if deviation indicates future issue.

CES trend analysis: use historical data for baseline, detect gradual degradation vs sudden shift
```

---

## 4. Operations Prompts

### 4.1 Alarm Rule Creation
```
Create CES alarm rule for {{namespace}}.{{metric_name}}:
- Resource dimension: {{dimension}} = {{dimension_value}}
- Comparison operator: {{operator}} (GT/GTE/LT/LTE/EQ)
- Threshold: {{threshold}} ({{unit}})
- Evaluation period: {{evaluation_period}} consecutive periods
- Period length: {{period_length}} seconds
- Alarm level: {{alarm_level}} (1=Critical, 2=Major, 3=Minor, 4=Info)
- Notification: {{notification_method}}
- OK actions: {{ok_actions}}
- Alarm actions: {{alarm_actions}}
Validate rule syntax and test with sample data.

CES alarm rule constraints: max 30 dimensions per rule, evaluation period must align with data granularity
```

### 4.2 Alarm Suppression Configuration
```
Configure CES alarm suppression for {{alarm_pattern}}:
- Suppression scope: {{scope}} (alarm/namespace/resource)
- Suppression duration: {{duration_minutes}} minutes
- Evaluation continues: {{evaluation_continues}} (yes/no)
- Auto-resume: {{auto_resume}} (yes/no)
- Related alarms to suppress: {{related_alarms}}
Design suppression strategy that prevents alarm fatigue without masking real issues.

CES suppression use cases: maintenance windows, planned changes, known false positives
```

### 4.3 Metric Dashboard Design
```
Design CES dashboard for {{use_case}}:
- Dashboard name: {{dashboard_name}}
- Required metrics:
  {{#each metrics}}
  - {{namespace}}.{{metric_name}} (dimension: {{dimension}})
  {{/each}}
- Visualization types: {{viz_types}} (line/graph/gauge/stat)
- Refresh interval: {{refresh_interval}} seconds
- Time range: {{time_range}}
- Comparison period: {{comparison_period}}
Create dashboard layout with optimal widget arrangement.

CES dashboard best practices: max 12 widgets per dashboard, use templates for consistency
```

### 4.4 Notification Policy Update
```
Update CES notification policy for {{topic_name}}:
- Current subscribers: {{current_subscribers}}
- New subscriber: {{new_subscriber}} ({{subscriber_type}})
- Notification content template: {{content_template}}
- Retry policy: {{retry_policy}} ({{max_retries}} retries, {{retry_interval}}s interval)
- Rate limiting: {{rate_limit}} notifications per minute
- Quiet hours: {{quiet_hours}} (UTC{{quiet_hours_offset}})
Validate policy changes won't cause notification gaps.

CES notification best practices: use SMN topic with multiple subscriber types for redundancy
```

---

## 5. Optimization Prompts

### 5.1 Alarm Efficiency Optimization
```
Optimize CES alarm efficiency:
- Current MTTD (mean time to detect): {{mttd}} minutes
- Current false positive rate: {{false_positive_rate}}%
- Alarm noise ratio: {{noise_ratio}}%
- Proposed changes:
  1. {{change_1}}: expected improvement {{improvement_1}}%
  2. {{change_2}}: expected improvement {{improvement_2}}%
Provide cost-benefit analysis of optimization.

CES alarm efficiency: tune evaluation periods, use multi-condition alarms, add data validation
```

### 5.2 Monitoring Cost Optimization
```
Optimize CES monitoring costs:
- Current monitoring cost: {{monthly_cost}} CNY
- Data ingestion: {{data_ingestion}} metrics/month
- Historical data storage: {{storage_gb}} GB
- Cost allocation by namespace:
  {{#each by_namespace}}
  - {{name}}: {{cost}} ({{percentage}}%)
  {{/each}}
- Optimization opportunities:
  - Reduce granularity for {{namespace}}: save {{savings_1}}
  - Delete unused rules: save {{savings_2}}
  - Adjust retention: save {{savings_3}}
Provide prioritized cost reduction plan.

CES cost optimization: reduce 1s metrics to 5min, delete unused rules, optimize retention periods
```

### 5.3 Threshold Tuning
```
Tune CES alarm thresholds based on historical data:
- Metric: {{namespace}}.{{metric_name}}
- Resource: {{resource_id}}
- Current threshold: {{current_threshold}}
- Historical data analysis (30 days):
  - Min: {{min_value}}, Max: {{max_value}}, Avg: {{avg_value}}
  - P50: {{p50}}, P90: {{p90}}, P99: {{p99}}
- Recommended threshold: {{recommended_threshold}}
- Confidence: {{confidence}}%
Validate recommendation against business requirements.

CES threshold tuning: use P95-P99 for critical alarms, avoid threshold near daily average
```

### 5.4 Multi-Condition Alarm Design
```
Design multi-condition CES alarm for {{use_case}}:
- Condition 1: {{namespace1}}.{{metric1}} {{op1}} {{threshold1}} for {{periods1}} periods
- Condition 2: {{namespace2}}.{{metric2}} {{op2}} {{threshold2}} for {{periods2}} periods
- Logical operator: {{operator}} (AND/OR)
- Alarm level: {{alarm_level}}
- Require all conditions: {{require_all}}
Design alarm that reduces false positives while maintaining detection sensitivity.

CES multi-condition use cases: high CPU + high memory, high latency + high error rate
```

---

## 6. Knowledge Base Prompts

### 6.1 Alarm Pattern Matching
```
Match CES alarm to known patterns:
- Alarm: {{alarm_name}} ({{alarm_id}})
- Namespace: {{namespace}}
- Metric: {{metric_name}}
- Value: {{current_value}} vs threshold {{threshold}}
- Duration: {{duration}} minutes
- Known patterns:
  1. Flash crowd: sudden spike, short duration (< 5min), no resource issue
  2. Gradual degradation: monotonic increase over hours, approaching threshold
  3. Metric collectors: CES agent issue, shows flatline or gaps
  4. Resource exhaustion: sustained high value, slow recovery
Identify most likely pattern and resolution guidance.
```

### 6.2 Resolution Guidance Retrieval
```
Retrieve resolution guidance for CES alarm:
- Alarm ID: {{alarm_id}}
- Alarm name: {{alarm_name}}
- Namespace: {{namespace}}
- Metric: {{metric_name}}
- Affected resource: {{resource_id}}
- Alarm history: {{alarm_history}}
Return applicable runbook and success metrics.

CES common resolutions: threshold tuning, resource scale-up, application restart, agent reinstall
```

### 6.3 Similar Alarm Analysis
```
Analyze similar past CES alarms:
- Current alarm: {{alarm_name}} on {{resource_id}}
- Namespace: {{namespace}}
- Time window: past 90 days
- Similarity criteria: same namespace, same metric, similar threshold
- Found {{similar_count}} similar alarms:
  {{#each similar_alarms}}
  - {{date}}: {{resolution}}, {{resolution_time}} to resolve
  {{/each}}
Predict resolution time and recommend approach.
```

### 6.4 Best Practice Recommendation
```
Recommend CES best practices for {{use_case}}:
- Current setup: {{current_setup}}
- Industry best practices: {{industry_best_practices}}
- Huawei Cloud Well-Architected monitoring alignment
- Common pitfalls: over-alarming, under-alarming, no escalation path
- Recommended alarm strategy: {{recommended_strategy}}
Provide implementation roadmap.

CES WAF alignment: Reliability (alarm response SLO), Cost (right-sized granularity), Operations (MTTD optimization)
```

---

## 7. Change Management Prompts

### 7.1 Change Impact Assessment
```
Assess impact of CES configuration change:
- Change type: {{change_type}} (create/update/delete alarm rule)
- Target: {{target}} (alarm rule / notification policy / dashboard)
- Change window: {{change_window}}
- Rollback plan: {{rollback_plan}}
- Affected monitoring coverage: {{affected_coverage}}
- Risk level: {{risk_level}} (Low/Medium/High/Critical)
Evaluate if change can proceed safely.
```

### 7.2 Change Correlation Analysis
```
Correlate CES changes with downstream impact:
- Issue observed: {{issue_description}}
- Time of observation: {{issue_time}}
- CES changes in past hour:
  - {{change_1}}: {{change_details_1}}
  - {{change_2}}: {{change_details_2}}
- Downstream impact: {{downstream_impact}}
Determine if CES changes caused or contributed to issue.

CES change correlation: alarm rule modifications, threshold changes, notification updates
```

### 7.3 Pre-Change Validation
```
Validate CES change readiness:
- Change details: {{change_details}}
- Syntax validation: {{syntax_valid}} (yes/no)
- Resource existence: {{resource_exists}} (yes/no)
- Permission validation: {{permission_ok}} (yes/no)
- Notification test: {{notification_test}} (sent/not sent)
- Monitoring data availability: {{data_available}} (yes/no)
Confirm change can proceed or list blocking issues.
```

### 7.4 Post-Change Verification
```
Verify CES change completed successfully:
- Change ID: {{change_id}}
- Change type: {{change_type}}
- Expected outcome: {{expected_outcome}}
- Verification steps:
  1. Alarm rule exists: {{rule_exists}} (yes/no)
  2. Metric data flowing: {{data_flowing}} (yes/no)
  3. Test alarm triggered: {{test_alarm_triggered}} (yes/no)
  4. Notification delivered: {{notification_delivered}} (yes/no)
Confirm success or flag issues.
```

---

## 8. Reporting Prompts

### 8.1 Daily Monitoring Report
```
Generate daily CES monitoring report:
- Total monitored resources: {{total_resources}}
- Active alarms: {{active_alarms}} (Critical: {{critical}}, Major: {{major}})
- Alarms triggered today: {{alarms_today}}
- Mean time to detect: {{mttd}} minutes
- Notification delivery rate: {{notification_rate}}%
- Data quality score: {{data_quality_score}}/100
- Action items: {{action_items}}
```

### 8.2 Weekly Alarm Analysis Report
```
Generate weekly CES alarm analysis report:
- Alarm volume: {{alarm_count}} (vs {{prev_week_count}} last week, {{trend}}%)
- Alarms by namespace:
  {{#each by_namespace}}
  - {{name}}: {{count}} ({{percentage}}%)
  {{/each}}
- Top alarm types:
  {{#each top_types}}
  - {{type}}: {{count}}
  {{/each}}
- False positive rate: {{false_positive_rate}}%
- Alarm noise ratio: {{noise_ratio}}%
- Recommendations: {{recommendations}}
```

### 8.3 Monthly Monitoring SLA Report
```
Generate monthly CES monitoring SLA report:
- Monitoring availability: {{monitoring_availability}}% (target: 99.9%)
- Data collection delay: {{avg_delay}}s (target: < 60s)
- Alarm delivery latency: {{delivery_latency}}s (target: < 30s)
- Alarm accuracy: {{alarm_accuracy}}% (no false positives)
- Data completeness: {{data_completeness}}% (target: > 99%)
- SLA violations: {{sla_violations}}
- Root causes: {{violation_root_causes}}
- Improvements: {{improvements}}
```

### 8.4 Executive Summary
```
Generate executive summary for monitoring operations:
- Monitoring scope: {{monitored_count}} resources, {{namespace_count}} services
- Alarm efficiency: {{alarm_efficiency}}% (actionable alarms / total)
- MTTD improvement: {{mttd_improvement}}% vs last month
- Cost performance: {{monthly_cost}} CNY for {{resource_count}} resources
- Strategic recommendations: {{strategic_recommendations}}
```

---

## Appendix: CES-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{namespace}}` | CES metric namespace | `SYS.ECS`, `SYS.RDS`, `SERVICE.custom` |
| `{{metric_name}}` | CES metric name | `cpu_util`, `memory_util`, `latency_p99` |
| `{{alarm_id}}` | CES alarm rule ID | `alarm-12345` |
| `{{topic_urn}}` | SMN topic URN | `urn:smn:cn-north-4:123456:topic-name` |
| `{{aggregation_method}}` | CES aggregation | `AVG`, `MAX`, `MIN`, `SUM` |
| `{{period_length}}` | Evaluation period in seconds | `60`, `300`, `900` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance (Prompt Handbook P1-3)*
