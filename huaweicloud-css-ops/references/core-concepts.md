# CSS Core Concepts

## Overview

Cloud Search Service (CSS) is Huawei Cloud's fully managed, distributed search and analytics engine service, fully compatible with open-source Elasticsearch and OpenSearch.

## Engine Types

### Elasticsearch
- **Version 7.6.2**: Stable release with security and performance improvements
- **Version 7.10.2**: Latest 7.x release with enhanced observability features

### OpenSearch
- **Version 1.3.6**: OpenSearch LTS release, Elasticsearch 7.10 fork
- **Version 2.17.1**: Latest OpenSearch release with advanced analytics

## Node Types

| Type | Code | Purpose | Characteristics |
|------|------|---------|-----------------|
| Data Node | `ess` | Stores data and executes queries | CPU/memory intensive |
| Master Node | `ess-master` | Cluster management and coordination | Low resource, high availability |
| Client Node | `ess-client` | Query load balancing | No data storage |
| Cold Node | `ess-cold` | Archive/infrequently accessed data | Cost-optimized storage |

## Cluster Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      CSS Cluster                             │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ Master Node │  │ Master Node │  │ Master Node │ (HA)     │
│  └──────┬──────┘  └─────────────┘  └─────────────┘          │
│         │                                                    │
│  ┌──────┴──────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ Data Node 1 │  │ Data Node 2 │  │ Data Node 3 │          │
│  │ (Primary)   │  │ (Replica)   │  │ (Replica)   │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Client Nodes (Optional) - Query coordination        │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Network Architecture

- **VPC Isolation**: Clusters deployed in user VPC
- **Subnet**: Private subnet recommended
- **Security Group**: Controls inbound/outbound traffic
- **EIP**: Optional for public access (not recommended for production)

## Storage Options

| Type | Performance | Use Case |
|------|-------------|----------|
| ULTRAHIGH | High IOPS SSD | Production workloads |
| HIGH | Standard SSD | Development, testing |
| COMMON | SATA | Cold data, archiving |

## Key Features

1. **Full-text Search**: Structured and unstructured data
2. **Aggregation**: Real-time analytics and metrics
3. **Near Real-time**: Sub-second indexing latency
4. **Horizontal Scaling**: Add/remove nodes dynamically
5. **Snapshot/Restore**: Backup to OBS
6. **IK Analyzer**: Chinese word segmentation
7. **Security**: HTTPS, encryption, access control

## Quotas and Limits

| Resource | Default Limit | Notes |
|----------|---------------|-------|
| Clusters per region | 20 | Can be increased |
| Nodes per cluster | 32 | Master + Data + Client |
| Max disk per node | 32 TB | Depends on flavor |
| Snapshots per cluster | 200 | Automatic cleanup |
| Indices per cluster | 10,000 | Depends on node specs |

## Supported Regions

- cn-north-1 (北京一)
- cn-north-4 (北京四)
- cn-east-2 (上海二)
- cn-south-1 (广州)
- ap-southeast-1 (香港)
- ap-southeast-3 (曼谷)
