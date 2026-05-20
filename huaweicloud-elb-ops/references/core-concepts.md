# Core Concepts — Huawei Cloud ELB

## Architecture

Huawei Cloud ELB distributes incoming traffic across multiple backend servers.

```
Internet / Client
    │
    ▼
┌─────────────────────────────────────┐
│        Elastic Load Balancer         │
│  ┌────────┐  ┌────────┐  ┌────────┐ │
│  │ Listener│  │ Listener│  │ Listener│ │
│  │ :80 HTTP│  │:443 HTTP│  │:9090 TCP│ │
│  └────┬───┘  └────┬───┘  └────┬───┘ │
│       │           │           │      │
│  ┌────▼───┐  ┌────▼───┐  ┌────▼───┐ │
│  │ Pool A │  │ Pool B │  │ Pool C │ │
│  │ROUND   │  │LEAST   │  │SOURCE  │ │
│  │_ROBIN  │  │_CONNECT│  │_IP     │ │
│  └────┬───┘  └────┬───┘  └────┬───┘ │
│       │           │           │      │
│  ┌────▼───┐  ┌────▼───┐  ┌────▼───┐ │
│  │Members │  │Members │  │Members │ │
│  │10.0.1.2│  │10.0.1.3│  │10.0.1.4│ │
│  │10.0.1.5│  │10.0.1.6│  │10.0.1.7│ │
│  └────────┘  └────────┘  └────────┘ │
│  ┌────────┐  ┌────────┐             │
│  │Health  │  │Health  │             │
│  │Monitor │  │Monitor │             │
│  │ /health│  │ :8080  │             │
│  └────────┘  └────────┘             │
└─────────────────────────────────────┘
```

## Key Concepts

### Load Balancer Types

| Type | Category | L4 (TCP/UDP) | L7 (HTTP/HTTPS) | Isolation | Best For |
|------|----------|-------------|-----------------|-----------|----------|
| **Shared** (共享型) | Classic | ✅ | ✅ | Multi-tenant | Dev/test, small workload |
| **Dedicated** (独享型) | ALB/NLB | ✅ | ✅ | Single-tenant | Production, high traffic |

### ELB Categories

| Category | Protocol | Features | Use Case |
|----------|----------|----------|----------|
| **Network (NLB)** | TCP, UDP, QUIC | Low latency, source IP preservation | Real-time apps, gaming, IoT |
| **Application (ALB)** | HTTP, HTTPS, gRPC | Host/Path routing, SSL offload, WAF integration | Web apps, microservices, APIs |

### Listener
Defines how LB listens for traffic:
- **Protocol**: HTTP, HTTPS, TCP, UDP, QUIC, gRPC
- **Port**: 1–65535
- **Default pool**: Pool to forward traffic to
- **SSL/TLS**: HTTPS requires certificate ref
- **Idle timeout**: Max connection idle time (60s)

### Backend Pool
Group of backend servers receiving traffic:
- **Algorithm**: `ROUND_ROBIN`, `LEAST_CONNECTIONS`, `SOURCE_IP`, `QUIC_CID`
- **Session persistence**: None, source IP, HTTP cookie, APP cookie
- **Slow start**: Gradual traffic increase to new members

### Health Monitor (Health Check)
Periodic check of member health:
- **Type**: TCP, HTTP, HTTPS, PING, UDP, gRPC
- **Delay**: Interval between checks (1–50s)
- **Timeout**: Max wait per check (1–50s)
- **Max retries**: Consecutive failures before marking unhealthy (1–10)
- **Path**: HTTP/HTTPS check path (e.g., `/health`)

### Backend Member
An ECS instance or private IP receiving traffic:
- **Address**: Private IP of backend server
- **Protocol port**: Port the backend service listens on
- **Subnet**: Must belong to LB's VPC
- **Weight**: 1–256 for traffic distribution weight

## Limits & Quotas

| Resource | Default Quota | Dedicated Quota |
|----------|--------------|-----------------|
| Load balancers | 10 | 30 (requestable) |
| Listeners per LB | 10 | 50 |
| Pools per LB | 10 | 50 |
| Members per pool | 100 | 500 |
| Health monitors per pool | 1 | 1 |
| Certificates per account | 100 | 100 |
| EIPs per LB (public) | 1 | 3 |

## Regional Endpoints

| Region | Endpoint |
|--------|----------|
| cn-north-4 | elb.cn-north-4.myhuaweicloud.com |
| cn-east-3 | elb.cn-east-3.myhuaweicloud.com |
| cn-south-1 | elb.cn-south-1.myhuaweicloud.com |
| ap-southeast-1 | elb.ap-southeast-1.myhuaweicloud.com |

## Dependency Graph

```
ELB
    ├── VPC (network)
    │   ├── Subnet (LB subnet)
    │   └── EIP (public access)
    ├── ECS (backend members)
    │   └── Security Group (member access)
    ├── WAF (HTTPS protection)
    ├── CES (monitoring metrics)
    ├── LTS (access logs)
    └── IAM (permissions)
```

## SPOF Analysis

| Component | Risk | Mitigation |
|-----------|------|------------|
| Single LB | Entire traffic fails | Deploy multi-region LB, use DNS failover |
| Single AZ | AZ failure knocks LB offline | Dedicated LB supports multi-AZ deployment |
| Single backend member | Individual server fails | ≥ 2 members per pool, multi-AZ spread |
| Health check misconfig | Healthy members marked unhealthy | Validate path, delay, timeout, max_retries |
| Certificate expiry | HTTPS listener fails | Set certificate expiry alarm, auto-renew |
