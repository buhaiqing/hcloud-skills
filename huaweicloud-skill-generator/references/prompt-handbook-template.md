# Prompt Handbook — Template

> **Purpose**: Categorized prompts for AI diagnosis and investigation.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Prompt Categories

| Category | Count | Description |
|----------|-------|-------------|
| Diagnosis | ≥5 | Root cause analysis prompts |
| Investigation | ≥5 | Evidence gathering prompts |
| Remediation | ≥3 | Fix action prompts |
| 巡检 | ≥3 | Health check prompts |
| 报告 | ≥2 | Summary report prompts |

## 2. Diagnosis Prompts

### 2.1 CPU High Diagnosis

```
You are analyzing a CPU high alert on {{resource_id}}.
Metric: {{metric_name}} = {{value}}%, threshold = {{threshold}}%
Duration: {{duration}}

Investigate:
1. Which process is consuming CPU?
2. Is this normal behavior or anomaly?
3. Any recent changes that could explain this?

Provide:
- Root cause assessment (High/Medium/Low confidence)
- Recommended actions
```

### 2.2 Memory Leak Detection

```
You are analyzing potential memory leak on {{resource_id}}.
Memory trend: {{trend}} over past {{hours}} hours
Current: {{current_value}}%, Peak: {{peak_value}}%

Investigate:
1. Is memory monotonically increasing?
2. Which process is leaking?
3. Expected leak rate?

Provide:
- Memory leak probability (High/Medium/Low)
- Estimated exhaustion time
- Recommended actions
```

### 2.3 Disk Full Warning

```
You are analyzing disk full warning on {{resource_id}}.
Disk usage: {{usage}}%
Growth rate: {{rate}}%/hour
Time to full: {{hours}} hours

Investigate:
1. Which directories are consuming space?
2. Any large files that can be cleaned?
3. Is this normal growth or abnormal?

Provide:
- Root cause assessment
- Cleanup recommendations
- 扩容建议
```

### 2.4 Connection Exhaustion

```
You are analyzing connection exhaustion on {{resource_id}}.
Connection usage: {{usage}}%
Current connections: {{current}}/{{max}}
Trend: {{trend}}

Investigate:
1. Which application is holding connections?
2. Any connection leaks (not properly closed)?
3. Normal traffic spike or abnormal?

Provide:
- Root cause assessment
- Quick fix recommendations
- Long-term optimization
```

### 2.5 Service Unavailable

```
You are analyzing service unavailable on {{resource_id}}.
Error rate: {{error_rate}}%
HTTP 5xx count: {{count}}
Time window: {{duration}}

Investigate:
1. Which backend service is failing?
2. Any recent deployments or config changes?
3. Dependencies status (RDS, Redis, etc.)?

Provide:
- Root cause assessment
- Impact scope
- Recovery recommendations
```

## 3. Investigation Prompts

### 3.1 Log Investigation

```
Search LTS logs for pattern "{{pattern}}" on {{resource_id}}.
Time range: {{start_time}} to {{end_time}}

Extract:
- Error frequency
- Error distribution
- Related events

Provide findings in structured format.
```

### 3.2 Change Correlation Investigation

```
Find CTS changes on {{resource_id}} between {{start_time}} and {{end_time}}.

Correlate with alarm at {{alarm_time}}.

Provide:
- List of changes
- Correlation score
- Most likely cause
```

### 3.3 Metrics Comparison

```
Compare current metrics with baseline on {{resource_id}}:
- CPU: {{current_cpu}}% vs baseline {{baseline_cpu}}%
- Memory: {{current_mem}}% vs baseline {{baseline_mem}}%
- Latency: {{current_lat}}ms vs baseline {{baseline_lat}}ms

Identify anomalies and rank by severity.
```

### 3.4 Dependency Health Check

```
Check health of dependencies for {{resource_id}}:
- Database: {{db_status}}
- Cache: {{cache_status}}
- Message Queue: {{mq_status}}
- Object Storage: {{obs_status}}

Identify which dependency might be causing issues.
```

### 3.5 Performance Bottleneck

```
Analyze performance bottleneck on {{resource_id}}.

Current metrics:
- CPU: {{cpu}}%
- Memory: {{mem}}%
- IO: {{io}}%
- Network: {{network}}%

Identify the bottleneck resource and recommend optimization.
```

## 4. Remediation Prompts

### 4.1 Scale Up Recommendation

```
Based on current metrics:
- CPU: {{cpu}}%
- Memory: {{memory}}%
- Trend: {{trend}}

Recommend:
1. Scale up or scale out?
2. Expected new size?
3. Estimated cost impact?

Provide recommendation with confidence level.
```

### 4.2 Cache Clear Recommendation

```
Cache analysis on {{resource_id}}:
- Hit rate: {{hit_rate}}%
- Memory used: {{used}}MB/{{total}}MB
- Eviction rate: {{eviction}}/s

Recommend:
1. Clear cache now?
2. Adjust cache size?
3. Change eviction policy?

Provide recommendation with risk assessment.
```

### 4.3 Configuration Optimization

```
Configuration analysis on {{resource_id}}:
- Current settings: {{settings}}
- Recommended settings: {{recommended}}
- Gap: {{gap}}

Recommend configuration changes with expected impact.
```

## 5. 巡检 Prompts

### 5.1 Daily Health Check

```
Perform daily health check for {{resource_id}}.

Check:
1. CPU/Memory/Disk metrics
2. Recent alarms
3. Pending changes
4. SLO compliance

Provide health score and findings.
```

### 5.2 Weekly Trend Report

```
Generate weekly trend report for {{resource_id}}.

Analyze:
1. Metric trends (CPU, Memory, Network)
2. Alarm frequency and severity
3. Change frequency
4. Performance comparison with last week

Provide summary and recommendations.
```

### 5.3 Monthly Capacity Review

```
Perform monthly capacity review for {{resource_id}}.

Review:
1. Resource utilization trends
2. Capacity headroom
3. Growth rate
4. Upcoming changes that may impact capacity

Provide capacity forecast and recommendations.
```

## 6. 报告 Prompts

### 6.1 Incident Summary

```
Generate incident summary for:
- Incident ID: {{incident_id}}
- Duration: {{start}} to {{end}}
- Impact: {{impact}}

Include:
- Timeline
- Root cause
- Actions taken
- Lessons learned
```

### 6.2 Health Report

```
Generate health report for {{resource_id}}.

Period: {{start}} to {{end}}

Sections:
1. Executive Summary
2. Availability Metrics
3. Performance Metrics
4. Incidents and Changes
5. Recommendations

Format as structured markdown.
```

## 7. Compliance Checklist

- [ ] ≥20 categorized prompts
- [ ] Each prompt includes context variables
- [ ] Each prompt specifies output format
- [ ] Diagnosis prompts include confidence level
