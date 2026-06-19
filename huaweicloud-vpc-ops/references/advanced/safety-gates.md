# VPC Safety Gates — High-Risk Operation Controls

> Advanced safety controls for Virtual Private Cloud.
> Load when modifying route tables, security groups, or EIP bindings.

## 1. Destructive Operation Catalogue

| Operation | Risk class | Default gate |
|-----------|-----------|--------------|
| `DeleteVpc` | cascade-delete subnets + SGs | drain + confirmation + snapshot route table |
| `DeleteSecurityGroupRule` | accidental allow / deny removal | rule staging + diff |
| `DeleteSubnet` | resource drain | maintenance window + drain ECS / RDS |
| `DisassociateEip` | external traffic cut | confirm alternate path |
| `UpdateRouteTable` | cross-VPC routing | dry-run + staged rollout |

## 2. Safety Gate Workflow

1. **Inventory**: list affected VPC / subnet / SG IDs
2. **Snapshot**: capture route table + SG rules
3. **Confirm**: collect `{{user.confirm_destructive}}` per resource
4. **Execute**: dry-run, then apply via SDK / CLI
5. **Verify**: poll `status` until `ACTIVE`; run smoke connectivity tests
6. **Rollback**: restore from snapshot if `HALT`

## 3. Cross-Skill Delegation

- `huaweicloud-vpc-ops → huaweicloud-ecs-ops` for SG rule impact analysis
- `huaweicloud-vpc-ops → huaweicloud-elb-ops` for EIP re-binding
- `huaweicloud-vpc-ops → huaweicloud-nat-ops` for NAT gateway changes
- `huaweicloud-vpc-ops → huaweicloud-iam-ops` for cross-account peering

> **Security-Sensitive**: every destructive operation above MUST pass the
> Safety Gate. Security group rule removal requires rule staging and a
> diff preview before commit.