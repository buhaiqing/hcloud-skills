# VPC Well-Architected + Three-Pillar Assessment

## 1. Security Pillar (安全支柱)

### IAM Minimum Permissions

| Role | Required Permissions | Use Case |
|------|---------------------|----------|
| VPC Viewer | `vpc:vpcs:list`, `vpc:vpcs:get`, `vpc:subnets:list`, `vpc:security-groups:list` | Read-only network inspection |
| VPC Operator | All Viewer + `vpc:vpcs:create`, `vpc:vpcs:update`, `vpc:vpcs:delete`, `vpc:security-groups:*` | Managing VPC resources |
| VPC Administrator | All Operator + `vpc:eip:*`, `vpc:nat:*`, `vpc:peerings:*` | Full network management |

### Security Group Best Practices

| Principle | Rule | Example |
|-----------|------|---------|
| Least privilege | Only allow specific source IPs, ports, protocols | Allow SSH from 10.0.0.0/8, not 0.0.0.0/0 |
| Default deny | No explicit deny rules; rely on default-deny behavior | Inbound: no "allow all" rules |
| Egress control | Restrict outbound traffic, not just inbound | Only allow 443 outbound for web servers |
| Role-based SGs | One SG per role (web, app, db), not per instance | Web-SG: 80/443 ingress, 5432 egress to app-SG |
| Sensitive ports | Never expose 22, 3389, 3306, 6379 to 0.0.0.0/0 | SSH: allow from bastion host IP only |

### VPC Isolation

- Each VPC is completely isolated from other VPCs by default
- Cross-VPC communication only via VPC Peering or VPN
- VPC Endpoint for private access to Huawei Cloud services
- Recommended: Use dedicated VPC per environment (prod, uat, staging)

### Encryption

- In-transit: TLS/SSL for application traffic; IPsec for VPN/peering
- At-rest: Not applicable to VPC itself; apply to attached resources (EVS volumes, RDS storage)

## 2. Stability Pillar (稳定支柱)

### Multi-AZ Design

| Practice | Recommendation |
|----------|---------------|
| Subnet distribution | Create at least 2 subnets in different AZs for each tier |
| NAT gateway | Deploy NAT gateway in primary AZ; use secondary for failover |
| Route table | Use identical route tables per AZ for consistent routing |

### Disaster Recovery

| Phase | Action |
|-------|--------|
| Phase 1: Detection | Monitor bandwidth utilization, EIP binding status, NAT gateway health |
| Phase 2: Containment | Isolate affected subnet; prevent cascading failures via security groups |
| Phase 3: Recovery | Restore from backup (infrastructure-as-code); verify connectivity |

### Backup Strategy

- Export VPC configurations (VPC, subnets, SGs, routes) as IaC templates
- Periodically document network topology for disaster recovery planning

## 3. Cost Pillar (成本支柱 / FinOps)

### Billing Model Comparison

| Resource | Billing Model | Description | Optimization Tip |
|----------|---------------|-------------|-----------------|
| VPC | Free | VPC itself has no cost | N/A |
| Subnet | Free | Subnets are free | N/A |
| EIP | Pay-per-use OR fixed bandwidth | Per-traffic billing or fixed monthly | For steady traffic → fixed bandwidth (cheaper); for spiky → pay-per-use |
| Bandwidth (Shared) | Fixed monthly | Shared across multiple EIPs | Use shared bandwidth when multiple EIPs exist |
| NAT Gateway | Hourly | Per-gateway hourly charge + bandwidth | Right-size NAT spec (Small/Medium/Large) |
| VPC Peering | Free (same region) / Charged (cross-region) | Traffic costs for cross-region | Minimize cross-region traffic; use CDN |

### Waste Detection Patterns

| Pattern | Detection | Remediation |
|---------|-----------|-------------|
| Idle EIPs | EIP exists but not bound to any resource | Release or bind; cost accumulates daily |
| Oversized bandwidth | EIP bandwidth >> actual usage | Downgrade to actual peak usage + 20% buffer |
| Unused NAT gateways | NAT gateway exists but 0 SNAT/DNAT connections | Delete if no longer needed |
| Orphaned VPCs | VPC with no subnets or resources older than 30 days | Delete after confirming no dependencies |

### Cost Attribution

- Tag VPCs with project/cost-center tags
- Tag EIPs with owner/purpose tags
- Use bandwidth billing reports to allocate costs to teams

## 4. Efficiency Pillar (效率支柱)

### CIDR Planning

- Plan CIDR allocation upfront to avoid overlap and future expansion issues
- Use /16 VPCs for flexibility; /24 subnets for most workloads
- Reserve subnet ranges for future AZ additions

### Automation

- Use IaC (Terraform, CloudFormation) for consistent VPC provisioning
- Template common patterns: 3-tier VPC, simple VPC, multi-region VPC
- Automate security group rule generation from application requirements

### Route Table Templates

| Template | Routes | Use Case |
|----------|--------|----------|
| Public subnet | Local + 0.0.0.0/0 → Internet Gateway | Web servers, bastion hosts |
| Private subnet | Local + 0.0.0.0/0 → NAT Gateway | App servers, databases |
| Peered subnet | Local + 172.16.0.0/16 → Peering Connection | Cross-VPC communication |

## 5. Performance Pillar (性能支柱)

### Bandwidth Tuning

| Scenario | Recommended Action |
|----------|-------------------|
| EIP bandwidth > 80% sustained | Upgrade bandwidth tier |
| EIP bandwidth < 20% sustained | Downgrade to save cost |
| Burst traffic | Use shared bandwidth pool for burst absorption |

### NAT Gateway Throughput

| Spec | Connections | Bandwidth | Use Case |
|------|-------------|-----------|----------|
| Small | 10,000 | 1 Gbps | Dev/test environments |
| Medium | 50,000 | 5 Gbps | Production workloads |
| Large | 200,000 | 10 Gbps | High-throughput production |

### VPC Peering vs Alternatives

| Option | Latency | Throughput | Cost | Best For |
|--------|---------|-----------|------|----------|
| VPC Peering | < 1ms (same region) | Up to 10 Gbps | Free (same region) | Cross-VPC in same region |
| Cloud Connect | ~5ms (cross-region) | Up to 10 Gbps | Billed by bandwidth | Cross-region, on-premises |
| VPN | ~10-50ms | Up to 1 Gbps | Billed by bandwidth | On-premises connection |

## FinOps Assessment

### Cost Visibility

- Tag all VPC resources with project/cost-center
- Separate VPCs per cost center for clean attribution
- Use shared bandwidth when multiple EIPs can aggregate

### Budget Alerts

- Set bandwidth budget alerts at 80%/90%/100%
- Monitor EIP count vs quota to avoid surprises

### Right-Sizing

- Analyze EIP bandwidth utilization: average < 30% → downgrade
- NAT gateway spec: connections << limit → downgrade spec

## SecOps Assessment (安全运营)

### Identity Security

- Minimum IAM: VPC Viewer for network inspection, VPC Operator for changes
- MFA recommended for VPC Administrator role
- AK/SK rotation every 90 days

### Threat Detection

- Monitor for unauthorized security group rule changes
- Alert on EIP unbind/bind events for production resources
- Use HSS to monitor for security group misconfigurations on ECS

### Compliance

- Document all security group rules with business justification
- Regular audit of rules allowing 0.0.0.0/0
- Network segmentation: production/separate VPCs from dev/staging

## AIOps Assessment (智能运营)

### Multi-Metric Correlation

| Anomaly Pattern | Detection Logic | Cross-Skill Delegation |
|-----------------|-----------------|------------------------|
| Network saturation | bandwidth_out > 90% + packet_loss > 1% | Delegating to huaweicloud-ces-ops for metric correlation |
| EIP exhaustion | EIP quota reached + new EIP requests failing | Delegating to huaweicloud-billing-ops for cost review |
| Cascading network failure | Multiple VPCs reporting connectivity issues | Investigate VPC peering, NAT gateway, and route table changes |
| Security breach | Unexpected security group rule addition | Delegating to huaweicloud-hss-ops for threat investigation |
| NAT gateway overload | SNAT connections > 90% of limit + high latency | Scale NAT gateway spec; delegate to huaweicloud-ecs-ops for connection optimization |

### Knowledge Base

| Fault Pattern | Root Cause | Resolution |
|---------------|------------|------------|
| Instance unreachable from internet | EIP not bound or SG blocking | Bind EIP; add inbound rule for required port |
| Private instance no internet | NAT gateway not configured or route missing | Add NAT gateway SNAT rule; add default route to NAT |
| Cross-VPC connectivity failure | Route table missing peering route | Add routes on both sides pointing to peering connection |
| EIP bind failure | EIP region mismatch with target resource | Create EIP in same region; move resource if needed |
| Security group allows unwanted traffic | Rule too broad (0.0.0.0/0) | Restrict source CIDR to known IPs |

### Self-Healing

- Auto-detect and alert on idle EIPs > 7 days
- Auto-flag security group rules with 0.0.0.0/0 on sensitive ports
- Auto-recommend bandwidth downgrades for under-utilized EIPs
