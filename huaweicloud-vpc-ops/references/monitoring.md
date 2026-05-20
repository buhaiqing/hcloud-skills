# VPC Monitoring — Huawei Cloud Virtual Private Cloud

## Monitoring VPC Resources

VPC network health directly impacts all dependent cloud services. Monitor the following:

### Bandwidth Utilization

| Metric | Threshold | Action |
|--------|-----------|--------|
| bandwidth_out > 90% of purchased bandwidth | Major alert | Scale bandwidth or implement traffic shaping |
| packet_out > 100,000 pps | Major alert | Check for DDoS or application anomalies |

### EIP Health

| Check | Frequency | Action |
|-------|-----------|--------|
| Unbound EIPs | Daily | Review and release unused EIPs to reduce cost |
| EIP bind/unbind events | Real-time via events | Alert on unexpected binding changes |
| Bandwidth utilization per EIP | Every 5 min | Identify over-utilized EIPs |

### Security Group Audit

| Check | Frequency | Action |
|-------|-----------|--------|
| Rules allowing 0.0.0.0/0 on sensitive ports | Weekly | Remediate immediately — restrict to specific IPs |
| Unused security groups (no instances attached) | Monthly | Delete to reduce management overhead |
| Overly permissive rules | Monthly | Apply least-privilege principle |

## VPC Peering Monitoring

- Monitor peering connection status changes (PENDING_ACCEPTANCE → ACTIVE → REJECTED)
- Alert on peering status changes for production connections
- Track cross-VPC traffic volume to identify dependency patterns

## Proactive Inspection (巡检)

### Weekly
- Audit security group rules for overly permissive access
- Review VPC CIDR utilization vs planned allocation
- Check for orphaned resources in subnets scheduled for decommission

### Monthly
- Bandwidth cost analysis: identify top bandwidth consumers
- EIP inventory: verify all EIPs are actively in use
- Route table audit: identify stale routes pointing to deleted resources
- NAT gateway utilization: check SNAT connection counts vs gateway capacity
