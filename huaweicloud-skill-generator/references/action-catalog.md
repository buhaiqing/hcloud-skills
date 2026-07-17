# Action Catalog — L5 Autonomous Operations

> **Purpose**: Risk-classified action registry for L5 autonomous operations. Defines which actions can auto-execute vs. require human approval.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §12 (Self-Healing)
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Risk Level Definitions

| Level | Label | Color | Auto-Execute | Human Approval | Rollback |
|-------|-------|-------|--------------|----------------|----------|
| 1 | **Low** | Green | ✅ Yes | Not required | Simple revert |
| 2 | **Medium** | Yellow | ✅ Yes | Not required | Scripted rollback |
| 3 | **High** | Orange | ❌ No | Required before execution | Manual rollback |
| 4 | **Critical** | Red | ❌ No | Emergency approval only | Irreversible |

### 1.1 Risk Assessment Criteria

| Dimension | Low (1) | Medium (2) | High (3) | Critical (4) |
|-----------|---------|------------|----------|--------------|
| Blast radius | Single instance | Single AZ | Multi-AZ | Region-wide |
| Data impact | None | Non-production | Production read | Production write |
| Reversibility | Instant | < 5 min | < 30 min | > 30 min / Irreversible |
| Cost impact | < $10 | < $100 | < $1000 | > $1000 |
| SLA impact | None | < 1% | 1-5% | > 5% |

---

## 2. ECS Action Catalog

| ID | Scenario | Action | Risk | Auto | Preconditions | Rollback |
|----|----------|--------|------|------|---------------|----------|
| ECS-A01 | Alarm disabled after deployment | Re-enable alarm | Low | ✅ | Alarm rule exists; target unchanged | Disable alarm |
| ECS-A02 | CPU持续高 (> 80% 持续 5min) | Scale up (vertical) | Medium | ✅ | Available flavor; quota available | Scale down |
| ECS-A03 | CPU持续高 (> 80% 持续 5min) | Scale up (AS) | Medium | ✅ | AS group exists; scaling rule defined | Remove instance |
| ECS-A04 | Memory持续高 (> 85%) | Restart process/service | Medium | ✅ | Process name known; health check defined | Revert service config |
| ECS-A05 | 磁盘使用率 > 90% | Expand disk | Medium | ✅ | Volume type supports expand; quota | Shrink (if supported) |
| ECS-A06 | 实例不健康 | Mark unhealthy + replace | Medium | ✅ | AS group or manual replacement | Restore original |
| ECS-A07 | 重启生产实例 | Instance reboot | High | ❌ | — | Power on |
| ECS-A08 | 删除生产实例 | Instance delete | Critical | ❌ | — | Recreate (if backup exists) |
| ECS-A09 | 修改安全组规则 | Security group rule change | High | ❌ | — | Revert rule |
| ECS-A10 | 修改VPC/子网 | VPC/subnet change | Critical | ❌ | — | Revert VPC config |

---

## 3. RDS Action Catalog

| ID | Scenario | Action | Risk | Auto | Preconditions | Rollback |
|----|----------|--------|------|------|---------------|----------|
| RDS-A01 | 连接池饱和 | Reset connection pool | Low | ✅ | Connection pooler enabled | Restore connections |
| RDS-A02 | 慢查询 (> 5s) | Kill slow query | Low | ✅ | Processlist accessible | Auto-reap |
| RDS-A03 | 磁盘使用率 > 85% | Expand storage | Medium | ✅ | Auto-scaling or manual expand | Cannot shrink |
| RDS-A04 | 实例不健康 | Reboot instance | Medium | ✅ | Master/standby switchable | Switch back |
| RDS-A05 | 备份失败 | Retry backup | Low | ✅ | Backup service normal | Manual backup |
| RDS-A06 | 主备切换 | RDS failover | Critical | ❌ | — | Manual switchback |
| RDS-A07 | 修改参数组 | Parameter group change | High | ❌ | — | Reset to default |
| RDS-A08 | 删除实例 | Instance delete | Critical | ❌ | — | Recreate (if backup exists) |
| RDS-A09 | 数据库导出 | Database export | High | ❌ | — | Delete export file |
| RDS-A10 | 强制备份 | Force backup | Medium | ✅ | Backup window defined | Delete backup |

---

## 4. CCE Action Catalog

| ID | Scenario | Action | Risk | Auto | Preconditions | Rollback |
|----|----------|--------|------|------|---------------|----------|
| CCE-A01 | Pod restart loop | Delete pod (force reschedule) | Low | ✅ | ReplicaSet > 1 | Restore pod |
| CCE-A02 | Node NotReady | Cordon + drain + replace | Medium | ✅ | New node available; drain timeout set | Uncordon node |
| CCE-A03 | 存储卷空间不足 | Expand PVC | Medium | ✅ | StorageClass supports expand | Cannot shrink |
| CCE-A04 | 镜像拉取失败 | Update image pull secret | Low | ✅ | Valid registry credentials | Restore secret |
| CCE-A05 | HPA target unreachable | Restart deployment | Medium | ✅ | ReplicaSet > 1 | Rollback deployment |
| CCE-A06 | 删除Namespace | Namespace delete | Critical | ❌ | — | Recreate namespace |
| CCE-A07 | 删除Deployment | Deployment delete | Critical | ❌ | — | Recreate from manifest |
| CCE-A08 | 修改node池 | Node pool resize | High | ❌ | — | Resize back |
| CCE-A09 | 升级集群版本 | Cluster upgrade | Critical | ❌ | — | Cannot downgrade |
| CCE-A10 | 滚动更新失败 | Rollback deployment | Medium | ✅ | Previous revision exists | Rollforward |

---

## 5. CES Action Catalog

| ID | Scenario | Action | Risk | Auto | Preconditions | Rollback |
|----|----------|--------|------|------|---------------|----------|
| CES-A01 | Alarm disabled | Re-enable alarm | Low | ✅ | Alarm rule exists | Disable alarm |
| CES-A02 | 告警阈值不合理 | Adjust threshold | Medium | ✅ | Previous threshold known | Restore threshold |
| CES-A03 | 告警重复触发 (alarm storm) | Suppress duplicate alarms | Low | ✅ | Suppression period defined | Remove suppression |
| CES-A04 | 监控指标丢失 | Recreate alarm rule | Medium | ✅ | Metric still available | Restore alarm rule |
| CES-A05 | 自定义指标不上报 | Restart monitoring agent | Low | ✅ | Agent accessible | Manual agent restart |
| CES-A06 | 删除告警规则 | Delete alarm rule | Critical | ❌ | — | Recreate alarm rule |
| CES-A07 | 修改告警通知方式 | Change notification path | Medium | ❌ | — | Restore notification path |
| CES-A08 | 批量创建告警规则 | Batch create alarms | High | ❌ | — | Delete created alarms |

---

## 6. ELB Action Catalog

| ID | Scenario | Action | Risk | Auto | Preconditions | Rollback |
|----|----------|--------|------|------|---------------|----------|
| ELB-A01 | 后端实例不健康 | Remove from pool + alert | Low | ✅ | Health check defined | Add back to pool |
| ELB-A02 | 后端实例恢复 | Add to pool | Low | ✅ | Instance healthy | Remove from pool |
| ELB-A03 | 证书即将过期 (< 30d) | Rotate certificate | Medium | ✅ | New certificate ready | Restore old cert |
| ELB-A04 | 连接超时率高 | Increase timeout | Medium | ✅ | Client behavior verified | Restore default |
| ELB-A05 | 带宽达到上限 | Upgrade bandwidth | Medium | ✅ | Quota available | Downgrade (if allowed) |
| ELB-A06 | 删除监听器 | Listener delete | Critical | ❌ | — | Recreate listener |
| ELB-A07 | 修改后端服务器组 | Backend group change | High | ❌ | — | Restore original group |
| ELB-A08 | 禁用负载均衡器 | Disable load balancer | Critical | ❌ | — | Re-enable |
| ELB-A09 | 添加后端实例 | Add backend instance | Medium | ✅ | Instance healthy | Remove instance |
| ELB-A10 | 修改健康检查 | Health check config change | High | ❌ | — | Restore health check |

---

## 7. Execution Preconditions (Common)

| Condition | Description | Check Command |
|-----------|-------------|---------------|
| `quota_available` | Resource quota not exceeded | `hcloud <product> listQuotas` |
| `health_check_ok` | Target resource is reachable | `hcloud <product> checkHealth` |
| `backup_exists` | Backup available for target | `hcloud <product> listBackups` |
| `no_active_incident` | No open incident for target | CES alarm query |
| `dry_run_success` | Dry-run execution passes | `hcloud <product> <action> --dry-run` |

---

## 8. Rollback Strategy Matrix

| Rollback Type | Definition | Typical Time | Used For |
|---------------|------------|--------------|----------|
| **Instant revert** | Single API call to restore previous state | < 1 min | Low risk actions |
| **Scripted rollback** | Pre-defined rollback procedure | 1-5 min | Medium risk actions |
| **Manual rollback** | Human operator intervention required | > 5 min | High/Critical actions |
| **Irreversible** | No rollback possible | N/A | Critical, destructive actions |

---

## 9. Skill Coverage Summary

| Skill | Actions Defined | Auto-Executable | Requires Approval |
|-------|----------------|-----------------|-------------------|
| ECS | 10 | 6 | 4 |
| RDS | 10 | 5 | 5 |
| CCE | 10 | 6 | 4 |
| CES | 8 | 5 | 3 |
| ELB | 10 | 5 | 5 |
| **Total** | **48** | **27 (56%)** | **21 (44%)** |

---

## 10. Adding New Actions

To add an action to the catalog, fill in the template:

```markdown
| NEW-A01 | [Scenario] | [Action] | [Low/Medium/High/Critical] | [✅/❌] | [Preconditions] | [Rollback] |
```

**Risk assessment steps**:
1. Evaluate blast radius (instance/AZ/region)
2. Evaluate data impact (none/read/write)
3. Evaluate reversibility (instant/scripted/manual/irreversible)
4. Evaluate cost impact (per hour)
5. Assign highest dimension risk level

**Approval routing**:
- Low/Medium → Auto-execute (no approval needed)
- High → Human approval required (Slack/PagerDuty)
- Critical → Emergency approval (on-call + manager)
