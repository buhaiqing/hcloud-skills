# EIP Idempotency Checklist

Use this checklist before retrying any EIP / bandwidth operation.

| Operation | Idempotent? | Pre-retry check | Safe-retry token |
|---|---|---|---|
| `allocate-eip` (按带宽) | No | `hcloud eip list` and dedupe by `public_ip_address` | n/a |
| `allocate-eip` (按流量) | No | same as above | n/a |
| `allocate-eip` (WHOLE shared) | No | `hcloud bandwidth list` and dedupe by `name` | n/a |
| `describe-eip` | Yes | n/a | n/a |
| `bind-eip` | Yes (idempotent at API level) | confirm target still has same `port_id` | `client_token` accepted |
| `unbind-eip` | Yes | confirm EIP is currently bound | n/a |
| `release-eip` | Yes (404 = success) | confirm `port_id == null` | n/a |
| `resize-bandwidth` | Yes (no-op if same size) | n/a | n/a |
| `add-eip-to-shared` | Yes (no-op if already in pool) | n/a | `client_token` accepted |
| `remove-eip-from-shared` | Yes | n/a | n/a |

## Allocation Dedupe Helper

```bash
# Before retrying allocate, see if a same-named EIP already exists
hcloud eip list --region {{env.HW_REGION_ID}} --output json \
  | jq -r '.publicips[] | select(.alias == "eip-prod-cn-north-4") | .id'
```

If a result is returned, use it instead of allocating.

## Bandwidth Dedupe Helper

```bash
# Check shared bandwidth pool name
hcloud bandwidth list --region {{env.HW_REGION_ID}} --output json \
  | jq -r '.bandwidths[] | select(.name == "bw-pool-cn-north-4") | .id'
```

## Release Flow Idempotency

```bash
# Safe to retry; 404 is success
hcloud eip delete --region {{env.HW_REGION_ID}} --eip-id "{{user.eip_id}}" \
  || echo "Already released (404 expected) — proceeding"
```

## Bandwidth Resize Cooldown Idempotency

If the resize returns `cooldown in progress`, do **not** retry. Wait for cooldown end,
then re-issue exactly once. More than one retry inside the cooldown is wasteful.
