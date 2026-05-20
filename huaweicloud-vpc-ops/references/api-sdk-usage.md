# VPC API & SDK Usage — Huawei Cloud Virtual Private Cloud

## API Base Information

- **Base URL (VPC)**: `https://vpc.{region_id}.myhuaweicloud.com`
- **Base URL (NAT)**: `https://nat.{region_id}.myhuaweicloud.com`
- **API Version**: V3
- **Protocol**: HTTPS
- **Content-Type**: application/json
- **Authentication**: IAM AK/SK signature (v4)

## Operation Endpoint Map

| Operation | Method | Endpoint | Description |
|-----------|--------|----------|-------------|
| CreateVpc | POST | `/v3/{project_id}/vpc/vpcs` | Create VPC |
| ListVpcs | GET | `/v3/{project_id}/vpc/vpcs` | List VPCs |
| GetVpc | GET | `/v3/{project_id}/vpc/vpcs/{vpc_id}` | Get VPC details |
| UpdateVpc | PUT | `/v3/{project_id}/vpc/vpcs/{vpc_id}` | Update VPC |
| DeleteVpc | DELETE | `/v3/{project_id}/vpc/vpcs/{vpc_id}` | Delete VPC |
| CreateSubnet | POST | `/v3/{project_id}/vpc/subnets` | Create subnet |
| ListSubnets | GET | `/v3/{project_id}/vpc/subnets` | List subnets |
| GetSubnet | GET | `/v3/{project_id}/vpc/subnets/{subnet_id}` | Get subnet details |
| UpdateSubnet | PUT | `/v3/{project_id}/vpc/subnets/{subnet_id}` | Update subnet |
| DeleteSubnet | DELETE | `/v3/{project_id}/vpc/subnets/{subnet_id}` | Delete subnet |
| CreateSecurityGroup | POST | `/v3/{project_id}/vpc/security-groups` | Create security group |
| ListSecurityGroups | GET | `/v3/{project_id}/vpc/security-groups` | List security groups |
| DeleteSecurityGroup | DELETE | `/v3/{project_id}/vpc/security-groups/{sg_id}` | Delete security group |
| CreateSecurityGroupRule | POST | `/v3/{project_id}/vpc/security-group-rules` | Create SG rule |
| ListSecurityGroupRules | GET | `/v3/{project_id}/vpc/security-group-rules` | List SG rules |
| DeleteSecurityGroupRule | DELETE | `/v3/{project_id}/vpc/security-group-rules/{rule_id}` | Delete SG rule |
| CreateRouteTable | POST | `/v3/{project_id}/vpc/routetables` | Create route table |
| ListRouteTables | GET | `/v3/{project_id}/vpc/routetables` | List route tables |
| DeleteRouteTable | DELETE | `/v3/{project_id}/vpc/routetables/{rt_id}` | Delete route table |
| CreatePublicip | POST | `/v2.1/{project_id}/publicips` | Allocate EIP |
| ListPublicips | GET | `/v2.1/{project_id}/publicips` | List EIPs |
| DeletePublicip | DELETE | `/v2.1/{project_id}/publicips/{eip_id}` | Release EIP |
| CreateBandwidth | POST | `/v2.1/{project_id}/bandwidths` | Create bandwidth |
| ListBandwidths | GET | `/v2.1/{project_id}/bandwidths` | List bandwidths |
| CreateVpcPeering | POST | `/v2.0/vpc/peerings` | Create VPC peering |
| ListVpcPeerings | GET | `/v2.0/vpc/peerings` | List peerings |
| DeleteVpcPeering | DELETE | `/v2.0/vpc/peerings/{peering_id}` | Delete peering |
| CreateNatGateway | POST | `/v2/{project_id}/nat_gateways` | Create NAT gateway |
| ListNatGateways | GET | `/v2/{project_id}/nat_gateways` | List NAT gateways |
| DeleteNatGateway | DELETE | `/v2/{project_id}/nat_gateways/{nat_id}` | Delete NAT gateway |
| CreateDnatRule | POST | `/v2/{project_id}/dnat_rules` | Create DNAT rule |
| ListDnatRules | GET | `/v2/{project_id}/dnat_rules` | List DNAT rules |
| DeleteDnatRule | DELETE | `/v2/{project_id}/dnat_rules/{dnat_id}` | Delete DNAT rule |
| CreateSnatRule | POST | `/v2/{project_id}/snat_rules` | Create SNAT rule |
| ListSnatRules | GET | `/v2/{project_id}/snat_rules` | List SNAT rules |
| DeleteSnatRule | DELETE | `/v2/{project_id}/snat_rules/{snat_id}` | Delete SNAT rule |

## Go SDK Setup

```go
import (
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2/model"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/nat/v2"
    nat_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/nat/v2/model"
)
```

## Key Response Patterns

### CreateVpc Response

```json
{
    "vpc": {
        "id": "vpc-abc123",
        "name": "prod-vpc",
        "cidr": "10.0.0.0/16",
        "status": "OK",
        "routes": []
    }
}
```

Response path: `$.vpc.id`

### ListPublicips Response

```json
{
    "publicips": [
        {
            "id": "eip-abc123",
            "public_ip_address": "123.45.67.89",
            "status": "ACTIVE",
            "bandwidth_id": "bw-xyz789",
            "port_id": "port-xxx111"
        }
    ]
}
```

### CreateSecurityGroupRule Response

```json
{
    "security_group_rule": {
        "id": "sg-rule-abc",
        "security_group_id": "sg-xyz",
        "direction": "ingress",
        "ethertype": "IPv4",
        "protocol": "tcp",
        "port_range_min": 22,
        "port_range_max": 22,
        "remote_ip_prefix": "10.0.0.0/8"
    }
}
```

## Pagination

- **List operations**: Return `*_links` with `next` URL for pagination.
- Use `limit` query parameter (max varies per resource) and `marker` for offset.
