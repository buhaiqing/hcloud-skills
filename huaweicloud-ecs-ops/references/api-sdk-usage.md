# API & SDK — Huawei Cloud ECS

## OpenAPI

- Spec: `https://support.huaweicloud.com/api-ecs/ecs_01_0043.html`
- Base endpoint: `ecs.{region}.myhuaweicloud.com`
- SDK: `github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2`

## SDK Operations Map

| Goal | API Operation | SDK Method | Async |
|------|--------------|-----------|-------|
| Create Single Server | `POST /v2/{project_id}/cloudservers` | `CreateServer` | Yes (`job_id`) |
| Create Multiple Servers | `POST /v2/{project_id}/cloudservers` (count>1) | `CreateServers` | Yes (`job_id`) |
| Describe Server | `GET /v2.1/{project_id}/servers/{server_id}` | `ShowServerDetail` | No |
| List Servers | `GET /v1/{project_id}/cloudservers/detail` | `ListServersDetail` | No |
| Start Server | `POST /v1/{project_id}/cloudservers/{server_id}/os-start` | `BatchStartServers` | Yes |
| Stop Server | `POST /v1/{project_id}/cloudservers/{server_id}/os-stop` | `BatchStopServers` | Yes |
| Delete Server | `POST /v1/{project_id}/cloudservers/delete` | `BatchDeleteServers` | Yes |
| Resize Server | `POST /v2.1/{project_id}/cloudservers/{server_id}/resize` | `ResizeServer` | Yes |
| List Flavors | `GET /v1/{project_id}/cloudservers/flavors` | `ListFlavors` | No |
| List Images | `GET /v2/{project_id}/images/detail` | `ListImages` (IMS) | No |
| List Volumes | `GET /v2/{project_id}/volumes/detail` | `ListVolumes` (EVS) | No |
| Attach Volume | `POST /v2/{project_id}/cloudserver/{server_id}/attach` | `AttachVolume` (EVS) | Yes |
| Detach Volume | `POST /v2/{project_id}/cloudserver/{server_id}/detach` | `DetachVolume` (EVS) | Yes |
| Show Job Status | `GET /v1/{project_id}/jobs/{job_id}` | `ShowJobStatus` | No |
| Show CloudCell Detail | `GET /v1/{project_id}/cloudservers/{server_id}/cloud_server_detail` | `ShowServerCloudCellDetail` | No |
| Execute CloudCell Command | `POST /v1/{project_id}/cloudservers/{server_id}/cloud_cell/execute_command` | `ExecuteCloudCellCommand` | Yes |
| Show Quota | `GET /v2/{project_id}/cloudservers/quotas` | `ShowQuota` | No |

## Async Job Pattern

Most ECS modification operations return a `job_id`. Poll until terminal state:

```go
func pollJob(client *ecs.EcsClient, jobID string, maxWait int) error {
    for i := 0; i < maxWait/5; i++ {
        resp, err := client.ShowJobStatus(&model.ShowJobStatusRequest{
            JobId: jobID,
        })
        if err != nil { return err }
        switch *resp.Status {
        case "SUCCESS": return nil
        case "FAIL":
            return fmt.Errorf("job failed: %v", *resp.FailReason)
        }
        time.Sleep(5 * time.Second)
    }
    return fmt.Errorf("job timeout after %ds", maxWait)
}
```

## Pagination

All list APIs support:
- `limit`: max items per page (max 1000)
- `offset`: start index (0-based)
- `marker`: pagination token for cursor-based pagination

## Key Request Fields (CreateServer)

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `name` | string | Yes | 1-64 chars, unique per account |
| `flavorRef` | string | Yes | Flavor UUID |
| `imageRef` | string | Yes | Image UUID (or `metadata.image_name`) |
| `vpcid` | string | Yes | VPC UUID |
| `nics[].subnetId` | string | Yes | Subnet UUID |
| `nics[].securityGroups[].id` | string | Recommended | Security group UUID |
| `rootVolume.volumetype` | string | Yes | `SSD`, `SAS`, `GPSSD`, `ESSD` |
| `rootVolume.size` | int32 | Recommended | System disk size (GB) |
| `availabilityZone` | string | Yes | e.g., `cn-north-4a` |
| `count` | int32 | Yes | Number of instances (1-500) |
| `clientToken` | string | No | Idempotency token |
| `userData` / `user_data` | string | No | Cloud-init script (base64) |
