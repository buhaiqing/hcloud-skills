# Well-Architected + Three-Pillar Assessment — Huawei Cloud ELB

## 1. Security (安全) — ELB

### IAM Minimum Permissions

| Operation | IAM Action | Resource Scope |
|-----------|-----------|---------------|
| ListLoadBalancers | `elb:*List*` | `*` |
| CreateLoadBalancer | `elb:*Create*`, `vpc:*Get*` | `*` |
| DeleteLoadBalancer | `elb:*Delete*` | `elb:loadbalancer:*` |
| CreateListener | `elb:*Create*` | `elb:listener:*` |
| Certificate Management | `elb:*Certificate*` | `elb:certificate:*` |

### Credential Management
- Use IAM agency for ELB → ECS cross-service health checks
- Rotate AK/SK every 90 days

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
