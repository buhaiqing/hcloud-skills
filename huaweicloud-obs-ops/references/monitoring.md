# Monitoring & Alerts — Huawei Cloud OBS

## Core Metrics Table

CES Namespace: **SYS.OBS**

| Metric Name | CES Metric ID | Unit | Description | Recommended Alert Threshold |
|-------------|--------------|------|-------------|---------------------------|
| Request Count (GET) | request_count_get | count | GET API calls per period | Monitor for baselining |
| Request Count (PUT) | request_count_put | count | PUT API calls per period | Monitor for baselining |
| Request Error Rate | request_error_rate | % | Failed requests / total | Warning: >5%, Critical: >20% |
| 4xx Error Count | error_4xx_count | count | Client-side errors | Warning: >100/min |
| 5xx Error Count | error_5xx_count | count | Server-side errors | Warning: >10/min, Critical: >50/min |
| Bandwidth In | bytes_in | bytes/s | Upload traffic rate | Monitor for baselining |
| Bandwidth Out | bytes_out | bytes/s | Download traffic rate | Warning: >80% of expected |
| Storage Usage | storage_bytes | bytes | Total storage consumed | Monitor growth trends |
| Object Count | object_count | count | Total objects in bucket | Monitor for unexpected growth |
| First Byte Latency | first_byte_latency | ms | Time to first byte | Warning: >500ms, Critical: >2000ms |
| Average Latency | average_latency | ms | Average API response time | Warning: >200ms, Critical: >1000ms |

## Alert Recommendations

| Metric | Warning | Critical | Window | Aggregation | Notification |
|--------|---------|----------|--------|-------------|--------------|
| request_error_rate | > 5% | > 20% | 5 min | Average | SMS + Email |
| error_4xx_count | > 100/min | > 500/min | 5 min | Sum | Email |
| error_5xx_count | > 10/min | > 50/min | 5 min | Sum | SMS + Email |
| bytes_out | > 80% expected | > 95% expected | 10 min | Average | Email (cost alert) |
| first_byte_latency | > 500 ms | > 2000 ms | 10 min | P95 | Email |
| storage_bytes growth | > 2x baseline rate | — | 24h | Trend | Email |

## Anomaly Patterns

### Pattern 1: Sudden 4xx Error Spike

**Detection**: `error_4xx_count > 10x baseline` within 5 minutes
**Possible Causes**: Invalid ACL change, bucket policy Deny rule, expired presigned URLs, client misconfiguration
**Action**: Check recent bucket policy/ACL changes, verify presigned URL expiry dates

### Pattern 2: 5xx Server Error

**Detection**: `error_5xx_count > 0` sustained for 5 minutes
**Possible Causes**: OBS service degradation, network partition to OBS endpoint
**Action**: Retry with backoff, check OBS service status, escalate with RequestId if persistent

### Pattern 3: Latency Degradation

**Detection**: `first_byte_latency` P95 > 500ms AND trending upward
**Possible Causes**: Network congestion, OBS endpoint overload, large object without CDN
**Action**: Enable CDN for frequently accessed objects, check network path to OBS endpoint

### Pattern 4: Unusual Egress Traffic Surge

**Detection**: `bytes_out` > 3x daily average within 1 hour
**Possible Causes**: Data exfiltration, CDN cache bypass, bulk download by misconfigured client
**Action**: Investigate source IPs via access logs, temporarily restrict bucket ACL, alert security team

### Pattern 5: Storage Growth anomaly

**Detection**: `storage_bytes` growth rate > 2x expected trend
**Possible Causes**: Uncontrolled uploads, missing lifecycle rules, abandoned multipart uploads
**Action**: Check lifecycle rules (abort stale multipart, expire old objects), review access patterns

## Cost-Related Metrics

| Metric | Cost Implication | Detection | Action |
|--------|----------------|-----------|--------|
| storage_bytes total | Storage cost per GB | Monthly growth trend | Review storage class distribution |
| bytes_out total | Egress traffic cost | Daily/monthly spike analysis | Use CDN, optimize transfer patterns |
| request_count | API request cost (negligible) | Extremely high counts | Batch operations where possible |
| storage_by_class | Cost per storage class | Lifecycle report effectiveness | Adjust lifecycle transition rules |

## Dashboards

### Daily Monitoring Dashboard

| Panel | Metrics | Time Range | Chart Type |
|-------|---------|------------|------------|
| Request Volume | request_count_get, request_count_put | Last 24h | Bar |
| Error Rate | error_4xx_count, error_5xx_count, request_error_rate | Last 24h | Line |
| Latency | first_byte_latency (avg, P95), average_latency | Last 24h | Line |
| Bandwidth | bytes_in, bytes_out | Last 24h | Area |
| Storage | storage_bytes, object_count | Last 30d | Line |

### Monthly Cost Dashboard

| Panel | Metrics | Time Range | Purpose |
|-------|---------|------------|---------|
| Storage Growth | storage_bytes daily | Last 30d | Project monthly storage cost |
| Egress Traffic | bytes_out daily sum | Last 30d | Track egress billing |
| Storage Class Distribution | storage by class | Current | Optimize lifecycle transitions |
