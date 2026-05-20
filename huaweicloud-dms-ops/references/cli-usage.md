# CLI Usage — Huawei Cloud DMS

## Coverage Map

| Operation | CLI Support | SDK Fallback | Notes |
|-----------|-------------|-------------|-------|
| CreateInstance | ✅ `hcloud DMS CreateInstance` | Go SDK | Confirm spec codes via API first |
| ShowInstance | ✅ `hcloud DMS ShowInstance` | Go SDK | |
| ListInstances | ✅ `hcloud DMS ListInstances` | Go SDK | Pagination via `--limit` and `--offset` |
| UpdateInstance | ✅ `hcloud DMS UpdateInstance` | Go SDK | Name, description, maintenance window only |
| DeleteInstance | ✅ `hcloud DMS DeleteInstance` | Go SDK | Irreversible — confirm first |
| CreateTopic | ✅ `hcloud DMS CreateTopic` | Go SDK | Kafka only |
| ListTopics | ✅ `hcloud DMS ListTopics` | Go SDK | |
| DeleteTopic | ✅ `hcloud DMS DeleteTopic` | Go SDK | |
| CreateQueue | ✅ `hcloud DMS CreateQueue` | Go SDK | RabbitMQ only |
| ListQueues | ✅ `hcloud DMS ListQueues` | Go SDK | |
| ListConsumerGroups | ✅ `hcloud DMS ListConsumerGroups` | Go SDK | |
| ShowConsumerGroupLag | ✅ `hcloud DMS ShowConsumerGroupLag` | Go SDK | |
| CreateBackup | ✅ `hcloud DMS CreateBackup` | Go SDK | |
| ListBackups | ✅ `hcloud DMS ListBackups` | Go SDK | |
| RestoreInstance | ✅ `hcloud DMS RestoreInstance` | Go SDK | |

## Authentication

All CLI commands require environment variables or inline credentials:

```bash
# Environment variables (recommended)
export HW_ACCESS_KEY_ID="your-ak"
export HW_SECRET_ACCESS_KEY="your-sk"
export HW_REGION_ID="cn-north-4"

# Or inline (not recommended for scripting)
hcloud DMS ListInstances --access_key="AK..." --secret_key="..." --region="cn-north-4"
```

## Common Patterns

### JSON Output for jq Pipeline

```bash
# List instances and extract IDs
hcloud DMS ListInstances --format=json | jq -r '.instances[].id'

# Get instance detail as JSON
hcloud DMS ShowInstance --instance_id="dms-abc123" --format=json

# Filter instances by engine type
hcloud DMS ListInstances --format=json | jq '.instances[] | select(.engine=="kafka")'
```

### Batch Operations

```bash
# Create topics in batch (bash loop)
for topic in "orders" "payments" "inventory"; do
  hcloud DMS CreateTopic \
    --instance_id="dms-abc123" \
    --name="$topic" \
    --partition_num=6 \
    --replication_factor=3
done
```

## Known CLI Limitations

| Limitation | Workaround |
|-----------|-----------|
| No `--dry-run` flag | Manually verify parameters before execution |
| CLI output varies by version | Use `--format=json` for consistent parsing |
| No batch import/export | Use SDK for bulk operations |
