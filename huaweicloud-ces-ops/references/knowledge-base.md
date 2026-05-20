# CES Knowledge Base — Huawei Cloud Cloud Eye Service

## Alarm Fault Patterns

### Pattern 1: Alarm Does Not Fire Despite Metric Breach

| Symptom | Metrics visible in dashboard but alarm stays "ok" |
|---------|---------------------------------------------------|
| Root Cause | Alarm dimension does not match resource dimension |
| Diagnosis | Compare alarm's `metric_dimension` value with actual resource ID |
| Resolution | Re-create alarm with correct dimension key/value |

### Pattern 2: Excessive Alarm Notifications (Noise)

| Symptom | Hundreds of SMS/email notifications per hour |
|---------|--------------------------------------------|
| Root Cause | Period too short (60s) + evaluation_periods = 1 |
| Diagnosis | Check alarm period and evaluation_periods |
| Resolution | Change period to 300s, evaluation_periods to 3; use hysteresis |

### Pattern 3: Metric Data Missing for Resource

| Symptom | Empty datapoints array for a known running resource |
|---------|-----------------------------------------------------|
| Root Cause | Resource in wrong region, agent not installed, or namespace mismatch |
| Diagnosis |
| 1. Check resource status in correct region → Use respective product skill |
| 2. For host metrics, check agent status → `systemctl status ces-agent` |
| 3. Verify namespace matches resource type → SYS.ECS for instances |
| Resolution | Install agent, fix namespace, or query correct region |

### Pattern 4: Alarm Created But Notification Never Received

| Symptom | Alarm state = "alarm" but no SMS/email received |
|---------|------------------------------------------------|
| Root Cause | SMN topic URN invalid or subscription not confirmed |
| Diagnosis | Verify SMN topic URN format and subscription status |
| Resolution | Re-create SMN topic and confirm all subscriptions |

## Cross-Product Cascade Faults

### Pattern 1: VPC Bandwidth Saturation Causes Cascading Alarms

```
Trigger: SYS.VPC > bandwidth_util > 90%
Cascading Alarms:
  → SYS.ELB > l7e_listener_qps drops (downstream)
  → SYS.ECS > cpu_util drops (application can't receive traffic)
  → SYS.RDS > rds007_qps drops (no queries coming in)
Root Cause: VPC bandwidth saturation
Resolution: Scale bandwidth quota or implement traffic shaping
```

### Pattern 2: Database Exhaustion Cascades to Application

```
Trigger: SYS.RDS > rds003_conn_usage > 90%
Cascading Alarms:
  → SYS.ECS > cpu_util drops (app threads blocking on connections)
  → SYS.ELB > 5xx errors spike (app returns errors)
Root Cause: Connection pool exhaustion
Resolution: Optimize connection pool, scale RDS, or add read replicas
```

## Historical Diagnosis Reference

| Date | Pattern | Impact | Resolution |
|------|---------|--------|------------|
| Generic | Deploy-triggered alarm storm | 50+ alarms during deployment | Add deployment window: disable non-critical alarms before deploy |
| Generic | Agent disconnect → blind spots | No metrics for 6 host instances | Implement agent health monitoring alarm |
| Generic | Threshold too tight | 20+ daily false positives | Recalibrate thresholds based on 30-day baseline |
