# Knowledge Base — Huawei Cloud RDS Fault Patterns

> ML feature definitions and anomaly scoring belong in
> [`huaweicloud-ces-ops/references/advanced/aiops-patterns.md`](../../huaweicloud-ces-ops/references/advanced/aiops-patterns.md).
> This file focuses on RDS-specific diagnosis and remediation.

## Pattern: RDS-001 — 主备切换导致连接中断

| Attribute | Content |
|-----------|---------|
| 触发指标 | `ha_switch_virtual_ip` 状态变化，或 `rds_ha_lag` > 5s |
| 典型特征 | 应用连接报错 `Lost connection to MySQL server`，数据库连接全部断开 |
| 关联指标 | `rds_replica_lag` 突增, `active_connections` 归零又重建 |
| 根因 | 1. 主备切换（规格变更/手动切换/故障切换） 2. 网络闪断 3. 主节点异常宕机 |
| 诊断步骤 | 1. `hcloud rds list ha instance` 检查主备状态 2. 查看 CTS 审计日志 `rds_ha_switch` 事件 3. 检查实例规格变更记录 |
| 修复方案 | 1. 立即: 确认新主节点已接管，业务重连机制自动恢复 2. 验证数据同步完整性 3. 检查应用连接池配置 |
| 预防措施 | 应用层配置连接池重连 + 健康检查，避免长连接直连主节点 |
| 自动修复触发 | `ha_status != primary` AND `duration > 30s` → Alert ops + 确认新主选举完成 |

## Pattern: RDS-002 — 连接池耗尽

| Attribute | Content |
|-----------|---------|
| 触发指标 | `rds_connections_used` / `rds_max_connections` > 90% |
| 典型特征 | 应用报 `Too many connections`，新建连接被拒绝 |
| 关联指标 | `rds_cpu_usage` 可能正常, `rds_active_sessions` 持续高位 |
| 根因 | 1. 应用连接泄漏（未关闭连接） 2. 连接池 max_connections 过小 3. 慢查询阻塞连接 4. 突发大量并发请求 |
| 诊断步骤 | 1. `hcloud rds list slow query` 检查慢查询 2. `show processlist` 分析活跃连接 3. 检查连接等待时间 `rds_connection_wait_time` |
| 修复方案 | 1. 立即: 杀掉长时间空闲连接 2. 短期: 临时扩大 max_connections 3. 长期: 修复应用连接泄漏，优化慢查询 |
| 预防措施 | 配置连接数告警阈值(80%)，连接池配置合理性检查，慢查询优化 |
| 自动修复触发 | `connection_usage > 90%` AND `duration > 5min` → Alert + 触发连接池健康报告 |

## Pattern: RDS-003 — 慢查询导致性能下降

| Attribute | Content |
|-----------|---------|
| 触发指标 | `rds_slow_query_count` 突增，或 `rds_query_duration_p99` > 预设阈值 |
| 典型特征 | 应用响应时间增加，数据库 CPU 使用率升高 |
| 关联指标 | `rds_disk_io_usage` 可能升高, `rds_lock_wait_count` 增加 |
| 根因 | 1. 缺失索引导致全表扫描 2. 查询语句不合理（SELECT *） 3. 数据量增长未优化 4. 统计信息过期 |
| 诊断步骤 | 1. `hcloud rds list slow query` 获取慢查询列表 2. `explain` 分析执行计划 3. 检查索引使用情况 4. 查看表统计信息 |
| 修复方案 | 1. 立即: 添加缺失索引 2. 短期: 优化查询语句 3. 长期: 定期分析表统计信息，调整查询 |
| 预防措施 | 上线前 SQL review，配置慢查询告警，定期执行 `analyze table` |
| 自动修复触发 | `slow_query_count > 100/min` OR `query_duration_p99 > 5000ms` → Alert + 自动收集执行计划 |

## Pattern: RDS-004 — 存储空间满

| Attribute | Content |
|-----------|---------|
| 触发指标 | `rds_disk_usage` > 85% |
| 典型特征 | 写入失败，报 `Disk full` 错误，数据库只读 |
| 关联指标 | `rds_binlog_size` 可能过大, `rds_temp_file_usage` 突增 |
| 根因 | 1. 业务数据快速增长 2. 慢查询产生大量临时表 3. binlog 未清理 4. 审计日志未归档 |
| 诊断步骤 | 1. `hcloud rds list instance detail` 查看磁盘使用详情 2. 检查 binlog 保留策略 3. 识别大表 `hcloud rds list database size` 4. 检查日志文件大小 |
| 修复方案 | 1. 立即: 删除无用数据或扩展磁盘 2. 清理 binlog 3. 启用自动存储扩展 4. 优化数据归档策略 |
| 预防措施 | 配置磁盘使用率告警(75%预警)，设置自动扩容策略，定期归档清理 |
| 自动修复触发 | `disk_usage > 85%` → Alert + 自动扩展磁盘（需开启自动扩容）+ 通知ops |

## Pattern: RDS-005 — 数据库只读

| Attribute | Content |
|-----------|---------|
| 触发指标 | `rds_readonly_status` = true 或写入操作失败 |
| 典型特征 | 应用写入报错 `Read-only connection`，主从复制可能异常 |
| 关联指标 | `rds_replica_lag` 可能异常, `rds_ha_role` 可能变化 |
| 根因 | 1. 主从复制延迟过大 2. 存储空间不足触发只读保护 3. 主备切换后角色变化 4. 参数组配置错误 |
| 诊断步骤 | 1. `hcloud rds list ha instance` 检查主从状态 2. 查看复制延迟 `rds_replica_lag` 3. 检查磁盘空间 4. 查看参数组配置 |
| 修复方案 | 1. 立即: 确认只读原因 2. 若是空间问题，清除数据或扩容 3. 若是复制问题，等待复制追上或跳过错误事务 |
| 预防措施 | 监控复制延迟，配置合理告警阈值，确保磁盘空间充足 |
| 自动修复触发 | `readonly_status = true` → Alert + 自动判断原因（空间/复制/角色）+ 提供修复建议 |

## Pattern: RDS-006 — 备份失败

| Attribute | Content |
|-----------|---------|
| 触发指标 | `rds_backup_status` = failed 或 `rds_backup_last_success_time` 超过预期周期 |
| 典型特征 | 备份任务未执行，RTO 窗口内无有效备份 |
| 关联指标 | `rds_disk_usage` 可能过高, `rds_instance_status` 可能异常 |
| 根因 | 1. 磁盘空间不足无法生成备份 2. 实例负载过高 3. 备份策略配置错误 4. 委托权限过期 |
| 诊断步骤 | 1. `hcloud rds list backup` 查看备份任务状态 2. 检查磁盘空间 3. 查看 CTS 审计日志 `rds_backup` 相关事件 4. 验证 OBS 委托权限 |
| 修复方案 | 1. 立即: 清理空间后手动触发备份 2. 检查备份策略配置 3. 更新 OBS 委托权限 4. 确认备份成功 |
| 预防措施 | 配置备份失败告警，监控磁盘空间，验证 OBS 委托有效期 |
| 自动修复触发 | `backup_failed = true` OR `time_since_last_backup > 25h` → Alert + 验证备份配置 + 通知ops |
