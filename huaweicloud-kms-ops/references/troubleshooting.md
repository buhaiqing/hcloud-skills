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

## 官方错误码参考（API Error Codes）

> **权威说明**：下表为华为云 KMS API **真实返回**的错误码（`KMS.*` 命名空间），故障定位时优先匹配。上文「Error Code Quick Reference」中的 `CMKAccessDenied` / `KeyNotFound` / `QuotaExceeded` / `InvalidParameter` 等为本 runbook 的**文档约定码**（语义化助记，非 API 原始返回码），仅用于正文 T1–T8 流程引用；当 `hcloud` 返回 `KMS.*` 码时以下表为准。

来源：[华为云 KMS API 参考 — 附录：错误码](https://support.huaweicloud.com/api-kms/ErrorCode.html)。以下为真实 `KMS.*` 错误码（CSMS.* 为凭据库错误码，不在此列）。

| Error Code | Meaning | Recovery |
|---|---|---|
| `KMS.0207` | 密钥不存在 | 选择有效密钥或先创建密钥 |
| `KMS.0209` | 密钥已被禁用 | 启用该密钥 |
| `KMS.0210` | 密钥处于计划删除状态，不可使用 | 启用密钥（取消计划删除） |
| `KMS.0211` | 默认主密钥不支持该操作 | 使用普通 CMK 执行操作 |
| `KMS.1104` | 密钥别名重复 | 使用其他别名 |
| `KMS.1105` | 密钥数量过多 | 提升配额或删除无用密钥 |
| `KMS.1201` | 密钥未处于禁用状态 | 先禁用密钥 |
| `KMS.1301` | 密钥未处于启用状态 | 启用密钥 |
| `KMS.1402` | 密钥已处于待删除状态 | 无需进一步操作 |
| `KMS.2601` | 导入令牌过期（BYOK） | 获取新令牌 |
| `KMS.2603` | 导入密钥与令牌中的 Key ID 不匹配 | 确保导入密钥的 Key ID 与令牌一致 |
| `KMS.2605` | 令牌校验失败 | 获取新令牌 |
| `KMS.1902` | `key_spec` 仅支持 AES_128 / AES_256 | 输入合法参数 |
| `KMS.0301` | X-Auth-Token 非法或为空 | 重新获取 token 并确保字符串完整 |
| `KMS.0303` | X-Auth-Token 已过期 | 重新获取 token |
| `KMS.0306` | 无访问权限 | 联系管理员授予所需权限 |
