# CCE Idempotency Checklist — Huawei Cloud Cloud Container Engine

## Idempotent Behavior by Operation

| Operation | Idempotent? | Duplicate Handling | Retry Semantics |
|-----------|-------------|-------------------|-----------------|
| CreateCluster | No | Returns `CCE.0002` (ClusterAlreadyExists) if name matches existing cluster | **Safe to retry** if no cluster created yet; use client request token |
| ListClusters | Yes | Returns full list; no side effects | Always safe to retry |
| DescribeCluster | Yes | Returns cluster details; 404 if not found | Always safe to retry |
| DeleteCluster | Partially | Idempotent — returns 200 even if cluster already deleted or in `Deleting` state | **Safe to retry** until cluster is fully removed |
| CreateNode | No | Returns `CCE.0011` (NodeAlreadyExists) if name matches | **Safe to retry** with client request token if no node created |
| ListNode | Yes | Returns full list; no side effects | Always safe to retry |
| DeleteNode | Partially | Idempotent — returns 200 even if node already deleted | **Safe to retry** until node is removed |
| CreateNodePool | No | Returns duplicate error if name matches existing pool | **Safe to retry** with unique request token |
| UpdateNodePool | Yes | Sets to desired state; repeated calls converge | **Safe to retry** — state converges to requested values |
| DeleteNodePool | Partially | Idempotent — safe to retry | **Safe to retry** until pool is removed |
| CreateAddonInstance | No | Returns duplicate error if addon already installed | **Safe to retry** with idempotent check: list addons first |
| ListAddonInstances | Yes | Returns full list; no side effects | Always safe to retry |
| GetClusterCert | Yes | Returns certificate; no side effects | Always safe to retry |

## Async Operation Retry Patterns

### Cluster Creation (Idempotent with Polling)

```
1. Send CreateCluster request
2. Store client_request_token (if supported)
3. If response = success → poll until Available
4. If response = 500/timeout → retry CreateCluster OR poll existing
   - Poll existing: if phase = Creating → continue waiting
   - If phase = Available → creation already succeeded
   - If phase = Error → delete failed cluster and retry
```

### Node Creation (Idempotent with Polling)

```
1. Send CreateNode request
2. If response = success → poll until Active
3. If response = 500/timeout → poll existing nodes by name
   - If node found and Active → creation already succeeded
   - If node found and Creating → continue waiting
   - If node not found → retry CreateNode
```

### Node Pool Creation (Idempotent with Polling)

```
1. Send CreateNodePool request
2. If response = success → poll until Active
3. If response = 500/timeout → poll existing pools by name
   - If pool found and Active → creation already succeeded
   - If pool found and Creating → continue waiting
   - If pool not found → retry CreateNodePool
```

## State Convergence Patterns

### UpdateNodePool (Convergent Operation)

- Setting `initialNodeCount` to desired value is convergent
- Setting autoscaling min/max is idempotent (overwrites previous values)
- Multiple UpdateNodePool calls with same values are no-ops

### Addon Installation (Idempotent Check Pattern)

```
Before installing addon:
1. List addons for cluster
2. Check if addon_name already exists
3. If exists → update values instead of create
4. If not exists → create new addon
```

### Cluster Update (Convergent)

- Description updates are idempotent (overwrite)
- Label updates are additive and idempotent for same keys

## Client Request Tokens

When available, use client request tokens for idempotent API calls:

- CCE supports client request tokens for CreateCluster via `metadata.annotations["clientToken"]`
- Tokens should be UUID-v4 formatted
- Token expiration: typically 24 hours
- If token is reused: same response returned (idempotent replay)

## Best Practices for Automation

1. **Always check before create:** List or describe before creating a resource
2. **Use unique names:** Include timestamps or hashes for non-idempotent resources
3. **Poll with timeout:** Async operations should poll until terminal state, not re-create
4. **Tag resources:** Use tags to track automation-owned resources
5. **Idempotent delete:** Safe to retry — always poll until 404
6. **Record state:** Store operation results for retry context
