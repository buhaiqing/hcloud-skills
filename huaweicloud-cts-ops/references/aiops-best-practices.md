# Huawei Cloud CTS AIOps Best Practices

## AIOps Goals for CTS

Huawei Cloud CTS is a key data source for automated incident detection, root cause analysis, and security operations. AIOps workflows should use CTS audit events as evidence for:

- anomaly detection in resource access patterns
- automated correlation of user operations and service failures
- incident investigation and reconstruction
- compliance validation and alert enrichment

## Recommended AIOps Patterns

### 1. Event-driven investigation

- Use CTS query results to identify unusual `Create`, `Delete`, `Update` actions on critical resources.
- Correlate audit events with recent ECS/RDS/OBS operations from other product skills.
- Convert event sequences into structured incident narratives for downstream remediation.

### 2. Adaptive query templates

- Provide prebuilt filter templates for common security checks:
  - `event_name=DeleteServer AND status=success`
  - `user_identity=*admin* AND event_name=UpdateSecurityGroup`
  - `resource_name=*production* AND event_name=CreateBucket`
- Allow the agent to tune query windows dynamically based on event density.

### 3. Alert enrichment

- Enrich alerts with CTS audit evidence such as actor, source IP, and request parameters.
- When trace queries show repeated failures or unauthorized attempts, escalate to SecOps.
- Attach direct CTS query links or audit event IDs to incident tickets.

### 4. Automation feedback loop

- Use CTS query success/failure as an automated validation step after remediation.
- If a remediation action changes audit trail delivery, verify the trail remains active and events are still captured.
- Detect drift when CTS delivery destination changes unexpectedly.

## Integration with Other AIOps Sources

- Combine CTS audit events with Cloud Eye metrics for operational context.
- Use OBS/LTS delivery health as a signal when event ingestion fails.
- Feed CTS event anomalies into anomaly detection models alongside network and KPI metrics.

## Practical Recommendations

- Prefer structured `query_events` outputs over manual log parsing.
- Keep filter complexity manageable to avoid slow queries.
- Periodically sample and validate critical audit queries to ensure coverage.
- Use CTS for forensic evidence, not as a real-time security stream.

## Metrics and Signals

- `audit_event_query_success_rate`
- `audit_delivery_failure_count`
- `trail_activation_latency`
- `invalid_filter_error_rate`
- `unauthorized_access_query_count`

## Example AIOps Scenario

1. A suspicious login comes from an unfamiliar IP.
2. CTS queries are executed for `event_name=Login AND source_ip=<ip>`.
3. The agent correlates the CTS results with recent IAM and ECS events.
4. If matched, generate an enriched incident report and recommend policy review.
