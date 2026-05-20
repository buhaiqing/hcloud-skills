# CES Troubleshooting Guide — Huawei Cloud Cloud Eye Service

## Error Code Taxonomy

| Error Code | HTTP Status | Name | Description | Recovery Action |
|------------|-------------|------|-------------|-----------------|
| CES.0003 | 400 | InvalidParameter | Request parameter validation failed | Fix parameter format and retry |
| CES.0006 | 429 | RequestLimitExceeded | API rate limit exceeded | Wait and retry with backoff |
| CES.0010 | 400 | InvalidRequestData | Request body format/type mismatch | Check JSON schema against API docs |
| CES.0012 | 409 | ResourceAlreadyExists | Alarm rule name already exists | Use different name or reuse existing |
| CES.0013 | 404 | ResourceNotFound | Metric or alarm resource not found | Verify namespace, metric_name, resource_id |
| CES.0016 | 403 | Forbidden | Project or resource unauthorized | Check IAM permissions, project_id |
| CES.0020 | 403 | QuotaExceeded | Alarm rule quota limit reached | Delete unused alarms or request quota increase |
| CES.0029 | 500 | InternalError | Internal server error | Retry with exponential backoff; HALT after 3 attempts |
| Auth.0001 | 401 | AuthenticationFailed | AK/SK authentication failed | Verify credentials; NEVER log secret key |
| Auth.0003 | 403 | AccessDenied | Insufficient permissions | Assign CES permissions via IAM |

## Ordered Diagnostic Steps

### Step 1: Authentication Issues

```
Symptom: 401 Unauthorized or 403 Forbidden
Check:   AK/SK validity, region consistency, project_id
Action:  Verify env vars exist; check IAM role has CES Administrator or CES Viewer
```

### Step 2: Parameter Validation

```
Symptom: 400 InvalidParameter or InvalidRequestData
Check:   Namespace format (SYS.xxx), metric_name validity, threshold is numeric
Action:  Cross-reference against official API docs for exact field requirements
```

### Step 3: Resource Not Found

```
Symptom: 404 ResourceNotFound
Check:   Resource exists in target region, namespace matches resource type
Action:  First verify resource via respective product skill (e.g., ECS for instances)
```

### Step 4: Alarm Not Triggering

```
Symptom: Resource metrics exceed threshold but alarm stays 'ok'
Check:
  1. Alarm is enabled (alarm_enabled = true)
  2. Evaluation periods met consistently
  3. Metric namespace and name are correct for the resource
  4. Dimension key/value match actual resource identifiers
  5. Period matches metric data collection interval
Action:
  - Verify metric data exists via ShowMetricData
  - Check alarm configuration matches actual resource
```

### Step 5: Notification Not Received

```
Symptom: Alarm state = 'alarm' but no notification received
Check:
  1. SMN topic exists and is accessible
  2. SMN subscriptions are active for the topic
  3. Alarm rule has alarm_actions configured with valid topic URN
Action:
  - Verify SMN topic URN format: urn:smn:{region}:{project_id}:{topic_name}
  - Check SMN subscription confirmations
```

### Step 6: Rate Limiting

```
Symptom: 429 RequestLimitExceeded
Check:   Request frequency exceeds 200 req/min per project
Action:
  - Implement exponential backoff: 1s → 2s → 4s → 8s
  - Use BatchQueryMetricData instead of individual queries
  - Cache metric data when appropriate
```

### Step 7: Empty Metric Data

```
Symptom: datapoints array is empty or has few entries
Check:
  1. Resource exists and is in RUNNING state
  2. Time range contains actual metric data
  3. Metric name is valid for the namespace
  4. For host metrics, agent is installed and running
Action:
  - Query metrics for last 1 hour to verify data availability
  - For SYS.* metrics, data is auto-collected; verify service is active
  - For AGT.* metrics, verify CES agent is installed and configured
```

## Multi-Round Diagnosis Flow

```
Alarm not working?
  ├── Is alarm enabled? ── No → Enable it
  │                           └── Done
  │                        ── Yes ↓
  ├── Does metric data exist? ── No → Check resource state / agent
  │                                  └── Fix resource
  │                               ── Yes ↓
  ├── Does threshold match data? ── No → Adjust threshold
  │                                     └── Done
  │                                  ── Yes ↓
  ├── Is notification configured? ── No → Configure SMN topic
  │                                      └── Done
  │                                   ── Yes ↓
  └── Is SMN topic working? ── No → Check SMN subscriptions
                                   └── Fix SMN
```
