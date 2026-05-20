# CCE Core Concepts — Huawei Cloud Cloud Container Engine

## Architecture Overview

CCE (Cloud Container Engine / 云容器引擎) is a managed Kubernetes service on Huawei Cloud. It consists of:

- **Control Plane (Managed by Huawei Cloud):** API server, scheduler, controller manager, etcd. Users do not manage these components directly.
- **Data Plane (Managed by User):** Worker nodes (ECS or BMS) running kubelet, kube-proxy, and container runtime (containerd/Docker).
- **Addons:** Cluster-level components like coredns, metrics-server, everest (CSI) installed via CCE addon system.

```
User → CCE API → Control Plane (Managed) → Worker Nodes (ECS/BMS) → Pods → Containers
```

## Cluster Types

| Cluster Type | Infrastructure | Use Case | Cost |
|-------------|---------------|----------|------|
| **VirtualMachine** | ECS instances | General purpose, most common | Standard ECS pricing |
| **BareMetal** | BMS (bare metal servers) | High-performance, latency-sensitive | Premium |
| **ARM** | Kunpeng processors | ARM-native workloads, cost-effective | Lower for ARM-compatible apps |

## Network Modes

| Mode | Description | When to Use |
|------|------------|-------------|
| **vpc** (VPC Network) | Pods have direct VPC IP addresses; best performance, limited by VPC subnet size | High-throughput services, direct VPC access needs |
| **tunnel** (Container Network Tunnel | VxLAN tunnel overlay; more pods per node, slight overhead | Large pod counts, IP conservation |

## Node Specifications

### Node Flavors

| Flavor Family | CPU:Memory | Examples | Best For |
|--------------|------------|----------|----------|
| **s6** | 1:2 to 32:128 | s6.small.1, s6.large.2 | General purpose |
| **c7** | 1:0.75 to 32:24 | c7.large.2, c7.xlarge.4 | CPU-intensive |
| **m7** | 1:4 to 48:384 | m7.large.2, m7.xlarge.8 | Memory-intensive |
| **hi3** | 1:1 to 64:448 | hi3.3xlarge.2 | High I/O databases |

### Operating Systems

| OS | Version | Compatible | Notes |
|----|---------|-----------|-------|
| EulerOS | 2.9, 2.10 | All flavors | Default, Huawei Cloud Linux |
| CentOS | 7.9 | Most flavors | Legacy support |
| Ubuntu | 20.04, 22.04 | Most flavors | Community preference |

## Node Pool Concepts

Node pools enable:
- **Autoscaling:** Automatic node count adjustment based on resource utilization
- **Flavor grouping:** Same ECS flavor across pool nodes for predictable scheduling
- **Lifecycle management:** Batch update, replace, or drain nodes
- **Key parameters:** `initial_node_count`, `min_node_count`, `max_node_count`, `scale_down_cooldown_time`

## Addon Ecosystem

| Addon | Function | Required? | Description |
|-------|----------|-----------|-------------|
| **coredns** | DNS resolution for services | Yes (auto-installed) | Kubernetes DNS service |
| **everest** | Container Storage Interface (CSI) | Yes (auto-installed) | PersistentVolume provisioning |
| **metrics-server** | Resource metrics | Recommended | HPA/VPA metrics source |
| **kube-proxy** | Service networking | Yes (auto-installed) | Service IP routing |
| **cloud-eye** | Monitoring agent | Recommended | CES metrics collection |
| **log-dis** | Log collection | Optional | LTS log forwarding |
| **network-policy** | Pod network policies | Optional | Calico-based network policies |
| **gpu-driver** | GPU support | GPU nodes only | NVIDIA driver + toolkit |

## Resource Relationships

```
Project
├── CCE Cluster (1 per project × many)
│   ├── Node Pool (0..N per cluster)
│   │   └── Node (0..N per pool)
│   ├── Addon (0..N per cluster)
│   └── Namespace (0..N per cluster)
└── VPC (shared dependency)
    └── Subnet (nodes attach here)
```

### External Dependencies

| Dependency | Service | Purpose |
|-----------|---------|---------|
| VPC | Virtual Private Cloud | Cluster networking, pod networking |
| Subnet | VPC Subnet | Node IP allocation |
| Security Group | VPC Security Group | Node firewall rules |
| ECS | Elastic Cloud Server | Node compute (VirtualMachine/BareMetal clusters) |
| EVS | Elastic Volume Server | PersistentVolume storage via everest CSI |
| EIP | Elastic IP | LoadBalancer service external access |
| ELB | Elastic Load Balance | Service type=LoadBalancer integration |
| SWR | Software Repository for Container | Container image registry |

## Regions, AZs, Quotas, Limits

| Resource | Default Limit | Can Increase | Notes |
|----------|--------------|-------------|-------|
| CCE Clusters per project | 50 | Yes | Contact support |
| Nodes per cluster | 1,000 | No | Hard limit |
| Pods per node | Depends on node flavor | No | VPC mode limited by subnet IPs |
| Node Pools per cluster | 200 | Yes | Contact support |
| Addons per cluster | 50 | No | System + user addons |

## SPOF Analysis

| Component | Single Point of Failure? | Mitigation |
|-----------|------------------------|-----------|
| CCE Control Plane | No (multi-AZ managed by Huawei) | Managed HA by Huawei Cloud |
| Worker Nodes | Yes (individual node failure) | Multi-node pools, pod anti-affinity |
| VPC/Subnet | Yes (if single subnet) | Multi-subnet across AZs |
| CoreDNS | No (replicated automatically) | Multiple replicas by default |
