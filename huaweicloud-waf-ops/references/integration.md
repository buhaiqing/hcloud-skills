# Huawei Cloud WAF Integration

## Recommended Delegation

- **ECS**: `huaweicloud-ecs-ops` should protect and verify backend instances behind WAF domains.
- **HSS**: `huaweicloud-hss-ops` should handle host-intrusion investigation on backend servers.
- **ELB**: `huaweicloud-elb-ops` should manage listeners and load balancing in front of WAF.
- **Billing**: `huaweicloud-billing-ops` should assess WAF bandwidth and policy cost impact.
- **IAM**: `huaweicloud-iam-ops` should manage account-level permissions for WAF policies.
- **VPC**: `huaweicloud-vpc-ops` should manage subnets and security groups for WAF deployment.
- **CES**: `huaweicloud-ces-ops` should configure WAF alarm rules and metric thresholds.

## Cross-Skill Patterns

- **Attack surge → ECS/HSS**: When WAF detects a backend-targeting attack, delegate host checks to ECS and intrusion scan to HSS.
- **Listener change → ELB**: When WAF domain topology changes, coordinate listener config with ELB.
- **Cert expiry → in-skill**: When a WAF domain certificate nears expiry, renew it via the WAF certificate APIs and re-bind the domain.
- **Alarm tuning → CES**: When WAF rule decay is detected, delegate threshold tuning to CES.

## Example Flow

1. User requests: "Handle the WAF attack surge on domain example.com and tighten defense."
2. WAF groups attack events by `sip` and classifies the attack type.
3. WAF delegates backend host checks to `huaweicloud-ecs-ops` and intrusion scan to `huaweicloud-hss-ops`.
4. WAF coordinates listener posture with `huaweicloud-elb-ops`.
5. If a certificate is near expiry, WAF renews it and re-binds the domain.
6. WAF applies CC tightening and reports outcome with cost impact from `huaweicloud-billing-ops`.
