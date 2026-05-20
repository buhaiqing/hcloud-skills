# VPC Troubleshooting Guide — Huawei Cloud Virtual Private Cloud

## Error Code Taxonomy

| Error Code | HTTP Status | Name | Description | Recovery Action |
|------------|-------------|------|-------------|-----------------|
| VPC.0003 | 400 | InvalidParameter | Invalid parameter format or value | Fix parameter; verify CIDR format |
| VPC.0010 | 409 | CidrConflict | CIDR overlaps with existing VPC | Choose non-overlapping CIDR |
| VPC.0013 | 404 | ResourceNotFound | VPC/subnet/SG not found | Verify ID in correct region |
| VPC.0016 | 403 | Forbidden | Unauthorized project/resource | Check IAM permissions |
| VPC.0020 | 403 | QuotaExceeded | Resource quota limit reached | Delete unused or request increase |
| VPC.0029 | 500 | InternalError | Internal server error | Retry with backoff; HALT after 3 |
| EIP.0003 | 400 | InvalidParameter | Invalid EIP parameters | Verify type, bandwidth size, share type |
| EIP.0012 | 409 | ResourceInUse | EIP is already bound to resource | Unbind EIP first |
| EIP.0020 | 403 | QuotaExceeded | EIP quota exceeded | Release unused EIPs |
| NAT.0013 | 404 | ResourceNotFound | NAT gateway or rule not found | Verify ID exists |
| Auth.0001 | 401 | AuthenticationFailed | AK/SK authentication failed | Verify credentials |
| Auth.0003 | 403 | AccessDenied | Insufficient permissions | Assign VPC Administrator role |

## Ordered Diagnostic Steps

### Step 1: VPC Creation Fails

```
Symptom: VPC creation returns 409 CidrConflict
Check:   Existing VPC CIDRs in project
Action:  list-vpcs and compare CIDRs; choose non-overlapping range
```

### Step 2: Subnet Creation Fails

```
Symptom: Subnet CIDR rejected
Check:   Subnet CIDR must be within VPC CIDR range
Action:  Verify subnet CIDR is a valid subset of VPC CIDR
```

### Step 3: Security Group Rule Not Allowing Traffic

```
Symptom: Instance cannot be reached despite security group rule
Check:
  1. Rule direction is correct (ingress vs egress)
  2. Protocol matches application (tcp/udp)
  3. Port range includes the application port
  4. Source CIDR is correct (not 0.0.0.0/0 unintentionally)
  5. No Network ACL blocking traffic at subnet level
  6. Instance OS firewall not blocking (iptables, firewalld)
Action:  Add rule incrementally; test connectivity after each change
```

### Step 4: EIP Cannot Bind to Resource

```
Symptom: EIP bind fails with ResourceInUse
Check:
  1. EIP is not already bound to another resource
  2. Target resource type supports EIP binding
  3. Target resource is in the same region as EIP
Action:  Check EIP status; unbind if necessary; verify region match
```

### Step 5: NAT Gateway Cannot Route Traffic

```
Symptom: Instances in private subnet cannot reach internet
Check:
  1. NAT gateway is in ACTIVE state
  2. SNAT rule exists covering the subnet CIDR
  3. Route table has 0.0.0.0/0 → NAT Gateway route
  4. Security group allows outbound traffic
Action:  Verify SNAT rule CIDR matches subnet; check routing
```

### Step 6: VPC Peering Not Working

```
Symptom: Can ping peering connection but cannot ping instances
Check:
  1. Route added in both VPC route tables pointing to peering
  2. Security groups allow cross-VPC traffic
  3. CIDRs do not overlap
  4. Peering status = "ACTIVE" (accepted by peer)
Action:  Add routes on both sides; verify SG rules
```

### Step 7: Route Not Effective

```
Symptom: Custom route added but traffic not routing as expected
Check:
  1. Route table is associated with the subnet
  2. Route priority (more specific routes take precedence)
  3. No conflicting route with same destination
Action:  Verify route table association; check route table entries order
```

## Multi-Round Diagnosis Flow

```
Cannot access instance?
  ├── Is instance in correct VPC/subnet? ── No → Verify placement
  │                                            └── Migrate if needed
  │                                         ── Yes ↓
  ├── Is security group allowing traffic? ── No → Add rule
  │                                              └── Test
  │                                           ── Yes ↓
  ├── Is route table correct? ── No → Add/fix routes
  │                                 └── Apply
  │                              ── Yes ↓
  ├── Is EIP/NAT configured (for public access)? ── No → Configure
  │                                                   └── Bind/create
  │                                                ── Yes ↓
  └── Is OS firewall blocking? ── Yes → Modify iptables/firewalld
                                     └── Done
```
