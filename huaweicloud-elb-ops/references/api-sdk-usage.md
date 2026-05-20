# API & SDK Usage — Huawei Cloud ELB

## JIT Go SDK Setup

```bash
mkdir -p /tmp/elb-sdk-workspace && cd /tmp/elb-sdk-workspace
go mod init elb-script
go get github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v3
```

## Operation Map (v3 API)

| Operation | Go SDK Method | API Path | CLI Command |
|-----------|--------------|----------|-------------|
| CreateLoadBalancer | `CreateLoadBalancer` | POST /v3/{project}/elb/loadbalancers | `hcloud elb create-loadbalancer` |
| ShowLoadBalancer | `ShowLoadBalancer` | GET /v3/{project}/elb/loadbalancers/{lb_id} | `hcloud elb show-loadbalancer` |
| ListLoadBalancers | `ListLoadBalancers` | GET /v3/{project}/elb/loadbalancers | `hcloud elb list-loadbalancers` |
| UpdateLoadBalancer | `UpdateLoadBalancer` | PUT /v3/{project}/elb/loadbalancers/{lb_id} | `hcloud elb update-loadbalancer` |
| DeleteLoadBalancer | `DeleteLoadBalancer` | DELETE /v3/{project}/elb/loadbalancers/{lb_id} | `hcloud elb delete-loadbalancer` |
| CreateListener | `CreateListener` | POST /v3/{project}/elb/listeners | `hcloud elb create-listener` |
| ShowListener | `ShowListener` | GET /v3/{project}/elb/listeners/{listener_id} | `hcloud elb show-listener` |
| ListListeners | `ListListeners` | GET /v3/{project}/elb/listeners | `hcloud elb list-listeners` |
| UpdateListener | `UpdateListener` | PUT /v3/{project}/elb/listeners/{listener_id} | `hcloud elb update-listener` |
| DeleteListener | `DeleteListener` | DELETE /v3/{project}/elb/listeners/{listener_id} | `hcloud elb delete-listener` |
| CreatePool | `CreatePool` | POST /v3/{project}/elb/pools | `hcloud elb create-pool` |
| ShowPool | `ShowPool` | GET /v3/{project}/elb/pools/{pool_id} | `hcloud elb show-pool` |
| ListPools | `ListPools` | GET /v3/{project}/elb/pools | `hcloud elb list-pools` |
| UpdatePool | `UpdatePool` | PUT /v3/{project}/elb/pools/{pool_id} | `hcloud elb update-pool` |
| DeletePool | `DeletePool` | DELETE /v3/{project}/elb/pools/{pool_id} | `hcloud elb delete-pool` |
| CreateMember | `CreateMember` | POST /v3/{project}/elb/pools/{pool_id}/members | `hcloud elb create-member` |
| ShowMember | `ShowMember` | GET /v3/{project}/elb/pools/{pool_id}/members/{member_id} | `hcloud elb show-member` |
| ListMembers | `ListMembers` | GET /v3/{project}/elb/pools/{pool_id}/members | `hcloud elb list-members` |
| UpdateMember | `UpdateMember` | PUT /v3/{project}/elb/pools/{pool_id}/members/{member_id} | `hcloud elb update-member` |
| DeleteMember | `DeleteMember` | DELETE /v3/{project}/elb/pools/{pool_id}/members/{member_id} | `hcloud elb delete-member` |
| CreateHealthMonitor | `CreateHealthMonitor` | POST /v3/{project}/elb/healthmonitors | `hcloud elb create-healthmonitor` |
| ShowHealthMonitor | `ShowHealthMonitor` | GET /v3/{project}/elb/healthmonitors/{monitor_id} | `hcloud elb show-healthmonitor` |
| UpdateHealthMonitor | `UpdateHealthMonitor` | PUT /v3/{project}/elb/healthmonitors/{monitor_id} | `hcloud elb update-healthmonitor` |
| DeleteHealthMonitor | `DeleteHealthMonitor` | DELETE /v3/{project}/elb/healthmonitors/{monitor_id} | `hcloud elb delete-healthmonitor` |
| ListCertificates | `ListCertificates` | GET /v3/{project}/elb/certificates | `hcloud elb list-certificates` |
| CreateCertificate | `CreateCertificate` | POST /v3/{project}/elb/certificates | `hcloud elb create-certificate` |
| DeleteCertificate | `DeleteCertificate` | DELETE /v3/{project}/elb/certificates/{cert_id} | `hcloud elb delete-certificate` |
| ListAvailabilityZones | `ListAvailabilityZones` | GET /v3/{project}/elb/availability-zones | `hcloud elb list-availability-zones` |
| ShowQuota | `ShowQuota` | GET /v3/{project}/elb/quotas | `hcloud elb show-quota` |

## Common Request/Response Patterns

### Create Load Balancer (Dedicated)

**Request**:
```json
{
    "loadbalancer": {
        "name": "prod-lb-01",
        "description": "Production application load balancer",
        "vpc_id": "vpc-1234",
        "elb_virsubnet_ids": ["subnet-5678"],
        "loadbalancer_type": "dedicated",
        "availability_zone_list": ["cn-north-4a", "cn-north-4b"],
        "admin_state_up": true
    }
}
```

**Response**:
```json
{
    "loadbalancer": {
        "id": "lb-abcdef-1234",
        "name": "prod-lb-01",
        "vpc_id": "vpc-1234",
        "provisioning_status": "ACTIVE",
        "operating_status": "ONLINE",
        "loadbalancer_type": "dedicated",
        "admin_state_up": true
    },
    "loadbalancer_id": "lb-abcdef-1234",
    "order_id": "CSABC12345"
}
```

### Create Listener (HTTPS)

**Request**:
```json
{
    "listener": {
        "name": "prod-https-443",
        "protocol_port": 443,
        "protocol": "HTTPS",
        "loadbalancer_id": "lb-abcdef-1234",
        "default_tls_container_ref": "cert-xxxxx",
        "default_pool_id": "pool-xxxxx",
        "admin_state_up": true
    }
}
```

### Create Pool

**Request**:
```json
{
    "pool": {
        "name": "prod-backend-pool",
        "protocol": "HTTP",
        "lb_algorithm": "ROUND_ROBIN",
        "listener_id": "listener-xxxxx",
        "session_persistence": {
            "type": "SOURCE_IP"
        },
        "slow_start": {
            "enable": true,
            "duration": 30
        }
    }
}
```

### Create Health Monitor

**Request**:
```json
{
    "healthmonitor": {
        "pool_id": "pool-xxxxx",
        "delay": 5,
        "timeout": 3,
        "max_retries": 3,
        "type": "HTTP",
        "url_path": "/health",
        "expected_codes": "200-399",
        "monitor_port": null
    }
}
```

## Pagination

List operations support `limit` + `marker` pagination (default 2000).

```go
request := &model.ListLoadBalancersRequest{
    Limit:  func() *int32 { v := int32(100); return &v }(),
    Marker: func() *string { v := ""; return &v }(),
}
```

## Idempotency

- LB creation: name uniqueness within VPC — duplicate returns error
- Listener: port + protocol + LB uniqueness
- Member: address + port + pool uniqueness — duplicate returns `ELB.3002`
- Delete: idempotent — deleting non-existent resource returns 404
