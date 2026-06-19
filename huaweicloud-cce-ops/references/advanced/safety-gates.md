# CCE Safety Gates — High-Risk Operation Controls

> Advanced safety controls for Cloud Container Engine.
> Load when issuing destructive cluster operations or executing rollbacks.

## 1. Destructive Operation Catalogue

| Operation | Risk class | Default gate |
|-----------|-----------|--------------|
| `DeleteCluster` | irreversible (data + workloads) | explicit confirmation + dry-run summary |
| `UpgradeCluster` | control-plane upgrade (downtime) | maintenance window + auto-rollback |
| `RemoveNode` | workload eviction | cordon + drain + maintain window |
| `ResetNode` | wipes kubelet | cordon + drain + safety snapshot |
| `DeleteAddon` | service interruption | helm diff + dependency check |

## 2. Safety Gate Workflow

1. **Dry-run**: emit `hcloud cce ... --dry-run` summary
2. **Confirm**: collect `{{user.confirm_destructive}}` from operator
3. **Snapshot**: capture etcd snapshot + node group state
4. **Execute**: issue the destructive call with `--retry-once`
5. **Verify**: poll cluster `status` until `Available` or `HALT`
6. **Rollback**: trigger documented rollback if `HALT`

## 3. Cross-Skill Delegation

- `huaweicloud-cce-ops → huaweicloud-iam-ops` for agency permissions
- `huaweicloud-cce-ops → huaweicloud-vpc-ops` for subnet changes
- `huaweicloud-cce-ops → huaweicloud-cts-ops` for audit trail

> **Security-Sensitive**: every destructive operation above MUST pass the
> Safety Gate workflow. The agent MUST surface `{{output.dry_run_report}}`
> before the destructive call and wait for `{{user.confirm_destructive}}`.