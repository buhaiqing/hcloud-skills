# Alarm Storm Handling — CTS

> **Purpose**: Guidance for detecting and mitigating alarm storms involving CTS events.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## What is an Alarm Storm

An alarm storm occurs when a large number of alerts fire in rapid succession, overwhelming operators and obscuring critical incidents. In CTS contexts, alarm storms typically arise from:

- Repeated failed authentication attempts from a single source
- Bulk deletion or modification operations triggering many rule violations
- Misconfigured automation generating excessive API calls
- External attack activity (DDoS, enumeration, credential stuffing)

---

## Detection

### Threshold Indicators

| Indicator | Threshold | Severity |
|-----------|-----------|----------|
| `auth_failure_rate` | > 20 failures/minute from same IP | Critical |
| `delete_event_rate` | > 50 deletes/minute | Critical |
| `api_call_volume` | > 10x baseline in 5 minutes | Warning |
| `error_trace_rate` | > 100 errors/minute | Warning |

### Detection Commands

```bash
# Check for auth failure spike
hcloud cts query-events \
  --event_name=Login \
  --status=fail \
  --start_time=$(date -v-30m +%Y-%m-%dT%H:%M:%SZ) \
  --limit 100

# Check for bulk delete activity
hcloud cts query-events \
  --event_name=Delete* \
  --start_time=$(date -v-60m +%Y-%m-%dT%H:%M:%SZ) \
  --limit 100

# Rate aggregation via CES
hcloud ces metric-data-query \
  --metric_name=cts_api_call_count \
  --dim.0=project_id:{{env.HW_PROJECT_ID}} \
  --start_time=$(date -v-30m +%s) \
  --end_time=$(date +%s) \
  --period=60
```

---

## Mitigation Playbook

### Phase 1: Triage (0-5 minutes)

1. **Identify the dominant pattern** — run the detection commands above
2. **Filter to top offenders** — identify the top 3 IPs or users responsible
3. **Classify severity**:
   - Credential stuffing / attack → Phase 2A
   - Misconfigured automation → Phase 2B
   - Legitimate bulk operation → Phase 2C

### Phase 2A: Attack Response

1. Block offending IP at VPC security group level (delegate to `huaweicloud-vpc-ops`)
2. Temporarily suspend affected user accounts (delegate to `huaweicloud-iam-ops`)
3. Escalate to SecOps with evidence from CTS query results

### Phase 2B: Automation Misconfiguration

1. Identify the automation source (source IP, user agent, IAM user)
2. Temporarily revoke CTS write permissions for the offending credential
3. Notify the team responsible for the automation

### Phase 2C: Legitimate Bulk Operation

1. Suppress CTS alerts temporarily via CES alarm suppression rules
2. Document the planned operation in the incident ticket
3. Re-enable monitoring after operation completes

### Phase 3: Post-Incident

1. Review CTS events to confirm the storm has subsided
2. Add new detection rules if the pattern was novel
3. Update this knowledge base if new root cause discovered

---

## Alarm Suppression

### CES Alarm Suppression

```bash
# Create suppression rule (example — adjust parameters)
hcloud ces alarm-action modify \
  --alarm_id <alarm-id> \
  --suppress_duration 3600
```

### Recommended Suppression Durations

| Scenario | Suppression Duration |
|----------|---------------------|
| Planned maintenance window | Until maintenance end |
| Known automation run | 2x expected duration |
| Attack in progress | 15 minutes (re-evaluate frequently) |

---

## Escalation Paths

| Condition | Escalate To |
|-----------|-------------|
| Confirmed attack with data breach risk | SecOps + Huawei Cloud support |
| CTS service-level outage | Huawei Cloud status page + support |
| Sustained > 1 hour alarm storm | Team lead + Cloud ops |
| Misconfigured automation causing ongoing cost | FinOps team |

---

## Prevention

- Configure CES alarms with rate-of-change thresholds, not just absolute thresholds
- Set up IP-based rate limiting via IAM and VPC security groups proactively
- Document known automation jobs that generate high API volume so they are not mistaken for attacks
- Periodically review CTS query results for early signals of enumeration or credential stuffing
