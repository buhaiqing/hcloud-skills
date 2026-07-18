# Alarm Storm Handling — HSS

> **Purpose**: Guidance for detecting and mitigating alarm storms involving Host Security Service.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

HSS alarm storms arise from brute-force floods, malware outbreaks, and unhandled-alert pile-ups. Signals come from HSS ListEvents and the host health score.

| Pattern | Indicators | Severity |
|---------|-----------|----------|
| Alert storm (mode 1) | `ListEvents` daily alerts > 200% of baseline OR > 50 alerts/day | Critical |
| Backlog (mode 2) | Unhandled alerts grow for 3 consecutive days | Warning |
| Vulnerability (mode 3) | Fix rate < 80% OR critical vuln unhandled | Critical |
| Health score drop | Agent offline / unprotected / unhandled > 10 / vulns > 20 (score deduction) | Warning |

```bash
# List recent security events
hcloud hss list-events \
  --host_name {{user.host_name}} \
  --start_time=$(date -v-1d +%Y-%m-%dT%H:%M:%SZ) --limit 100

# Group alerts by source IP to spot brute-force floods
hcloud hss list-events --limit 200 \
  | jq -s 'group_by(.src_ip) | map({ip: .[0].src_ip, count: length})'
```

---

## 2. Aggregation Rules

- **Group by host / src_ip**: Use `jq group_by(.host_name)` and `group_by(.src_ip)` to collapse per-event noise into per-source incidents.
- **Brute-force auto-ban**: Same `src_ip` with `login_fail` >= 10 → single "brute-force" incident + auto block, suppress individual login-fail alarms.
- **Malware auto-isolate**: Same host with 3 consecutive `malware` detections → single outbreak incident + auto isolate.

---

## 3. Suppression Rules

| Scenario | Suppression |
|----------|-------------|
| Known scanner / pen-test IP | Suppress login-fail alarms for test window |
| Auto-ban active | Suppress subsequent login-fail from banned IP (15 min) |
| Auto-isolate active | Suppress malware duplicates from isolated host |

```bash
# Suppress a CES/HSS alarm during response (example — adjust args)
hcloud ces alarm-action modify \
  --alarm_id <alarm-id> \
  --suppress_duration 1800
```

---

## 4. Response Procedures

### Phase 1: Triage (0-5 min)
1. Group events by `src_ip` / `host_name`; identify dominant pattern.

### Phase 2: Brute-Force Flood
```bash
# Auto-block offending source IP (delegate to VPC security group)
hcloud hss block-ip --ip {{output.src_ip}}   # placeholder — verify subcommand
```

### Phase 3: Malware Outbreak
- Auto-isolate host; trigger vulnerability scan; verify health score recovery.

### Phase 4: Post-Incident
- Confirm backlog cleared; tune detection thresholds if novel pattern.

---

## 5. Delegation Matrix

| Trigger | Delegate To |
|---------|-------------|
| Instance state before isolation | `huaweicloud-ecs-ops` |
| Web tamper coordination | `huaweicloud-waf-ops` |
| Container escape | `huaweicloud-cce-ops` |
| Cost of response | `huaweicloud-billing-ops` |
| Account-level permission | `huaweicloud-iam-ops` |
| DDoS protection | `huaweicloud-antiddos-ops` |
