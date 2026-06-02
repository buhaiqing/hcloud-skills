# Knowledge Base — Common Billing Fault Patterns

## Fault Pattern 1: Unexpected High Cost

**Symptom:** Monthly bill significantly higher than previous months.
**Possible causes:**
- New resource deployment without budget check
- Resource leak (forgotten test/staging instances)
- DDoS attack (bandwidth cost spike)
- Reserved instance expired → fell back to on-demand
**Diagnosis:** Run Op 11 (Anomaly Detection) → Op 8 (Cost Analysis by service) → Op 12 (Optimization Mining)

## Fault Pattern 2: Budget Exceeded

**Symptom:** Budget alert fired before expected.
**Possible causes:**
- Resource misconfiguration (over-provisioned)
- Unplanned deployment (dev instance in production)
- Price change or billing model switch
**Diagnosis:** Check Op 6 budget → Op 8 cost breakdown → Op 12 pattern detection

## Fault Pattern 3: Resource Package Underutilized

**Symptom:** Bought a resource package but usage is low.
**Possible causes:**
- Overestimated workload
- Workload migrated to different service
**Action:** Run Op 7 → Op 12 (P7 idle package) → consider refund or downgrade

## Fault Pattern 4: Cannot Refund Resource Package

**Symptom:** `BSS.0302` — package not refundable.
**Possible causes:**
- Package type has "no refund" policy (typically annual packages)
- Package already partially consumed
**Action:** Inform user → suggest waiting for expiry → configure smaller package next time

## Fault Pattern 5: Bill Delay

**Symptom:** Current month bills not showing.
**Possible causes:**
- Billing system aggregation delay (up to 24h for pay-per-use)
- Cross-region data synchronization (up to 72h for some services)
**Action:** Wait and retry → if > 72h, contact support

## Cross-Product Cascade

| Trigger | Affected | Cascade |
|---------|----------|---------|
| ECS instance stopped by budget limit | All apps on that ECS | Service disruption → monitoring alert → cost stop |
| Reserved instance expired | ECS/RDS billing | On-demand cost spike → budget overshoot |
| Storage lifecycle policy applied | OBS costs | Storage cost drops → retrieval costs may rise |
| Log retention shortened | LTS costs | Storage cost drops → audit compliance risk |