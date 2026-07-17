# Chaos Engineering — ELB

> **Purpose**: Document fault injection experiments for ELB (Elastic Load Balance) resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/chaos-engineering-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Backend failure | Stop backend ECS instance | Request success rate, health check | Unhealthy backend removed, traffic redistributed | Request failure >5% for >2min |
| AZ failure | Stop all backends in one AZ | Cross-AZ distribution, request success | Requests route to other AZs | Success rate <90% for >3min |
| Health check failure | Block health check port | Backend status, active connections | Backend marked unhealthy, connections drained | Unhealthy state persists >5min |
| Connection exhaustion | Exhaust ELB connection limit | New connection rejection rate | Connection queued or rejected with 503 | Rejection rate >10% for >2min |
| SSL certificate expiry | Simulate cert validation failure | HTTPS request success rate | Alert triggered, cert rotation initiated | HTTPS success rate <50% for >1min |
| Listener rule failure | Remove backend target group | Request routing, 502 rate | Requests routed per fallback rule | 502 rate >20% for >3min |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from backend failure to health check detection | 20% |
| Fault isolation ability | Explosion radius (single backend vs entire group) | 20% |
| Recovery automation | Auto-unhealthy, re-add, scaling effectiveness | 25% |
| Degradation quality | Availability during partial backend failure | 15% |
| Data consistency | Session persistence, request idempotency | 20% |

### Resilience Grades

| Score | Grade | Recommendation |
|-------|-------|---------------|
| 8-10 | A (Excellent) | Regular chaos validation, maintain |
| 6-8 | B (Good) | Supplement missing fault scenarios |
| 4-6 | C (Fair) | Increase self-healing, improve degradation |
| 0-4 | D (Weak) | Prioritize critical resilience gaps |

## 3. Chaos Experiment Workflow

```yaml
chaos_experiment:
  name: "elb-backend-failure"
  objective: "Verify ELB removes unhealthy backend within health check interval"

  preconditions:
    - "ELB with ≥2 backend servers across AZs"
    - "Health check configured (interval 30s, threshold 3)"
    - "CES alarm on ELB backend server status"

  steps:
    - inject_fault: "Stop primary backend ECS instance"
    - observe_metrics: "Monitor health check status, active connections"
    - verify_behavior: "Backend marked unhealthy ≤ 90s (3x interval)"
    - rollback_fault: "Restart backend, verify re-registration"

  success_criteria:
    - "Backend unhealthy detected ≤ 90s"
    - "Traffic redistributed to healthy backends"
    - "No request failures (verified via success rate)"

  emergency_rollback:
    - "Restart backend immediately"
    - "Force re-add backend to target group if needed"
    - "Scale ELB if connection limit reached"
```

## 4. ELB-Specific Experiment Details

### 4.1 Backend Failure (Primary Scenario)

**Objective**: Verify ELB health check and traffic redistribution.

**Injection**:
```bash
# Stop backend ECS instance
hcloud ECS StopServers --instance-ids <backend-id> --force
```

**Metrics to Monitor**:
- `elb_backend_server_status` via CES
- `elb_connection_count` per backend
- `elb_request Success_rate` overall

**Expected**: Backend marked unhealthy after 3 consecutive failures (90s), traffic redistributed.

### 4.2 AZ Failure

**Objective**: Verify cross-AZ redundancy and availability.

**Injection**:
```bash
# Stop all ECS instances in target AZ
# Or block ELB health check to backends in AZ via SG rule
hcloud VPC CreateSecurityGroupRule --security-group-id <sg-id> \
  --direction ingress --remote-ip-prefix <elb-subnet-cidr> \
  --protocol tcp --port <health-check-port> --description "CHAOS: AZ isolation"
```

**Metrics**: Cross-AZ request success rate, backend count per AZ.

### 4.3 Health Check Failure

**Objective**: Verify health check threshold and graceful drain.

**Injection**:
```bash
# Block health check port on backend
iptables -A INPUT -p tcp --dport <hc-port> -j REJECT
```

**Metrics**: Health check status transitions, active connection count during drain.

### 4.4 SSL Certificate Validation Failure

**Objective**: Verify cert expiry alerting and rotation workflow.

**Injection**:
```bash
# Simulate cert validation failure (in practice: use expired test cert)
# Monitor via CES: elb_ssl_cert_expiry_days
```

**Metrics**: `elb_ssl_cert_expiry_days`, HTTPS success rate, alarm firing time.

## 5. Emergency Rollback Procedures

| Scenario | Rollback Action |
|----------|----------------|
| Backend not re-registering | Force restart backend, manually add to target group |
| Health check stuck | Reset health check configuration, verify security group |
| Connection limit reached | Scale ELB, enable connection draining |
| Cert rotation failure | Rollback to previous cert, manual intervention |
| AZ failure persists | Manually move backends to healthy AZ |

## 6. Compliance Checklist

- [x] ≥5 fault injection experiments designed (6 experiments)
- [x] Resilience scoring model defined (5 dimensions, 20% weights)
- [x] Experiment workflow documented (YAML format)
- [x] Emergency rollback procedures defined (5 scenarios)
