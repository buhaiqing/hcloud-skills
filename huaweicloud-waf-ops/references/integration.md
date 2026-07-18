# Huawei Cloud WAF Integration

## Recommended Delegation

- **ECS**: `huaweicloud-ecs-ops` should protect and verify backend instances behind WAF domains.
- **HSS**: `huaweicloud-hss-ops` should handle host-intrusion investigation on backend servers.
- **ELB**: `huaweicloud-elb-ops` should manage listeners and load balancing in front of WAF.
- **Billing**: `huaweicloud-billing-ops` should assess WAF bandwidth and policy cost impact.
- **IAM**: `huaweicloud-iam-ops` should manage account-level permissions for WAF policies.
- **VPC**: `huaweicloud-vpc-ops` should manage subnets and security groups for WAF deployment.
- **SCM**: `huaweicloud-scm-ops` should manage SSL certificates bound to WAF domains.
- **CES**: `huaweicloud-ces-ops` should configure WAF alarm rules and metric thresholds.
- **Anti-DDoS**: `huaweicloud-antiddos-ops` should provide DDoS protection for WAF-facing IPs.

## Cross-Skill Patterns

- **Attack surge → ECS/HSS**: When WAF detects a backend-targeting attack, delegate host checks to ECS and intrusion scan to HSS.
- **Listener change → ELB**: When WAF domain topology changes, coordinate listener config with ELB.
- **Cert expiry → SCM**: When a WAF domain certificate nears expiry, delegate renewal to SCM.
- **Alarm tuning → CES**: When WAF rule decay is detected, delegate threshold tuning to CES.

## Example Flow

1. User requests: "Handle the WAF attack surge on domain example.com and tighten defense."
2. WAF groups attack events by `sip` and classifies the attack type.
3. WAF delegates backend host checks to `huaweicloud-ecs-ops` and intrusion scan to `huaweicloud-hss-ops`.
4. WAF coordinates listener posture with `huaweicloud-elb-ops`.
5. If certificate is near expiry, WAF delegates renewal to `huaweicloud-scm-ops`.
6. WAF applies CC tightening and reports outcome with cost impact from `huaweicloud-billing-ops`.
