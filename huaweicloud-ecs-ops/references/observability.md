# Observability Integration — Huawei Cloud ECS

## Metrics → Logs → Traces Pipeline

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│ Metrics   │────▶│  Logs    │────▶│ Traces   │
│  (CES)    │     │  (LTS)   │     │  (AOM)   │
└──────────┘     └──────────┘     └──────────┘
                       ▼
              ┌─────────────────┐
              │ Unified Report  │
              └─────────────────┘
```

## Metrics → Logs Linkage

| CES Metric Anomaly | LTS Query Target | Purpose |
|-------------------|-----------------|---------|
| `cpu_util` spike | Application error logs | Confirm error burst causing CPU surge |
| `mem_usedPercent` leak | Application memory logs | Confirm allocation pattern |
| `diskUsage_percent` > 90% | /var/log/ large file search | Identify file consuming space |
| `load1` > vCPU count | System audit log | Confirm process explosion |

### LTS Log Query Examples

```
# Query ECS application errors in last 1 hour
SELECT * FROM lts_log
WHERE stream_name = "{{user.instance_name}}-applog"
AND content LIKE '%ERROR%'
AND timestamp > now() - INTERVAL 1 HOUR

# Query system logs for OOM events
SELECT * FROM lts_log
WHERE stream_name = "{{user.instance_name}}-syslog"
AND (content LIKE '%oom%' OR content LIKE '%killed%')

# Query SSH login attempts
SELECT * FROM lts_log
WHERE stream_name = "{{user.instance_name}}-auth.log"
AND content LIKE '%sshd%'
```

## Metrics → Traces Linkage

| CES Anomaly | AOM Trace Target | Purpose |
|-------------|-----------------|---------|
| `cpu_util` spike | Application Trace ID → Span | Locate hot methods |
| High request latency | ECS→ELB→Backend Trace | Locate bottleneck service |
| Error rate increase | Error Trace by service | Locate error root cause |

### Degradation Strategy

If LTS/AOM skills are unavailable:
1. Use CloudShell remote execution to collect logs directly (`journalctl`, `tail -f`)
2. Use LTS API directly via Go SDK
3. Provide console link for manual troubleshooting
