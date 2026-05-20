# Well-Architected Assessment — Huawei Cloud DMS

## 1. Security (安全)

### IAM Minimum Permissions

| Operation | Required Policy | Action |
|-----------|----------------|--------|
| List instances | `DMS ReadOnlyAccess` | `dms:instances:list` |
| Create instance | `DMS FullAccess` | `dms:instances:create` |
| Delete instance | `DMS FullAccess` | `dms:instances:delete` |
| Create topic/queue | `DMS FullAccess` | `dms:resources:create` |
| Reset password | `DMS FullAccess` | `dms:instances:resetPassword` |
| Create backup | `DMS FullAccess` | `dms:backup:create` |
| Restore backup | `DMS FullAccess` | `dms:backup:restore` |

### Network Security

- Deploy DMS instances inside VPC, not with public access
- Use security group rules to restrict client access to specific CIDR ranges
- Enable TLS for client-to-broker encryption (port 9093 for Kafka, 5671 for RabbitMQ)
- Enable SASL/PLAIN authentication for Kafka clients
- Use VPC endpoints for cross-VPC access without public internet

### Data Protection

- KMS encryption for data at rest (storage encryption)
- TLS 1.2+ encryption for data in transit
- Backup encryption inherits from source instance

## 2. Stability (稳定)

### High Availability

- Use cluster mode (3+ brokers) for production workloads
- Replication factor 3 for critical topics (tolerates 2 broker failures)
- Multi-AZ deployment distributes brokers across AZs
- Automatic broker failover with leader re-election

### Backup & Recovery

| RPO | RTO | Method |
|-----|-----|--------|
| <24 hours | <2 hours | Daily automated backup |
| <1 hour | <30 min | Manual on-demand backup before changes |
| Near-zero | Multi-region | DMS migration tool + MirrorMaker (Kafka) |

### Disaster Recovery

| Phase | Action | Responsible |
|-------|--------|-------------|
| Phase 1 (1h) | Create new DMS instance in DR region | Agent |
| Phase 2 (2h) | Restore latest backup to DR instance | Agent |
| Phase 3 (1h) | Switch producers/consumers to DR endpoint | User/Application |
| Phase 4 (ongoing) | Setup MirrorMaker for continuous replication | User |

## 3. Cost (成本)

### Billing Model Comparison

| Model | Commitment | Discount | Best For |
|-------|-----------|----------|----------|
| 按需 (On-demand) | None | 0% | Dev/test, variable workloads |
| 包月 (Monthly) | 1 month | ~30% | Stable production workloads |
| 包年 (Yearly) | 1 year | ~50-70% | Long-running production |
| 3-year | 3 years | ~85% | Mission-critical, stable |

### Cost Optimization

| Pattern | Detection | Recommendation |
|---------|-----------|---------------|
| Idle instance | CPU < 5% for 7 days | Switch to smaller spec or delete |
| Over-provisioned | Max CPU < 20% for 30 days | Downgrade to lower spec |
| Under-provisioned | CPU > 80% for 24 hours | Upgrade to higher spec |
| Unused topics | Topic with 0 messages for 7 days | Delete topic and reclaim partitions |

## 4. Efficiency (效率)

- Use batch operations for creating multiple topics/queues
- Pipe CLI JSON output through `jq` for automated processing
- Integrate with CI/CD pipelines for infrastructure-as-code
- Use tags (`env`, `project`, `team`) for resource organization

## 5. Performance (性能)

### Scaling Triggers

| Metric | Threshold | Action |
|--------|-----------|--------|
| CPU usage | >80% for 10min | Scale up instance spec |
| Disk usage | >85% | Increase storage or clean messages |
| Consumer lag | >100,000 | Scale consumer application or add partitions |
| Connection count | >80% of max | Add brokers to distribute connections |

### Performance Baselines

| Instance Spec | Max TPS | Max Partitions | Max Storage |
|--------------|---------|----------------|-------------|
| `kafka.2u4g.cluster` | 30,000 | 300 | 1.8 TB |
| `kafka.4u8g.cluster` | 80,000 | 600 | 3.6 TB |
| `kafka.8u16g.cluster` | 200,000 | 1,200 | 6 TB |
| `kafka.16u32g.cluster` | 500,000 | 2,400 | 9 TB |

## 6. FinOps (财务运营)

### Cost Visibility

| Tool | Purpose |
|------|---------|
| BSS (Billing Center) | View monthly DMS costs per instance |
| CES billing metrics | Track storage and bandwidth costs |
| Cost tags | Tag instances with `cost_center`, `project` |

### Waste Detection

```bash
# Find idle Kafka instances (no messages in 7 days)
hcloud DMS ListInstances --format=json | jq '.[] | select(.total_messages==0)'
```

## 7. SecOps (安全运营)

| Control | Implementation |
|---------|---------------|
| IAM least privilege | Use fine-grained policies, not `DMS FullAccess` by default |
| Credential rotation | Rotate AK/SK every 90 days |
| Network isolation | VPC + security group only; no public access |
| Encryption at rest | KMS SSE-KMS for storage |
| Encryption in transit | TLS 1.2+ with SASL/SCRAM authentication |
| Audit logging | Enable CTS to track all DMS API calls |

## 8. AIOps (智能运营)

### Anomaly Patterns

| Pattern | Detection Logic | Severity |
|---------|----------------|----------|
| Consumer lag spike | Lag > 100,000 AND increasing for 10min | Critical |
| Partition skew | One partition > 2x average messages | Warning |
| Connection storm | Connections > 80% max in 5min | Critical |
| Disk I/O bottleneck | Disk wait time > 50ms for 5min | Warning |
| Produce failure rate | Produce errors > 1% in 5min | Critical |
| Broker offline | Broker count < configured count | Critical |

### Cross-Skill Diagnosis

| Symptom | Primary Skill | Supporting Skills |
|---------|--------------|-------------------|
| Producer timeout | `huaweicloud-dms-ops` | `huaweicloud-vpc-ops` (network) |
| High consumer lag | `huaweicloud-dms-ops` | `huaweicloud-ces-ops` (metrics) |
| Instance creation failure | `huaweicloud-dms-ops` | `huaweicloud-iam-ops` (permissions) |
| Backup failure | `huaweicloud-dms-ops` | `huaweicloud-cbr-ops` (backup) |
