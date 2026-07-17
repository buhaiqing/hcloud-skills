# Well-Architected + Three-Pillar Assessment — Huawei Cloud ELB

## 1. Security (安全) — ELB

### IAM Minimum Permissions

| Operation | IAM Action | Resource Scope |
|-----------|-----------|---------------|
| ListLoadBalancers | `elb:loadbalancers:list` | `*` |
| ShowLoadBalancer | `elb:loadbalancers:get` | `elb:loadbalancer:${lb_id}` |
| CreateLoadBalancer | `elb:loadbalancers:create`, `vpc:vpcs:get`, `vpc:subnets:get` | `elb:loadbalancer:*`, `vpc:vpc:*` |
| UpdateLoadBalancer | `elb:loadbalancers:update` | `elb:loadbalancer:${lb_id}` |
| DeleteLoadBalancer | `elb:loadbalancers:delete` | `elb:loadbalancer:${lb_id}` |
| ListListeners | `elb:listeners:list` | `*` |
| CreateListener | `elb:listeners:create` | `elb:listener:*` |
| UpdateListener | `elb:listeners:update` | `elb:listener:${listener_id}` |
| DeleteListener | `elb:listeners:delete` | `elb:listener:${listener_id}` |
| ListPools | `elb:pools:list` | `*` |
| CreatePool | `elb:pools:create` | `elb:pool:*` |
| UpdatePool | `elb:pools:update` | `elb:pool:${pool_id}` |
| DeletePool | `elb:pools:delete` | `elb:pool:${pool_id}` |
| ListMembers | `elb:members:list` | `*` |
| CreateMember | `elb:members:create`, `ecs:servers:get` | `elb:pool:${pool_id}`, `ecs:server:*` |
| UpdateMember | `elb:members:update` | `elb:member:${member_id}` |
| DeleteMember | `elb:members:delete` | `elb:member:${member_id}` |
| CreateHealthMonitor | `elb:healthmonitors:create` | `elb:healthmonitor:*` |
| UpdateHealthMonitor | `elb:healthmonitors:update` | `elb:healthmonitor:${hm_id}` |
| DeleteHealthMonitor | `elb:healthmonitors:delete` | `elb:healthmonitor:${hm_id}` |
| ListCertificates | `elb:certificates:list` | `*` |
| CreateCertificate | `elb:certificates:create` | `elb:certificate:*` |
| DeleteCertificate | `elb:certificates:delete` | `elb:certificate:${cert_id}` |
| ListAvailabilityZones | `elb:availability-zones:list` | `*` |
| ShowQuota | `elb:quotas:get` | `*` |

### Credential Management
- Use IAM agency for ELB → ECS cross-service health checks
- Rotate AK/SK every 90 days

### Certificate Expiry Monitoring

Certificate expiry alarms prevent HTTPS listener failures due to expired SSL certificates.

| Alarm Name | Threshold | Notification | Priority |
|------------|-----------|--------------|----------|
| cert-expiry-30d | ≤ 30 days remaining | Email | Warning |
| cert-expiry-7d | ≤ 7 days remaining | SMS + Email | Critical |
| cert-expiry-1d | ≤ 1 day remaining | All channels + Webhook | Emergency |

#### CLI Configuration

```bash
# Create certificate expiry alarm (requires custom monitoring script)
# Step 1: Create SMN topic for certificate alerts
TOPIC_URN=$(hcloud smn topic create \
  --name "elb-cert-expiry-alerts" \
  --display-name "ELB证书过期告警" \
  --query "topic_urn")

# Step 2: Subscribe notifications
hcloud smn subscription create --topic-urn "$TOPIC_URN" --endpoint "ops@company.com" --protocol "email"
hcloud smn subscription create --topic-urn "$TOPIC_URN" --endpoint "+8613800138000" --protocol "sms"

# Step 3: Certificate expiry check script (run daily via cron)
#!/bin/bash
# Check certificate expiry and send alerts
for CERT_ID in $(hcloud elb list-certificates --region $REGION --query "certificates[].id"); do
  EXPIRY_DATE=$(hcloud elb show-certificate --certificate-id "$CERT_ID" --query "expire_time")
  DAYS_LEFT=$(( ($(date -d "$EXPIRY_DATE" +%s) - $(date +%s)) / 86400 ))

  if [ $DAYS_LEFT -le 30 ]; then
    hcloud smn message publish \
      --topic-urn "$TOPIC_URN" \
      --subject "ELB Certificate Expiry Warning: $DAYS_LEFT days" \
      --message "Certificate $CERT_ID expires in $DAYS_LEFT days. Renew immediately."
  fi
done
```

#### Best Practices

| Practice | Recommendation | Rationale |
|----------|----------------|-----------|
| Check frequency | Daily check via cron | Early detection of expiry |
| Renewal buffer | Renew at 30 days remaining | Avoid last-minute emergencies |
| Auto-renew | Enable auto-renew for managed certs | Eliminate manual renewal process |
| Backup certificate | Keep backup cert ready | Instant swap if renewal delayed |

### Network Security
- **Least privilege**: Only open required ports on listener
- **Security group**: Backend members' SG must allow traffic from LB subnet
- **HTTPS mandatory**: All production listeners should use HTTPS with valid certificate
- **TLS version**: Minimum TLS 1.2, recommended TLS 1.3
- **WAF integration**: Enable WAF for HTTPS listeners protecting web apps

### Encryption
- **In transit**: HTTPS listeners with TLS termination at LB
- **Certificate management**: Upload to ELB Certificate, rotate before expiry
- **Access logs**: Enable LTS access log streaming (log encryption enabled)

## 2. Stability (稳定) — ELB

### High Availability

| Pattern | Description | Recommendation |
|---------|-------------|---------------|
| Multi-AZ LB | Deploy LB across ≥ 2 AZs | Mandatory for production |
| Multi-member pool | ≥ 2 backend members per pool | Mandatory for production |
| Cross-region failover | DNS-based failover to different region | For critical applications |
| Health check | Properly configured health checks | Required for all pools |

### Backup & Recovery

| Scenario | Recovery Method | RTO |
|----------|----------------|-----|
| LB failure | Failover to standby LB (DNS) | < 5min |
| Configuration corruption | Recreate via IaC template | < 15min |
| AZ outage | Multi-AZ LB survives | < 1min |
| Accidental deletion | Restore from backup configuration | < 20min |

### DR Runbook

**Phase 1: Detection**
1. Check `provisioning_status` of LB
2. Check `operating_status` — should be ONLINE
3. Verify backend members' health via `ListMembers`
4. Check CES metrics for connectivity

**Phase 2: Recovery**
1. If single member unhealthy: remove from pool, recover via ECS skill
2. If pool all unhealthy: verify health check config
3. If LB down: failover to standby LB (DNS update)
4. If multi-AZ issue: divert traffic to healthy AZ

**Phase 3: Verification**
1. Test connectivity through LB
2. Verify backend members show ONLINE
3. Check error rate returns to normal

### Multi-AZ Best Practices
- Dedicated LB: deploy across ≥ 2 AZs natively
- Shared LB: single AZ by default — use multiple shared LBs for multi-AZ
- Backend members: distribute evenly across AZs
- Health check: cross-AZ health check enabled

## 3. Cost (成本) — ELB (FinOps)

### Billing Model Comparison

| LB Type | Billing Model | Cost Characteristics | Best For |
|---------|--------------|---------------------|----------|
| **Shared** | 按需 | Free (pay only for data processed) | Dev/test, low traffic |
| **Dedicated (小型)** | 按需/包月 | ¥~300/month | Low traffic production |
| **Dedicated (中型)** | 按需/包月 | ¥~600/month | Medium traffic |
| **Dedicated (大型)** | 按需/包月 | ¥~1200/month | High traffic |

### Waste Detection

| Waste Pattern | Detection Method | Action |
|--------------|-----------------|--------|
| Idle LB (no traffic) | `m1_cps` = 0 for 7+ days | Delete or stop |
| Over-provisioned LB | `m1_cps` < 10% of LB capacity | Downgrade to smaller LB type |
| Unused listeners | Listener has 0 connections | Remove unused listeners |
| Orphaned pools | Pool has no members | Delete pool |
| Idle EIPs | EIP bound to LB but LB not used | Release EIP |

### Right-Sizing Guidance

| Average CPS | Current Type | Recommendation |
|-------------|-------------|---------------|
| < 1000 | Dedicated 大型 | Downgrade to dedicated 小型 |
| < 5000 | Dedicated 大型 | Downgrade to dedicated 中型 |
| 5000–10000 | Dedicated 中型 | Keep |
| > 10000 | Dedicated 小型 | Upgrade to dedicated 中型/大型 |

### Unit Economics

| Metric | Formula | Target |
|--------|---------|--------|
| Cost per request | Monthly cost / total requests | < ¥0.000001 |
| Cost per GB transferred | Monthly cost / total bytes | < ¥0.01/GB |
| Active connections per ¥ | Active connections / monthly cost | Maximize |

## 4. Efficiency (效率) — ELB

### CI/CD Integration
- Infrastructure as Code (Terraform): LB config as code
- CI/CD pipeline: update pool members during deployment
- Rolling update: gradually replace backend members
- Blue-green deployment: switch pool between versions

### Automation Patterns
- Auto-scaling: AS service adds/removes pool members automatically
- Canary deployment: weight-based traffic distribution via multiple pools
- Health check auto-recovery: unhealthy members automatically removed and re-added

## 5. Performance (性能) — ELB

### Performance Baselines

| Metric | Expected Range | Optimization |
|--------|---------------|--------------|
| New connections/s (CPS) | 2000–40000 (type-dependent) | Upgrade LB type |
| Throughput | 100Mbps–10Gbps | Upgrade LB type |
| L7 latency (P99) | < 500ms backend | Optimize backend, add caching |
| L4 latency (P99) | < 100ms | Check network path |
| Connection idle timeout | 60s default | Adjust per app needs |

### Optimization Patterns
1. **Connection pooling**: Reuse connections to reduce CPS
2. **SSL termination at LB**: Offload SSL from backend
3. **Compression**: Enable gzip at LB level
4. **Caching**: Add CDN or cache layer before LB
5. **Keepalive**: Enable HTTP keepalive on backend
6. **Health check tuning**: Adjust delay/timeout to balance accuracy vs overhead
### SLO/SLI Definition — ELB

#### SLI (Service Level Indicator) Metrics

| SLI Name | Formula | Data Source | Collection Frequency |
|----------|---------|-------------|---------------------|
| Availability | Successful requests / Total requests × 100% | CES + ELB | 1min |
| Latency P99 | 99th percentile response time (ms) | AOM Trace | 1min |
| Error Rate | 5xx responses / Total requests × 100% | ELB + AOM | 1min |
| Saturation | CPU utilization / Connection utilization / Disk utilization | CES | 5min |

#### SLO Targets

| SLI | SLO Target | Error Budget (Monthly) | Alert Threshold |
|-----|------------|-----------------------|-----------------|
| Availability | ≥ 99.9% | 43.2 min/month | < 99.95% triggers Warning |
| Latency P99 | ≤ 200ms | — | > 300ms triggers Warning |
| Error Rate | ≤ 0.1% | — | > 0.5% triggers Critical |
| Saturation | ≤ 80% | — | > 85% triggers Warning |

#### Error Budget Burn Rate Alerts

| Burn Rate | Consumption Speed | Alert Level | Meaning |
|-----------|------------------|-------------|---------|
| 1× | Normal consumption (43.2 min/month) | — | Normal |
| 2× | 21.6 min exhausted | Info | Attention needed |
| 5× | 8.6 min exhausted | Warning | Intervention needed |
| 14.4× | 3h exhausted | Critical | Immediate action required |


---

## Worker Output Contract (Read-Only Assessment Mode)

> Invoked when Well-Architected review sets `{{user.mode}}=well-architected-readonly`.
> Return **`{{output.product_assessment}}`** — field names MUST match the canonical schema.

**Canonical schema:** [worker-output-schema.md](../../huaweicloud-skill-generator/references/worker-output-schema.md)

| Constant | Value |
|----------|-------|
| `skill_id` | `huaweicloud-elb-ops` |
| `product` | `elb` |
| Finding `id` pattern | `elb-{rel|sec|cost|eff}-NNN` |

### Pillar → checklist map

| `pillars` key | Checklist source in this document |
|---------------|-------------------------------------|
| `reliability` | Stability / DR / backup sections |
| `security` | IAM / network / encryption sections |
| `cost` | FinOps / billing / idle detection sections |
| `efficiency` | Automation / batch / CI/CD sections |

### Example `{{output.product_assessment}}`

```json
{
  "skill_id": "huaweicloud-elb-ops",
  "product": "elb",
  "region": "cn-north-4",
  "scope": "account-wide",
  "assessment_date": "2026-06-19T10:00:00+08:00",
  "status": "OK",
  "partial": false,
  "resource_count": 1,
  "pillars": {
    "cost": {
      "score": 80,
      "status": "assessed",
      "findings": []
    },
    "efficiency": {
      "score": 80,
      "status": "assessed",
      "findings": []
    },
    "reliability": {
      "score": 80,
      "status": "assessed",
      "findings": []
    },
    "security": {
      "score": 80,
      "status": "assessed",
      "findings": []
    }
  },
  "recommendations": [],
  "trace": {
    "commands": [
      "hcloud elb read-only-list --region cn-north-4 (HW_SECRET_ACCESS_KEY=<masked>)"
    ],
    "request_ids": [
      "0123456789abcdef0123456789abcdef"
    ]
  },
  "errors": []
}
```
