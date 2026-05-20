# Core Concepts — Huawei Cloud LTS

## Architecture Overview

LTS (Log Tank Service) collects, stores, searches, and analyzes logs from Huawei Cloud services and user applications. The data model is hierarchical:

```
Project
 └── Log Group (logical container, determines retention TTL)
      └── Log Stream (actual log data container)
           ├── Raw logs (ingested via ICAgent/API/SDK/Kafka)
           └── Structured/indexed logs (for search & analysis)
```

### Key Concepts

| Concept | Description | Limits |
|---------|-------------|--------|
| **Log Group** | Logical container for log streams. Defines retention period (`ttl_in_days`). | 100 per project |
| **Log Stream** | Actual log data container within a group. Logs are written to streams. | 200 per group |
| **ICAgent** | Log collection agent installed on ECS/bare-metal hosts. Collects file/standard output/system logs. | 1 per host |
| **Structured Parsing** | Converts raw logs to structured KV/JSON for indexed search. | 10 parsing rules per stream |
| **Index Configuration** | Enables SQL analysis and fast search on structured fields. | Must be enabled per stream |
| **Log Transfer** | Exports logs to OBS or DMS for long-term storage or external processing. | 10 transfer rules per stream |
| **Dashboard** | Visualizes log search/analysis results with charts and tables. | 50 per project |
| **Quick Search (Saved Search)** | Persists frequently-used search queries. | 100 per project |

### Resource Hierarchy

```
huaweicloud-lts-ops
├── Log Group Management
│   ├── CreateLogGroup    → POST /v2/{project_id}/groups
│   ├── ListLogGroups     → GET  /v2/{project_id}/groups
│   ├── UpdateLogGroup    → PUT  /v2/{project_id}/groups/{log_group_id}
│   └── DeleteLogGroup    → DELETE /v2/{project_id}/groups/{log_group_id}
├── Log Stream Management
│   ├── CreateLogStream   → POST /v2/{project_id}/groups/{log_group_id}/streams
│   ├── ListLogStreams    → GET  /v2/{project_id}/log-streams
│   └── DeleteLogStream   → DELETE /v2/{project_id}/groups/{log_group_id}/streams/{log_stream_id}
└── Log Operations
    ├── ListLogs          → POST /v2/{project_id}/groups/{log_group_id}/streams/{log_stream_id}/content/query
    ├── CreateTransfer    → POST /v2/{project_id}/transfers
    ├── ListTransfers     → GET  /v2/{project_id}/transfers
    └── DeleteTransfer    → DELETE /v2/{project_id}/transfers/{log_transfer_id}
```

### Regions

LTS is available in all Huawei Cloud regions. Cross-region transfer is not supported — transfer targets must be in the same region.

### Quotas

| Resource | Default Quota | Adjustable |
|----------|---------------|------------|
| Log Groups per project | 100 | Yes (support ticket) |
| Log Streams per group | 200 | No |
| Transfer rules per stream | 10 | Yes (support ticket) |
| Dashboards per project | 50 | No |
| Quick searches per project | 100 | No |
| ICAgent per host | 1 | No |

### Billing Model

| Component | Charged By | Notes |
|-----------|-----------|-------|
| Log ingestion | GB/month (compressed) | Different tiers for text vs structured |
| Log storage | GB/month | Based on `ttl_in_days` — shorter TTL = less cost |
| Index volume | GB/month | Indexing structured fields incurs additional cost |
| Log transfer | GB transferred | Free within region; inter-region transfer charged |
| Dashboard/Analysis | Query CU | Charged per query complexity/data scanned |

### Dependencies

| Service | Relationship |
|---------|-------------|
| **ECS/BMS** | Hosts running ICAgent for log collection |
| **CCE** | Container logs collected via ICAgent or sidecar |
| **OBS** | Log transfer target for long-term/cold storage |
| **DMS** | Log transfer target for real-time log consumption |
| **CTS** | Records all LTS API calls for audit |
| **CES** | LTS metrics (log volume, transfer status) reported to CES |

### Single Points of Failure (SPOF) Analysis

| Component | Risk | Mitigation |
|-----------|------|------------|
| ICAgent failure | Logs not collected | Multi-path ingestion (SDK fallback); heartbeat monitoring via CES |
| Log group deleted | All streams & logs lost | IAM deny policy; CTS monitoring on DeleteLogGroup |
| Storage full | Log ingestion throttled | Monitor storage via CES; set CES alarm at 80% |
| Transfer failure | Log data not exported | CES alarm on transfer failures; retry mechanism |
