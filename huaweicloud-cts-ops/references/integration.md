# Huawei Cloud CTS Integration

## Recommended Delegation

- **IAM**: `huaweicloud-iam-ops` should manage service roles and permissions for CTS to write to OBS/SMN/LTS.
- **OBS**: `huaweicloud-obs-ops` should provision buckets and validate bucket policies for audit delivery.
- **SMN**: `huaweicloud-smn-ops` should manage topics and subscriptions when CTS sends notifications.
- **LTS**: `huaweicloud-lts-ops` should manage log group sinks and retention.
- **CES**: `huaweicloud-ces-ops` should monitor delivery health and alert on trace ingestion issues.

## CTS and Compliance Workflow

1. Create or update CTS trails with explicit audit destinations.
2. Delegate destination creation to the respective storage/notification skill.
3. Use CTS queries to validate event capture for sensitive actions.
4. If audit queries fail, inspect IAM and destination skills.

## Cross-Skill Patterns

- **Audit destination validation**: After CTS trail creation, call bucket/topic validation routines from destination skills.
- **Permission troubleshooting**: If CTS cannot write audit data, delegate to IAM skill to inspect policies and trust relationships.
- **Security incident investigation**: Use CTS query results alongside ECS/RDS/OBS resource timelines from other product skills.

## Example Flow

1. User requests: "Create a CTS trail for security audit and deliver to OBS."
2. CTS skill creates the trail and verifies destination config.
3. OBS skill confirms bucket and policy.
4. IAM skill verifies CTS service principal permission.
5. CTS skill validates event delivery with a sample query.
