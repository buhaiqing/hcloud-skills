# API & SDK Usage — Huawei Cloud OBS

## Go SDK

```go
import (
    "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

client, err := obs.New(ak, sk, endpoint)
```

Note: OBS uses a separate Go SDK (`huaweicloud-sdk-go-obs`) — NOT the `huaweicloud-sdk-go-v3` monorepo.

## Core API Operations

### Bucket Operations

| Operation | HTTP | Path | Description |
|-----------|------|------|-------------|
| CreateBucket | PUT | `/{bucket}` | Create bucket |
| ListBuckets | GET | `/` | List all buckets |
| DeleteBucket | DELETE | `/{bucket}` | Delete bucket (must be empty) |
| HeadBucket | HEAD | `/{bucket}` | Check bucket exists & accessible |
| GetBucketLocation | GET | `/{bucket}?location` | Get bucket's region |

### Object Operations

| Operation | HTTP | Path | Description |
|-----------|------|------|-------------|
| PutObject | PUT | `/{bucket}/{key}` | Upload object (≤ 5GB) |
| GetObject | GET | `/{bucket}/{key}` | Download object |
| DeleteObject | DELETE | `/{bucket}/{key}` | Delete object (or create delete marker) |
| HeadObject | HEAD | `/{bucket}/{key}` | Get object metadata |
| CopyObject | PUT | `/{bucket}/{key}` | Copy object intra/inter region |
| ListObjects | GET | `/{bucket}` | List objects in bucket |
| ListObjectsVersions | GET | `/{bucket}?versions` | List all versions |

### Multipart Upload

| Operation | HTTP | Path | Description |
|-----------|------|------|-------------|
| InitiateMultipartUpload | POST | `/{bucket}/{key}?uploads` | Start multipart upload |
| UploadPart | PUT | `/{bucket}/{key}?partNumber=N&uploadId=X` | Upload a part |
| CompleteMultipartUpload | POST | `/{bucket}/{key}?uploadId=X` | Complete upload |
| AbortMultipartUpload | DELETE | `/{bucket}/{key}?uploadId=X` | Abort multipart upload |
| ListMultipartUploads | GET | `/{bucket}?uploads` | List in-progress uploads |
| ListParts | GET | `/{bucket}/{key}?uploadId=X` | List parts of upload |

### ACL & Policy

| Operation | HTTP | Path | Description |
|-----------|------|------|-------------|
| SetBucketAcl | PUT | `/{bucket}?acl` | Set bucket ACL |
| GetBucketAcl | GET | `/{bucket}?acl` | Get bucket ACL |
| SetBucketPolicy | PUT | `/{bucket}?policy` | Set bucket policy (JSON) |
| GetBucketPolicy | GET | `/{bucket}?policy` | Get bucket policy |
| SetObjectAcl | PUT | `/{bucket}/{key}?acl` | Set object ACL |

### Configuration

| Operation | HTTP | Path | Description |
|-----------|------|------|-------------|
| SetBucketLifecycle | PUT | `/{bucket}?lifecycle` | Set lifecycle rules |
| GetBucketLifecycle | GET | `/{bucket}?lifecycle` | Get lifecycle rules |
| SetBucketVersioning | PUT | `/{bucket}?versioning` | Enable/suspend versioning |
| GetBucketVersioning | GET | `/{bucket}?versioning` | Get versioning status |
| SetBucketCors | PUT | `/{bucket}?cors` | Set CORS rules |
| GetBucketCors | GET | `/{bucket}?cors` | Get CORS rules |
| SetBucketWebsite | PUT | `/{bucket}?website` | Configure static website |
| GetBucketWebsite | GET | `/{bucket}?website` | Get website config |

## Pagination

ListObjects supports marker-based pagination:

```
GET /{bucket}?prefix=logs/&delimiter=/&max-keys=1000&marker=xxx
```

| Parameter | Type | Default | Max |
|-----------|------|---------|-----|
| prefix | string | "" | — (filters by prefix) |
| delimiter | string | "" | — (groups by "folder") |
| max-keys | int | 1000 | 1000 |
| marker | string | "" | — (start after this key) |

Response includes `IsTruncated` (bool) and `NextMarker` for subsequent pages.

## Multipart Upload Flow

```
1. InitiateMultipartUpload → returns UploadId
2. UploadPart concurrently (partNumber 1-N, each ≥ 5MB except last)
3. CompleteMultipartUpload with UploadId + part list (ETag + PartNumber)
4. On failure: AbortMultipartUpload or wait for lifecycle cleanup
```

## Presigned URL

Generate temporary access URL (valid for configurable duration, max 7 days):

```go
input := &obs.CreateSignedUrlInput{
    Method:  obs.HttpMethodGet,
    Bucket:  "my-bucket",
    Key:     "path/to/object",
    Expires: 3600, // 1 hour
}
output, _ := client.CreateSignedUrl(input)
fmt.Println(output.SignedUrl)
```

## Error Response

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Error>
  <Code>NoSuchKey</Code>
  <Message>The specified key does not exist.</Message>
  <RequestId>abc-123-def</RequestId>
  <Resource>/my-bucket/path/object</Resource>
  <HostId>obs.cn-north-4.myhuaweicloud.com</HostId>
</Error>
```
