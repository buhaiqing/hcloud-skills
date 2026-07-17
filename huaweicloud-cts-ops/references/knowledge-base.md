# CTS Knowledge Base

> **Purpose**: Structured fault patterns and remediation knowledge for CTS operations.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## Fault Pattern 1: Trail Delivery Destination Unreachable

### Symptoms
- Trail status is `ACTIVE` but OBS/SMN/LTS receives no new events
- CTS console shows delivery failures with error code `Cts.0101`

### Root Cause
- OBS bucket deleted or access policy changed
- SMN topic removed or IAM permissions revoked
- LTS log group deleted or ingest endpoint changed

### Resolution
1. Validate destination exists: `hcloud obs ls`
2. Check CTS IAM role has `ObsOperator` or equivalent
3. Recreate trail with verified destination

---

## Fault Pattern 2: Query Returns Empty Despite Active Trail

### Symptoms
- Valid time range query returns zero events
- Trail status confirmed `ACTIVE`

### Root Cause
- Query filter is too restrictive (wrong event_name, resource_type)
- Event source service not covered by trail scope
- Time range offset (clock skew between trail and query window)

### Resolution
1. Simplify filter: query without filters first
2. Expand time window by ±10 minutes
3. Confirm service is in trail's event source list

---

## Fault Pattern 3: Duplicate Trail Name

### Symptoms
- Trail creation fails with `Cts.0401`

### Root Cause
- Trail name already exists in the project
- Deleted trail name still in deletion grace period

### Resolution
1. List existing trails: `hcloud cts list-trails`
2. Use unique naming convention: `trail-<project>-<purpose>-<timestamp>`
3. Wait for grace period (typically 7 days) or use a variant name

---

## Fault Pattern 4: Unauthorized Access Despite Valid Credentials

### Symptoms
- CLI returns `AccessDenied` or `Unauthorized`
- Credentials are valid for other services

### Root Cause
- Access key lacks CTS-specific IAM policy
- Cross-project access without proper delegation
- Token expired for temporary credentials

### Resolution
1. Verify CTS policy attached: `huawei iam GetUserPolicies`
2. Ensure policy includes `CTS ReadWrite` or `CTS FullAccess`
3. For cross-project, ensure `HW_PROJECT_ID` matches trail's project

---

## Fault Pattern 5: Quota Exceeded for New Trail

### Symptoms
- Trail creation fails with `Cts.0405`

### Root Cause
- Project-level trail quota already exhausted
- Default quota: 1 trail per project in most regions

### Resolution
1. List current trails: `hcloud cts list-trails`
2. Delete unused trails to free quota
3. Contact Huawei Cloud support to increase quota if all trails are in use

---

## Fault Pattern 6: Delivery Latency Spike

### Symptoms
- Events appear in OBS/SMN/LTS with > 5 minute delay
- CES metric `cts_delivery_latency` shows sustained high values

### Root Cause
- OBS bucket throttling or maintenance
- Network path degradation between CTS and delivery target
- High event volume exceeding delivery pipeline capacity

### Resolution
1. Check OBS bucket status: `hcloud obs list`
2. Check Huawei Cloud service health at status.huaweicloud.com
3. Temporarily reduce event volume by narrowing trail scope

---

## Fault Pattern 7: Trace Volume Spike Without Business Justification

### Symptoms
- Unusual spike in audit event volume
- No corresponding business activity in logs

### Root Cause
- Misconfigured automation generating excessive API calls
- Security event (e.g., port scan, enumeration attack)
- Third-party integration looping on API calls

### Resolution
1. Query top event_types by volume: `hcloud cts query-events --limit 10`
2. Identify the source IP or user responsible
3. Correlate with IAM and VPC flow logs to determine root cause
