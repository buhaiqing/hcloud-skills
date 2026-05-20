# VPC Idempotency Checklist — Huawei Cloud Virtual Private Cloud

## Idempotent Operations

| Operation | Idempotent? | Mechanism | Notes |
|-----------|-------------|-----------|-------|
| CreateVpc | ❌ | Name not unique; same CIDR may overlap | Use describe-before-create pattern |
| ListVpcs | ✅ | GET operation | Safe to retry |
| GetVpc | ✅ | GET operation | Safe to retry |
| UpdateVpc | ✅ | PUT operation (replace-based) | Safe to retry |
| DeleteVpc | ✅ | DELETE operation | Second delete returns 404 |
| CreateSubnet | ❌ | Same CIDR in same VPC returns error | Check existence first |
| DeleteSubnet | ✅ | DELETE operation | Safe to retry |
| CreateSecurityGroup | ❌ | Name may conflict; describe first | Check existence first |
| CreateSecurityGroupRule | ❌ | Duplicate rule returns error | Check existing rules first |
| DeleteSecurityGroupRule | ✅ | DELETE operation | Safe to retry |
| CreateRouteTable | ❌ | Check existence first | |
| DeleteRouteTable | ✅ | DELETE operation | Safe to retry |
| CreatePublicip | ❌ | Allocate new IP each time | Track ID after creation |
| DeletePublicip | ✅ | DELETE operation | Must be unbound first |
| CreateNatGateway | ❌ | Check existence in subnet | |
| DeleteNatGateway | ✅ | DELETE operation | Safe to retry |

## Idempotent VPC Creation Pattern

```
1. GET /v3/{project_id}/vpc/vpcs
2. If VPC with matching name and CIDR exists → Return existing vpc.id (SKIP)
3. If VPC with matching CIDR exists (different name) → Report conflict; HALT
4. If no VPC with this CIDR exists → POST /v3/{project_id}/vpc/vpcs (CREATE)
```

## Idempotent Subnet Creation Pattern

```
1. GET /v3/{project_id}/vpc/subnets?vpc_id={vpc_id}
2. If subnet with matching name AND CIDR exists → Return existing subnet.id (SKIP)
3. If no match → POST /v3/{project_id}/vpc/subnets (CREATE)
```

## Idempotent Security Group Rule Creation Pattern

```
1. GET /v3/{project_id}/vpc/security-group-rules?security_group_id={sg_id}
2. If rule exists with same direction, protocol, port_range, and remote_ip_prefix → Return existing rule.id (SKIP)
3. If no match → POST /v3/{project_id}/vpc/security-group-rules (CREATE)
```

## Retry Safety

- All GET and DELETE operations are inherently idempotent — safe to retry.
- POST operations require pre-existence checks to avoid duplicates or conflicts.
- PUT operations are replace-based — inherently idempotent.
