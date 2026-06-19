# CTS SecOps — Audit Trail & Forensic Deep Dive

> Advanced security audit + forensic patterns for Cloud Trace Service.
> Load when handling incident response, compliance reports, or threat-hunt queries.

## 1. Forensic Query Patterns

### 1.1 Privilege escalation trace
- Filter: `event_type = IAM` AND `request.operation = "create agency"`
- Group by `user.name` over 7 days
- Action: flag users with > 5 agency creations; check with Security-Sensitive gate

### 1.2 Data exfiltration trace
- Filter: `event_type = OBS` AND `request.operation = "getObject"` AND `response.size > 1 GB`
- Cross-reference with `user.tenant_id` and source IP
- Action: trigger SecOps HALT; rotate AK/SK; preserve CTS evidence

### 1.3 Login anomaly trace
- Filter: `event_type = IAM` AND `request.operation = "login"`
- Group by source IP; flag > 5 distinct users from same IP / 24h
- Action: block IP via VPC security group; force MFA rotation

## 2. Compliance Reports

| Standard | Retention | Required events |
|----------|-----------|-----------------|
| 等保 2.0 | ≥ 180 days | IAM, OBS, ECS, CTS itself |
| GDPR | ≥ 365 days | data access, deletion, transfer |
| SOC 2 | ≥ 365 days | privileged actions, configuration changes |

## 3. Trace Storage Encryption

- Enable SSE-KMS for the OBS bucket storing CTS export
- Rotate the dedicated KMS key every 365 days
- Restrict bucket access via bucket policy + IAM agency only

> **Security-Sensitive**: CTS event deletion, tracker disabling, or log group
> purge MUST require explicit operator confirmation. The agent MUST surface
> `{{output.preservation_hold_id}}` whenever evidence is held.