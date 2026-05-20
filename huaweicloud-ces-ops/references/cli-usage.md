# CES CLI Usage — Huawei Cloud Cloud Eye Service

## CLI Command Map

| Operation | CLI Command | Description |
|-----------|-------------|-------------|
| List alarms | `hcloud ces list-alarms` | List alarm rules with optional filters |
| Describe alarm | `hcloud ces describe-alarm` | Get specific alarm rule details |
| Create alarm | `hcloud ces create-alarm-rule` | Create new alarm rule |
| Enable alarm | `hcloud ces enable-alarm` | Enable an alarm rule |
| Disable alarm | `hcloud ces disable-alarm` | Disable an alarm rule |
| Delete alarm | `hcloud ces delete-alarm` | Delete an alarm rule |
| Query metric data | `hcloud ces query-metric-data` | Query metric data for a resource |
| List dashboards | `hcloud ces list-dashboards` | List monitoring dashboards |
| Describe dashboard | `hcloud ces describe-dashboard` | Get specific dashboard details |
| Create dashboard | `hcloud ces create-dashboard` | Create a monitoring dashboard |
| Delete dashboard | `hcloud ces delete-dashboard` | Delete a dashboard |
| List events | `hcloud ces list-events` | List cloud service events |
| Show quotas | `hcloud ces show-quotas` | Query CES resource quotas |

## CLI Coverage Gap Table

| Operation | CLI Support | SDK Only? | Notes |
|-----------|-------------|-----------|-------|
| CreateAlarmRule | ✅ | No | Full support |
| ListAlarms | ✅ | No | Full support |
| DescribeAlarm | ✅ | No | Full support |
| Enable/DisableAlarm | ✅ | No | Full support |
| DeleteAlarm | ✅ | No | Full support |
| QueryMetricData | ✅ | No | Full support |
| BatchQueryMetrics | ❌ | Yes | Use Go SDK for batch queries |
| CreateDashboard | ✅ | No | Basic creation |
| ListDashboards | ✅ | No | Full support |
| ShowDashboard | ✅ | No | Full support |
| DeleteDashboard | ✅ | No | Full support |
| ListEvents | ✅ | No | Full support |
| AddEventData | ❌ | Yes | Custom events via SDK only |
| ShowQuotas | ✅ | No | Full support |
| BatchListMetrics | ❌ | Yes | Batch metric listing via SDK only |

## Invocation Patterns

### Common Parameters

All CES CLI commands require:
```bash
--region "{{user.region}}"      # Target region
```

### List Alarms with Filters

```bash
hcloud ces list-alarms \
  --region "cn-north-4" \
  --alarm-name "cpu" \
  --alarm-enabled "true"
```

### Create Alarm — JSON Output

```bash
hcloud ces create-alarm-rule \
  --region "cn-north-4" \
  --alarm-name "prod-cpu-alarm" \
  --alarm-enabled true \
  --alarm-action-name "urn:smn:cn-north-4:project123:ces-alerts" \
  --alarm-resources "i-abc123" \
  --metric-namespace "SYS.ECS" \
  --metric-name "cpu_util" \
  --metric-dimension.0.name "instance_id" \
  --metric-dimension.0.value "i-abc123" \
  --comparison-operator "GE" \
  --threshold "80" \
  --evaluation-periods "3" \
  --period "300" \
  --alarm-level "2"
```

Parse alarm_id: `response.alarm_id`

### Batch Query Metrics via SDK (CLI not supported)

When batch querying is needed and CLI doesn't support it, use the JIT Go SDK fallback documented in the SDK usage reference.

## JSON Path Mappings

| CLI Output Field | JSON Path | Description |
|-----------------|-----------|-------------|
| alarm_id | `$.alarm_id` | Rule identifier |
| alarm_name | `$.alarm_name` | Rule display name |
| alarm_enabled | `$.alarm_enabled` | Whether rule is active |
| datapoints | `$.datapoints` | Array of metric data points |
| id (dashboard) | `$.id` | Dashboard identifier |
| total | `$.meta_data.total` | Total count for paginated results |
