# KMS API & SDK Usage — Huawei Cloud Key Management Service

## Go SDK Import

```go
import (
    kms "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/kms/v2"
    kms_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/kms/v2/model"
)
```

## Endpoints

| Region | Endpoint |
|---|---|
| cn-north-4 | `kms.cn-north-4.myhuaweicloud.com` |
| cn-east-3 | `kms.cn-east-3.myhuaweicloud.com` |
| ap-southeast-1 | `kms.ap-southeast-1.myhuaweicloud.com` |

> **TE-1:** Construct endpoint from `{{env.HW_REGION_ID}}` — never hardcode.

## JSON Paths (Top-of-File Declaration)

```text
.output.key_id    = key_metadata.key_id
.output.key_arn   = key_metadata.key_arn
.output.key_state = key_metadata.key_state   # ENABLED / DISABLED / PENDING_DELETION
.output.grant_id  = grant.grant_id
.output.key_type  = key_metadata.key_type   # SYMMETRIC_DEFAULT / ASYMMETRIC_...
```

## Common Operations (Go SDK)

### List Keys

```go
//go:build ignore
req := &kms_model.ListKeysRequest{
    Limit: ptr.Int32(100),
}
resp, err := client.ListKeys(req)
// resp.Keys[].KeyId
```

### Create Key

```go
//go:build ignore
req := &kms_model.CreateKeyRequest{
    Body: &kms_model.CreateKeyRequestBody{
        Alias:           ptr.String("my-alias"),
        KeyUsage:        ptr.String("ENCRYPT_DECRYPT"),
        KeyType:         ptr.String("SYMMETRIC_DEFAULT"),
        RotationEnabled: ptr.Bool(false),
    },
}
resp, err := client.CreateKey(req)
// resp.KeyMetadata.KeyId / KeyArn / KeyState
```

### Describe Key

```go
//go:build ignore
req := &kms_model.ShowKeyRequest{KeyId: keyID}
resp, err := client.ShowKey(req)
// resp.KeyMetadata.* all fields
```

### Enable / Disable Key

```go
//go:build ignore
enableReq := &kms_model.EnableKeyRequest{KeyId: keyID}
_, err := client.EnableKey(enableReq)

disableReq := &kms_model.DisableKeyRequest{KeyId: keyID}
_, err = client.DisableKey(disableReq)
```

### Schedule Deletion

```go
//go:build ignore
req := &kms_model.ScheduleKeyDeletionRequest{
    KeyId:             keyID,
    PendingWindowDays: ptr.Int32(30), // 7-1096
}
resp, err := client.ScheduleKeyDeletion(req)
```

### Create Grant

```go
//go:build ignore
req := &kms_model.CreateGrantRequest{
    Body: &kms_model.CreateGrantRequestBody{
        KeyId:            keyID,
        GranteePrincipal: granteePrincipal,
        Operations:       []string{"Encrypt", "Decrypt", "GenerateDataKey"},
    },
}
resp, err := client.CreateGrant(req)
// resp.GrantId
```

### List / Revoke Grants

```go
//go:build ignore
listReq := &kms_model.ListGrantsRequest{KeyId: keyID}
listResp, err := client.ListGrants(listReq)
// listResp.Grants[].GrantId

revokeReq := &kms_model.RevokeGrantRequest{
    KeyId:   keyID,
    GrantId: grantID,
}
_, err = client.RevokeGrant(revokeReq)
```

### Import Key Material

```go
//go:build ignore
req := &kms_model.ImportKeyMaterialRequest{
    KeyId:             keyID,
    ImportToken:       importToken,
    EncryptedKeyMaterial: encryptedMaterial,
}
_, err = client.ImportKeyMaterial(req)
```

### Create Data Key

```go
//go:build ignore
req := &kms_model.CreateDatakeyRequest{
    KeyId:           keyID,
    DatakeyPlinLength: ptr.Int32(32),
}
resp, err := client.CreateDatakey(req)
// resp.Plaintext  (base64)
// resp.CipherText (base64)
```

### Decrypt Data Key

```go
//go:build ignore
req := &kms_model.DecryptDataKeyRequest{
    KeyId:       keyID,
    CipherText:  cipherText,
}
resp, err := client.DecryptDataKey(req)
// resp.PlainText (base64)
```

## Error Mapping (Top 10 — one per row, ≤3 cols)

| HTTP / `error_code` | Cause | Agent Action |
|---|---|---|
| 400 `InvalidParameterValue` | Bad alias / window days | Fix from OpenAPI; retry 0–1 |
| 403 `CMKAccessDenied` | IAM missing | HALT; add `kms:*` policy |
| 404 `KeyNotFound` | CMK does not exist | HALT; verify key_id |
| 409 `InvalidKeyState` | Op not allowed in current state | HALT; check key_state |
| 409 `KeyExistsException` | Alias already in use | Return existing key_id |
| 409 `GrantNotFound` | Grant already revoked | Treat as success |
| 409 `QuotaExceeded` | CMK limit hit | HALT; raise quota |
| 429 `Throttling` | Rate-limited | Backoff w/ Retry-After |
| 500 `InternalServerError` | HSM error | Retry 3×; escalate |
| 503 `ServiceUnavailable` | Region degraded | Backoff + report |

> **TE-3:** Error table is 4 columns max, one error per row.
