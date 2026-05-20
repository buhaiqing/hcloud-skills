# GaussDB KooCLI Syntax Reference

## Global Parallel Parameters

```
--cli-region          : region (e.g., ap-southeast-1)
--project_id          : project ID (auto-resolved if configured)
--cli-query           : JMESPath query for output filtering
--cli-output-format   : json (default) | table | tsv
```

## Common Flag Patterns

```bash
# List instances with pagination
hcloud GaussDB ListInstances \
  --cli-region="{{env.REGION}}" \
  --offset=0 \
  --limit=100

# Show single instance detail
hcloud GaussDB ShowInstanceDetail \
  --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
  --cli-region="{{env.REGION}}"

# Create manual backup
hcloud GaussDB CreateManualBackup \
  --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
  --name="manual-backup-$(date +%Y%m%d)" \
  --description="Daily manual backup" \
  --cli-region="{{env.REGION}}"

# List backups filtered by type
hcloud GaussDB ListBackups \
  --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
  --backup_type="manual" \
  --cli-region="{{env.REGION}}" \
  --cli-query="backups[?status=='COMPLETED']"

# Apply parameter template
hcloud GaussDB ApplyConfiguration \
  --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
  --configuration_id="{{env.GAUSSDB_CONFIG_ID}}" \
  --cli-region="{{env.REGION}}"

# List databases
hcloud GaussDB ListDatabases \
  --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
  --cli-region="{{env.REGION}}"

# Create database user
hcloud GaussDB CreateDbUser \
  --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
  --name="app_user" \
  --password="{{env.DB_USER_PASSWORD}}" \
  --cli-region="{{env.REGION}}"

# List parameter templates
hcloud GaussDB ListConfigurations \
  --cli-region="{{env.REGION}}" \
  --cli-query="configurations[].{name:name,type:datastore_type,updated:updated}"
```

## JMESPath Query Patterns

```bash
# Get instance ID by name
--cli-query="instances[?name=='prod-gauss-01'].id | [0]"

# Count instances by status
--cli-query="length(instances[?status=='ACTIVE'])"

# Show backup summary
--cli-query="backups[].{id:id,type:type,size:size,status:status}"

# Filter with pagination awareness
--cli-query="instances[?disk_usage>='80'].{id:id,name:name,disk_usage:disk_usage}"
```

## Pagination

Most List APIs support `--offset` and `--limit`:

```bash
# Page 1: 100 items
hcloud GaussDB ListInstances --offset=0 --limit=100
# Page 2: next 100
hcloud GaussDB ListInstances --offset=100 --limit=100
```

For large datasets, iterate:

```bash
offset=0
limit=100
while true; do
  result=$(hcloud GaussDB ListInstances --offset=$offset --limit=$limit \
    --cli-region="{{env.REGION}}" --cli-query="instances[].id" --cli-output-format=json)
  [ "$result" = "[]" ] && break
  echo "$result"
  offset=$((offset + limit))
done
```

## CLI Field Name Mapping

API JSON field names and their CLI equivalents are identical in KooCLI's `GaussDB` service (auto-generated from OpenAPI). Multi-word fields use `snake_case` (e.g., `disk_encryption_id`, `db_user_name`, `backup_used_space`).

## Troubleshooting CLI Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `Missing parameter: instance_id` | Required param omitted | Add `--instance_id` |
| `Invalid region` | Wrong region name | Check supported regions |
| `AccessDenied` | AK/SK insufficient | Verify IAM policy includes `gaussdb:*` |
| `ResourceNotFound` | Instance doesn't exist | Verify instance_id and region |
