# CES Well-Architected + Three-Pillar Assessment

## 1. Security Pillar (安全支柱)

### IAM Minimum Permissions

| Role | Required Permissions | Use Case |
|------|---------------------|----------|
| CES Viewer | `ces:alarm:list`, `ces:alarm:show`, `ces:metric:list`, `ces:metric:show`, `ces:dashboard:list`, `ces:dashboard:show` | Read-only monitoring access |
| CES Operator | All Viewer + `ces:alarm:create`, `ces:alarm:update`, `ces:alarm:delete` | Managing alarm rules |
| CES Administrator | All Operator + `ces:dashboard:create`, `ces:dashboard:update`, `ces:dashboard:delete` | Full CES management |

### Credential Safety

- AK/SK stored in environment variables only — never in SKILL.md, logs, or error messages
- For JIT Go SDK: construct `basic.CredentialsBuilder()` from env vars; never echo `sk`
- Regular AK/SK rotation recommended (every 90 days)

### Network Isolation

- CES API accessible via VPC Endpoint for private network access
- Recommended: Use VPC Endpoint to avoid public network exposure

## 2. Stability Pillar (稳定支柱)

### Multi-AZ Alarm Redundancy

- Alarm rules are region-level, not AZ-level — a single alarm covers all AZs in the region
- For critical services spanning multiple regions, create duplicate alarm rules per region

### Disaster Recovery

| Phase | Action |
|-------|--------|
| Phase 1: Detection | Critical alarms notify SMN topic with SMS + email |
| Phase 2: Containment | Disable automated responses that might cascade (e.g., auto-scaling loops) |
| Phase 3: Recovery | Restore monitoring after resources recover; verify alarm rules re-enabled |

### Backup Strategy

- Export alarm configurations periodically (alarm rules are lightweight but critical)
- Use CLI `list-alarms` + JSON export for backup: `hcloud ces list-alarms --region cn-north-4 --output json > ces-alarms-backup.json`

## 3. Cost Pillar (成本支柱 / FinOps)

### Billing Model

| Item | Pricing |
|------|---------|
| System metrics (SYS.*) | Free — included with cloud service subscription |
| Agent metrics (AGT.*) | Free with CES agent installed |
| Custom metrics | Charged per metric per month |
| Alarm rules | Free up to quota limit; excess not billed (quota limit enforced) |
| Dashboards | Free |
| API calls | Free within rate limits |

### Waste Detection Patterns

| Pattern | Detection | Remediation |
|---------|-----------|-------------|
| Unused alarm rules | Alarm enabled but never triggered in 90 days | Review whether alarm is still needed; consider disabling |
| Overly granular monitoring | 5-second interval for non-critical metrics | Change to 5-minute or 1-hour period |
| Duplicate alarms | Multiple alarms monitoring same metric on same resource | Consolidate into single alarm with multiple thresholds |

### Cost Attribution

- Tag alarm rules with project/cost-center tags to attribute monitoring costs
- Use batch queries to reduce API call volume (and associated overhead)

## 4. Efficiency Pillar (效率支柱)

### Batch Operations

- Use `BatchQueryMetricData` for querying metrics across multiple resources (up to 10 per batch)
- Use `BatchListMetrics` for discovering available metrics in bulk

### Alarm Templates

- Create alarm templates for common patterns (CPU > 80%, Memory > 90%, Disk > 85%)
- Apply templates to new resources during provisioning via CI/CD

### Automation

- Integrate alarm creation into IaC (Terraform, CloudFormation)
- Use CES APIs for automated alarm setup during resource provisioning

## 5. Performance Pillar (性能支柱)

### Alarm Evaluation Tuning

| Parameter | Recommended | Rationale |
|-----------|-------------|-----------|
| Period | 300s (5 min) for standard, 60s for critical | Balances responsiveness with noise reduction |
| Evaluation periods | 3 (standard), 1 (critical) | Reduces flapping while catching real issues |
| Threshold | 80% for standard alerts, 95% for critical | Based on typical resource utilization baselines |

### Query Performance

- Limit time range for metric queries to reduce latency (≤ 24 hours for detailed, ≤ 30 days for aggregated)
- Use appropriate filter: `average` for general trends, `max` for peak identification

### Metric Data Volume

- High-volume metric queries (> 1000 datapoints) should be paginated or use batch queries
- Consider caching frequently queried metrics (e.g., daily dashboards)

## FinOps Assessment (财务运营)

### Cost Visibility

- CES system metrics: included free with cloud services
- Custom metrics: billed separately; track per project
- Recommend enabling cost center tags for custom metrics

### Budget Alerts

- Monitor CES API call volume to stay within free tier limits
- Set budget alerts at 80%/90%/100% for custom metric spending

### Right-Sizing Guidance

- Review alarm thresholds periodically to avoid over-monitoring
- Consolidate duplicate alarms to reduce resource usage

## SecOps Assessment (安全运营)

### Identity Security

- Minimum IAM: CES Viewer for monitoring only, CES Operator for alarm management
- MFA recommended for CES Administrator role
- AK/SK rotation every 90 days

### Data Security

- Metric data is internal to the project; not shared cross-project
- Dashboard data may contain sensitive resource information — restrict access via IAM

### Threat Detection

- CES alarms can monitor HSS (Host Security Service) metrics for threat detection
- Integrate CES with HSS alerts for comprehensive security monitoring

## AIOps Assessment (智能运营)

### Multi-Metric Correlation

| Anomaly Pattern | Detection Logic | Cross-Skill Delegation |
|-----------------|-----------------|------------------------|
| Resource pressure | CPU > 90% + Memory > 85% + Disk I/O high simultaneously | Delegating to huaweicloud-ecs-ops |
| Network degradation | bandwidth_util > 90% + packet loss > 1% | Delegating to huaweicloud-vpc-ops |
| Database bottleneck | QPS drop + connection_util > 90% + IOPS > 90% | Delegating to huaweicloud-rds-ops |
| Cascading failure | Downstream service alarms after upstream alarm | Multi-skill diagnosis |

### Knowledge Base

| Fault Pattern | Root Cause | Resolution |
|---------------|------------|------------|
| Alarm never triggers | Metric namespace mismatch | Verify namespace matches resource type |
| False positive alarms | Evaluation period too short | Increase evaluation_periods to 3+ |
| Alarm storm during deployment | Resource restart causes metric spikes | Disable alarms during deployment window |
| Metric data gaps | Agent disconnected or not installed | Verify CES agent status on target resource |
| Notification delay | SMN topic misconfigured | Verify topic URN and subscription status |

### Self-Healing

- Auto-re-enable alarm rules after deployment completes
- Auto-adjust thresholds based on historical baseline (advanced, requires custom logic)
- Auto-create alarms for newly provisioned resources (CI/CD integration)
