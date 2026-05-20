# VPC CLI Usage — Huawei Cloud Virtual Private Cloud

## CLI Command Map

| Operation | CLI Command | Product |
|-----------|-------------|---------|
| List VPCs | `hcloud vpc list` | VPC |
| Create VPC | `hcloud vpc create` | VPC |
| Describe VPC | `hcloud vpc describe` | VPC |
| Delete VPC | `hcloud vpc delete` | VPC |
| List Subnets | `hcloud vpc list-subnets` | VPC |
| Create Subnet | `hcloud vpc create-subnet` | VPC |
| Delete Subnet | `hcloud vpc delete-subnet` | VPC |
| List Security Groups | `hcloud vpc list-security-groups` | VPC |
| Create Security Group | `hcloud vpc create-security-group` | VPC |
| Delete Security Group | `hcloud vpc delete-security-group` | VPC |
| List Security Group Rules | `hcloud vpc list-security-group-rules` | VPC |
| Create SG Rule | `hcloud vpc create-security-group-rule` | VPC |
| Delete SG Rule | `hcloud vpc delete-security-group-rule` | VPC |
| List Route Tables | `hcloud vpc list-route-tables` | VPC |
| Create Route Table | `hcloud vpc create-route-table` | VPC |
| Delete Route Table | `hcloud vpc delete-route-table` | VPC |
| List EIPs | `hcloud eip list` | EIP |
| Create EIP | `hcloud eip create` | EIP |
| Describe EIP | `hcloud eip describe` | EIP |
| Bind EIP | `hcloud eip bind` | EIP |
| Unbind EIP | `hcloud eip unbind` | EIP |
| Delete EIP | `hcloud eip delete` | EIP |
| List Bandwidths | `hcloud bandwidth list` | EIP |
| Create Bandwidth | `hcloud bandwidth create` | EIP |
| List NAT Gateways | `hcloud nat list-gateway` | NAT |
| Create NAT Gateway | `hcloud nat create-gateway` | NAT |
| Delete NAT Gateway | `hcloud nat delete-gateway` | NAT |
| List SNAT Rules | `hcloud nat list-snat-rule` | NAT |
| Create SNAT Rule | `hcloud nat create-snat-rule` | NAT |
| Delete SNAT Rule | `hcloud nat delete-snat-rule` | NAT |
| List DNAT Rules | `hcloud nat list-dnat-rule` | NAT |
| Create DNAT Rule | `hcloud nat create-dnat-rule` | NAT |
| Delete DNAT Rule | `hcloud nat delete-dnat-rule` | NAT |
| List VPC Peerings | `hcloud vpc list-peering` | VPC |
| Create VPC Peering | `hcloud vpc create-peering` | VPC |
| Delete VPC Peering | `hcloud vpc delete-peering` | VPC |

## CLI Coverage Gap Table

| Operation | CLI Support | SDK Only? | Notes |
|-----------|-------------|-----------|-------|
| VPC CRUD | ✅ | No | Full support |
| Subnet CRUD | ✅ | No | Full support |
| Security Group CRUD | ✅ | No | Full support |
| Security Group Rules | ✅ | No | Full support |
| Route Tables | ✅ | No | Full support |
| EIP CRUD | ✅ | No | Full support |
| Bandwidth CRUD | ✅ | No | Full support |
| NAT Gateway CRUD | ✅ | No | Full support |
| SNAT/DNAT Rules | ✅ | No | Full support |
| VPC Peering CRUD | ✅ | No | Full support |
| Cross-region peering | ❌ | Yes | Requires SDK / console |

## JSON Path Mappings

| CLI Output Field | JSON Path | Description |
|-----------------|-----------|-------------|
| vpc.id | `$.vpc.id` | VPC identifier |
| subnet.id | `$.subnet.id` | Subnet identifier |
| security_group.id | `$.security_group.id` | Security group ID |
| security_group_rule.id | `$.security_group_rule.id` | Rule ID |
| publicip.id | `$.publicip.id` | EIP identifier |
| publicip.public_ip_address | `$.publicip.public_ip_address` | EIP address |
| nat_gateway.id | `$.nat_gateway.id` | NAT gateway ID |
| bandwidth.id | `$.bandwidth.id` | Bandwidth ID |
