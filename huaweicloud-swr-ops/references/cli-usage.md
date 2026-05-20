# CLI Usage — Huawei Cloud SWR

## Coverage Map

| Operation | CLI Support | Docker CLI | SDK Fallback |
|-----------|-------------|-----------|-------------|
| CreateOrganization | ✅ `hcloud SWR CreateOrganization` | N/A | Go SDK |
| ListOrganizations | ✅ `hcloud SWR ListOrganizations` | N/A | Go SDK |
| DeleteOrganization | ✅ `hcloud SWR DeleteOrganization` | N/A | Go SDK |
| CreateRepository | ✅ `hcloud SWR CreateRepository` | N/A | Go SDK |
| ListRepositories | ✅ `hcloud SWR ListRepositories` | N/A | Go SDK |
| DeleteRepository | ✅ `hcloud SWR DeleteRepository` | N/A | Go SDK |
| ListImages | ✅ `hcloud SWR ListImages` | N/A | Go SDK |
| DeleteImageTag | ✅ `hcloud SWR DeleteImageTag` | N/A | Go SDK |
| CreateRetentionPolicy | ✅ `hcloud SWR CreateRetentionPolicy` | N/A | Go SDK |
| ListRetentionPolicies | ✅ `hcloud SWR ListRetentionPolicies` | N/A | Go SDK |
| CreateImageSync | ✅ `hcloud SWR CreateImageSync` | N/A | Go SDK |
| GenerateLoginToken | ✅ `hcloud SWR GenerateLoginToken` | Used with `docker login` | Go SDK |
| Pull Image | ❌ N/A | ✅ `docker pull` | N/A |
| Push Image | ❌ N/A | ✅ `docker push` | N/A |

## Authentication

```bash
# Step 1: Generate login token (valid ~1 hour)
LOGIN_CMD=$(hcloud SWR GenerateLoginToken --region="cn-north-4")

# Step 2: Execute docker login
docker login -u {{env.HW_ACCESS_KEY_ID}} --password-stdin \
  swr.{{env.HW_REGION_ID}}.myhuaweicloud.com <<< "$LOGIN_CMD"
```

## Common Patterns

### JSON Output for jq Pipeline

```bash
# List all repos with image count
hcloud SWR ListRepositories --format=json | \
  jq '.[] | {name: .name, images: .num_images, size: .size}'

# Find repos with >100 images needing cleanup
hcloud SWR ListRepositories --format=json | \
  jq '.[] | select(.num_images > 100) | .name'

# List image tags sorted by push date
hcloud SWR ListImages --organization="my-org" --repository="nginx" --format=json | \
  jq 'sort_by(.pushed_at) | reverse'
```

### Docker Push/Pull Pipeline

```bash
# Tag and push
docker build -t my-app:latest .
docker tag my-app:latest swr.cn-north-4.myhuaweicloud.com/my-org/my-app:latest
docker push swr.cn-north-4.myhuaweicloud.com/my-org/my-app:latest

# Pull and run
docker pull swr.cn-north-4.myhuaweicloud.com/my-org/my-app:1.0.0
docker run -d -p 8080:80 swr.cn-north-4.myhuaweicloud.com/my-org/my-app:1.0.0
```

### Batch Image Cleanup

```bash
# Delete all tags older than 30 days (requires jq)
hcloud SWR ListImages --organization="my-org" --repository="nginx" --format=json | \
  jq -r '.[] | select(.pushed_at < (now - 30*86400 | strftime("%Y-%m-%dT%H:%M:%SZ"))) | .name' | \
  while read tag; do
    hcloud SWR DeleteImageTag \
      --organization="my-org" \
      --repository="nginx" \
      --tag="$tag"
  done
```

## Known CLI Limitations

| Limitation | Workaround |
|-----------|-----------|
| No `docker login` directly in CLI | Use `GenerateLoginToken` + `docker login` |
| No batch image delete | Script loop with `DeleteImageTag` |
| No built-in image cleanup | Use retention policy or manual script |
| No vulnerability report via CLI | Use SDK or SWR web console |
