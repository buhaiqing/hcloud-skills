# Huawei Cloud CTS Core Concepts

## What is CTS

Huawei Cloud CTS (Cloud Trace Service / 云审计) captures cloud API calls, user operations, and resource changes across Huawei Cloud. It is designed for audit, compliance, security investigation, and change tracking.

## Key Concepts

- **Trail**: A CTS trail defines which audit events to collect, where they are delivered, and how long they are retained.
- **Audit Event**: A single record of an operation or API call, including user identity, resource, action, result, timestamp, and source IP.
- **Delivery Target**: CTS can deliver events to OBS, SMN, or Log Tank Service (LTS) depending on region and use case.
- **Query**: Search audit events by time range, user, action, resource, status, or custom filters.
- **Retention**: How long CTS retains event data before expiration.

## Architecture

1. **Event Source**: Cloud service APIs and management plane operations generate audit events.
2. **CTS Collection**: The CTS service ingests these events and stores them temporarily for query.
3. **Delivery Layer**: Events are forwarded to audit destinations such as OBS/SMN/LTS.
4. **Query & Analytics**: Operators query events for security analysis, investigation, and compliance reporting.

## Common Use Cases

- **Compliance Audit**: Verify who changed resources and when.
- **Forensic Investigation**: Trace suspicious access or unauthorized operations.
- **Change Tracking**: Track configuration changes across a project.
- **Security Monitoring**: Detect anomalous API calls or operations.

## Limits and Boundaries

- **Trail concurrency**: Projects may have limits on the number of active trails.
- **Delivery destination**: Not all regions support all target types; OBS is most common.
- **Query window**: CTS query ranges are typically bounded by retention and storage delays.
- **Region scope**: Trails and events are region-specific; cross-region correlation requires separate trails.

## When to Use CTS vs Logs

- Use **CTS** when you need **audit-level event history** and **traceable user actions**.
- Use **Cloud Eye / Cloud Monitor** for **metric-based monitoring**.
- Use **Application Logs** for **application-specific payload details**.
- Use **OBS**/LTS for **long-term storage** of audit data.
