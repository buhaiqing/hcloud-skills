# Knowledge Base — Huawei Cloud DMS Fault Patterns

> ML feature definitions and anomaly scoring belong in
> [`huaweicloud-ces-ops/references/advanced/aiops-patterns.md`](../../huaweicloud-ces-ops/references/advanced/aiops-patterns.md).
> This file focuses on DMS-specific diagnosis and remediation.

## Pattern: DMS-001 — 消息堆积 (Message Backlog)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `queue_depth` > 10000消息持续 > 5min，或 `backlog_growth_rate` > 1000消息/min |
| 典型特征 | 消费者处理速度跟不上生产者，消息延迟增加，队列深度监控告警 |
| 关联指标 | `produce_rate` 正常，`consume_rate` 下降，`consumer_count` 可能减少 |
| 根因 | 1. 消费者故障/崩溃 2. 消费者处理逻辑变慢 3. 消费者数量不足 4. 消费者OOM |
| 诊断步骤 | 1. `hcloud dms list_consumers <group_id>` 查看消费者状态 2. 检查消费者日志 3. `hcloud dms get_queue <id>` 查看队列指标 4. 监控消费者lag变化趋势 |
| 修复方案 | 1. 立即: 重启异常消费者 2. 扩容消费者数量 3. 优化消费者处理逻辑 4. 临时: 增加消费并发度 |
| 预防措施 | 配置消费者健康检查，设置队列深度告警，监控消费延迟，配置死信队列 |
| 自动修复触发 | `queue_depth > 50000` AND `consumer_lag > 10min` → 自动扩容消费者实例并告警 |

## Pattern: DMS-002 — 消费者组失效 (Consumer Group Failure)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `active_consumers` = 0 持续 > 2min，或 `consumer_group_status` = dead |
| 典型特征 | 消费者组无法消费消息，队列消息持续堆积，组状态显示inactive |
| 关联指标 | `queue_depth` 增长，`consumer_count` = 0，`produce_rate` 正常 |
| 根因 | 1. 消费者进程崩溃 2. 网络分区导致心跳超时 3. 消费者配置错误(group.id重复) 4. Kafka Rebalance风暴 |
| 诊断步骤 | 1. `hcloud dms get_consumer_group <group_id>` 查看组状态 2. 检查消费者实例存活 3. `kafka-consumer-groups.sh --describe` 查看详细状态 4. 分析Rebalance日志 |
| 修复方案 | 1. 重启消费者进程 2. 修复网络问题 3. 修正group.id配置 4. 调整session.timeout/ms和heartbeat.interval.ms |
| 预防措施 | 配置消费者自动重启，监控心跳，设置合理的Rebalance超时参数，使用新版消费者SDK |
| 自动修复触发 | `active_consumers = 0` AND `duration > 3min` → 自动重启消费者并发送告警 |

## Pattern: DMS-003 — 分区再平衡风暴 (Partition Rebalance Storm)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `rebalance_count` > 10次/分钟，或 `rebalance_duration` > 30s |
| 典型特征 | 消费者反复加入/离开组，消息消费中断，分区分配频繁变化，延迟波动大 |
| 关联指标 | `heartbeat_timeout` 升高，`session_timeout` 频繁触发，`network_io` 波动 |
| 根因 | 1. 网络抖动 2. GC pause导致心跳超时 3. 消费者配置不一致 4. 部署时配置变更 |
| 诊断步骤 | 1. 查看consumer日志定位Rebalance原因 2. `kafka-consumer-groups.sh --describe` 查看REBALANCEING状态 3. 检查网络延迟和GC日志 4. 对比消费者配置版本 |
| 修复方案 | 1. 临时: 禁止消费者启动，等待Rebalance结束 2. 调整`session.timeout`(10s→30s)和`max.poll.interval.ms` 3. 优化JVM减少GC |
| 预防措施 | 使用独立线程发送心跳，调整合理的Rebalance超时，禁用JVM GC优化，设置Consistent member.id |
| 自动修复触发 | `rebalance_count > 5/min` 持续5min → 发送告警并建议调整超时参数 |

## Pattern: DMS-004 — 磁盘空间满 (Disk Space Full)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `disk_usage` > 90%，或 `storage_used` / `storage_quota` > 95% |
| 典型特征 | 消息写入失败，生产者报磁盘满错误，新分区无法创建，Kafka Broker不可写入 |
| 关联指标 | `produce_error_rate` 升高，`write_latency` 突增，`retention_cleanup` 停止 |
| 根因 | 1. 消息保留策略过宽 2. 消费者group offset未提交 3. 临时文件堆积 4. 分区副本不同步 |
| 诊断步骤 | 1. `hcloud dms show_instance <id>` 查看磁盘使用 2. `kafka-log-dirs.sh` 查看各分区占用 3. 检查retention配置 4. 定位大topic |
| 修复方案 | 1. 立即: 清理过期消息或增加磁盘 2. 调整retention时间 3. 手动执行log cleanup 4. 联系华为云扩容 |
| 预防措施 | 配置磁盘使用率告警，设置合理的retention策略，配置自动cleanup，监控分区大小 |
| 自动修复触发 | `disk_usage > 85%` → 自动触发日志cleanup，发送预警；>95% → 阻止生产并告警 |

## Pattern: DMS-005 — API限流 (API Rate Limiting)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `api_throttle_count` > 0，或 `api_error_code` = 429 |
| 典型特征 | API调用返回429错误，Producer/Consumer初始化失败，SDK报rate limit exceeded |
| 关联指标 | `api_calls` 接近配额，`produce/consume` 速率异常，`error_rate` 升高 |
| 根因 | 1. 突发大量API调用 2. API配额配置过小 3. 客户端重试风暴 4. 监控查询过于频繁 |
| 诊断步骤 | 1. 查看API调用统计: `hcloud dms show_quota <project_id>` 2. 分析调用来源和时间分布 3. 检查客户端重试配置 4. 定位高频调用操作 |
| 修复方案 | 1. 等待限流窗口过去(通常1分钟) 2. 申请提高API配额 3. 优化客户端重试策略(exponential backoff) 4. 减少监控轮询频率 |
| 预防措施 | 配置API配额告警，实现指数退避重试，合理拆分API调用，缓存查询结果 |
| 自动修复触发 | `api_throttle_count > 10` → 发送告警并自动触发客户端退避 |

## Pattern: DMS-006 — 生产者失败 (Producer Failure)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `produce_error_rate` > 5%，或 `produce_error_code` 非零持续 > 1min |
| 典型特征 | 消息无法发送，队列深度不增长，Producer日志显示错误，消息丢失 |
| 关联指标 | `queue_depth` 不增长，`broker_available` 可能false，`network_latency` 可能升高 |
| 根因 | 1. Broker不可达 2. 消息大小超限 3. 权限不足 4. Topic不存在 5. 消息格式错误 |
| 诊断步骤 | 1. 检查Broker健康: `hcloud dms list_brokers <instance_id>` 2. 查看Producer日志具体错误码 3. 验证Topic存在和权限 4. 检查消息大小限制 |
| 修复方案 | 1. 修复网络连通性 2. 压缩或拆分大消息 3. 修正权限配置 4. 创建Topic或修复Topic配置 5. 修复消息序列化 |
| 预防措施 | 配置Producer重试和幂等，监控Produce成功率，验证Topic配置，设置合理的acks配置 |
| 自动修复触发 | `produce_error_rate > 10%` → 发送告警并触发生产者健康检查 |

## Pattern: DMS-007 — Topic分区不均衡 (Partition Imbalance)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `partition_skew_ratio` > 2:1，或最大分区消息量/最小分区消息量 > 3 |
| 典型特征 | 部分Broker磁盘使用率高，部分低，热点Topic处理不均匀，整体吞吐下降 |
| 关联指标 | `disk_usage` 分布不均，`leader_partition_skew` > 1.5 |
| 根因 | 1. Key分配不均 2. 分区副本分布不均 3. Leader选举不均衡 4. 分区迁移未完成 |
| 诊断步骤 | 1. `hcloud dms list_topic_partitions <topic>` 查看分布 2. `kafka-topics.sh --describe` 分析分区详情 3. 检查各Broker负载 4. 查看preferredreplica分布 |
| 修复方案 | 1. 执行`kafka-preferred-replica-election.sh` 2. 重新分配分区: `kafka-reassign-partitions.sh` 3. 使用自定义Partitioner均衡Key |
| 预防措施 | 使用均衡的Partitioner策略，定期执行preferred replica election，监控分区分布 |
| 自动修复触发 | `partition_skew > 2.5` → 自动触发preferred replica election并告警 |

## Pattern: DMS-008 — 消费者延迟 (Consumer Lag)

| Attribute | Content |
|-----------|---------|
| 触发指标 | `consumer_lag` > 50000消息，或 `lag_time` > 30min |
| 典型特征 | 消费者处理速度落后生产者，新消息消费延迟增加，监控显示lag增长 |
| 关联指标 | `produce_rate` > `consume_rate`，`consumer_cpu` 可能高，`poll_latency` 可能高 |
| 根因 | 1. 消费者处理能力不足 2. 单条消息处理时间过长 3. 消费者实例不足 4. 内存不足导致频繁GC |
| 诊断步骤 | 1. `hcloud dms get_consumer_group <group>` 查看各分区lag 2. 分析消费者处理时间分布 3. 检查CPU/内存指标 4. 查看GC日志 |
| 修复方案 | 1. 增加消费者实例 2. 优化处理逻辑减少单条消息时间 3. 扩容消费者资源 4. 批量处理+异步提交offset |
| 预防措施 | 配置lag告警，监控消费者资源，评估消费者吞吐量，预估容量 |
| 自动修复触发 | `consumer_lag > 100000` AND `lag_growth > 10000/min` → 发送告警并建议扩容消费者 |

---

*Knowledge Base version 1.0.0 — for DMS AIOps L2 compliance*
