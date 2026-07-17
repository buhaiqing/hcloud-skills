# Observability Trinity — Template

> **Purpose**: Template for Metrics → Logs → Traces linkage rules.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Observability Trinity Overview

| Component | Data Source | Purpose |
|-----------|-------------|---------|
| Metrics | CES (Cloud Eye Service) | Quantitative measurements (CPU%, latency ms) |
| Logs | LTS (Log Tank Service) | Discrete events, errors, debug info |
| Traces | APM (Application Performance Management) | Request flow across services |

## 2. Linkage Rules

### 2.1 Metric → Log Linkage

| When metric alerts | Check logs |
|--------------------|-----------|
| CPU > 90% | LTS: `top` output, process logs |
| Latency > 500ms | LTS: slow query log, GC logs |
| Error rate > 1% | LTS: error logs, stack traces |
| Connection saturated | LTS: connection pool logs |

### 2.2 Log → Metric Linkage

| When log pattern detected | Check metrics |
|--------------------------|--------------|
| OOM errors | Memory usage metric |
| Connection timeouts | Connection count metric |
| Slow queries | Query latency metric |
| Disk full | Disk usage metric |

### 2.3 Trace → Metric/Log Linkage

| When trace shows latency | Check metrics + logs |
|-------------------------|---------------------|
| Span duration > threshold | CPU/memory of specific service |
| Error in trace | Error logs of specific service |
| Timeout in trace | Downstream service health |

## 3. Data Source Mapping

| Product | Metrics (CES) | Logs (LTS) | Traces (APM) |
|---------|--------------|-------------|---------------|
| ECS | SYS.ECS | Instance logs, system logs | APM traces |
| CCE | SYS.CCE | Container logs, pod logs | APM traces |
| RDS | SYS.RDS | Slow query log, error log | APM traces |
| DCS | SYS.DCS | Redis command log | — |
| ELB | SYS.ELB | Access log | — |

## 4. Correlation Query Examples

```bash
# 1. Metric alert → Find related logs
ALERT_METRIC="cpu_usage"
ALERT_VALUE=95
RESOURCE_ID="ecs-xxxxx"
TIME_RANGE="-30m"

# Query LTS for process logs
hcloud lts query-log \
  --log-group-id "$LOG_GROUP" \
  --log-stream-id "$LOG_STREAM" \
  --start-time "$(( $(date +%s) * 1000 - 30 * 60 * 1000 ))" \
  --end-time "$(date +%s)" \
  --keywords "cpu|process" \
  --output json

# 2. Log pattern → Find related metrics
ERROR_PATTERN="Connection timeout"
TIME_RANGE="-5m"

# Query CES for connection metrics
hcloud ces query-metric-data \
  --namespace "SYS.ECS" \
  --metric-name "tcp_curr_estab" \
  --dimension "instance_id=$RESOURCE_ID" \
  --period 60 \
  --from "$(( $(date +%s) - 300 ))" \
  --to "$(date +%s)"
```

## 5. Compliance Checklist

- [ ] Metrics → Logs linkage defined
- [ ] Logs → Metrics linkage defined
- [ ] Trace → Metric/Log linkage defined
- [ ] Data source mapping documented
- [ ] Correlation query examples provided
