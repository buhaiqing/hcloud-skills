# Alarm Storm Handling — WAF

> **Purpose**: Guidance for detecting and mitigating alarm storms involving Web Application Firewall.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

WAF alarm storms arise from attack spikes, rule bypass/decay, and certificate expiry. Signals come from WAF event logs and CES.

| Pattern | Indicators | Severity |
|---------|-----------|----------|
| Attack surge (mode 1) | Attack-event change rate > 100% | Critical |
| Rule decay (mode 2) | Rule hit-rate deviates > 3σ from mean OR 0 hits for 3 consecutive days | Warning |
| Cert expiry (mode 3) | Certificate expires in 30 / 7 days | Warning / Critical |
| Domain health | Attack volume > 10000 / 7d | Critical |

```bash
# Query attack events from WAF
hcloud waf list-events \
  --domain {{user.domain}} \
  --start_time=$(date -v-1d +%Y-%m-%dT%H:%M:%SZ) --limit 200

# Group by source IP to spot multi-domain attack source
hcloud waf list-events --limit 500 \
  | jq -s 'group_by(.sip) | map({ip: .[0].sip, count: length})'
```

---

## 2. Aggregation Rules

- **Source grouping**: `group_by(.sip)` to recognize one attacker hitting multiple domains → single incident, not per-domain alarms.
- **CC attack tightening**: If `attacks` contains `"cc"` and count > 500 → single CC-flood incident + auto tighten rule, suppress per-request CC alarms.
- **Rule decay merge**: Consecutive 0-hit days collapse into one "rule decay" alert, not daily duplicates.

---

## 3. Suppression Rules

| Scenario | Suppression |
|----------|-------------|
| Planned load test / scan | Suppress attack-surge alarms for test window |
| CC auto-tighten active | Suppress CC duplicates from offending IP (15 min) |
| Known good bot | Suppress rule-decay alarm if allowlist confirmed |

```bash
# Suppress a CES/WAF alarm during response (example — adjust args)
hcloud ces alarm-action modify \
  --alarm_id <alarm-id> \
  --suppress_duration 1800
```

---

## 4. Response Procedures

### Phase 1: Triage (0-5 min)
1. Group events by `sip`; classify attack type (CC / SQLi / XSS / bot).

### Phase 2: Attack Surge
```bash
# Tighten CC rule for offending source (placeholder — verify subcommand)
hcloud waf modify-policy --domain {{user.domain}} --cc-threshold 500
```

### Phase 3: Cert Expiry
- Renew certificate via SCM; bind to domain; verify no expiry alarm remains.

### Phase 4: Post-Incident
- Confirm attack volume subsided; update rule set if bypass detected.

---

## 5. Delegation Matrix

| Trigger | Delegate To |
|---------|-------------|
| Backend instance protection | `huaweicloud-ecs-ops` |
| Host intrusion on backend | `huaweicloud-hss-ops` |
| Listener / LB config | `huaweicloud-elb-ops` |
| Cost of WAF | `huaweicloud-billing-ops` |
| Account permission | `huaweicloud-iam-ops` |
| Network / subnet | `huaweicloud-vpc-ops` |
| Certificate management | `huaweicloud-scm-ops` |
| Alarm rule config | `huaweicloud-ces-ops` |
| DDoS protection | `huaweicloud-antiddos-ops` |
