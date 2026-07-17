# Cross-Skill Delegation Matrix — Template

> **Purpose**: Standardized alarm-to-skill routing for cross-product incident handling.
> **Usage**: Copy this template to each skill's `references/integration.md` §Cross-Skill Delegation,
>   replacing placeholders with product-specific values.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Type → Skill Mapping

| Alarm Type | Target Skill | Delegation Trigger | Priority |
|------------|--------------|-------------------|----------|
| CPU High | `huaweicloud-ecs-ops` | CPU > 90% sustained 5min | P1 |
| Memory High | `huaweicloud-ecs-ops` | Memory > 90% sustained 5min | P1 |
| Disk Full | `huaweicloud-ecs-ops` | Disk > 95% | P0 |
| Instance Down | `huaweicloud-ecs-ops` | Status = ACTIVE false | P0 |
| **To be replaced** | **Replace with product-specific mappings** | **Replace with actual trigger conditions** | **P0/P1** |

### 1.1 Delegation Format

```yaml
# In integration.md §Cross-Skill Delegation:
delegation:
  - alarm_type: "<alarm type from CES>"
    target_skill: "<skill-name-ops>"
    trigger:
      metric: "<metric name>"
      threshold: "<value>"
      duration: "<time>"
    priority: P0|P1
    fallback: "<fallback skill if primary unavailable>"
```

### 1.2 Example (ECS → RDS)

```yaml
delegation:
  - alarm_type: "rds_connection_exhausted"
    target_skill: "huaweicloud-rds-ops"
    trigger:
      metric: "rds045_connection_usage"
      threshold: "95"
      duration: "5m"
    priority: P0
    fallback: "huaweicloud-dbaas-ops"
```

---

## 2. Cross-Product Cascade Routing

### 2.1 Cascade Patterns

| Cascade Path | Propagation Condition | Blocking Action |
|--------------|----------------------|------------------|
| ECS → RDS | DB connection errors from ECS | Check RDS status first |
| ECS → ELB | ELB backend errors | Check ECS health first |
| CCE → ECS | Node pressure | Check CCE node pool |
| **To be replaced** | **Replace with actual cascade conditions** | **Replace with blocking actions** |

### 2.2 Cascade Detection Rules

```yaml
cascade:
  detection:
    window: "30m"
    correlation_threshold: "3 alarms within window"
  blocking:
    - name: "Block ECS restart during RDS failover"
      condition: "rds_failover_in_progress = true"
      action: "delay_ecs_restart"
```

---

## 3. Delegation Priority Matrix

| Priority | Meaning | Response SLA | Auto-Escalate |
|----------|---------|-------------|---------------|
| **P0** | Critical, service down | 5 min | 15 min |
| **P1** | High, degradation | 15 min | 30 min |
| **P2** | Medium, warning | 1 hour | 4 hour |
| **P3** | Low, info | 24 hour | 48 hour |

---

## 4. Skill Reference Table

| Skill | Handles | Namespace |
|-------|---------|-----------|
| `huaweicloud-ecs-ops` | Compute instances, bare metal | `SYS.ECS` |
| `huaweicloud-cce-ops` | Container workloads | `SYS.CCE` |
| `huaweicloud-rds-ops` | Database instances | `SYS.RDS` |
| `huaweicloud-dcs-ops` | Redis/Cache | `SYS.DCS` |
| `huaweicloud-dms-ops` | Message queues | `SYS.DMS` |
| `huaweicloud-ces-ops` | Monitoring/Alarm | `SYS.CES` |
| `huaweicloud-elb-ops` | Load balancing | `SYS.ELB` |
| `huaweicloud-vpc-ops` | Network, VPN, NAT | `SYS.VPC` |
| `huaweicloud-obs-ops` | Object storage | `SYS.OBS` |
| `huaweicloud-css-ops` | Search service | `SYS.CSS` |
| `huaweicloud-gaussdb-ops` | Database (GaussDB) | `SYS.GAUSSDB` |
| **To be replaced** | **Add product-specific skills** | **Add CES namespaces** |

---

## 5. Implementation Notes

### 5.1 How to Reference This Template

1. Copy this file's content to your skill's `references/integration.md`
2. Replace placeholder rows with actual alarm mappings for your product
3. Keep the cascade patterns that apply to your product
4. Update the Skill Reference Table with products you interact with

### 5.2 Delegation Constraints

- **Max Hops**: 3 (prevent infinite delegation loops)
- **Timeout**: 5 min per delegation (fallback to next priority if timeout)
- **Circular Prevention**: Track delegation chain, block if cycle detected

### 5.3 Fallback Rules

| Scenario | Fallback Behavior |
|----------|-------------------|
| Primary skill unavailable | Route to fallback skill |
| All skills unavailable | Log event, create incident, alert on-call |
| Delegation timeout | Retry once, then escalate |

---

## 6. Compliance Checklist

- [ ] All P0/P1 alarms have delegation targets
- [ ] No circular delegation paths (use `check_delegation_chain.sh`)
- [ ] Fallback skills are defined for all P0 delegations
- [ ] Cascade blocking rules are documented
- [ ] Priority matrix matches incident severity levels

---

## 7. Validation Command

```bash
# Validate delegation matrix (run from repo root)
python3 scripts/validate_delegation_matrix.py --skill <skill-name>
```
