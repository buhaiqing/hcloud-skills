# CSS API & SDK Usage

## SDK Package

```go
import (
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/css/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/css/v2/model"
)
```

## Client Initialization

```go
import (
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
)

func createCSSClient(ak, sk, region string) *css.CssClient {
    cfg := config.DefaultHttpConfig()
    return css.CssClientBuilder().
        WithEndpoint(fmt.Sprintf("css.%s.myhuaweicloud.com", region)).
        WithCredential(basic.NewCredentialsBuilder().
            WithAk(ak).WithSk(sk).Build()).
        WithHttpConfig(cfg).Build()
}
```

## Operation Map

### Cluster Management

| Operation | SDK Method | Request Type | Response Type |
|-----------|------------|--------------|---------------|
| List Clusters | `ListClusters` | `ListClustersRequest` | `ListClustersResponse` |
| Show Cluster | `ShowClusterDetail` | `ShowClusterDetailRequest` | `ShowClusterDetailResponse` |
| Create Cluster | `CreateCluster` | `CreateClusterRequest` | `CreateClusterResponse` |
| Delete Cluster | `DeleteCluster` | `DeleteClusterRequest` | `DeleteClusterResponse` |
| Restart Cluster | `RestartCluster` | `RestartClusterRequest` | `RestartClusterResponse` |
| Extend Cluster | `ExtendCluster` | `ExtendClusterRequest` | `ExtendClusterResponse` |
| Scale Out | `ScaleOut` | `ScaleOutRequest` | `ScaleOutResponse` |
| Reset Password | `ResetPassword` | `ResetPasswordRequest` | `ResetPasswordResponse` |

### Snapshot Management

| Operation | SDK Method | Request Type | Response Type |
|-----------|------------|--------------|---------------|
| List Snapshots | `ListSnapshots` | `ListSnapshotsRequest` | `ListSnapshotsResponse` |
| Create Snapshot | `CreateSnapshot` | `CreateSnapshotRequest` | `CreateSnapshotResponse` |
| Delete Snapshot | `DeleteSnapshot` | `DeleteSnapshotRequest` | `DeleteSnapshotResponse` |
| Restore Snapshot | `RestoreSnapshot` | `RestoreSnapshotRequest` | `RestoreSnapshotResponse` |
| Show Snapshot Policy | `ShowSnapshotPolicy` | `ShowSnapshotPolicyRequest` | `ShowSnapshotPolicyResponse` |
| Set Snapshot Policy | `SetSnapshotPolicy` | `SetSnapshotPolicyRequest` | `SetSnapshotPolicyResponse` |

### Dictionary Management

| Operation | SDK Method | Request Type | Response Type |
|-----------|------------|--------------|---------------|
| List IK Dicts | `ListIkDicts` | `ListIkDictsRequest` | `ListIkDictsResponse` |
| Add IK Dict | `AddIkDict` | `AddIkDictRequest` | `AddIkDictResponse` |
| Update IK Dict | `UpdateIkDict` | `UpdateIkDictRequest` | `UpdateIkDictResponse` |
| Delete IK Dict | `DeleteIkDict` | `DeleteIkDictRequest` | `DeleteIkDictResponse` |

## Request/Response Examples

### Create Cluster

```go
request := &model.CreateClusterRequest{
    Body: &model.CreateClusterReq{
        Name: "prod-es-cluster",
        Datastore: &model.CreateClusterReqDatastore{
            Type:    "elasticsearch",
            Version: "7.10.2",
        },
        Instance: &model.CreateClusterReqInstance{
            FlavorRef: "ess.spec-4u8g",
            Nics: &model.CreateClusterReqInstanceNics{
                VpcId:           "vpc-xxx",
                NetId:           "subnet-xxx",
                SecurityGroupId: "sg-xxx",
            },
            Volume: &model.CreateClusterReqInstanceVolume{
                Type: "ULTRAHIGH",
                Size: 100,
            },
            AvailabilityZone: "cn-north-4a",
        },
        InstanceNum: 3,
        HttpsEnable: true,
        DiskEncryptionEnabled: true,
        DiskEncryptionKey: "kms-key-xxx",
        BackupStrategy: &model.CreateClusterReqBackupStrategy{
            Period: "00:00 GMT+08:00",
            Prefix: "snapshot",
            Keepday: 7,
            Bucket: "css-snapshots",
            BasePath: "css-backup",
            Agency: "css_obs_agency",
        },
    },
}

response, err := client.CreateCluster(request)
if err != nil {
    // Handle error
}
// response.Cluster.Id contains created cluster ID
```

### List Clusters

```go
request := &model.ListClustersRequest{
    Start: int32(1),
    Limit: int32(10),
}

response, err := client.ListClusters(request)
if err != nil {
    // Handle error
}
// response.Clusters contains cluster list
```

### Create Snapshot

```go
request := &model.CreateSnapshotRequest{
    ClusterId: "cluster-xxx",
    Body: &model.CreateSnapshotReq{
        Name:        "manual-snapshot-20260527",
        Description: "Manual backup before upgrade",
        Indices:     "*,-.kibana*",
        BackupObsPath: "obs://bucket-name/snapshots/",
    },
}

response, err := client.CreateSnapshot(request)
```

## Pagination

```go
var allClusters []model.ClusterList
var start int32 = 1
const limit int32 = 100

for {
    request := &model.ListClustersRequest{
        Start: start,
        Limit: limit,
    }
    response, err := client.ListClusters(request)
    if err != nil {
        break
    }
    allClusters = append(allClusters, *response.Clusters...)
    if len(*response.Clusters) < int(limit) {
        break
    }
    start += limit
}
```

## Error Handling

```go
response, err := client.CreateCluster(request)
if err != nil {
    if sdkError, ok := err.(*sdkerr.ServiceError); ok {
        switch sdkError.Code {
        case "CSS.0001":
            // Invalid parameter
        case "CSS.0002":
            // Quota exceeded
        case "CSS.0003":
            // Insufficient balance
        default:
            // Unknown error
        }
    }
}
```

## Async Operations Polling

```go
func waitForClusterAvailable(client *css.CssClient, clusterId string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        request := &model.ShowClusterDetailRequest{
            ClusterId: clusterId,
        }
        response, err := client.ShowClusterDetail(request)
        if err != nil {
            return err
        }
        if response.Status == "AVAILABLE" {
            return nil
        }
        if response.Status == "FAILED" {
            return fmt.Errorf("cluster creation failed")
        }
        time.Sleep(60 * time.Second)
    }
    return fmt.Errorf("timeout waiting for cluster")
}
```
