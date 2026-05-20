# Huawei Cloud CTS Monitoring

## What to Monitor

- **Trail status**: `ACTIVE`, `CREATING`, `FAILED`, `DELETING`
- **Event delivery success**: verify that audit data reaches OBS/SMN/LTS
- **Query latency**: monitor response times for audit queries
- **Error counts**: frequent `Cts.*` failures indicate misconfiguration or permission issues

## Recommended Metrics

1. **Trail Creation Rate**
2. **Trail Delivery Failures**
3. **Query Success Rate**
4. **Event Ingestion Delay**
5. **Destination Write Errors**

## Monitoring Patterns

- Use CTS trail state change as an operational alert trigger.
- Alert on repeated audit delivery failures to prevent blind spots.
- Monitor OBS/SMN/LTS sink health for storage or notification problems.
- Correlate CTS errors with IAM permission changes.

## Visibility Tips

- Confirm regions with CTS support before deploying trails.
- Use separate trails for production and audit/reporting workloads.
- Keep retention aligned with compliance requirements.
- Periodically validate a sample query to ensure events are capturable.
