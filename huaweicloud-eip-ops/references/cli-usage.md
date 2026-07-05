# EIP CLI Usage — Huawei Cloud Elastic IP

## CLI Command Map

| Operation | CLI Command | Notes |
|---|---|---|
| List EIPs | `hcloud eip list` | All EIPs in current region |
| Allocate EIP | `hcloud eip create` | Idempotency caveat: see SKILL.md |
| Describe EIP | `hcloud eip describe` | Single EIP detail |
| Update EIP | `hcloud eip update` | Rename / update alias |
| Bind EIP | `hcloud eip bind` | ECS / ENI / NAT / ELB |
| Unbind EIP | `hcloud eip unbind` | Detach |
| Release EIP | `hcloud eip delete` | **Irreversible** — see safety gate |
| Resize bandwidth | `hcloud eip update-bandwidth` | Mbps change |
| Quota | `hcloud eip describe-quota` | Region-level EIP quota |
| List shared bandwidths | `hcloud bandwidth list` | All shared bandwidth pools |
| Create shared bandwidth | `hcloud bandwidth create` | WHOLE type |
| Add EIP to shared | `hcloud bandwidth add-eip` | Move EIP into pool |
| Remove EIP from shared | `hcloud bandwidth remove-eip` | Move EIP back to PER |
| Resize shared bandwidth | `hcloud bandwidth update` | Pool-level Mbps |

> **Verify before use:** `hcloud eip --help` and `hcloud bandwidth --help`. The exact
> subcommand shape evolves between `hcloud` versions; if a subcommand is missing, fall
> back to the **JIT Go SDK** path in `references/api-sdk-usage.md`.

## Common Recipes

### Allocate a 按带宽 EIP (production default)

```bash
hcloud eip create \
  --region "{{user.region}}" \
  --name "eip-prod-{{user.region}}" \
  --type "5_bgp" \
  --billing-mode "bandwidth" \
  --bandwidth-size 5 \
  --charge-type "postpaid"
```

### Allocate a 按流量 EIP (dev/test)

```bash
hcloud eip create \
  --region "{{user.region}}" \
  --name "eip-dev-{{user.region}}" \
  --type "5_bgp" \
  --billing-mode "traffic" \
  --bandwidth-size 100
```

### Bind to ECS

```bash
# 1) find ECS port id
PORT_ID=$(hcloud ecs describe --server-id "{{user.ecs_id}}" \
  --region "{{user.region}}" \
  --output json | jq -r '.server.addresses[][].port_id' | head -1)

# 2) bind
hcloud eip bind \
  --region "{{user.region}}" \
  --eip-id "{{user.eip_id}}" \
  --port-id "$PORT_ID"
```

### Move EIP into shared bandwidth (CLI may not exist; prefer SDK)

```bash
# 1) Create shared bandwidth (WHOLE)
BANDWIDTH_ID=$(hcloud bandwidth create \
  --region "{{user.region}}" \
  --name "bw-pool-{{user.region}}" \
  --share-type "WHOLE" \
  --size 200 \
  --charge-mode "bandwidth" \
  --output json | jq -r '.bandwidth.id')

# 2) Add EIP to pool (subcommand may require SDK fallback)
hcloud bandwidth add-eip \
  --region "{{user.region}}" \
  --bandwidth-id "$BANDWIDTH_ID" \
  --eip-id "{{user.eip_id}}"
```

### Resize bandwidth

```bash
hcloud eip update-bandwidth \
  --region "{{user.region}}" \
  --eip-id "{{user.eip_id}}" \
  --bandwidth-size 20
```

### Release EIP (irreversible)

```bash
# Verify unbound
hcloud eip describe --region "{{user.region}}" --eip-id "{{user.eip_id}}" \
  --output json | jq '.publicip.port_id'   # MUST be null

# Release
hcloud eip delete --region "{{user.region}}" --eip-id "{{user.eip_id}}"
```

## Output Conventions

All commands accept `--output json`. Parse with `jq`:

```bash
# Get public IP + binding state
hcloud eip list --region {{env.HW_REGION_ID}} --output json \
  | jq '.publicips[] | {id, public_ip_address, status, port_id, bandwidth_size: .bandwidth.size}'
```

## When to Fall Back to SDK

| CLI missing? | Use SDK call |
|---|---|
| Move EIP into shared bandwidth programmatically | `eip.UpdatePublicip` with `bandwidth.id` |
| 95计费 subscription change | `bandwidth.UpdateBandwidth` (charge_mode=95) |
| Cross-region EIP (read-only snapshot) | `eip.ShowPublicip` with `?region=…` |
