# Huawei Cloud HSS Integration

## Recommended Delegation

- **ECS**: `huaweicloud-ecs-ops` should verify instance state before HSS protects or isolates a host.
- **WAF**: `huaweicloud-waf-ops` should coordinate web tamper protection for hosts serving web traffic.
- **CCE**: `huaweicloud-cce-ops` should handle container escape and pod-level security incidents.
- **Billing**: `huaweicloud-billing-ops` should assess cost impact of host protection and isolation actions.
- **IAM**: `huaweicloud-iam-ops` should manage account-level permissions for HSS agents and console access.
- **Anti-DDoS**: `huaweicloud-antiddos-ops` should provide DDoS protection when attacks target host IPs.

## Cross-Skill Patterns

- **Host security alert → ECS check**: When HSS raises an alert on a host, delegate instance health/state verification to ECS before isolation.
- **Web tamper → WAF**: When HSS detects web tampering on a host, coordinate rule tightening with WAF.
- **Container escape → CCE**: When HSS detects container escape, delegate pod/cluster isolation to CCE.

## Example Flow

1. User requests: "Investigate host security alert and contain the affected server."
2. HSS analyzes the alert, groups by `host_name` / `src_ip`, and classifies the pattern.
3. HSS delegates instance-state verification to `huaweicloud-ecs-ops`.
4. If web tampering is found, HSS coordinates with `huaweicloud-waf-ops`.
5. If container escape is found, HSS delegates to `huaweicloud-cce-ops`.
6. HSS applies isolation/block and reports outcome with cost impact from `huaweicloud-billing-ops`.
