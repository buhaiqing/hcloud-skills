# CDN CLI Usage — Huawei Cloud CDN

## CLI Command Map

| Operation | CLI Command | Notes |
|---|---|---|
| List domains | `hcloud cdn list-domain` | All domains in current project |
| Create domain | `hcloud cdn create-domain` | Async; poll for `online` |
| Describe domain | `hcloud cdn show-domain` | Single domain detail |
| Delete domain | `hcloud cdn delete-domain` | **Irreversible** |
| Start domain | `hcloud cdn start-domain` | Transition to `online` |
| Stop domain | `hcloud cdn stop-domain` | Transition to `offline` |
| Modify domain config | `hcloud cdn modify-domain-config` | Cache rules, HTTPS, origin |
| Refresh cache | `hcloud cdn refresh-cache` | Async job; poll with `list-tasks` |
| Preheat cache | `hcloud cdn preheat-cache` | Async job; poll with `list-tasks` |
| List tasks | `hcloud cdn list-tasks` | Refresh/preheat job status |
| Query stats | `hcloud cdn list-stats` | Bandwidth, traffic, hit rate |
| Show quota | `hcloud cdn show-quota` | Domain count limits |

> **Verify before use:** `hcloud cdn --help`. CLI subcommands evolve between versions;
> fall back to **JIT Go SDK** path in `references/api-sdk-usage.md`.

## Common Recipes

### Create a CDN domain

```bash
# 1) Verify CNAME is set at your DNS registrar
# 2) Create domain
hcloud cdn create-domain \
  --region "{{user.region}}" \
  --domain-name "example.com" \
  --business-type "web" \
  --service-area "mainland_china" \
  --origin "1.2.3.4" \
  --origin-type "ipaddr"

# 3) Poll until online
hcloud cdn list-domain --region "{{user.region}}" \
  --output json | jq '.result[] | select(.domain_name=="example.com") | {id, status}'
```

### Refresh cache

```bash
hcloud cdn refresh-cache \
  --region "{{user.region}}" \
  --type "file" \
  --urls "https://example.com/css/style.css"

# Poll job status
hcloud cdn list-tasks --region "{{user.region}}" \
  --task-type "refresh_cache" --output json | jq '.result[] | {id, status, create_time}'
```

### Query bandwidth and hit rate

```bash
hcloud cdn list-stats \
  --region "{{user.region}}" \
  --domain-id "$(hcloud cdn list-domain --region {{user.region}} --output json | jq -r '.result[0].id')" \
  --start-time "$(date -d '1 day ago' +%Y%m%d%H%M%S)" \
  --end-time "$(date +%Y%m%d%H%M%S)" \
  --stat-type "bandwidth,hit_rate" \
  --output json
```

## When to Fall Back to SDK

| CLI missing? | Use SDK call |
|---|---|
| Batch refresh (>100 URLs) | `RefreshCache` with array of URLs |
| Detailed statistics (per-province) | `ShowBandwidthInterval` with dimension filters |
| Advanced origin config (OBS + authentication) | `CreateDomain` / `ModifyDomainConfig` with full body |
| Quota detail per project | `ShowDomainDetailQuota` |
