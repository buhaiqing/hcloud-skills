# Knowledge Base — Huawei Cloud GaussDB Fault Patterns

> ML feature definitions and anomaly scoring belong in
> [`huaweicloud-ces-ops/references/advanced/aiops-patterns.md`](../../huaweicloud-ces-ops/references/advanced/aiops-patterns.md).
> This file focuses on GaussDB-specific diagnosis and remediation.

## Pattern: GaussDB-001 — 主备切换导致连接中断

| Attribute | Content |
|-----------|---------|
| 触发指标 | `replica_lag` > 10MB持续30s，或 `conn_count`/`max_connections` > 90% |
| 典型特征 | 应用报连接失败，SQL执行报"Connection lost"，主备状态均为"standalone" |
| 关联指标 | `cpu_usage` 正常，`disk_usage` 正常，`tlog_size` 增长 |
| 根因 | 1. 华为云AZ级故障 2. 主备间网络抖动 3. 主节点CPU/内存过载导致HA心跳超时 |
| 诊断步骤 | 1. 检查CES告警历史 2. `hcloud gaussdb show_instance <id>` 查看实例状态 3. `hcloud gaussdb list_instances` 对比AZ分布 4. 检查CTS审计日志最近的配置变更 |
| 修复方案 | 1. 等待HA自动恢复(通常<60s) 2. 若超过3分钟无恢复: `hcloud gaussdb restart_instance <id>` 3. 切换应用重连逻辑 |
| 预防措施 | 多AZ部署，选择性配置`multi_az_policy=balance`，开启连接池代理(HWSQL) |
| 自动修复触发 | `ha_heartbeat_timeout` AND `primary_switch_required` → 自动触发主备切换并发送告警 |

## Pattern: GaussDB-002 — 连接池耗尽 (Connection Pool Exhaustion)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `conn_count`/`max_connections` > 95% 持续 > 5min，或 `conn_error_rate` > 10/s |
| 典型特征 | 应用报"Too many connections"，新连接建立失败，已有连接正常响应 |
| 关联指标 | `active_sessions` > 500，`waiting_queries` > 50，`query_response_time` 上升 |
| 根因 | 1. 突发流量激增 2. 慢查询占用连接 3. 应用连接泄漏(未关闭连接) 4. `max_connections` 配置过小 |
| 诊断步骤 | 1. `hcloud gaussdb list_connections <id>` 查看连接分布 2. `show processlist` 定位占用连接的SQL 3. 检查应用侧连接池配置 4. 分析连接创建/销毁趋势 |
| 修复方案 | 1. 立即: `hcloud gaussdb restart_instance <id>` 重建连接 2. 长期: 调大`max_connections`，优化慢查询，修复连接泄漏 |
| 预防措施 | 设置`max_connections_alarm_threshold=80%`，配置连接池最小空闲连接数，应用侧正确关闭连接 |
| 自动修复触发 | `conn_util > 95%` AND `duration > 5min` → 发送高危告警并触发自动kill空闲连接脚本 |

## Pattern: GaussDB-003 — 存储空间满 (Storage Full)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `disk_usage` > 90%，或 `tlog_size`/`data_disk_ratio` > 80% |
| 典型特征 | 写入SQL报错"磁盘满"，事务无法提交，实例变为只读，SSH可能正常但数据库操作失败 |
| 关联指标 | `wal_size` 持续增长，`write_iops` 下降，`cpu_usage` 可能正常或升高(Compaction) |
| 根因 | 1. 数据文件快速增长(批量导入) 2. 事务日志堆积(WAL) 3. 临时文件未清理 4. 批量DELETE未回收空间 |
| 诊断步骤 | 1. `hcloud gaussdb show_instance <id>` 查看存储使用详情 2. `select pg_size_pretty(pg_database_size())` 3. 检查表膨胀: `SELECT tablename, pg_size_pretty(pg_total_relation_size()) FROM pg_tables ORDER BY DESC LIMIT 10` |
| 修复方案 | 1. 立即: DELETE批量清理无用数据，或VACUUM FULL回收空间 2. 联系华为云扩容磁盘 3. 开启自动扩容: `hcloud gaussdb modify_instance_auto_scale <id>` |
| 预防措施 | 配置磁盘使用率告警(80%预警)，设置`autovacuum`自动清理，开启表级存储配额 |
| 自动修复触发 | `disk_usage > 85%` → 自动触发异步 vacuum，主动扩容磁盘 |

## Pattern: GaussDB-004 — CPU过载 (CPU Overload)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `cpu_usage` > 90% 持续 > 10min，`qps` 正常或下降 |
| 典型特征 | 查询响应变慢，连接建立延迟，CES指标显示CPU使用率红色告警 |
| 关联指标 | `active_sessions` 高，`lock_wait_count` 可能升高，`io_util` 可能升高 |
| 根因 | 1. 复杂JOIN/全表扫描 2. 统计信息陈旧 3. 批量复杂分析查询 4. 并发连接过多 |
| 诊断步骤 | 1. `hcloud gaussdb show_queries <id>` 查看当前查询 2. `SELECT * FROM pg_stat_activity WHERE state='active'` 3. `EXPLAIN ANALYZE` 分析慢SQL 4. 检查`pg_stat_user_indexes`看索引使用率 |
| 修复方案 | 1. 立即: `SELECT pg_terminate_backend(pid)` 终止高消耗查询 2. 长期: 优化SQL，添加索引，更新统计信息`ANALYZE` |
| 预防措施 | 配置慢查询日志(>1s记录)，开启查询限流，设置`statement_timeout`，定期`ANALYZE` |
| 自动修复触发 | `cpu_usage > 95%` AND `active_session > 100` → 自动终止最慢的5个查询 |

## Pattern: GaussDB-005 — 事务锁等待/死锁 (Transaction Lock Wait/Deadlock)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `lock_wait_count` > 50，或 `deadlock_count` > 0 |
| 典型特征 | 多个事务同时等待，SQL执行超时，应用报事务等待超时，单个事务回滚 |
| 关联指标 | `active_sessions` 稳定，`cpu_usage` 正常，`io_util` 可能升高 |
| 根因 | 1. 不同事务交叉更新相同行 2. 长时间运行事务未提交 3. 显式锁冲突(行锁升级为表锁) |
| 诊断步骤 | 1. `SELECT * FROM pg_locks WHERE granted=false` 查看等待锁 2. `SELECT pg_blocking_pids(pid)` 3. `SELECT * FROM pg_stat_activity WHERE wait_event_type='Lock'` 4. 定位持有锁的事务并分析 |
| 修复方案 | 1. 终止等待事务: `SELECT pg_cancel_backend(pid)` 2. 回滚大事务: `BEGIN; ROLLBACK;` 3. 优化事务范围，减少锁持有时间 |
| 预防措施 | 减少事务内操作数量，按固定顺序访问数据，避免长事务，设置`lock_timeout` |
| 自动修复触发 | `deadlock_count > 0` → 自动终止检测到的死锁事务并发送告警 |

## Pattern: GaussDB-006 — 复制延迟 (Replication Lag)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `replica_lag` > 1GB，或 `replica_lag_time` > 300s |
| 典型特征 | 备机数据落后主机，读取备机应用看到过期数据，HA切换时RPO增加 |
| 关联指标 | `wal_generate_rate` 高，`replay_rate` 低，`network_latency` 可能高 |
| 根因 | 1. 备机IO瓶颈 2. 主库写入过快(大事务) 3. 网络带宽不足 4. 备机CPU/内存过载 |
| 诊断步骤 | 1. `hcloud gaussdb show_replication <id>` 查看延迟详情 2. 在备机执行`pg_stat_replication`查看复制状态 3. `select * from pg_stat_replication` 对比`sent_lsn`和`write_lsn` |
| 修复方案 | 1. 备机IO高: 考虑升级备机规格或迁移到SSD 2. 大事务: 拆分为小事务 3. 网络问题: 检查VPC带宽或启用压缩复制 |
| 预防措施 | 监控复制延迟，设置延迟告警，大事务使用分批提交，选择合适的复制模式(同步/异步) |
| 自动修复触发 | `replica_lag > 500MB` AND `duration > 10min` → 发送预警并触发复制优化建议 |

## Pattern: GaussDB-007 — 慢查询堆积 (Slow Query Accumulation)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `slow_query_count` > 100/hour，或平均执行时间 > 5s |
| 典型特征 | 新查询排队等待，QPS下降，连接数可能增加，CPU使用率可能升高 |
| 关联指标 | `avg_query_time` 升高，`active_sessions` 稳定，`lock_wait` 可能升高 |
| 根因 | 1. 缺少索引 2. 统计信息陈旧 3. 查询计划选择错误 4. 大表JOIN |
| 诊断步骤 | 1. `hcloud gaussdb list_slow_queries <id>` 获取慢查询列表 2. `EXPLAIN ANALYZE` 分析执行计划 3. `pg_stats` 查看表统计信息 4. 检查索引有效性 |
| 修复方案 | 1. 立即: 终止问题查询 2. 长期: 优化SQL，添加索引，`ANALYZE`更新统计信息，调整`random_page_cost` |
| 预防措施 | 开启慢查询日志，设置合理的`random_page_cost`，定期`ANALYZE`，使用覆盖索引 |
| 自动修复触发 | `slow_query_count > 50` in 10min → 自动生成优化建议报告并发送告警 |

## Pattern: GaussDB-008 — 备份失败 (Backup Failure)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `backup_status` = failed，或 `last_backup_age` > 25小时 |
| 典型特征 | CBR备份任务失败，备份历史显示红色失败标记，恢复点RPO增大 |
| 关联指标 | `disk_usage` 可能>90%，`backup_duration` 异常长，`vault_quota` 可能不足 |
| 根因 | 1. 磁盘空间不足 2. CBR Vault配额不足 3. 实例状态非running 4. 网络问题 |
| 诊断步骤 | 1. `hcloud cbr vault list` 检查Vault状态 2. `hcloud gaussdb show_backup <id>` 查看失败详情 3. 检查磁盘使用率 4. 检查Vault配额 |
| 修复方案 | 1. 清理磁盘空间或扩容 2. 清理过期备份释放Vault空间 3. 重新触发备份: `hcloud gaussdb create_backup <id>` |
| 预防措施 | 配置CBR Vault自动扩容，监控磁盘使用率，设置备份成功/失败告警 |
| 自动修复触发 | `backup_failed = true` → 自动重试备份(最多3次)，失败后发送告警 |

---

*Knowledge Base version 1.0.0 — for GaussDB AIOps L2 compliance*
