# KMS Knowledge Base — Fault Patterns

## Pattern K1: Dependent Service Encryption Fails After Key Disabled

| Symptom | OBS upload / RDS query fails with "encryption key unavailable" |
|---|---|
| Root Cause | CMK is `DISABLED` or in `PENDING_DELETION` state; dependent services cannot encrypt/decrypt |
| Diagnosis | `hcloud kms describe-key --key-id <id>` → check `key_state` |
| Resolution | (a) Re-enable key via `enable-key` if in `DISABLED`. (b) Cancel deletion via `cancel-key-deletion` if in `PENDING_DELETION`. (c) Verify dependent services recover within 5 min. |
| Cross-skill | `huaweicloud-obs-ops` (OBS failure), `huaweicloud-rds-ops` (RDS failure) |

## Pattern K2: Key Deletion Scheduled Without Grant Revoke

| Symptom | `schedule-key-deletion` succeeded; dependent service loses access during window |
|---|---|
| Root Cause | Grants remain active during deletion window; revocation during window breaks service |
| Diagnosis | `list-grants` shows active grants after `schedule-key-deletion` |
| Resolution | (a) Cancel deletion immediately via `cancel-key-deletion`. (b) Revoke all grants. (c) Re-schedule deletion only after confirming no dependent services remain. |

## Pattern K3: BYOK Import Fails with Expired Token

| Symptom | `import-key-material` returns error or silently fails |
|---|---|
| Root Cause | Import token validity is 24 hours; expired token = operation rejected |
| Diagnosis | Check token creation timestamp vs current time (> 24h = expired) |
| Resolution | Generate a new import token via `create-import-token` and retry within 24h |

## Pattern K4: Data Unrecoverable After Deletion Window Expires

| Symptom | OBS objects / RDS databases become unreadable after deletion window |
|---|---|
| Root Cause | Key deletion is permanent; data encrypted with that key is unrecoverable |
| Diagnosis | Verify `key_state = DELETED` and deletion_date is in the past |
| Resolution | (a) Prevention: never delete a key with active dependent resources. (b) Recovery: none — data is permanently lost. |

## Pattern K5: Key Rotation Doesn't Re-encrypt Existing Data

| Symptom | After rotation, old data still decrypts with old key version |
|---|---|
| Root Cause | Rotation rotates the CMK wrapper (DEK envelope), not the data encryption keys themselves |
| Diagnosis | `show-key-rotation-policy` confirms rotation is enabled |
| Resolution | Manual re-encryption: decrypt with old key version, encrypt with new key version (agent must implement) |
