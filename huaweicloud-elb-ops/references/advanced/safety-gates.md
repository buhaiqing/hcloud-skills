# ELB Safety Gates — High-Risk Operation Controls

> Advanced safety controls for Elastic Load Balance.
> Load when deleting listeners / pools / load balancers or rotating certificates.

## 1. Destructive Operation Catalogue

| Operation | Risk class | Default gate |
|-----------|-----------|--------------|
| `DeleteLoadBalancer` | cascade-delete backends + listeners | drain + confirmation |
| `DeleteListener` | immediate traffic cut | pre-cutover + confirmation |
| `DeletePool` | backend deregistration | drain health-check |
| `BatchUpdatePolicies` | cross-listener impact | dry-run + staged rollout |
| `DeleteCertificate` | TLS break for HTTPS | confirm replacement cert |

## 2. Safety Gate Workflow

1. **Inventory**: list all LBs / listeners / pools in scope
2. **Drain**: set `connection_drain_timeout = 300s`
3. **Confirm**: collect `{{user.confirm_destructive}}` per resource
4. **Execute**: dry-run, then apply
5. **Verify**: poll LB `provisioning_status` until `ACTIVE`
6. **Rollback**: redeploy from terraform state if `HALT`

## 3. Cross-Skill Delegation

- `huaweicloud-elb-ops → huaweicloud-ecs-ops` for backend health checks
- `huaweicloud-elb-ops → huaweicloud-waf-ops` for WAF policy binding
- `huaweicloud-elb-ops → huaweicloud-iam-ops` for certificate delegation

> **Security-Sensitive**: every destructive operation above MUST pass the
> Safety Gate. Certificate deletion requires replacement cert confirmation.