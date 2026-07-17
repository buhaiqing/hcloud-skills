# KMS AIOps Patterns

> Tier-2 advanced content per **TE-7**: AIOps depth lives here, not in SKILL.md.

## Pattern 1: Key State Anomaly (Spike)

| Field | Value |
|---|---|
| Metric | `key_state` change events via CTS audit log |
| Window | Event-level |
| Threshold | Any `DISABLE_KEY` or `SCHEDULE_KEY_DELETION` without user-initiated trigger |
| Action | (a) Query CTS for `disable-key` / `schedule-key-deletion` events. (b) Cross-check with IAM logs for unauthorized principal. (c) Delegate to `huaweicloud-iam-ops` for permission audit. |
| Cross-skill | `huaweicloud-iam-ops` (permission audit), `huaweicloud-cts-ops` (audit log) |

## Pattern 2: Grant Proliferation (Pressure)

| Field | Value |
|---|---|
| Metric | `grant_count` per CMK |
| Window | Daily |
| Threshold | `grant_count > 10` on a single key |
| Action | (a) Query `list-grants` for all grants on the key. (b) Identify stale grants (grantee service decommissioned). (c) Revoke stale grants after confirmation. |
| Cross-skill | `huaweicloud-iam-ops` (grantee identity), `huaweicloud-billing-ops` (cost) |

## Pattern 3: Deletion Pending Storm (Trend)

| Field | Value |
|---|---|
| Metric | `key_state=PENDING_DELETION` count within 7-day window |
| Window | Daily |
| Threshold | ≥ 3 keys entering `PENDING_DELETION` within 7 days |
| Action | (a) Query `list-keys` + filter `key_state=PENDING_DELETION`. (b) For each: verify no dependent OBS/RDS/EVS resources. (c) Cancel deletion for keys still in use. |
| Cross-skill | `huaweicloud-billing-ops` (cost leak), `huaweicloud-obs-ops`, `huaweicloud-rds-ops` |

## Pattern 4: API Throttling (Pressure)

| Field | Value |
|---|---|
| Metric | `kms_key_api_fail_count` / `kms_key_api_invoke_count` |
| Window | 5 min |
| Threshold | Fail rate > 10% OR `ThrottlingException` in logs |
| Action | (a) Back off retry interval. (b) Distribute calls across keys if batch operation. (c) If persistent, open a Huawei Cloud ticket for quota increase. |
| Cross-skill | `huaweicloud-ces-ops` (alarm wiring) |

## Cross-Skill Delegation Matrix (AIOps)

| Pattern | KMS | CTS | IAM | Billing | OBS | RDS | CES |
|---|:-:|:-:|:-:|:-:|:-:|:-:|:-:|
| 1 State anomaly | ✅ | ✅ | ✅ | | | | |
| 2 Grant proliferation | ✅ | | ✅ | ✅ | | | |
| 3 Deletion storm | ✅ | | | ✅ | ✅ | ✅ | |
| 4 Throttling | ✅ | | | | | | ✅ |
