# Well-Architected + Three-Pillar Assessment — Huawei Cloud FunctionGraph

## 1. Security (安全) — FunctionGraph

### IAM Minimum Permissions

| Operation | IAM Action | Resource Scope |
|-----------|-----------|---------------|
| ListFunctions | `functiongraph:*List*` | `*` |
| CreateFunction | `functiongraph:*Create*`, `obs:object:GetObject` | `*` |
| DeleteFunction | `functiongraph:*Delete*` | `*` |
| InvokeFunction | `functiongraph:*Invoke*` | `functiongraph:function:*` |
| ManageTriggers | `functiongraph:*Trigger*` | `functiongraph:function:*` |

### Credential Management
- Use IAM agency (委托) for cross-service access (OBS, SMN, DMS, etc.)
- Do NOT embed AK/SK in function code or environment variables
- Rotate AK/SK every 90 days
- For VPC access: use VPC endpoint + security group, not public IP

### Network Security
- VPC access: only enable when function needs to access private resources
- Security group: allow only required outbound traffic
- Function code: scan for secrets before deployment
- Environment variables: use KMS encryption for sensitive values

## 2. Stability (稳定) — FunctionGraph

### Backup & Recovery

| Operation | Method | Description |
|-----------|--------|-------------|
| Export function code | `ShowFunctionCode` + download from OBS | Save code ZIP |
| Export function config | `ShowFunctionConfig` → JSON | Save config as template |
| Cross-region replication | Manual via CI/CD pipeline | Deploy to multiple regions |
| Version rollback | Update alias to point to previous version | Instant rollback |

**RTO target:** < 5 minutes (alias version rollback)
**RPO target:** < 1 version (each deployment is a recovery point)

### DR Runbook

**Phase 1: Detection**
1. Check function state via `ShowFunctionConfig`
2. Check invocation error rate via CES metric `fail_count`
3. Check function code integrity via `ShowFunctionCode`

**Phase 2: Quick Recovery**
1. Rollback alias to previous stable version
2. Verify via test invocation
3. If alias rollback fails, redeploy from code backup

**Phase 3: Root Cause**
1. Compare current vs previous version code/config
2. Check LTS logs for error details
3. Implement fix and deploy new version

### Versioning Strategy
- Always use alias (`prod`) for production traffic
- Canary deployments: 10% → 50% → 100% traffic shift
- Keep minimum 3 recent versions for rollback

## 3. Cost (成本) — FunctionGraph (FinOps)

### Billing Model

FunctionGraph is pay-per-use (按需). No subscription/spot options.

**Cost components:**
| Component | Unit | Rate (example) |
|-----------|------|----------------|
| Compute time | GB-second (memory × duration) | ¥0.001/GBs |
| Invocations | Per million requests | ¥1.00/million |
| Outbound traffic | GB | ¥0.80/GB |
| Reserved instances | Per hour | Instance type-dependent |

### Waste Detection

| Waste Pattern | Detection Method | Action |
|--------------|-----------------|--------|
| Over-provisioned memory | `duration` short but `memory_size` large | Reduce memory (lower cost per invocation) |
| Rarely invoked functions | `count` < 100/day for 30 days | Archive or delete |
| Reserved instance idle | `concurrent_executions` < 10% of reserved | Remove reserved instance |
| Excessive timeout | `max_duration` < 10% of timeout | Reduce timeout value |

### Right-Sizing Guidance

| Current Memory (MB) | Avg Duration (ms) | Suggestion |
|--------------------|-------------------|------------|
| 256 | < 100 | Consider 128 MB |
| 512 | < 50 | Reduce to 256 MB |
| 1024 | < 100 | Keep or reduce to 512 MB |
| 2048+ | < 200 | Reduce memory, measure impact |

### Unit Economics

| Metric | Formula | Target |
|--------|---------|--------|
| Cost per invocation | Monthly cost / invocation count | < ¥0.0001 |
| Cost per GB-second | Monthly cost / (memory_gb × duration_s) | < ¥0.001/GBs |
| Memory efficiency | Avg duration × memory_mb / expected | < 1.0 (lower is better) |

## 4. Efficiency (效率) — FunctionGraph

### CI/CD Integration
- Code stored in OBS → CodePipeline or custom CI/CD
- Deploy via SDK `UpdateFunctionCode` / `UpdateFunctionConfig`
- Version promotion: `CreateFunctionVersion` → `UpdateAlias`
- Rollback: `UpdateAlias` to previous version

### Automation Patterns
- Scheduled functions via Timer trigger (cron expressions)
- Event-driven processing via OBS/SMN/CTS triggers
- Auto-scaling: native (no manual scaling needed)
- Batch operations: `ListFunctions` → iterate → apply changes

## 5. Performance (性能) — FunctionGraph

### Performance Baselines

| Metric | Expected Range | Optimization |
|--------|---------------|--------------|
| Cold start duration | 100ms–2s (runtime-dependent) | Reserved instances |
| Warm invocation duration | 10ms–500ms | Optimize code, reduce dependencies |
| Memory usage | 50–80% of allocated | Right-size memory |
| Concurrent executions | Baseline-dependent | Increase concurrency quota |

### Optimization Patterns
1. **Reduce cold start**: Reserved instances, provisioned concurrency
2. **Optimize dependencies**: Minimize package size, use layers
3. **Connection pooling**: Reuse connections across invocations (global scope)
4. **Async processing**: Use async invoke for non-blocking workloads
5. **Streaming**: Use function streaming for large payloads
