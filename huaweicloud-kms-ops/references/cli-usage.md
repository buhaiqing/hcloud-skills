# KMS CLI Usage — Huawei Cloud Key Management Service

## CLI Command Map

| Operation | CLI Command | Notes |
|---|---|---|
| List keys | `hcloud kms list-keys` | All CMKs in region |
| Create key | `hcloud kms create-key` | Idempotent by alias |
| Describe key | `hcloud kms describe-key` | Single key metadata |
| Enable key | `hcloud kms enable-key` | Idempotent by state |
| Disable key | `hcloud kms disable-key` | Idempotent by state |
| Schedule deletion | `hcloud kms schedule-key-deletion` | 7–1096 day window |
| Create grant | `hcloud kms create-grant` | Idempotent by (key, principal) |
| List grants | `hcloud kms list-grants` | All grants on a key |
| Revoke grant | `hcloud kms revoke-grant` | Grant_id from list output |
| Create import token | `hcloud kms create-import-token` | BYOK第一步 |
| Import key material | `hcloud kms import-key-material` | BYOK第二步 |
| Create data key | `hcloud kms create-datakey` | Returns plaintext + ciphertext |
| Decrypt data key | `hcloud kms decrypt-datakey` | Decrypt DEK ciphertext |

> **Verify before use:** `hcloud kms --help` and `hcloud kms <subcommand> --help`.

## Common Recipes

### List all CMKs
```bash
hcloud kms list-keys --region {{env.HW_REGION_ID}} --output json \
  | jq '.keys[] | {key_id, key_state: .key_state}'
```

### Create a CMK with rotation
```bash
hcloud kms create-key \
  --region "{{user.region}}" \
  --alias "{{user.key_alias}}" \
  --key-type "SYMMETRIC_DEFAULT" \
  --rotation-enabled
```

### Disable a CMK (safety gate required)
```bash
hcloud kms disable-key \
  --region "{{user.region}}" \
  --key-id "{{user.key_id}}"
```

### Schedule key deletion
```bash
# 1) Verify no active grants
hcloud kms list-grants --key-id "{{user.key_id}}" --region "{{user.region}}"

# 2) Schedule deletion (30-day window)
hcloud kms schedule-key-deletion \
  --region "{{user.region}}" \
  --key-id "{{user.key_id}}" \
  --pending-window-days 30
```

### Generate a data key (for client-side encryption)
```bash
hcloud kms create-datakey \
  --region "{{user.region}}" \
  --key-id "{{user.key_id}}" \
  --datakey-plain-length 32 \
  --output json | jq '{plaintext, ciphertext}'
```

## Output Conventions

All commands accept `--output json`. Parse with `jq`:

```bash
# Key state summary
hcloud kms list-keys --region {{env.HW_REGION_ID}} --output json \
  | jq '.keys[] | select(.key_state == "ENABLED") | .key_id'
```

## When to Fall Back to SDK

| CLI missing? | Use SDK call |
|---|---|
| BYOK import token generation | `CreateImportToken` via SDK |
| Key quota query | `ShowKeyQuotas` via SDK |
| Grant with constraints | `CreateGrant` via SDK (supports retiring_principal) |
| List key rotation policy | `ShowKeyRotationPolicy` via SDK |
