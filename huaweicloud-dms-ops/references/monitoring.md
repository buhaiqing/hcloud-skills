# Monitoring — Huawei Cloud DMS

## CES Metrics

DMS metrics are available in namespace `SYS.DMS`.

### Kafka Metrics

| Metric | Unit | Description | Recommended Alarm Threshold |
|--------|------|-------------|---------------------------|
| `kafka_messages_in_total` | Count | Total messages produced | Monitor trend |
| `kafka_messages_in_rate` | msg/s | Message produce rate | Scale if >70% of max TPS |
| `kafka_messages_out_total` | Count | Total messages consumed | Compare with `_in_total` for lag |
| `kafka_messages_consumer_lag` | Count | Total consumer lag across all groups | Alarm if >100,000 |
| `broker_cpu_usage` | % | Broker CPU utilization | >80% for 10min |
| `broker_memory_usage` | % | Broker memory utilization | >80% for 10min |
| `broker_disk_usage` | % | Broker disk utilization | >85% for 5min |
| `total_connection_count` | Count | Total client connections | >80% of max connections |
| `partition_count` | Count | Total partitions across all topics | Monitor for quota planning |
| `request_time_average` | ms | Average request latency | >100ms for 5min |
| `network_incoming_rate` | bytes/s | Network incoming throughput | Plan bandwidth |
| `network_outgoing_rate` | bytes/s | Network outgoing throughput | Plan bandwidth |

### RabbitMQ Metrics

| Metric | Unit | Description | Recommended Alarm Threshold |
|--------|------|-------------|---------------------------|
| `rabbitmq_messages_ready` | Count | Messages waiting to be consumed | Monitor for backlog |
| `rabbitmq_messages_unacknowledged` | Count | Delivered but unacknowledged | >10,000 |
| `rabbitmq_messages_total` | Count | Total messages in queues | Monitor trend |
| `rabbitmq_queue_length` | Count | Queue depth | >50,000 |
| `node_cpu_usage` | % | Node CPU | >80% for 10min |
| `node_memory_usage` | % | Node memory | >80% for 10min |
| `node_disk_usage` | % | Node disk | >85% for 5min |
| `node_fd_usage` | % | File descriptor usage | >80% |
| `node_socket_usage` | % | Socket usage | >80% |

## Recommended Alarm Rules

```bash
# Alarm: High consumer lag (Kafka)
hcloud CES CreateAlarm \
  --name="dms-high-consumer-lag" \
  --namespace="SYS.DMS" \
  --metric_name="kafka_messages_consumer_lag" \
  --threshold=100000 \
  --comparison_operator="gt" \
  --period=300 \
  --evaluation_periods=2

# Alarm: Disk usage critical
hcloud CES CreateAlarm \
  --name="dms-disk-usage-critical" \
  --namespace="SYS.DMS" \
  --metric_name="broker_disk_usage" \
  --threshold=85 \
  --comparison_operator="gt" \
  --period=300 \
  --evaluation_periods=1
```

## Dashboard Suggestion

| Panel | Metrics | Period |
|-------|---------|--------|
| Message Throughput | `kafka_messages_in_rate`, `kafka_messages_out_rate` | 5min |
| Consumer Lag | `kafka_messages_consumer_lag` | 5min |
| Broker Health | `broker_cpu_usage`, `broker_memory_usage`, `broker_disk_usage` | 5min |
| Network | `network_incoming_rate`, `network_outgoing_rate` | 5min |
| Connections | `total_connection_count` | 5min |
