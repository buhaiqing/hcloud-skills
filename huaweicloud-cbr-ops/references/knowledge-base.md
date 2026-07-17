# Knowledge Base — Huawei Cloud CBR Fault Patterns

> Fault patterns for Cloud Backup and Recovery (CBR). Each entry describes
> symptoms, root cause, diagnosis steps, and remediation.

## Pattern: CBR-001 — 备份任务超时

| Attribute | Content |
|-----------|---------|
| 触发指标 | `backup_job_status = failed` 且 `failure_reason` 包含 `timeout` |
| 典型特征 | 备份任务长时间处于 `running` 状态后最终失败 |
| 关联指标 | `backup_duration` 持续增长, `vault_storage_used` 正常 |
| 根因 | 1. 备份数据量大超过超时阈值 2. 存储后端响应慢 3. 并发备份过多 |
| 诊断步骤 | 1. `hcloud cbr backup-list --vault-id <vault_id>` 检查任务状态 2. 检查vault容量是否接近上限 3. 检查目标存储OBS连通性 |
| 修复方案 | 1. 增加备份超时时间 `backup_timeout` 参数 2. 减少单次备份数据量 3. 错峰执行备份 |
| 预防措施 | 配置备份任务监控告警, 合理规划备份窗口和并发数 |
| 自动修复触发 | `backup_duration > 7200s` AND `status = running` → Alert ops team, do not auto-retry |

## Pattern: CBR-002 — 恢复失败

| Attribute | Content |
|-----------|---------|
| 触发指标 | `restore_job_status = failed` |
| 典型特征 | 恢复任务启动后立即失败或进度停滞 |
| 关联指标 | `restore_objects_count` 与预期不符, `vault_backup_count` 减少 |
| 根因 | 1. 备份集损坏或不完整 2. 目标资源不存在或状态异常 3. 权限不足 |
| 诊断步骤 | 1. `hcloud cbr restore-list --backup-id <backup_id>` 查看失败原因 2. 验证目标资源状态 `hcloud cbr vault-list` 3. 检查IAM策略是否包含 `cbr:restores:create` |
| 修复方案 | 1. 重新执行备份确保完整性 2. 确认目标资源状态正常 3. 修正IAM权限 |
| 预防措施 | 定期执行备份恢复演练, 验证备份完整性 |
| 自动修复触发 | `restore_status = failed` → Alert ops team, capture failure reason |

## Pattern: CBR-003 — 存储配额不足

| Attribute | Content |
|-----------|---------|
| 触发指标 | `vault_storage_used / vault_storage_total > 90%` |
| 典型特征 | 新备份任务失败, vault状态变为 `limited` |
| 关联指标 | `backup_failure_rate` 上升, `vault_object_count` 达到配额 |
| 根因 | 1. 备份保留策略过宽松 2. 备份数据量增长未预估 3. 存储包规格选择过小 |
| 诊断步骤 | 1. `hcloud cbr vault-list` 检查各vault存储使用率 2. `hcloud cbr backup-list --vault-id <vault_id>` 统计备份数量 3. 分析备份保留策略 |
| 修复方案 | 1. 立即: 删除过期备份 `hcloud cbr backup-delete` 2. 扩容存储包 3. 调整备份保留周期 |
| 预防措施 | 配置存储使用率告警 (80%预警), 定期审计备份保留策略 |
| 自动修复触发 | `storage_used > 90%` → Alert ops team, do not auto-delete (data loss risk) |

## Pattern: CBR-004 — 策略调度失败

| Attribute | Content |
|-----------|---------|
| 触发指标 | `policy_trigger_status = failed` 或 `policy_last_run_time` 停滞 |
| 典型特征 | 定时备份未执行, vault备份数量未增加 |
| 关联指标 | `backup_schedule_missed_count` > 0, `policy_enabled = true` |
| 根因 | 1. 策略资源被删除但调度仍触发 2. 备份策略与vault绑定关系异常 3. 系统调度服务瞬时故障 |
| 诊断步骤 | 1. `hcloud cbr policy-list` 检查策略状态 2. `hcloud cbr vault-list` 检查vault绑定的策略 3. 查看CTS审计日志定位删除操作 |
| 修复方案 | 1. 重新创建并绑定策略 2. 手动触发一次备份验证 3. 检查系统健康状态 |
| 预防措施 | 策略变更需关联CTS告警, 策略删除前检查依赖关系 |
| 自动修复触发 | `policy_last_run_time` stale > 48h → Alert ops team |

## Pattern: CBR-005 — 复制延迟

| Attribute | Content |
|-----------|---------|
| 触发指标 | `replication_lag > 3600s` (复制延迟超过1小时) |
| 典型特征 | 跨区域复制备份长时间处于 `replicating` 状态 |
| 关联指标 | `replication_bandwidth` 低于预期, 目标区域vault存储未增加 |
| 根因 | 1. 跨区域网络带宽受限 2. 目标区域存储资源不足 3. 复制任务排队等待 |
| 诊断步骤 | 1. `hcloud cbr replication-list` 查看复制状态 2. 检查源和目标区域网络连通性 3. 检查目标vault配额 |
| 修复方案 | 1. 错峰执行复制任务 2. 扩容跨区域带宽 3. 清理目标区域过期备份 |
| 预防措施 | 配置复制延迟监控告警, 跨区域复制选择低峰时段 |
| 自动修复触发 | `replication_lag > 7200s` → Alert ops team |

## Pattern: CBR-006 — Vault资源锁定

| Attribute | Content |
|-----------|---------|
| 触发指标 | `vault_status = locked` 且所有备份操作返回 403 |
| 典型特征 | 无法创建新备份, 恢复操作也失败 |
| 关联指标 | `vault_protection_status = protection_enabled`, 账户存在欠费 |
| 根因 | 1. 华为云账户欠费触发资源锁定 2. 合规策略自动锁定 3. 手动执行锁定操作 |
| 诊断步骤 | 1. 检查华为云账户余额和账单状态 2. `hcloud cbr vault-list` 查看vault状态 3. 检查CTS日志查找锁定操作来源 |
| 修复方案 | 1. 充值账户解除欠费 2. 如为合规锁定, 联系技术支持 3. 手动解锁vault |
| 预防措施 | 配置账户余额告警, 开启自动充值 |
| 自动修复触发 | `vault_status = locked` → Alert billing team immediately |
