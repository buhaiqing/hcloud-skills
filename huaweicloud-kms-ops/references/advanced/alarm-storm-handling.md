# Alarm Storm Handling — KMS

> **Purpose**: Guidance for detecting and mitigating alarm storms involving Key Management Service.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

KMS alarm storms arise from API throttling, unauthorized key state changes, and deletion campaigns. Signals combine CES metrics with CTS audit trails.

| Pattern | Indicators | Severity |
|---------|-----------|----------|
| API failure storm | `kms_key_api_fail_count` >= 10 / 5min OR failure rate > 10% OR `ThrottlingException` | Critical |
| Unauthorized state change | `DISABLE_KEY` / `SCHEDULE_KEY_DELETION` with no user trigger | Critical |
| Over-grant | Single key `grant_count` > 10 | Warning |
| Deletion storm | >= 3 keys enter `PENDING_DELETION` within 7 days | Critical |
| Quota pressure | `kms_quota_usage_ratio` >= 0.8 | Warning |

Query API failures via `hcloud ces metric-data-query --namespace=SYS.KMS --metric_name=kms_key_api_fail_count`; audit state changes via `hcloud cts query-events --event_name=ScheduleKeyDeletion`.

---

## 2. Aggregation Rules

- **Audit linkage**: State-change alarms are correlated with CTS trails and IAM to judge whether the actor was authorized; suppress if triggered by a known admin change ticket.
- **Deletion storm collapse**: All keys entering `PENDING_DELETION` within 7 days are aggregated per key into a single "deletion storm" incident rather than N separate alarms.
- **Throttling merge**: `ThrottlingException` + `kms_key_api_fail_count` spikes are merged into one API-throttling incident.

---

## 3. Suppression Rules

| Scenario | Suppression |
|----------|-------------|
| Planned key rotation campaign | Suppress state-change alarms for campaign window |
| Known bulk encryption job | Suppress throttling alarm if quota headroom exists (re-evaluate 15 min) |
| Authorized deletion (ticketed) | Suppress deletion-storm alarm for that batch |

Suppress a CES alarm during maintenance:

```bash
hcloud ces alarm-action modify --alarm_id <alarm-id> --suppress_duration 3600
```

---

## 4. Response Procedures

### Phase 1: Triage (0-5 min)
1. Run detection commands; correlate with CTS to separate attack from planned op.

### Phase 2: Unauthorized Change
- Inspect who disabled/scheduled deletion via `hcloud cts query-events --event_name=DisableKey --limit 50`; cancel scheduled deletion / re-enable key; rotate potentially exposed key.

### Phase 3: Deletion Storm
- Halt further `ScheduleKeyDeletion`; verify with key owners; check dependent services.

### Phase 4: Post-Incident
- Review IAM grants; confirm storm subsided.

---

## 5. Delegation Matrix

| Trigger | Delegate To |
|---------|-------------|
| Encrypted bucket impact | `huaweicloud-obs-ops` |
| Encrypted DB impact | `huaweicloud-rds-ops` |
| Encrypted disk impact | `huaweicloud-evs-ops` |
| Permission / grant review | `huaweicloud-iam-ops` |
| Cost of key ops | `huaweicloud-billing-ops` |
| Audit trail validation | `huaweicloud-cts-ops` |
