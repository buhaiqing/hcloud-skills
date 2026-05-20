# CES API & SDK Usage — Huawei Cloud Cloud Eye Service

## API Base Information

- **Base URL**: `https://ces.{region_id}.myhuaweicloud.com`
- **API Version**: V1.0
- **Protocol**: HTTPS
- **Content-Type**: application/json
- **Authentication**: IAM AK/SK signature (v4)

## Operation Endpoint Map

| Operation | Method | Endpoint | Description |
|-----------|--------|----------|-------------|
| CreateAlarmRule | POST | `/V1.0/{project_id}/alarms` | Create or update alarm rule |
| ListAlarmRules | GET | `/V1.0/{project_id}/alarms` | List alarm rules |
| DescribeAlarmRule | GET | `/V1.0/{project_id}/alarms/{id}` | Get specific alarm rule |
| DeleteAlarmRule | DELETE | `/V1.0/{project_id}/alarms/{id}` | Delete alarm rule |
| AlarmAction | PUT | `/V1.0/{project_id}/alarms/{id}/action` | Enable/disable alarm |
| BatchListMetrics | POST | `/V1.0/{project_id}/batch-list-metrics` | Batch list metrics |
| ListMetrics | GET | `/V1.0/{project_id}/metrics` | List metric details |
| ShowMetricData | GET | `/V1.0/{project_id}/metric-data` | Query metric data |
| BatchQueryMetricData | POST | `/V1.0/{project_id}/metric-data/batch-query` | Batch query metric data |
| CreateDashboard | POST | `/V1.0/{project_id}/dashboards` | Create dashboard |
| ListDashboards | GET | `/V1.0/{project_id}/dashboards` | List dashboards |
| ShowDashboard | GET | `/V1.0/{project_id}/dashboards/{id}` | Get dashboard details |
| DeleteDashboard | DELETE | `/V1.0/{project_id}/dashboards/{id}` | Delete dashboard |
| ListEvents | GET | `/V1.0/{project_id}/events` | List cloud service events |
| AddEventData | POST | `/V1.0/{project_id}/event-data` | Add custom event data |
| ShowQuotas | GET | `/V1.0/{project_id}/quotas` | Query CES quotas |

## Go SDK Setup

```go
package main

import (
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ces/v1"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ces/v1/model"
)

func NewCESClient(region, ak, sk string) *v1.CesClient {
    return v1.CesClientBuilder().
        WithEndpoint(fmt.Sprintf("ces.%s.myhuaweicloud.com", region)).
        WithCredential(basic.NewCredentialsBuilder().WithAk(ak).WithSk(sk).Build()).
        WithHttpConfig(config.DefaultHttpConfig()).Build()
}
```

## Key Request/Response Patterns

### CreateAlarmRule Response

```json
{
    "alarm_id": "al1234567890abcdef",
    "alarm_name": "ecs-cpu-alarm",
    "alarm_enabled": true,
    "alarm_level": 2,
    "metric_name": "cpu_util",
    "metric_namespace": "SYS.ECS",
    "comparison_operator": ">="
}
```

Response path: `$.alarm_id`

### ShowMetricData Response

```json
{
    "datapoints": [
        {
            "average": 65.2,
            "max": 89.1,
            "min": 42.3,
            "sum": 652.0,
            "variance": 156.8,
            "timestamp": 1600000000000
        }
    ],
    "metric_name": "cpu_util",
    "namespace": "SYS.ECS"
}
```

Response path: `$.datapoints`

### ListAlarmRules Response

```json
{
    "metric_alarms": [
        {
            "alarm_id": "al1234567890abcdef",
            "alarm_name": "ecs-cpu-alarm",
            "alarm_enabled": true,
            "alarm_level": 2,
            "metric_name": "cpu_util",
            "metric_namespace": "SYS.ECS",
            "comparison_operator": ">=",
            "threshold": 80,
            "evaluation_periods": 3,
            "period": 300,
            "alarm_actions": [
                "urn:smn:region:project_id:topic_name"
            ],
            "alarm_resources": ["instance_id_1"]
        }
    ],
    "meta_data": {
        "total": 1
    }
}
```

## Pagination

- **List operations**: Return `meta_data.total` for count, use `limit` (max 100) and `marker` for pagination.
- **Metric data**: Up to 1000 datapoints per request.

## Batch Query

For querying metrics across multiple resources, use `BatchQueryMetricData` POST endpoint:

```json
{
    "metrics": [
        {
            "namespace": "SYS.ECS",
            "metric_name": "cpu_util",
            "dimensions": [{"name": "instance_id", "value": "instance-1"}]
        },
        {
            "namespace": "SYS.ECS",
            "metric_name": "cpu_util",
            "dimensions": [{"name": "instance_id", "value": "instance-2"}]
        }
    ],
    "from": 1600000000000,
    "to": 1600100000000,
    "period": "1",
    "filter": "average"
}
```
