# Knowledge Base — Huawei Cloud OBS Fault Patterns

## Product Fault Pattern Library

### FP-001: Access Denied After Policy Change

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-001 |
| **Symptom** | Previously working application suddenly gets 403 when accessing OBS objects |
| **Trigger Conditions** | `error_4xx_count spike` + specific `AccessDenied` errors + recent bucket policy change |
| **Root Cause** | Bucket policy or ACL changed, removing previously granted access |
| **Diagnosis Flow** | 1. `obsutil get-acl obs://bucket` → check current ACL<br>2. `obsutil get-policy obs://bucket` → check for Deny statements<br>3. Review CTS audit log for recent policy changes<br>4. Verify IAM policy still covers OBS actions |
| **Resolution Steps** | 1. If ACL changed: restore to previous ACL<br>2. If policy changed: add Allow statement for affected user/action<br>3. Verify access restored: `obsutil ls obs://bucket`<br>4. If presigned URLs expired: regenerate |
| **Prevention** | CTS monitoring for bucket policy changes, change management process, test policy before applying |
| **CES Metrics** | error_4xx_count, request_error_rate |

### FP-002: Multipart Upload Stuck

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-002 |
| **Symptom** | Upload never completes, incomplete parts consume storage, no usable object |
| **Trigger Conditions** | InitiateMultipartUpload succeeded but CompleteMultipartUpload failed or was never called |
| **Root Cause** | Network interruption during part upload, application crash before completion, or wrong part numbers |
| **Diagnosis Flow** | 1. `obsutil listuploads obs://bucket` → find stuck uploads<br>2. `obsutil listparts obs://bucket/key --uploadId=xxx` → check uploaded parts<br>3. Compare uploaded parts with expected (all parts ≥ 5MB except last) |
| **Resolution Steps** | 1. `obsutil abortupload obs://bucket/key --uploadId=xxx` → clean up<br>2. Retry upload with multipart<br>3. If network unstable: reduce threadNum, increase retry count |
| **Prevention** | Configure lifecycle rule to auto-abort incomplete uploads after 7 days:<br>`{"AbortIncompleteMultipartUpload": {"DaysAfterInitiation": 7}}` |
| **CES Metrics** | storage_bytes (growing from stale parts) |

### FP-003: Lifecycle Rule Conflict

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-003 |
| **Symptom** | Objects not transitioning to expected storage class, or expired prematurely |
| **Trigger Conditions** | Multiple lifecycle rules with overlapping prefixes |
| **Root Cause** | Conflicting rules: one rule transitions to Warm at 30d, another expires at 20d → expiration wins |
| **Diagnosis Flow** | 1. `obsutil get-lifecycle obs://bucket` → list all rules<br>2. Check for overlapping prefixes and conflicting actions<br>3. Rule priority: Expiration > Transition; more specific prefix > less specific |
| **Resolution Steps** | 1. Reorder rules: specific prefixes first, then general<br>2. Remove conflicting expiration that fires before transition<br>3. Test with sample objects matching each prefix |
| **Prevention** | Validate lifecycle rules before applying, use non-overlapping prefixes, document rule intent |

### FP-004: Versioning Mass Delete Recovery

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-004 |
| **Symptom** | Thousands of objects disappeared from bucket, application errors |
| **Trigger Conditions** | DeleteObjects or batch delete operation executed on versioned bucket |
| **Root Cause** | Accidentally deleted all objects (or wildcard delete) on versioned bucket |
| **Diagnosis Flow** | 1. Check if versioning was enabled: `obsutil get-versioning obs://bucket`<br>2. `obsutil listversions obs://bucket` → look for objects before delete markers<br>3. Count delete markers vs actual object versions |
| **Resolution Steps** | 1. If versioning was enabled: delete the delete markers (delete marker has versionId)<br>2. `obsutil rm obs://bucket/key --versionId={delete_marker_versionId}`<br>3. For bulk recovery: script to find and remove all delete markers<br>4. Verify objects reappear: `obsutil ls obs://bucket` |
| **Prevention** | Never use wildcard delete in production, require confirmation for batch deletes, enable MFA delete |

### FP-005: Cross-Region Replication Desync

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-005 |
| **Symptom** | Destination bucket missing recently created objects from source |
| **Trigger Conditions** | CRR status shows lag, object count mismatch between regions |
| **Root Cause** | Network disruption between regions, destination bucket versioning was disabled, or CRR rule was modified |
| **Diagnosis Flow** | 1. `obsutil get-replication obs://bucket` → check CRR status<br>2. Compare object counts and latest modification times between regions<br>3. Check if versioning is enabled on destination (required for CRR) |
| **Resolution Steps** | 1. Fix network connectivity between regions<br>2. If versioning was disabled on destination: re-enable, objects created during disabled period will NOT replicate (need manual sync)<br>3. For missing objects: manually copy from source to destination<br>4. Monitor replication catching up |
| **Prevention** | Monitor CRR status via CES, alert on replication lag > 1 hour, verify versioning on both buckets |

### FP-006: CDN Cache Invalidation Failure

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-006 |
| **Symptom** | Updated content not served through CDN, users see stale data |
| **Trigger Conditions** | Object updated in OBS, CDN still serving old version |
| **Root Cause** | CDN cache TTL not expired after OBS update, or CDN purge failed |
| **Diagnosis Flow** | 1. Access object via CDN URL → check content<br>2. Access object via direct OBS URL → compare content<br>3. If different: CDN cache has not been invalidated<br>4. Check CDN cache purge status/history |
| **Resolution Steps** | 1. Purge CDN cache for affected URL(s)<br>2. Set appropriate Cache-Control headers on OBS objects<br>3. For dynamic content: use Cache-Control: no-cache + ETag validation |
| **Prevention** | Automate CDN purge after OBS upload, use versioned object keys for immutable assets, set appropriate max-age |

## Cross-Product Cascade Faults

### CF-001: IAM Permission Revoked → OBS Access Denied

| Field | Value |
|-------|-------|
| **Cascade ID** | CF-001 |
| **Root Product** | IAM (`huaweicloud-iam-ops`) |
| **Affected Product** | OBS |
| **Symptom** | All OBS API calls return 403 simultaneously |
| **Trigger** | IAM policy attached to AK's user was modified or revoked |
| **Diagnosis** | 1. Verify AK is still active: check IAM user status<br>2. List IAM user policies → check for OBS permissions<br>3. Cross-reference with CTS audit log for policy change timestamp |
| **Resolution** | 1. Re-attach OBS policy to IAM user<br>2. If root AK was disabled: use secondary AK or create new IAM user<br>3. Verify OBS access restored |
| **Prevention** | CTS monitoring for IAM policy changes, dual AK configuration, no single point of failure for credentials |

### CF-002: Network Route Change → VPC Endpoint Unreachable

| Field | Value |
|-------|-------|
| **Cascade ID** | CF-002 |
| **Root Product** | VPC (`huaweicloud-vpc-ops`) |
| **Affected Product** | OBS |
| **Symptom** | ECS accessing OBS via VPC Endpoint times out |
| **Trigger** | VPC route table modified, removing OBS Endpoint route |
| **Diagnosis** | 1. VPC Endpoint status → check if endpoint still ACTIVE<br>2. Route table check → OBS prefix route exists?<br>3. DNS resolution for endpoint → resolves to correct IP? |
| **Resolution** | 1. Restore OBS route in VPC route table<br>2. If endpoint was deleted: recreate VPC Endpoint for OBS<br>3. Verify ECS can reach OBS endpoint via VPC |
| **Prevention** | CTS monitoring for route table changes, infrastructure-as-code for network configuration |

## Additional OBS-Specific Fault Patterns

### FP-007: Storage Quota Exhaustion

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-007 |
| **Symptom** | Upload fails with 403 or "quota exceeded", bucket appears full |
| **Trigger Conditions** | Bucket size approaching or exceeding bucket quota limit |
| **Root Cause** | Bucket quota not set appropriately for data growth, or unexpected data accumulation |
| **Diagnosis Flow** | 1. `obsutil stat obs://bucket` → check current bucket size and quota<br>2. Compare with historical growth rate<br>3. List large objects: `obsutil ls obs://bucket -s -t` |
| **Resolution Steps** | 1. Increase bucket quota in OBS console or via API<br>2. Clean up unnecessary objects or move to another bucket<br>3. Implement lifecycle rules to archive or delete old data |
| **Prevention** | Set quota with 20% buffer above current usage, monitor growth trend, configure CES alarm at 80% |
| **CES Metrics** | bucket_size_bytes, bucket_object_count |

### FP-008: Request Throttling Active

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-008 |
| **Symptom** | HTTP 503 errors, slow request latency, request timeouts |
| **Trigger Conditions** | Request rate exceeds bucket or account limits, bandwidth saturation |
| **Root Cause** | Traffic spike, poorly designed application (excessive requests), account throttling limits |
| **Diagnosis Flow** | 1. Check CES metrics for 503 response rate<br>2. `obsutil stat obs://bucket` → check bandwidth<br>3. Review request patterns for anomalies |
| **Resolution Steps** | 1. Implement request retry with exponential backoff<br>2. Reduce request rate from client side<br>3. Contact support to increase throttling limits if necessary |
| **Prevention** | Implement client-side rate limiting, use CDN for static content, design for expected traffic |
| **CES Metrics** | request_5xx_count, request_count, bandwidth |
| **AIOps Correlation** | See `advanced/aiops-patterns.md` for throttling detection pattern |

### FP-009: Cross-Region Replication Lag

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-009 |
| **Symptom** | Destination bucket lags behind source, objects not replicated in expected time |
| **Trigger Conditions** | CRR status shows lag > 1 hour, replication task stuck or failed |
| **Root Cause** | Network issues between regions, destination bucket issues, CRR configuration changed |
| **Diagnosis Flow** | 1. `obsutil get-replication obs://bucket` → check CRR status<br>2. Check bandwidth between regions<br>3. Verify destination bucket versioning is still enabled |
| **Resolution Steps** | 1. Fix network connectivity<br>2. Re-enable versioning on destination if disabled<br>3. Manually sync missing objects: `obsutil sync` |
| **Prevention** | Monitor CRR lag via CES, alert at 30 minute lag threshold |
| **CES Metrics** | replicate_byte_lag, replicate_object_lag |
| **AIOps Correlation** | See `advanced/aiops-patterns.md` for latency spike pattern |
