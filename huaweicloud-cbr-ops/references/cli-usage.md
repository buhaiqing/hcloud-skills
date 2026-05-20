# CLI Usage — Huawei Cloud CBR

## Coverage Map

| Operation | CLI Support | SDK Fallback | Notes |
|-----------|-------------|-------------|-------|
| CreateVault | ✅ `hcloud CBR CreateVault` | Go SDK | |
| ShowVault | ✅ `hcloud CBR ShowVault` | Go SDK | |
| ListVaults | ✅ `hcloud CBR ListVaults` | Go SDK | Pagination via `--limit` and `--offset` |
| UpdateVault | ✅ `hcloud CBR UpdateVault` | Go SDK | Resize, name, spec |
| DeleteVault | ✅ `hcloud CBR DeleteVault` | Go SDK | Irreversible |
| CreatePolicy | ✅ `hcloud CBR CreatePolicy` | Go SDK | |
| ListPolicies | ✅ `hcloud CBR ListPolicies` | Go SDK | |
| UpdatePolicy | ✅ `hcloud CBR UpdatePolicy` | Go SDK | |
| DeletePolicy | ✅ `hcloud CBR DeletePolicy` | Go SDK | |
| CreateBackup | ✅ `hcloud CBR CreateBackup` | Go SDK | |
| ListBackups | ✅ `hcloud CBR ListBackups` | Go SDK | |
| ShowBackup | ✅ `hcloud CBR ShowBackup` | Go SDK | |
| DeleteBackup | ✅ `hcloud CBR DeleteBackup` | Go SDK | |
| RestoreBackup | ✅ `hcloud CBR RestoreBackup` | Go SDK | |
| ReplicateBackup | ✅ `hcloud CBR ReplicateBackup` | Go SDK | |

## Common Patterns

### JSON Output for jq Pipeline

```bash
# List vaults with name and usage
hcloud CBR ListVaults --format=json | \
  jq '.vaults[] | {name: .name, used: .billing.used, size: .billing.size, status: .billing.status}'

# Find vaults above 80% capacity
hcloud CBR ListVaults --format=json | \
  jq '.vaults[] | select(.billing.used / .billing.size > 0.8) | .name'
```

### Batch Operations

```bash
# Create vaults for multiple environments
for env in "dev" "staging" "prod"; do
  hcloud CBR CreateVault \
    --name="${env}-ecs-backup" \
    --type="server" \
    --storage_size="500" \
    --billing_mode="postPaid"
done
```

## Known CLI Limitations

| Limitation | Workaround |
|-----------|-----------|
| No vault capacity forecast | Use CES metrics for trend analysis |
| Batch backup creation limited to 10 per call | Chain multiple `CreateBackup` calls |
