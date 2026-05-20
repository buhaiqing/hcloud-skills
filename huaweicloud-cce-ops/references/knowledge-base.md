# CCE Knowledge Base — Huawei Cloud Cloud Container Engine

## Fault Patterns

### FP-1: Cluster Creation Timeout

| Field | Value |
|-------|-------|
| **Symptom** | Cluster remains in `Creating` phase for > 15 minutes |
| **Root Causes** | 1. VPC/subnet connectivity issues 2. Insufficient IPs in subnet 3. Security group blocking control plane traffic 4. AZ resource exhaustion |
| **Diagnosis Steps** | 1. Check cluster `statusReason` field 2. Verify subnet available IPs: `hcloud vpc describe-subnet` 3. Verify security group allows TCP 6443 4. Check AZ resource availability |
| **Fix** | 1. Fix network configuration and retry 2. Use a subnet with more available IPs 3. Select different AZ 4. Delete failed cluster before retry |
| **Prevention** | Pre-verify network prerequisites before cluster creation; use /24 or larger subnets |

### FP-2: Node NotReady After Joining

| Field | Value |
|-------|-------|
| **Symptom** | Node phase = `NotReady`, pods fail to schedule |
| **Root Causes** | 1. kubelet crash-loop 2. Network between node and API server broken 3. Certificate expired 4. Disk pressure on node 5. Insufficient memory |
| **Diagnosis Steps** | 1. SSH into node, check kubelet: `systemctl status kubelet` 2. Check API server reachability: `curl https://<api-server>:6443/healthz` 3. Check disk usage: `df -h` 4. Check memory: `free -m` 5. Check kubelet logs: `journalctl -u kubelet --since "10 min ago"` |
| **Fix** | 1. `systemctl restart kubelet` 2. Fix security group rules for node-to-control-plane 3. Regenerate node certificate 4. Clean disk (logs/images) 5. Drain and replace node |
| **Prevention** | Monitor node resource utilization; set resource limits on workloads; use separate data volumes |

### FP-3: Addon Installation Failure

| Field | Value |
|-------|-------|
| **Symptom** | Addon stuck in `installing` or `failed` state |
| **Root Causes** | 1. Addon version incompatible with cluster K8s version 2. Insufficient cluster resources (CPU/memory) 3. Addon configuration values contain errors 4. Network unable to pull addon image from SWR |
| **Diagnosis Steps** | 1. Check addon compatibility matrix for K8s version 2. Check cluster resource utilization 3. Validate JSON config values 4. Check SWR connectivity |
| **Fix** | 1. Use compatible addon version: `hcloud cce list-addon-templates` 2. Scale up cluster or add nodes 3. Fix addon values and reinstall 4. Verify SWR endpoint reachability |
| **Prevention** | Always check addon template version before installation; test addon configs in staging first |

### FP-4: Pod Stuck in Pending

| Field | Value |
|-------|-------|
| **Symptom** | Pod status = `Pending` for > 5 minutes |
| **Root Causes** | 1. Insufficient node resources (CPU/memory) 2. NodeSelector/affinity rules too restrictive 3. PV/C PVC not bound 4. Resource quota exceeded in namespace 5. No taint-tolerating nodes |
| **Diagnosis Steps** | 1. `kubectl describe pod <pod>` — check Events for scheduling messages 2. `kubectl top nodes` — check available resources 3. `kubectl get pvc -A` — check PVC binding 4. `kubectl describe quota -n <namespace>` 5. `kubectl taint nodes` — check taints |
| **Fix** | 1. Scale up node pool 2. Relax nodeSelector/affinity or add matching nodes 3. Fix storage class or PV provisioning 4. Increase namespace quota 5. Add tolerations or remove taints |
| **Prevention** | Set resource requests/limits accurately; use HPA; monitor namespace quotas; pre-provision PVs for stateful apps |

### FP-5: PersistentVolumeClaim Stuck in Pending

| Field | Value |
|-------|-------|
| **Symptom** | PVC status = `Pending`, cannot bind to PV |
| **Root Causes** | 1. StorageClass does not exist or is misconfigured 2. everest CSI addon not installed 3. EVS quota exhausted 4. StorageClass provisioner mismatch |
| **Diagnosis Steps** | 1. `kubectl get sc` — verify StorageClass exists 2. `kubectl describe sc <name>` — check provisioner 3. `kubectl get pod -n kube-system \| grep everest` — verify CSI is running 4. Check EVS quota |
| **Fix** | 1. Install or fix StorageClass 2. Reinstall everest addon 3. Delete unused EVS volumes 4. Match StorageClass provisioner to everest configuration |
| **Prevention** | Install everest addon during cluster creation; define StorageClass before stateful deployments |

### FP-6: Node Pool Autoscaling Not Triggering

| Field | Value |
|-------|-------|
| **Symptom** | Pods remain Pending but no new nodes are created |
| **Root Causes** | 1. Autoscaling disabled on node pool 2. Current node count = max_node_count 3. Scale-up cooldown period active 4. CA (Cluster Autoscaler) addon not installed 5. Resource request exceeds largest node flavor |
| **Diagnosis Steps** | 1. `hcloud cce describe-nodepool` — check `autoscaling.enable` 2. Check current vs max node count 3. Check CA addon logs for scale-up blocks 4. Check pod resource requests vs max node capacity |
| **Fix** | 1. Enable autoscaling on node pool 2. Increase max_node_count 3. Wait for cooldown to expire 4. Install/verify CA addon 5. Use larger node flavor or set `resourceLimits` in CA config |
| **Prevention** | Set appropriate min/max ranges; install Cluster Autoscaler addon; set accurate resource requests on workloads |

## Cascade Failure Patterns

### CCF-1: VPC Subnet Exhaustion Cascade

```
VPC subnet runs out of available IPs (e.g., /28 with 11 usable IPs)
  → New nodes cannot be created in the subnet
    → Node pool cannot scale up
      → Pods remain Pending (can't schedule to new nodes)
        → Services experience degraded availability
          → Autoscaler fails silently

Detection:   CES metric: subnet available IPs approaching 0; pod Pending count increasing
Root cause:  Subnet CIDR too small for workload growth
Resolution:  1. Create new subnet with larger CIDR 2. Move new node pool to new subnet 3. Drain old nodes 4. Delete old subnet
```

### CCF-2: EVS Volume Limit Reached Cascade

```
EVS volume quota reached (max 50 volumes per project in default quota)
  → PVC creation fails → applications lose data access
    → everest CSI controller logs show CreateVolume errors
      → Pods stuck in ContainerCreating
        → Rolling updates blocked (new pod can't start)
          → Service degradation across all stateful workloads

Detection:   PVC events show `Failed to provision volume`; EVS quota monitoring
Root cause:  EVS volume quota too low for number of stateful workloads
Resolution:  1. Delete orphaned EVS volumes 2. Request EVS quota increase 3. Use fewer PVCs with shared access modes
Prevention:  Set EVS monitoring alarm on volume count approaching quota; implement PVC lifecycle management
```

### CCF-3: Control Plane Upgrade Cascade

```
CCE cluster control plane upgrade initiated
  → API server temporarily unavailable during upgrade
    → kubelet on nodes loses connection to API server
      → Nodes may enter NotReady state
        → Pod scheduling blocked during upgrade window
          → If PDB not configured, workloads may be disrupted

Prevention:  1. Configure PDBs for critical workloads 2. Perform upgrade during low-traffic windows 3. Monitor upgrade progress via cluster status 4. Do NOT modify nodes during control plane upgrade
Detection:   Cluster phase = `Upgrading`; API server health check failures
Resolution:  1. Wait for upgrade to complete (typically 10-30 min) 2. Verify API server health 3. Check node statuses post-upgrade
```

## Historical Incident Patterns

| Incident | Pattern | Frequency | Impact | Remediation Time |
|----------|---------|-----------|--------|-----------------|
| Cluster upgrade node incompatibility | Post-upgrade nodes report version mismatch | Rare (per release cycle) | Medium — workloads degraded | 1-4 hours (rolling node upgrade) |
| AZ outage single-AZ cluster | Complete cluster unavailability | Very rare | Critical — full outage | 2-8 hours (redeploy to another AZ) |
| Node pool deletion blocked by PDB | Nodes stuck in deleting state | Common | Low — delayed cleanup | Minutes to hours (force delete after eviction) |
| CoreDNS failure | DNS resolution fails for all services | Rare | Critical — service discovery broken | 10-30 minutes (restart CoreDNS pods) |
