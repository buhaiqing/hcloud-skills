# Huawei Cloud CTS API & SDK Usage

## Service API Overview

Huawei Cloud CTS exposes audit trail and event query APIs. The core service package is `huaweicloud-sdk-go-v3/services/cts/v3`.

### Common operations

- `CreateTrail` / `DeleteTrail`
- `ListTrails` / `ShowTrail`
- `UpdateTrail`
- `QueryEvents`
- `ShowEvent`
- `ListDeliveryConfigs` (destination discovery)

## Go SDK Operation Map

### Create Trail

```go
request := &model.CreateTrailRequest{
    Body: &model.CreateTrailRequestBody{
        Name:           apiName,
        DeliveryConfig: deliveryConfig,
        RetentionDays:  func() *int32 { v := int32(365); return &v }(),
    },
}
response, err := client.CreateTrail(context.TODO(), request)
```

### List Trails

```go
request := &model.ListTrailsRequest{
    Limit: func() *int32 { v := int32(50); return &v }(),
}
response, err := client.ListTrails(context.TODO(), request)
```

### Show Trail

```go
request := &model.ShowTrailRequest{
    TrailId: trailID,
}
response, err := client.ShowTrail(context.TODO(), request)
```

### Query Events

```go
request := &model.QueryEventsRequest{
    TrailId:   trailID,
    StartTime: &startTime,
    EndTime:   &endTime,
    Filter:    &filterExpression,
    Limit:     func() *int32 { v := int32(100); return &v }(),
}
response, err := client.QueryEvents(context.TODO(), request)
```

## Response Parsing

- `CreateTrailResponse` contains `TrailId`, `Name`, `Status`, `DeliveryConfig`.
- `ListTrailsResponse` contains `Trails` and pagination metadata.
- `ShowTrailResponse` contains the full trail configuration.
- `QueryEventsResponse` contains `Events` and result count.

## Error Handling

- `Cts.0401` — Duplicate trail name or invalid request.
- `Cts.0402` — Delivery destination invalid.
- `Cts.0403` — Region not supported.
- `Cts.0404` — Destination unreachable.
- `Cts.0405` — Quota exceeded.
- `Cts.0406` — Account balance insufficient.

## Import Example

```go
import (
    "context"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    cts "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cts/v3"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cts/v3/model"
    ctsregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cts/v3/region"
)
```

## Best Practices

- Use `ShowTrail` after `CreateTrail` or `UpdateTrail` to verify final state.
- Keep trail names unique and descriptive.
- Prefer `OBS` delivery for long-term retention and archival.
- Validate query time windows and filter expressions before execution.
