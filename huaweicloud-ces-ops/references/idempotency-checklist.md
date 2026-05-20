# CES Idempotency Checklist — Huawei Cloud Cloud Eye Service

## Idempotent Operations

| Operation | Idempotent? | Mechanism | Notes |
|-----------|-------------|-----------|-------|
| CreateAlarmRule | ❌ | Alarm name must be unique | Duplicate name returns CES.0012; check existence first |
| ListAlarms | ✅ | GET operation | Safe to retry |
| DescribeAlarm | ✅ | GET operation | Safe to retry |
| EnableAlarm | ✅ | Toggle to enabled state | Repeated calls are safe |
| DisableAlarm | ✅ | Toggle to disabled state | Repeated calls are safe |
| DeleteAlarm | ✅ | DELETE operation | Second delete returns 404 Not Found |
| QueryMetricData | ✅ | GET operation | Safe to retry |
| CreateDashboard | ❌ | Dashboard title must be unique | Check existence first |
| DeleteDashboard | ✅ | DELETE operation | Safe to retry |

## Idempotent Alarm Creation Pattern

```
1. GET /V1.0/{project_id}/alarms?alarm_name={name}
2. If alarm exists with same configuration → Return existing alarm_id (SKIP)
3. If alarm exists with different configuration → Update via PUT or recreate
4. If alarm does not exist → POST /V1.0/{project_id}/alarms (CREATE)
```

## Idempotent Dashboard Creation Pattern

```
1. GET /V1.0/{project_id}/dashboards
2. If dashboard with matching title exists → Return existing dashboard_id (SKIP)
3. If no match → POST /V1.0/{project_id}/dashboards (CREATE)
```

## Retry Safety

All GET and DELETE operations are inherently idempotent and safe to retry.
POST operations require pre-creation checks to avoid duplicates.
PUT operations are replace-based and inherently idempotent.
