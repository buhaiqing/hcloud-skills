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

### Budget Alert Configuration

Budget alerts enable proactive cost monitoring for CES spending, preventing unexpected billing and enabling timely cost optimization decisions.

#### Integration Architecture

```
CES Custom Metrics → BMS Budget Service → SMN Notifications → Email/SMS/Webhook
```

**Key Components:**
- **BMS (Budget Management Service)**: Huawei Cloud's budget tracking service
- **SMN (Simple Message Notification)**: Multi-channel notification delivery
- **CES Metrics**: Custom metric usage data for budget tracking

#### CLI Commands for Budget Setup

**1. Create Budget with Thresholds:**

```bash
# Create budget for CES custom metrics spending
hcloud bms budget create \
  --budget-name "ces-custom-metrics-budget" \
  --budget-type "COST" \
  --budget-limit 100.00 \
  --currency "CNY" \
  --time-period "MONTHLY" \
  --thresholds '[{"threshold_type":"ABSOLUTE","threshold_value":80.00,"notification_method":"EMAIL"},{"threshold_type":"ABSOLUTE","threshold_value":90.00,"notification_method":"SMS"},{"threshold_type":"ABSOLUTE","threshold_value":100.00,"notification_method":"ALL"}]'
```

**2. Configure SMN Topic for Notifications:**

```bash
# Create SMN topic for budget alerts
hcloud smn topic create \
  --name "ces-budget-alerts" \
  --display-name "CES预算告警通知"

# Subscribe email notification
hcloud smn subscription create \
  --topic-urn "urn:smn:cn-north-4:PROJECT_ID:ces-budget-alerts" \
  --endpoint "ops-team@company.com" \
  --protocol "email"

# Subscribe SMS notification (for critical thresholds)
hcloud smn subscription create \
  --topic-urn "urn:smn:cn-north-4:PROJECT_ID:ces-budget-alerts" \
  --endpoint "+8613800138000" \
  --protocol "sms"
```

**3. Query Budget Status:**

```bash
# List all budgets
hcloud bms budget list \
  --query "budget_name=ces-custom-metrics-budget"

# Get budget details and threshold status
hcloud bms budget show \
  --budget-id "BUDGET_ID"
```

#### Threshold Configuration Strategy

| Threshold | Notification | Action Required | Priority |
|-----------|--------------|-----------------|----------|
| 80% | Email notification | Review custom metric usage; identify optimization opportunities | Medium |
| 90% | SMS + Email | Immediate action: disable non-essential custom metrics; optimize granularity | High |
| 100% | SMS + Email + Webhook | Emergency: disable all custom metrics; investigate unexpected spikes | Critical |

#### Advanced Configuration: Webhook Integration

For automated response to budget breaches, configure webhook notifications:

```bash
# Subscribe webhook for automated response
hcloud smn subscription create \
  --topic-urn "urn:smn:cn-north-4:PROJECT_ID:ces-budget-alerts" \
  --endpoint "https://automation.company.com/budget-handler" \
  --protocol "https"

# Webhook payload structure
{
  "budget_name": "ces-custom-metrics-budget",
  "threshold_triggered": 90,
  "current_usage": 92.5,
  "timestamp": "2026-05-26T10:30:00Z",
  "notification_type": "BUDGET_THRESHOLD_EXCEEDED"
}
```

#### Example Workflow: End-to-End Budget Setup

**Scenario**: Team wants to monitor CES custom metrics spending with 100 CNY monthly budget.

```bash
# Step 1: Create SMN topic
TOPIC_URN=$(hcloud smn topic create \
  --name "ces-budget-alerts" \
  --display-name "CES预算告警" \
  --query "topic_urn")

# Step 2: Subscribe notifications
hcloud smn subscription create --topic-urn "$TOPIC_URN" --endpoint "ops@company.com" --protocol "email"
hcloud smn subscription create --topic-urn "$TOPIC_URN" --endpoint "+8613800138000" --protocol "sms"

# Step 3: Create budget with linked SMN topic
BUDGET_ID=$(hcloud bms budget create \
  --budget-name "ces-monthly-budget" \
  --budget-type "COST" \
  --budget-limit 100.00 \
  --time-period "MONTHLY" \
  --notification-topic "$TOPIC_URN" \
  --thresholds '[
    {"threshold":80,"notification":"EMAIL"},
    {"threshold":90,"notification":"SMS"},
    {"threshold":100,"notification":"ALL"}
  ]' \
  --query "budget_id")

# Step 4: Verify setup
hcloud bms budget show --budget-id "$BUDGET_ID"

# Expected output:
# Budget Name: ces-monthly-budget
# Budget Limit: 100.00 CNY
# Thresholds: 80% (Email), 90% (SMS), 100% (All channels)
# Notification Topic: ces-budget-alerts
```

#### Best Practices

| Practice | Recommendation | Rationale |
|----------|----------------|-----------|
| Budget scope | Project-specific budget for CES | Enables granular cost attribution per project |
| Threshold spacing | 80/90/100 with escalating actions | Graduated response prevents over-reaction at minor usage |
| Notification channels | Email for monitoring, SMS for critical | Reduces notification fatigue while ensuring critical alerts reach operators |
| Budget review | Weekly budget status check during sprint review | Proactive monitoring prevents threshold surprises |
| Integration | Link budget webhook to automation platform | Enables auto-disable of non-essential custom metrics at 90% threshold |

#### Alternative: Manual Budget Monitoring

If BMS is unavailable, implement manual budget monitoring via CES metrics:

```bash
# Query CES API call count (proxy for custom metric usage)
hcloud ces metric-data list \
  --namespace "SYS.CES" \
  --metric-name "api_call_count" \
  --dimensions '[{"name":"service","value":"ces"}]' \
  --from "2026-05-01T00:00:00Z" \
  --to "2026-05-26T00:00:00Z"

# Manual threshold check script example
#!/bin/bash
API_CALLS=$(hcloud ces metric-data list --namespace SYS.CES --metric-name api_call_count --query "datapoints[0].avg")
LIMIT=100000  # Free tier limit
USAGE_PERCENT=$((API_CALLS * 100 / LIMIT))

if [ $USAGE_PERCENT -ge 80 ]; then
  echo "Warning: CES API usage at ${USAGE_PERCENT}%"
  # Trigger notification via SMN
  hcloud smn message publish --topic-urn "$TOPIC_URN" --subject "CES Budget Alert" --message "API usage: ${USAGE_PERCENT}%"
fi
```

**Note**: BMS integration is recommended for production environments; manual monitoring is suitable for development/testing or environments without BMS access.

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

#### Execution Flows (Implemented in SKILL.md)

| Self-Healing Type | Trigger | Execution Flow | Validation |
|-------------------|---------|----------------|------------|
| **Auto Re-enable Alarms** | Post-deployment webhook or scheduled check | SKILL.md → "Operation: Self-Healing — Auto Re-enable Alarms" | List alarms with `alarm_enabled=false` → empty |
| **Auto-adjust Thresholds** | False positive rate > 5% or manual trigger | SKILL.md → "Operation: Self-Healing — Auto-adjust Alarm Thresholds" | Monitor trigger rate 24h post-adjustment |
| **Auto-create Alarms** | CI/CD resource provisioning event | Alarm templates + respective product skill integration | Verify alarm exists for new resource |

#### Self-Healing Reference

- **Auto Re-enable**: Full CLI + SDK execution flow in SKILL.md (lines 460-560)
- **Auto-adjust Thresholds**: Historical baseline analysis + threshold calculation flow in SKILL.md (lines 560-620)
- **Auto-create Alarms**: CI/CD integration via alarm templates; delegate to respective product skill for resource creation
