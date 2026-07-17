# KMS Troubleshooting Guide — Huawei Cloud Key Management Service

## 1. Top Failure Patterns

| # | Pattern | Symptom | Root Cause | Fix |
|---|---|---|---|---|
| T1 | Dependent service encryption fails | OBS upload / RDS query fails with "encryption error" | CMK is `DISABLED` or `PENDING_DELETION` | Re-enable key or cancel deletion |
| T2 | Key deletion doesn't apply | `schedule-key-deletion` returns success but key still works | Window not expired; key still usable during window | Understand PENDING_DELETION semantics |
| T3 | Grant revoked too early | Service loses access to encrypted data | Grant revoked before data migration | Re-grant or re-encrypt data |
| T4 | BYOK import fails | `import-key-material` returns `InvalidParameter` | Import token expired (24h validity) | Generate new import token |
| T5 | CMK quota exhausted | `create-key` returns `QuotaExceeded` | 100 CMK limit reached | Delete unused keys or raise quota |
| T6 | IAM permission denied | `CMKAccessDenied` on all KMS calls | IAM policy missing `kms:*` | Delegate to `huaweicloud-iam-ops` |
| T7 | Rotation not working | Data still decrypts with old key | Rotation rotates wrapper only; data not re-encrypted | Manual re-encryption needed |
| T8 | Key state conflict | `enable-key` fails on `PENDING_DELETION` key | Cannot enable a scheduled-for-deletion key | Cancel deletion first via `cancel-key-deletion` |

## 2. Diagnostic Order

```
1) hcloud kms describe-key --key-id {{user.key_id}}
   → Check key_state: ENABLED / DISABLED / PENDING_DELETION

2) hcloud kms list-grants --key-id {{user.key_id}}
   → Check active grants (revoke grants before deletion)

3) hcloud kms list-keys --region {{env.HW_REGION_ID}}
   → Check CMK count vs quota

4) Check dependent services (OBS / RDS / EVS) for encryption failures
   → If dependent service fails: re-enable CMK
```

## 3. Error Code Quick Reference

| Error Code | Meaning | Action |
|---|---|---|
| `CMKAccessDenied` | No IAM permission | Add `kms:*` policy |
| `KeyNotFound` | Key doesn't exist | Verify key_id |
| `InvalidKeyState` | Operation not allowed in current state | Check key_state |
| `KeyExistsException` | Alias already in use | Use existing key or unique alias |
| `QuotaExceeded` | CMK limit reached | Delete unused keys |
| `GrantNotFound` | Grant already revoked | Idempotent — ignore |
| `InvalidParameter` | Bad parameter value | Fix from error message |
| `ThrottlingException` | Rate limit | Back off |
| `InternalServerError` | HSM error | Retry; escalate |

## 4. Key State → Allowed Operations Matrix

| Op | ENABLED | DISABLED | PENDING_DELETION |
|---|---|---|---|
| Encrypt | ✅ | ❌ | ❌ |
| Decrypt | ✅ | ❌ | ❌ |
| GenerateDataKey | ✅ | ❌ | ❌ |
| EnableKey | ✅ (no-op) | ✅ | ❌ (must cancel first) |
| DisableKey | ✅ | ✅ (no-op) | ❌ |
| CreateGrant | ✅ | ❌ | ❌ |
| RevokeGrant | ✅ | ❌ | ❌ |
| ScheduleKeyDeletion | ✅ | ✅ | ❌ (already scheduled) |
| CancelKeyDeletion | ❌ | ❌ | ✅ |
