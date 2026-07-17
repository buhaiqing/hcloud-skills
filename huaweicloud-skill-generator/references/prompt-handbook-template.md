# Prompt Handbook Template — Huawei Cloud Skill Generator

> **Purpose:** Reusable prompt templates for AIOps L4 compliance. Each skill-specific `prompts.md` should adapt these templates.
> **Version:** 1.0.0
> **Status:** Template — replace placeholders with product-specific content

---

## 1. Diagnostic Prompts

### 1.1 Instance Health Check
```
Analyze {{product}} instance {{resource_id}} health status:
- Current metric values: CPU {{cpu_usage}}%, Memory {{mem_usage}}%, Disk {{disk_usage}}%
- Recent alert history: {{alert_count}} alerts in past {{time_window}}
- Related events: {{event_summary}}
Determine if instance is healthy and recommend actions.
```

### 1.2 Root Cause Analysis
```
Given {{product}} {{resource_type}} {{resource_id}} shows:
- Symptom: {{symptom_description}}
- First observed: {{first_observed_time}}
- Metric anomaly: {{anomaly_metrics}}
- correlated events: {{correlated_events}}
Perform root cause analysis and provide ranked hypothesis list with confidence scores.
```

### 1.3 Performance Degradation Diagnosis
```
{{product}} {{resource_type}} {{resource_id}} performance degraded:
- Latency increased from {{baseline_latency}}ms to {{current_latency}}ms
- Error rate changed from {{baseline_error}}% to {{current_error}}%
- Traffic {{traffic_change_direction}} by {{traffic_change_percent}}%
- Resource utilization: {{resource_metrics}}
Diagnose root cause and suggest remediation steps.
```

### 1.4 Availability Incident Response
```
{{product}} {{resource_type}} {{resource_id}} availability issue detected:
- Availability: {{current_availability}}% (SLO target: {{slo_target}}%)
- Impact duration: {{duration_minutes}} minutes
- Affected scope: {{affected_scope}}
- Current status: {{current_status}}
Generate incident response playbook with next steps.
```

---

## 2. Inspection Prompts

### 2.1 Routine Health Inspection
```
Perform routine inspection on {{product}} {{resource_type}}:
- List all resources in {{scope}}
- Check for resources with CPU > {{threshold_cpu}}% OR Memory > {{threshold_mem}}% sustained for {{duration}}
- Identify resources with no monitoring data for > {{no_data_hours}} hours
- Flag any resources with active alerts
Report findings in structured format.
```

### 2.2 Cost Optimization Scan
```
Scan {{product}} {{resource_type}} for cost optimization opportunities:
- Identify idle resources (CPU < {{idle_cpu}}%, Network < {{idle_network}}% for {{idle_period_days}} days)
- Find oversized resources (utilization < {{undersized_threshold}}% over {{undersized_period}} days)
- Check for overdue reserved capacity transitions
- Verify tag compliance for cost attribution
Provide prioritized action list with estimated savings.
```

### 2.3 Security Compliance Check
```
Audit {{product}} {{resource_type}} security compliance:
- Verify IAM policies follow least-privilege principle
- Check for exposed credentials or insecure configurations
- Validate network isolation (VPC endpoints, security groups)
- Confirm encryption at rest and in transit is enabled
- Check for unpatched vulnerabilities
Report compliance status and remediation priorities.
```

### 2.4 Capacity Planning Review
```
Review capacity for {{product}} {{resource_type}}:
- Current utilization: CPU {{cpu_util}}%, Memory {{mem_util}}%, Storage {{storage_util}}%
- Growth trend: {{growth_rate}}% weekly average
- Projected capacity exhaustion: {{exhaustion_date}}
- Current capacity headroom: {{headroom_days}} days
Provide scaling recommendations with timeline.
```

---

## 3. Anomaly Detection Prompts

### 3.1 Metric Anomaly Analysis
```
Analyze metric anomaly for {{product}} {{resource_id}}:
- Metric: {{metric_name}}
- Current value: {{current_value}} (baseline: {{baseline_value}})
- Deviation: {{deviation_percent}}% from baseline
- Duration: {{anomaly_duration}} minutes
- Similar historical patterns: {{historical_patterns}}
Determine if this is a true anomaly or false positive, and assess severity.
```

### 3.2 Alarm Storm Triage
```
Triage alarm storm affecting {{product}}:
- Total alarms: {{alarm_count}} in past {{time_window}}
- Alarms by type: {{alarm_type_distribution}}
- Primary affected resources: {{affected_resources}}
- Probable root cause: {{probable_root_cause}}
- Recommended triage actions: {{recommended_actions}}
Prioritize alarms and suggest suppression strategy.
```

### 3.3 Trend Analysis
```
Analyze metric trend for {{product}} {{resource_type}} {{resource_id}}:
- Metric: {{metric_name}}
- Time range: {{start_time}} to {{end_time}}
- Trend direction: {{trend_direction}} (slope: {{trend_slope}}%/day)
- Seasonal pattern: {{seasonal_pattern_detected}}
- Forecasted value at {{forecast_date}}: {{forecasted_value}}
Assess if trend indicates impending issue.
```

### 3.4 Cross-Metric Correlation
```
Correlate metrics for {{product}} {{resource_id}}:
- Primary symptom: {{primary_metric}} {{primary_direction}} to {{primary_value}}
- Candidate causes:
  - CPU: {{cpu_value}}% (normal: {{cpu_normal}})
  - Memory: {{mem_value}}% (normal: {{mem_normal}})
  - Network: {{net_value}}% (normal: {{net_normal}})
  - Disk: {{disk_value}}% (normal: {{disk_normal}})
Identify most likely correlation and suggest investigation path.
```

---

## 4. Operations Prompts

### 4.1 Backup Verification
```
Verify backup status for {{product}} {{resource_type}} {{resource_id}}:
- Last successful backup: {{last_backup_time}}
- Backup retention: {{retention_days}} days
- Backup size: {{backup_size}}GB
- Backup schedule: {{backup_schedule}}
- Restoration test status: {{restoration_test_status}}
Validate backup completeness and recommend verification steps.
```

### 4.2 Scaling Decision
```
Evaluate scaling needs for {{product}} {{resource_type}} {{scaling_group_id}}:
- Current instances: {{current_instances}}
- Current average CPU: {{avg_cpu}}%
- Scaling policy: {{scaling_policy}}
- Cooldown period: {{cooldown_minutes}} minutes
- Recent scaling history: {{scaling_history}}
Recommend whether to scale out, scale in, or take no action.
```

### 4.3 Configuration Drift Detection
```
Detect configuration drift for {{product}} {{resource_type}} {{resource_id}}:
- Expected configuration: {{expected_config_hash}}
- Actual configuration: {{actual_config_hash}}
- Drift details: {{drift_details}}
- Last config update: {{last_update_time}}
- Related monitoring alerts: {{related_alerts}}
Recommend reconciliation actions.
```

### 4.4 Disaster Recovery Drill
```
Execute DR drill for {{product}} {{resource_type}}:
- Primary region: {{primary_region}}
- DR region: {{dr_region}}
- RTO target: {{rto_target}} minutes
- RPO target: {{rpo_target}} minutes
- Test type: {{test_type}} (table-top/functional/full)
Validate DR procedures and document gaps.
```

---

## 5. Optimization Prompts

### 5.1 Performance Tuning Recommendation
```
Recommend performance tuning for {{product}} {{resource_type}} {{resource_id}}:
- Current performance: {{current_metrics}}
- Target performance: {{target_metrics}}
- Resource constraints: {{resource_constraints}}
- Recent changes: {{recent_changes}}
Propose tuning actions with expected impact and risk assessment.
```

### 5.2 Architecture Review
```
Review architecture of {{product}} deployment:
- Current architecture: {{architecture_description}}
- Workload characteristics: {{workload_type}}
- Performance requirements: {{performance_requirements}}
- Scalability requirements: {{scalability_requirements}}
- Identify architectural improvements and trade-offs.
```

### 5.3 Resource Right-Sizing
```
Right-size {{product}} {{resource_type}} {{resource_id}}:
- Current flavor/size: {{current_flavor}}
- Current average utilization: CPU {{avg_cpu}}%, Memory {{avg_mem}}%
- Peak utilization: CPU {{peak_cpu}}%, Memory {{peak_mem}}%
- Application requirements: {{app_requirements}}
Recommend optimal flavor/size with justification.
```

### 5.4 Cost-Benefit Analysis
```
Perform cost-benefit analysis for {{product}} {{resource_type}} {{resource_id}}:
- Current monthly cost: {{monthly_cost}}
- Current performance: {{performance_metrics}}
- Alternative options: {{alternative_options}}
- Cost per unit of performance: {{cost_per_unit}}
Provide recommendation with ROI calculation.
```

---

## 6. Knowledge Base Prompts

### 6.1 Fault Pattern Matching
```
Match current situation to known fault patterns:
- Current symptoms: {{symptoms}}
- Affected resources: {{affected_resources}}
- Time of occurrence: {{occurrence_time}}
- Known patterns in knowledge base:
  {{#each patterns}}
  - {{name}}: {{description}} (similarity: {{similarity_score}}%)
  {{/each}}
Identify most similar pattern and suggest resolution path.
```

### 6.2 Resolution Guidance Retrieval
```
Retrieve resolution guidance for {{product}} issue:
- Issue type: {{issue_type}}
- Error code/message: {{error_code}}
- Resource state: {{resource_state}}
- Recent actions taken: {{recent_actions}}
Return relevant knowledge base entries with success rates.
```

### 6.3 Similar Incident Search
```
Search for similar past incidents:
- Current incident: {{incident_description}}
- Affected product: {{product}}
- Time window: {{lookback_period}}
- Similarity criteria: {{similarity_criteria}}
Return past incidents with resolution approaches and outcomes.
```

### 6.4 Best Practice Recommendation
```
Recommend best practices for {{product}} {{resource_type}}:
- Current configuration: {{current_config}}
- Industry best practices: {{industry_best_practices}}
- Huawei Cloud Well-Architected Framework alignment: {{waf_alignment}}
- Common pitfalls for this resource type: {{common_pitfalls}}
Provide prioritized recommendation list.
```

---

## 7. Change Management Prompts

### 7.1 Change Impact Assessment
```
Assess impact of planned change:
- Change type: {{change_type}}
- Target resources: {{target_resources}}
- Change window: {{change_window}}
- Rollback plan: {{rollback_plan}}
- Affected services: {{affected_services}}
Evaluate risk and provide approval recommendation.
```

### 7.2 Change Correlation Analysis
```
Correlate recent changes with {{product}} issues:
- Issue observed: {{issue_description}}
- Time of observation: {{issue_time}}
- Changes in past {{lookback_window}}:
  {{#each changes}}
  - {{change_type}} at {{change_time}}: {{change_details}}
  {{/each}}
Determine if any change likely caused the issue.
```

### 7.3 Pre-Change Validation
```
Validate readiness for {{product}} change:
- Change details: {{change_details}}
- Prerequisites met: {{prerequisites_status}}
- Resource health: {{resource_health_status}}
- Backup status: {{backup_status}}
- Monitoring coverage: {{monitoring_coverage}}
Confirm change can proceed or list blocking issues.
```

### 7.4 Post-Change Verification
```
Verify change completed successfully:
- Change ID: {{change_id}}
- Expected changes: {{expected_changes}}
- Actual changes observed: {{actual_changes}}
- Monitoring metrics post-change: {{post_change_metrics}}
- Alerts triggered: {{alerts_triggered}}
Confirm success or flag issues requiring attention.
```

---

## 8. Reporting Prompts

### 8.1 Daily Operations Report
```
Generate daily operations report for {{product}}:
- Total resources monitored: {{total_resources}}
- Resources with active alerts: {{alert_count}}
- Major incidents: {{incident_count}}
- Changes deployed: {{change_count}}
- Performance summary: {{performance_summary}}
- Action items: {{action_items}}
```

### 8.2 Weekly Trend Report
```
Generate weekly trend report for {{product}}:
- Alert volume trend: {{alert_trend}}% vs previous week
- Top alert types: {{top_alert_types}}
- Incident summary: {{incident_summary}}
- Capacity utilization trend: {{capacity_trend}}
- Cost trend: {{cost_trend}}
- Recommendations: {{recommendations}}
```

### 8.3 Monthly SLA Report
```
Generate monthly SLA report for {{product}} {{resource_type}}:
- SLO targets: {{slo_targets}}
- Actual performance: {{actual_performance}}
- Error budget consumed: {{error_budget_consumed}}%
- SLA violations: {{sla_violations}}
- Root cause of violations: {{violation_root_causes}}
- Improvement actions: {{improvement_actions}}
```

### 8.4 Executive Summary
```
Generate executive summary for {{product}} operations:
- Key metrics: {{key_metrics}}
- Incidents and resolution: {{incident_summary}}
- Service level achievement: {{sla_achievement}}
- Cost performance: {{cost_performance}}
- Strategic recommendations: {{strategic_recommendations}}
```

---

*Template version 1.0.0 — adapt with product-specific placeholders and values*
