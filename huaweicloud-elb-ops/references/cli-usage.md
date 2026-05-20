# CLI Usage — Huawei Cloud ELB

## Overview

Huawei Cloud ELB is supported via `hcloud elb` CLI commands. This document covers the CLI invocation patterns used throughout the skill as the primary execution path.

## CLI Command Map

| Operation | CLI Command | SDK Equivalent | Notes |
|-----------|-------------|---------------|-------|
| Create LB | `hcloud elb create-loadbalancer` | `CreateLoadBalancer` | Supports shared & dedicated |
| Describe LB | `hcloud elb show-loadbalancer` | `ShowLoadBalancer` | Requires `--loadbalancer-id` |
| List LBs | `hcloud elb list-loadbalancers` | `ListLoadBalancers` | Supports `--limit`/`--marker` |
| Update LB | `hcloud elb update-loadbalancer` | `UpdateLoadBalancer` | — |
| Delete LB | `hcloud elb delete-loadbalancer` | `DeleteLoadBalancer` | Supports `--cascade` |
| Create Listener | `hcloud elb create-listener` | `CreateListener` | — |
| List Listeners | `hcloud elb list-listeners` | `ListListeners` | Filter by `--loadbalancer-id` |
| Create Pool | `hcloud elb create-pool` | `CreatePool` | — |
| List Pools | `hcloud elb list-pools` | `ListPools` | Filter by `--loadbalancer-id` |
| Create Member | `hcloud elb create-member` | `CreateMember` | Requires pool-id |
| List Members | `hcloud elb list-members` | `ListMembers` | Shows health status |
| Create Health Monitor | `hcloud elb create-healthmonitor` | `CreateHealthMonitor` | — |
| Show Health Monitor | `hcloud elb show-healthmonitor` | `ShowHealthMonitor` | — |
| List Certificates | `hcloud elb list-certificates` | `ListCertificates` | — |
| Create Certificate | `hcloud elb create-certificate` | `CreateCertificate` | — |
| List AZs | `hcloud elb list-availability-zones` | `ListAvailabilityZones` | Check LB type support |
| Show Quota | `hcloud elb show-quota` | `ShowQuota` | Check before creating |

## CLI Invocation Patterns

### Common flags

| Flag | Description | Required |
|------|-------------|----------|
| `--region` | Region ID | Yes (or env) |
| `--loadbalancer-id` | LB identifier | For LB-specific operations |
| `--listener-id` | Listener identifier | For listener operations |
| `--pool-id` | Pool identifier | For pool/member operations |
| `--healthmonitor-id` | Health monitor identifier | For health check operations |

### JSON Output

All CLI commands support JSON output via the `--format json` flag (or configured in `hcloud` config). Parse required fields from the JSON output.

```bash
# Example: JSON output for listing LBs
hcloud elb list-loadbalancers --region cn-north-4 --format json
```

## Coverage Gap Table

| Operation | CLI Support | SDK Support | Preferred Path |
|-----------|-------------|-------------|----------------|
| Create LB | ✅ Full | ✅ Full | CLI |
| Delete LB (cascade) | ✅ Full | ✅ Full | CLI |
| Create Listener | ✅ Full | ✅ Full | CLI |
| Create HTTPS Listener | ✅ Full | ✅ Full | CLI (certificate ref) |
| Create Pool | ✅ Full | ✅ Full | CLI |
| Session Persistence config | ✅ | ✅ | CLI |
| Slow Start config | ✅ | ✅ | CLI |
| Add Member | ✅ Full | ✅ Full | CLI |
| Batch Add Members | ⚠️ Partial | ✅ Full | SDK for batch |
| Create Health Monitor | ✅ Full | ✅ Full | CLI |
| Create Certificate | ✅ Full | ✅ Full | CLI |
| Update Certificate | ⚠️ Partial | ✅ Full | SDK for complex updates |
| List Quotas | ✅ Full | ✅ Full | CLI |
| List AZs | ✅ Full | ✅ Full | CLI |
| Member weight update | ✅ Full | ✅ Full | CLI |
| LB IP type change | ⚠️ Partial | ✅ Full | SDK |

## CLI Output Parsing Example

```bash
# Get LB ID from list output
hcloud elb list-loadbalancers --region cn-north-4 --format json | jq -r '.loadbalancers[] | select(.name=="prod-lb-01") | .id'

# Get member health status
hcloud elb list-members --pool-id {{pool_id}} --region cn-north-4 --format json | jq -r '.members[] | "\(.address):\(.protocol_port) → \(.operating_status)"'
```
