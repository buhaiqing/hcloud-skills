# CBR Backup — Reliability & Disaster Recovery Deep Dive

> Advanced stability + DR patterns layered below the runbook.
> Load this file when designing cross-region DR, RTO/RPO targets, or
> backup retention policies.

## 1. RTO / RPO Targets

| Tier | RTO target | RPO target | Replication | Use case |
|------|-----------|-----------|-------------|----------|
| Critical | < 15 min | < 5 min | Cross-region CBR + async replication | Production OLTP |
| Important | < 1 h | < 30 min | Cross-region CBR (daily) | Internal services |
| Standard | < 4 h | < 24 h | Same-region CBR only | Dev / test |

## 2. Cross-Region DR Pattern

```text
Primary region (cn-north-4)        DR region (cn-south-1)
┌──────────────────────┐         ┌──────────────────────┐
│ ECS / RDS / EVS      │ ──────► │ Replicated vault     │
│ Source vault         │ CBR     │ Standby resources    │
└──────────────────────┘ copy    └──────────────────────┘
```

1. Replicate vault to DR region every 30 min
2. Run DR drill monthly: restore last replica into isolated VPC
3. Document `RTO` measured during drill, surface drift via CES alarm

## 3. Backup Verification Runbook

1. Pick latest snapshot → `status: available`
2. Verify `size_bytes > 0` and `consistency: crash-consistent`
3. Spin up isolated ECS from snapshot, run smoke tests
4. Tear down DR environment within `verify_window`

> **Security-Sensitive**: vault deletion and replication deletion MUST require
> operator confirmation (`{{user.confirm_destructive}}`) before the call is
> issued. Encrypt DR replicas with a dedicated KMS key rotated ≥ 365d.