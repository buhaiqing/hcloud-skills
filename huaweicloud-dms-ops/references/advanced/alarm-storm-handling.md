# Alarm Storm Handling — DMS

> **Purpose**: Guidance for detecting and mitigating alarm storms involving Distributed Message Service (Kafka / RabbitMQ).
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

DMS alarm storms come from queue backlog, broker resource exhaustion, and throughput spikes. Signals sourced from CES namespaces `SYS.DMS` (Kafka / RabbitMQ).

| Pattern | Indicators | Severity |
|---------|-----------|----------|
| Consumer lag pile-up | `kafka_messages_consumer_lag` > 100000 | Critical |
| Broker disk pressure | `broker_disk_usage` > 85% | Critical |
| Broker resource | `broker_cpu` / `broker_memory` > 80% | Warning |
| Producing burst | `kafka_messages_in_rate` > 70% of max TPS | Warning |
| RabbitMQ backlog | `rabbitmq_queue_length` > 50000 OR `messages_unack` > 10000 | Critical |

```bash
# Query consumer lag from CES
hcloud ces metric-data-query \
  --metric_name=kafka_messages_consumer_lag \
  --namespace=SYS.DMS \
  --dim.0=instance_id:{{user.instance_id}} \
  --start_time=$(date -v-30m +%s) --end_time=$(date +%s) --period=300

# Compare in/out rate to distinguish producer burst vs consumer stall
hcloud ces metric-data-query \
  --metric_name=kafka_messages_in_total \
  --namespace=SYS.DMS \
  --dim.0=instance_id:{{user.instance_id}} \
  --start_time=$(date -v-30m +%s) --end_time=$(date +%s) --period=300
```

---

## 2. Aggregation Rules

- **Producer vs consumer disambiguation**: Compare `kafka_messages_consumer_lag` against `_in_total` / `_out_total`. If `_in_total` spikes while lag grows → "producing burst"; if `_out_total` flat while lag grows → "consumer stall". Suppress the wrong-class alarm.
- **Broker collapse**: `broker_cpu` + `broker_memory` + `broker_disk_usage` on the same broker within 5 min → single broker-incident, not 3 alarms.
- **Topic grouping**: Aggregate lag alarms per topic partition to avoid per-partition noise.

---

## 3. Suppression Rules

| Scenario | Suppression |
|----------|-------------|
| Planned rebalance / partition migration | Suppress broker resource alarms 2x duration |
| Known producer campaign | Suppress lag alarm if `_in_total` burst is expected (15 min window) |
| Consumer stall under maintenance | Suppress lag duplicates once root cause confirmed |

```bash
# Suppress a CES alarm during maintenance (example — adjust args)
hcloud ces alarm-action modify \
  --alarm_id <alarm-id> \
  --suppress_duration 3600
```

---

## 4. Response Procedures

### Phase 1: Triage (0-5 min)
1. Run detection commands; classify as producer burst or consumer stall.

### Phase 2: Consumer Stall
```bash
# Inspect consumer groups (placeholder — verify subcommand)
hcloud dms show-consumer-group --instance {{user.instance_id}} --group {{user.group}}
# Scale consumer pods via CCE or restart stalled consumers
```

### Phase 3: Broker Pressure
- Expand broker storage / add broker; verify `broker_disk_usage` < 85%.

### Phase 4: Post-Incident
- Confirm lag trending to zero; document novel pattern.

---

## 5. Delegation Matrix

| Trigger | Delegate To |
|---------|-------------|
| Network / subnet for brokers | `huaweicloud-vpc-ops` |
| Permission / AK issues | `huaweicloud-iam-ops` |
| Metric gaps / alarm config | `huaweicloud-ces-ops` |
| Audit of admin actions | `huaweicloud-cts-ops` |
| Consumer pod scaling | `huaweicloud-cce-ops` |
| Image / package pull | `huaweicloud-swr-ops` |
