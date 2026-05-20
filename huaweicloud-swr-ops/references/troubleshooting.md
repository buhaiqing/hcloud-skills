# Troubleshooting — Huawei Cloud SWR

## Error Code Reference

| Code | Message | Cause | Diagnostic | Resolution |
|------|---------|-------|-----------|------------|
| `SWR.0001` | Organization already exists | Organization name taken globally | `ListOrganizations` to verify | Use unique name |
| `SWR.0002` | Organization not found | Org doesn't exist or wrong name | `ListOrganizations` | Use correct org name |
| `SWR.0003` | Repository already exists | Repository name exists in org | `ListRepositories` | Use unique repo name |
| `SWR.0004` | Repository not found | Repo doesn't exist | `ListRepositories` | Use correct repo name |
| `SWR.0005` | Image not found | Image tag doesn't exist | `ListImages` | Verify tag name |
| `SWR.0006` | Organization quota exceeded | Too many orgs | `ListOrganizations` to count | Delete unused orgs or request limit increase |
| `SWR.0007` | Repository quota exceeded | Too many repos in org | `ListRepositories` | Delete unused repos |
| `SWR.0008` | Storage quota exceeded | Total image storage too large | Check `repo_storage_usage` metric | Delete old images or request quota increase |
| `SWR.0009` | Invalid image name format | Name doesn't meet Docker standards | Validate against Docker naming | Use lowercase, no special chars except `-`/`_`/`/` |
| `SWR.0010` | Sync rule already exists | Rule for (repo+target) already exists | `ListImageSync` | Modify existing rule |
| `SWR.0011` | Target region not supported | SWR not available in target region | Check region list | Use supported region |
| `SWR.0012` | Retention policy conflict | Policy already exists for repo | `ListRetentionPolicies` | Delete existing policy first |

## Diagnostic Procedures

### Scenario 1: Docker Pull Fails

```bash
# 1. Test authentication
echo "your-sk" | docker login -u "{{env.HW_ACCESS_KEY_ID}}" --password-stdin \
  swr.{{env.HW_REGION_ID}}.myhuaweicloud.com

# 2. Check if image exists
hcloud SWR ListImages \
  --organization="{{user.organization_name}}" \
  --repository="{{user.repository_name}}" \
  --region="{{env.HW_REGION_ID}}"

# 3. Verify image URL format
# Correct: swr.cn-north-4.myhuaweicloud.com/my-org/nginx:1.25
# Wrong (missing tag): swr.cn-north-4.myhuaweicloud.com/my-org/nginx

# 4. Check DNS resolution
nslookup swr.{{env.HW_REGION_ID}}.myhuaweicloud.com

# 5. Check network connectivity
curl -I https://swr.{{env.HW_REGION_ID}}.myhuaweicloud.com/v2/
```

### Scenario 2: Docker Push Fails

```bash
# 1. Verify write permissions
# Check IAM policy includes SWR FullAccess

# 2. Check repository exists
hcloud SWR ListRepositories \
  --organization="{{user.organization_name}}" \
  --region="{{env.HW_REGION_ID}}"

# 3. Check quota usage
hcloud CES ShowMetricData \
  --namespace="SYS.SWR" \
  --metric_name="repo_storage_usage" \
  --dim="repo_name={{user.repository_name}}" \
  --period="3600" --from="-24h" --to="now"

# 4. Verify image size < 10GB limit
docker images "{{user.image_name}}" --format "{{.Size}}"
```

### Scenario 3: Image Not Found in CCE

```bash
# 1. Verify image exists in SWR
hcloud SWR ListImages \
  --organization="{{user.organization_name}}" \
  --repository="{{user.repository_name}}"

# 2. Check image URL in CCE deployment
kubectl describe pod <pod-name> | grep -A5 "Image:"

# 3. Verify SWR credentials in CCE
# Ensure imagePullSecrets is configured correctly
kubectl get secrets -n <namespace> | grep swr
```

### Scenario 4: Vulnerability Found

```bash
# 1. Get vulnerability report
hcloud SWR ListImageVulnerabilities \
  --organization="{{user.organization_name}}" \
  --repository="{{user.repository_name}}" \
  --tag="{{user.tag_name}}"

# 2. Check severity
# Critical/High → immediate rebuild
# Medium/Low → schedule update
```

## Known Issues

| Issue | Symptom | Workaround | Fix Version |
|-------|---------|-----------|-------------|
| Long login token expiry | Login expires after 1 hour | Re-run `GenerateLoginToken` | Use `docker login` with long-term AK/SK |
| Large image push timeout | Push fails after 30 min | Use smaller layers or better network | Split image into multiple layers |
| Sync rule not triggering | New tag not synced | Wait up to 15 min or trigger manually | Auto-trigger within 15 min of push |
