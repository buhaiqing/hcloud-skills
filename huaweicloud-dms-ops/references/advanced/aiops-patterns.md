# AIOps Patterns — DMS

> **Purpose**: DMS (消息队列 Kafka/RabbitMQ) 专属异常检测模式，基于 CES 命名空间 `SYS.DMS` 真实指标。
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `broker_disk_full` | `broker_disk_usage > 85%` | Critical | 清理过期消息/扩容磁盘，防写阻塞 |
| `broker_overload` | `broker_cpu > 80%` 或 `broker_memory > 80%` | Major | 排查热点分区，水平扩容 broker |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `consumer_lag_growth` | `kafka_messages_consumer_lag > 100000` 且持续上升 | Major | 扩容消费组/优化消费逻辑 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `produce_burst` | `kafka_messages_in_rate > 70%` 实例 max TPS | Major | 区分生产突增：比 `_in_total`/`_out_total` 判别，限流上游 |
| `consume_stall` | `consumer_lag` 升但 `_in_total` 平稳（消费停滞） | Critical | 查消费端死锁/慢处理，重启消费实例 |
| `mq_backlog` | `rabbitmq_queue_length > 50000` 或 `messages_unack > 10000` | Major | 检查消费者健康与 prefetch 配置 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `lag_in_out_correlation` | 对比 `consumer_lag` 与 `_in_total`/`_out_total` 区分"生产突增" vs "消费停滞" | Major | 按根因分流：上游限流 or 消费端扩容 |

---

## 2. Alarm Storm Handling

仅交叉引用，不重复内容：`详见 references/advanced/alarm-storm-handling.md`

---

## 3. Root Cause Analysis

1. **堆积** → 比对 `consumer_lag` 与 `_in_total`/`_out_total` → 判定生产突增或消费停滞 → 对应限流上游或扩容消费端。
2. **磁盘打满** → 查 `broker_disk_usage` 与保留策略 → 缩短 retention 或扩磁盘 → 防生产者被阻塞。
3. **Broker 过载** → 关联 `broker_cpu`/`broker_memory` 与热点分区 → 重平衡分区。
4. **RabbitMQ 积压** → 查 `queue_length`/`messages_unack` → 消费端异常或 prefetch 过低 → 修复消费者。
