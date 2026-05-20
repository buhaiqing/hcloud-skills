# Core Concepts — Huawei Cloud ECS

## Architecture

Huawei Cloud ECS (Elastic Cloud Server) is built on a virtualized compute layer with the following components:

```
┌─────────────────────────────────────┐
│              ECS Instance            │
│  ┌────────┐ ┌────────┐ ┌─────────┐  │
│  │  vCPU  │ │ Memory │ │  VBD    │  │
│  │ (KVM)  │ │ (RAM)  │ │ (I/O)   │  │
│  └────────┘ └────────┘ └─────────┘  │
│                                     │
│  ┌────────┐ ┌────────┐ ┌─────────┐  │
│  │  VPC   │ │  SG    │ │  EIP    │  │
│  │ Network│ │ Rules  │ │ (NAT)   │  │
│  └────────┘ └────────┘ └─────────┘  │
└─────────────────────────────────────┘
         │              │              │
         ▼              ▼              ▼
    EVS Volume    Security      Elastic IP
    (Block)       Groups        (Public)
```

## Resource Limits

| Resource | Default Limit | Notes |
|----------|--------------|-------|
| ECS instances per account | 20 (per region) | Increase via quota request |
| Volumes per instance | 24 | Including root disk |
| Security groups per instance | 5 | — |
| EIPs per instance | 1 | Via NAT gateway can share |
| Snapshots per volume | 32 | Per volume |
| Images per account | 500 | Private images |

## Regions and Availability Zones

| Region Code | Location | AZs Available |
|-------------|----------|--------------|
| `cn-north-4` | Beijing4 | 2+ AZs |
| `cn-north-1` | Beijing1 | 2 AZs |
| `cn-east-3` | Shanghai1 | 2 AZs |
| `cn-south-1` | Guangzhou | 2 AZs |
| `ap-southeast-1` | Singapore | 2 AZs |
| `ap-southeast-3` | Bangkok | 1 AZ |

## ECS Flavors

Flavors follow the naming pattern: `family.generation.size`

| Series | Use Case | vCPU:Memory |
|--------|----------|-------------|
| `s3` (General) | Web servers, dev/test | 1:2, 1:4 |
| `c3` (Compute) | HPC, rendering | 1:1, 1:2 |
| `m3` (Memory) | Databases, caching | 1:4, 1:8 |
| `d3` (Disk) | Big data, storage | 1:4 with large local disks |
| `i3` (High I/O) | NoSQL, search engines | 1:4, high IOPS |
| `t3` (Burst) | Sporadic workloads | 1:1, 1:2 |

## Dependency Graph

```
ECS Instance
├── Requires: VPC + Subnet
├── Requires: Security Group
├── Requires: Image (EVS system disk)
├── Optional: EVS data disk(s)
├── Optional: EIP (public access)
├── Optional: Cloud-Cell Agent (remote exec)
├── Optional: Key Pair (SSH access)
└── Monitored by: CES metrics → LTS logs → AOM traces
```

## Single Point of Failure Analysis

- Single ECS instance = SPOF (no built-in HA)
- Mitigation: Multi-AZ deployment via AS (Auto Scaling) or multiple instances behind ELB
- Single AZ = regional outage risk
- Mitigation: Cross-region DR via CBS replication or DNS failover

## Cloud-Cell Agent (云主机助手)

The Cloud-Cell Agent enables remote command execution and file transfer without SSH exposure:
- Pre-installed on some marketplace images
- Manual installation: download agent binary, run install script
- Requires outbound HTTPS (443) to `*.myhuaweicloud.com`
- Security group must allow outbound connections
- IAM permission: `ECS:ecsCloudCell:executeCommand`
