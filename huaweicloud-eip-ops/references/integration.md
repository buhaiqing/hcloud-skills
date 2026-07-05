# EIP Integration — Cross-Skill Delegation Matrix

## Delegation Map

| Scenario | Primary skill | Delegate to | Reason |
|---|---|---|---|
| Allocate EIP, bind to ECS | `huaweicloud-eip-ops` | `huaweicloud-ecs-ops` (target verify) | EIP needs ECS `RUNNING` + port_id |
| EIP + NAT Gateway | `huaweicloud-eip-ops` | `huaweicloud-nat-ops` | EIP first, then create NAT + SNAT |
| EIP + ELB (Enhanced) | `huaweicloud-eip-ops` | `huaweicloud-elb-ops` | EIP binds to ELB listener |
| EIP + DDoS mitigation | `huaweicloud-ddos-ops` (when present) | `huaweicloud-eip-ops` | DDoS first, EIP bound to protected IP |
| Idle EIP cost | `huaweicloud-billing-ops` | `huaweicloud-eip-ops` (list) | EIP list, billing attribution |
| Bandwidth alarm | `huaweicloud-ces-ops` | `huaweicloud-eip-ops` (resize) | CES detects, EIP resizes |
| EIP exposure check | `huaweicloud-hss-ops` | `huaweicloud-vpc-ops` (SG) | HSS scans, VPC fixes SG |
| EIP release from prod ECS | `huaweicloud-ecs-ops` (impact) | `huaweicloud-eip-ops` (release) | ECS confirms no live traffic, EIP releases |
| Cross-region EIP inquiry | `huaweicloud-eip-ops` (per-region) | `huaweicloud-billing-ops` (consolidate) | EIP is region-scoped; consolidate per region |

## Anti-Delegation (do NOT do)

- Do NOT change the security group from `huaweicloud-eip-ops` — delegate to
  `huaweicloud-vpc-ops`. Mixing EIP and SG in one flow obscures blast radius.
- Do NOT mutate NAT / SNAT / DNAT from this skill — delegate to `huaweicloud-nat-ops`.
- Do NOT touch DDoS policies from this skill — even if a release "frees" the
  protected IP, the DDoS policy must be cleaned separately.

## Chained Runbook (canonical example)

User: "为新 ECS 申请一个按带宽 EIP 并允许外部 SSH"

```
1) huaweicloud-vpc-ops     # ensure SG allows 22 from <bastion CIDR>
2) huaweicloud-ecs-ops     # create / verify ECS, get port_id
3) huaweicloud-eip-ops     # allocate 按带宽 5 Mbps; bind to port_id
4) huaweicloud-ces-ops     # alarm on outgoing_bandwidth > 80% × 5
5) huaweicloud-billing-ops # tag cost-center; budget alert 80/90/100
```

## Chained Runbook (cleanup example)

User: "下个月下线 dev 环境，所有 EIP 释放"

```
1) huaweicloud-eip-ops     # list all EIPs in region; cross-check port_id
2) huaweicloud-ecs-ops     # confirm dev ECS will be deleted
3) huaweicloud-vpc-ops     # confirm SG / route table cleanup
4) huaweicloud-eip-ops     # unbind + release
5) huaweicloud-billing-ops # verify cost ledger reflects release
```
