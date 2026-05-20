# VPC Knowledge Base — Huawei Cloud Virtual Private Cloud

## Network Fault Patterns

### Pattern 1: Instance Unreachable from Public Network

| Symptom | ECS instance with EIP cannot be pinged or SSH'd to |
|---------|---------------------------------------------------|
| Root Cause Chain |
| 1. EIP not actually bound to instance network card | Check `port_id` on EIP |
| 2. Security group missing inbound rule for ICMP (ping) or port 22 (SSH) | SG blocks traffic |
| 3. Instance OS firewall (iptables/firewalld) blocking | OS-level firewall |
| 4. Wrong route table — no default route to internet | Route misconfiguration |
| Diagnosis Steps |
| 1. Verify EIP `port_id` matches instance network card ID |
| 2. Check SG inbound rules for required port |
| 3. SSH via VPC internal IP (if possible) and check OS firewall |
| 4. Verify route table has 0.0.0.0/0 → Internet Gateway |
| Resolution | Fix the specific misconfiguration in the chain |

### Pattern 2: Private Instances Cannot Reach Internet

| Symptom | Instances without EIP cannot download packages or reach external APIs |
|---------|------------------------------------------------------------------------|
| Root Cause |
| 1. No NAT gateway in VPC | Need to create NAT gateway |
| 2. NAT gateway exists but no SNAT rule for the subnet | SNAT rule must cover subnet CIDR |
| 3. Route table doesn't route 0.0.0.0/0 to NAT gateway | Add route entry |
| 4. Security group blocks outbound DNS (port 53) | Allow outbound DNS |
| Resolution | Create NAT gateway → Add SNAT rule → Add default route → Verify SG egress

### Pattern 3: VPC Peering Connectivity Failure

| Symptom | Two VPCs peered but instances cannot communicate |
|---------|--------------------------------------------------|
| Root Cause Chain |
| 1. Peering not accepted (status: PENDING_ACCEPTANCE) | Peer project must accept |
| 2. No routes in route tables pointing to peering | Add routes on both sides |
| 3. Security groups block cross-VPC traffic | Add SG rules for peer CIDR |
| 4. Overlapping CIDRs between VPCs | Cannot fix — CIDRs must not overlap |
| Resolution | Accept peering → Add routes → Update SG rules |

### Pattern 4: EIP Bandwidth Saturation

| Symptom | Application timeouts, high latency on EIP-bound resources |
|---------|----------------------------------------------------------|
| Root Cause | Bandwidth purchased is lower than actual traffic demand |
| Diagnosis | Check CES metric `eip_bandwidth_out` vs purchased bandwidth size |
| Resolution | Upgrade EIP bandwidth; consider shared bandwidth pool for multiple EIPs |

## Cross-Product Cascade Faults

### Pattern 1: VPC Deletion Breaks Dependent Resources

```
Trigger: VPC deleted (or subnet deleted)
Cascading Impact:
  → ECS instances lose network connectivity
  → RDS instances become unreachable
  → ELB listeners return 502
Root Cause: Network dependency on deleted VPC/subnet
Prevention: Delete dependent resources first; or verify no resources exist
```

### Pattern 2: NAT Gateway Failure Affects All Private Instances

```
Trigger: NAT gateway becomes unhealthy or is deleted
Cascading Impact:
  → All instances in private subnets lose internet access
  → Application deployments fail (can't download dependencies)
  → Cron jobs fail (can't reach external APIs)
Root Cause: Single NAT gateway SPOF
Prevention: Deploy redundant NAT gateways; monitor NAT gateway health
```

### Pattern 3: Security Group Change Breaks Application

```
Trigger: Security group rule modified (e.g., removed inbound rule for port 80)
Cascading Impact:
  → Web server rejects all inbound HTTP traffic
  → ELB health checks fail (return 503)
  → Users see "Service Unavailable"
Root Cause: Overly aggressive SG change without testing
Prevention: Test SG changes on subset of instances first; use change windows
```

## Historical Diagnosis Reference

| Issue | Root Cause | Resolution Time | Prevention |
|-------|------------|-----------------|------------|
| Prod servers unreachable | SG inbound rule accidentally deleted | 15 min | SG change requires approval workflow |
| Cron jobs failing across 50 hosts | NAT gateway quota exceeded, gateway deleted | 1 hour | Monitor NAT gateway health and quota |
| Cross-VPC API calls failing | Peer VPC CIDR changed without route update | 30 min | Document peering dependencies; alert on CIDR changes |
