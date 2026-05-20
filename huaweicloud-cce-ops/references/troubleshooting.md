# CCE Troubleshooting Guide — Huawei Cloud Cloud Container Engine

## Error Code Taxonomy

| Error Code | HTTP Status | Name | Description | Recovery Action |
|------------|-------------|------|-------------|-----------------|
| CCE.0001 | 400 | BadRequest | Generic request validation failed | Check request format and required fields |
| CCE.0002 | 409 | ClusterAlreadyExists | Cluster name already exists in project | Use different cluster name or reuse existing |
| CCE.0003 | 400 | InvalidParameter | Parameter format or value is invalid | Fix parameter; cross-reference API docs |
| CCE.0004 | 404 | ClusterNotFound | Cluster ID does not exist | Verify cluster_id in target region |
| CCE.0005 | 400 | ClusterStatusInvalid | Cluster not in valid state for operation | Wait for cluster to reach Available state |
| CCE.0006 | 429 | RequestThrottled | API rate limit exceeded | Implement exponential backoff; retry |
| CCE.0007 | 400 | VPCNotFound | Referenced VPC ID does not exist | Verify VPC ID or create VPC first |
| CCE.0008 | 400 | SubnetNotFound | Referenced subnet ID does not exist | Verify subnet ID or create subnet first |
| CCE.0009 | 400 | InsufficientResources | No available resources in AZ | Try different AZ or flavor family |
| CCE.0010 | 400 | InvalidFlavor | ECS flavor not valid or unavailable | Check available flavors for target AZ |
| CCE.0011 | 400 | NodeAlreadyExists | Node with same name already in cluster | Use different node name |
| CCE.0012 | 404 | NodeNotFound | Node ID does not exist in cluster | Verify node_id and cluster_id |
| CCE.0013 | 400 | NodePoolLimitExceeded | Maximum node pools per cluster reached | Delete unused node pools or request increase |
| CCE.0014 | 400 | AddonVersionMismatch | Addon version incompatible with cluster K8s version | Use compatible addon version |
| CCE.0015 | 404 | AddonNotFound | Addon instance does not exist | Verify addon_id or install addon |
| CCE.0016 | 403 | ProjectNotAuthorized | IAM permission insufficient | Grant CCE Administrator or CCE Viewer role |
| CCE.0017 | 400 | NodePoolNotFound | Node pool ID does not exist | Verify nodepool_id and cluster_id |
| CCE.0020 | 403 | QuotaExceeded | CCE resource quota exceeded | Delete unused resources or request quota increase |
| CCE.0029 | 500 | InternalError | CCE service internal error | Retry with exponential backoff; HALT after 3 attempts |
| CCE.0030 | 500 | AsyncOperationFailed | Async cluster/node operation failed | Check statusReason; resolve root cause and retry |
| Auth.0001 | 401 | AuthenticationFailed | AK/SK authentication failed | Verify credentials; NEVER log secret key |
| Auth.0003 | 403 | AccessDenied | IAM permission denied | Assign required CCE IAM permissions |

## Ordered Diagnostic Steps

### Step 1: Authentication Issues

```
Symptom: 401 Unauthorized or 403 Forbidden
Check:   AK/SK validity, region consistency, project_id, IAM role
Action:  Verify env vars exist; check IAM role has CCE Administrator or CCE Viewer
```

### Step 2: Cluster Creation Failure

```
Symptom: Cluster stuck in Error phase or creation timeout
Check:
  1. VPC and subnet exist in target region
  2. Security group exists and allows required ports (API server 6443)
  3. Sufficient quota in project
  4. Account balance sufficient
  5. Cluster name is unique
  6. K8s version is supported
Action:
  - Verify VPC connectivity: ping between nodes
  - Check subnet has sufficient available IPs
  - Verify security group has required inbound rules
  - Check quota via quota API
  - Delete failed cluster before retrying
```

### Step 3: Node NotReady

```
Symptom: Node phase = NotReady or node cannot join cluster
Check:
  1. Node ECS instance is RUNNING
  2. Network connectivity between node and cluster control plane
  3. kubelet process is running on node
  4. Node can reach API server (port 6443)
  5. Certificate has not expired
  6. Node is not tainted NoSchedule
Action:
  - SSH into node, check kubelet status: systemctl status kubelet
  - Check network: ping cluster API endpoint
  - Review kubelet logs: journalctl -u kubelet
  - Verify security group allows node-to-control-plane traffic
```

### Step 4: Addon Installation Failure

```
Symptom: Addon stuck in installing/failed state
Check:
  1. Addon version is compatible with cluster K8s version
  2. Cluster has sufficient resources (CPU, memory)
  3. Network connectivity for addon image pull
  4. Correct addon configuration values
Action:
  - Verify addon compatibility matrix
  - Check cluster resource utilization
  - Review addon installation logs via CCE console
  - Delete failed addon before reinstalling
```

### Step 5: Pod Scheduling Failures

```
Symptom: Pod stuck in Pending state
Check:
  1. Node resource availability (CPU, memory)
  2. NodeSelector/tolerations match
  3. PersistentVolumeClaim is bound
  4. Resource quotas not exceeded
Action:
  - Describe pod: kubectl describe pod <name> -n <namespace>
  - Check node resources: kubectl top nodes
  - Review PV/PVC status: kubectl get pvc -A
  - Verify storage class exists: kubectl get sc
```

### Step 6: Node Pool Autoscaling Issues

```
Symptom: Autoscaler not creating/removing nodes as expected
Check:
  1. Autoscaling is enabled on the node pool
  2. Current node count is within min/max range
  3. Resource requests exceed available capacity
  4. PDB (Pod Disruption Budget) is not blocking scale-down
Action:
  - Verify node pool autoscaling config
  - Check if scale operation is within cooldown period
  - Review scale-up/scale-down events in cluster
  - Check if PDBs are preventing evictions
```

### Step 7: Rate Limiting

```
Symptom: 429 RequestThrottled
Check:   Request frequency exceeds API limits
Action:
  - Implement exponential backoff: 1s → 2s → 4s → 8s
  - Cache responses when appropriate
  - Use batch operations instead of individual calls
```

## Multi-Round Diagnosis Flow

```
Cluster creation failed?
  ├── Is VPC configured? ── No → Create VPC via huaweicloud-vpc-ops
  │                           └── Done
  │                        ── Yes ↓
  ├── Does subnet have IPs? ── No → Use different subnet or expand
  │                                └── Expand subnet
  │                             ── Yes ↓
  ├── Is quota sufficient? ── No → Request increase or delete unused
  │                               └── Request quota
  │                            ── Yes ↓
  ├── Is balance sufficient? ── No → Recharge account
  │                               └── HALT
  │                            ── Yes ↓
  └── Check cluster status phase ── Error → Check statusReason for details
                                      └── Fix root cause and retry

Node NotReady?
  ├── Is ECS instance RUNNING? ── No → Start/recreate node
  │                                   └── Start instance
  │                                ── Yes ↓
  ├── Can node reach API server? ── No → Check security group/network
  │                                      └── Fix network rules
  │                                   ── Yes ↓
  ├── Is kubelet running? ── No → Restart kubelet or reinstall node
  │                          └── systemctl restart kubelet
  │                       ── Yes ↓
  └── Check node taints/conditions ── DiskPressure/MemoryPressure → Fix resource
                                       └── Clean disk or add memory

Addon installation failed?
  ├── Is addon version compatible? ── No → Use compatible version
  │                                     └── Check addon template
  │                                  ── Yes ↓
  ├── Does cluster have resources? ── No → Scale up or clean workloads
  │                                      └── Add nodes
  │                                   ── Yes ↓
  └── Check addon values config ── Invalid → Fix JSON values
                                    └── Validate JSON syntax
```
