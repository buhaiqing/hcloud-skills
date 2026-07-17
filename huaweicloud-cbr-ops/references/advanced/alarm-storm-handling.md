# Alarm Storm Handling — CBR

> **Purpose**: Guidance for detecting and suppressing CBR alarm storms during
> large-scale incidents. Load when multiple CBR alarms fire within a short
> window or when CES reports a CBR alarm surge.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

### 1.1 Detection Criteria

| Signal | Threshold | Window |
|--------|-----------|--------|
| CBR alarms per vault | > 10 | 5 min |
| CBR alarms across account | > 50 | 5 min |
| Duplicate alarm ratio | > 80% identical messages | 5 min |

### 1.2 Trigger Sources

- Mass backup failure across multiple vaults (storage backend issue)
- Vault locking cascade (billing/欠费)
- Policy misconfiguration triggering repeated retries
- Cross-region replication backlog

---

## 2. Suppression Strategy

### 2.1 Per-Vault Coalescing

When > 10 alarms fire for the same `vault_id` within 5 minutes:

1. Suppress further per-backup alarms for that vault
2. Emit a single summary alarm: `CBR_VAULT_ALARM_STORM: <vault_id>, <count> alarms coalesced`
3. Continue monitoring for resolution

### 2.2 Cross-Vault Aggregation

When alarms fire across > 5 vaults simultaneously:

1. Identify common root cause (check CTS for bulk operations, check billing status)
2. Emit: `CBR_CROSS_VAULT_ALARM_STORM: <count> vaults affected, suspected cause: <root_cause>`
3. Suppress per-vault notifications until root cause is resolved

### 2.3 Time-Window Deduplication

| Alarm Type | Suppression Window |
|------------|-------------------|
| `backup_failed` | 15 min (allow one retry) |
| `restore_failed` | 30 min (investigation expected) |
| `storage_quota_warning` | 60 min (non-urgent) |
| `policy_trigger_missed` | 120 min (batch job window) |

---

## 3. Escalation Paths

### 3.1 Alarm Storm Severity Tiers

| Tier | Condition | Action |
|------|-----------|--------|
| T1 — Critical | Vault locked + > 5 vaults affected | Page ops immediately, escalate to CBR on-call |
| T2 — High | Backup failure rate > 50% + > 3 vaults | Alert CBR team lead, begin incident bridge |
| T3 — Medium | Isolated vault issue, < 3 vaults | Normal alarm delivery, track in incident db |

### 3.2 Alarm Routing

```
T1 alarm storm
  → PagerDuty: CBR on-call
  → Slack: #cbr-incidents, #cbr-oncall
  → Stop: per-vault alarm spam

T2 alarm storm
  → PagerDuty: CBR on-call (low urgency)
  → Slack: #cbr-alerts
  → Continue: one alarm per vault (no coalescing)

T3 alarm
  → Slack: #cbr-alerts
  → Email: CBR team distribution list
  → Continue: normal alarm delivery
```

---

## 4. Recovery Validation

After root cause resolution:

1. Verify backup jobs resume normally: `hcloud cbr backup-list --vault-id <id> --status running`
2. Confirm storage metrics stabilize: CES dashboard for vault I/O
3. Check replication lag clears: `hcloud cbr replication-list`
4. Manually clear suppressed alarm cache to resume normal alerting
5. Post incident report: alarm count, duration, root cause, action taken
