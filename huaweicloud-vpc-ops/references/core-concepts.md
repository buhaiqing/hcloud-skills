# VPC Core Concepts — Huawei Cloud Virtual Private Cloud

## Architecture

Virtual Private Cloud (VPC / 虚拟私有云) is the foundational networking service on Huawei Cloud. It provides:

1. **Isolated Network**: Dedicated virtual network per project with customizable IP addressing.
2. **Subnet Segmentation**: Subdivide VPC into subnets within availability zones.
3. **Traffic Control**: Security groups (instance-level) and network ACLs (subnet-level).
4. **Connectivity**: EIPs for public access, VPC Peering for cross-VPC, NAT Gateway for outbound, VPN for on-premises.

### VPC Resource Hierarchy

```
VPC (1 CIDR block)
├── Subnet (1 AZ, subnet CIDR within VPC CIDR)
│   ├── Security Group Rules (stateful, instance-level)
│   ├── ECS Instances
│   └── RDS / ELB / NAT Gateway
├── Route Table (routes traffic between subnets and external)
│   ├── Local route (default for subnet traffic)
│   ├── Custom routes (to peering, NAT, etc.)
│   └── Default route (0.0.0.0/0 → Internet/NAT)
├── Security Groups (firewall rules)
│   ├── Inbound Rules (allow traffic into instances)
│   └── Outbound Rules (allow traffic out of instances)
└── EIP / Bandwidth / NAT Gateway
```

## CIDR Planning

### Allowed CIDR Ranges

| CIDR Range | Available IPs | Typical Use |
|------------|---------------|-------------|
| 10.0.0.0/8 – /28 | 16,777,200 – 14 | Large enterprises, multi-tier |
| 172.16.0.0/12 – /28 | 1,048,560 – 14 | Medium deployments |
| 192.168.0.0/16 – /28 | 65,534 – 14 | Standard, most common |

### Recommended VPC Design

| Scenario | VPC CIDR | Subnet Layout |
|----------|----------|---------------|
| 3-tier app (web/app/db) | 10.0.0.0/16 | 10.0.1.0/24 (web), 10.0.2.0/24 (app), 10.0.3.0/24 (db) |
| Multi-AZ deployment | 10.0.0.0/16 | /24 subnets per AZ (cn-north-4a: 10.0.1.0/24, cn-north-4b: 10.0.2.0/24) |
| Simple single-tier | 192.168.0.0/16 | 192.168.1.0/24 (all resources) |

## Security Group vs Network ACL

| Feature | Security Group | Network ACL |
|---------|---------------|-------------|
| Scope | Instance-level | Subnet-level |
| State | **Stateful** (auto-allow return traffic) | Stateless |
| Rule evaluation | All rules evaluated; allow if any match | Rules evaluated in priority order |
| Default behavior | Deny all inbound, allow all outbound | Deny all traffic |
| Use case | Fine-grained per-instance access | Broad subnet-wide filtering |

## Key VPC Metrics

### Bandwidth Metrics (SYS.VPC namespace)
| Metric Name | Unit | Description |
|-------------|------|-------------|
| bandwidth_in | bps | Inbound bandwidth |
| bandwidth_out | bps | Outbound bandwidth |
| packet_in | pps | Inbound packets per second |
| packet_out | pps | Outbound packets per second |

### EIP Metrics (SYS.EIP namespace)
| Metric Name | Unit | Description |
|-------------|------|-------------|
| eip_bandwidth_in | bps | EIP inbound |
| eip_bandwidth_out | bps | EIP outbound |

## Limits and Quotas

| Resource | Default Limit | Notes |
|----------|--------------|-------|
| VPCs per project | 5 | Adjustable |
| Subnets per VPC | 100 | Adjustable |
| Security groups per VPC | 100 | Adjustable |
| Security group rules per SG | 50 (ingress + egress) | |
| Route entries per route table | 200 | |
| EIPs per project | 20 | Adjustable |
| VPC peerings per project | 50 | |
| NAT gateways per VPC | 10 | |
| SNAT rules per NAT gateway | 20 | |
| DNAT rules per NAT gateway | 20 | |

## Dependency Graph

```
Cloud Resource → VPC (network isolation)
              → Subnet (AZ placement)
              → Security Group (access control)
              → Route Table (traffic routing)
              → EIP (public access) ←→ Bandwidth
              → NAT Gateway (outbound)
              → VPC Peering (cross-VPC)
```

## SPOF Analysis

- **Single VPC = single failure domain**: All resources in one VPC share the same network plane.
- **Multi-AZ subnets**: Deploy subnets across multiple AZs within a VPC for resilience.
- **Redundant NAT**: Use multiple NAT gateways for high-throughput requirements.
- **VPC Peering limitations**: Non-transitive peering — if A peers with B and B with C, A cannot reach C.
