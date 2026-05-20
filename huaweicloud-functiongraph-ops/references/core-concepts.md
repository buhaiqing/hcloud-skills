# Core Concepts — Huawei Cloud FunctionGraph

## Architecture

FunctionGraph is a serverless compute service that executes code in response to events. Key architectural concepts:

```
Event Source (APIG/OBS/SMN/Timer/etc.)
    │
    ▼
FunctionGraph Service
    │
    ├── Trigger ──► Function (Code + Config)
    │                   │
    │                   ├── Version/Alias (immutable snapshots)
    │                   ├── Reserved Instance (concurrency control)
    │                   └── VPC Access (optional)
    │
    ▼
Execution Logs (LTS) ──► Metrics (CES)
```

## Key Concepts

### Function
The core compute unit. Defined by:
- **Function URN**: `urn:fss:{region}:{project_id}:function:{name}:{version}`
- **Runtime**: Python 2.7/3.6/3.9/3.10, Node.js 14.18/16.17/18.15, Java 8/11/17, Go 1.x, C#, PHP
- **Handler**: Entry point (e.g., `index.handler`, `com.example.MyHandler::handleRequest`)
- **Timeout**: 1–300 seconds (default 30s)
- **Memory**: 128–4096 MB, step 128 (default 256)
- **Ephemeral storage**: 512 MB–10 GB (default 512 MB)

### Trigger
Event source that invokes the function:

| Trigger Type | Event Source | Use Case |
|-------------|-------------|----------|
| TIMER | CloudTiming | Scheduled execution |
| APIG | API Gateway | HTTP API endpoint |
| OBS | Object Storage | File upload events |
| SMN | Simple Message Notification | Message notifications |
| LTS | Log Tank Service | Log streaming processing |
| CTS | Cloud Trace Service | Resource operation events |
| DMS | Distributed Message Service | Kafka/RocketMQ message consumption |
| DEDICATEDGATEWAY | APIG Dedicated | Dedicated API gateway |

### Version & Alias
- **Version**: Immutable snapshot of function code + config (`v1.0.0`, `v2.0.0`)
- **Alias**: Mutable pointer to version (`prod`, `staging`) with traffic distribution support

### Reserved Instance
Guaranteed concurrency for a function. Avoids cold starts but incurs cost regardless of invocation.

## Limits & Quotas

| Resource | Default Quota | Max |
|----------|--------------|-----|
| Functions per account | 100 | 500 (request) |
| Code size (inline) | 10 MB | — |
| Code size (OBS) | 10 GB | — |
| Timeout | 300s | 300s |
| Memory | 4096 MB | 4096 MB |
| Concurrent executions | 300 per account | 1000 (request) |
| Trigger types per function | 10 | 10 |
| Version count per function | — | 50 |
| Alias count per function | — | 5 |

## Regional Endpoints

| Region | Endpoint |
|--------|----------|
| cn-north-4 | functiongraph.cn-north-4.myhuaweicloud.com |
| cn-east-3 | functiongraph.cn-east-3.myhuaweicloud.com |
| cn-south-1 | functiongraph.cn-south-1.myhuaweicloud.com |
| ap-southeast-1 | functiongraph.ap-southeast-1.myhuaweicloud.com |

## Dependency Graph

```
FunctionGraph
    ├── OBS (code package storage)
    ├── APIG (HTTP triggers)
    ├── SMN (notification triggers)
    ├── LTS (execution logs)
    ├── CTS (async invocation tracing)
    ├── DMS / Kafka (message event triggers)
    ├── CES (monitoring metrics)
    ├── VPC / Subnet (VPC access)
    └── IAM (permissions & agency)
```

## SPOF Analysis

| Component | Risk | Mitigation |
|-----------|------|------------|
| Single function | Function bug breaks all invocations | Version/alias with traffic shifting, canary deployments |
| No VPC access | Cannot access RDS/DCS in VPC | Enable VPC access for private resources |
| Cold start latency | >1s P99 latency for infrequent functions | Reserved instances for latency-sensitive functions |
| Quota exhaustion | 300 concurrent limit reached | Request quota increase, use async invocations |
| Code package corruption | OBS URL wrong or deleted | Use versioned OBS buckets, verify checksums |
