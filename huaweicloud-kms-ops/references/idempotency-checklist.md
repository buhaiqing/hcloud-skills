# KMS Idempotency Checklist — Huawei Cloud Key Management Service

## Idempotent Operations

| Operation | Idempotent? | Safe-retry token | Pre-retry check |
|---|---|---|---|
| `create-key` | Yes (by alias) | n/a | `list-keys` + dedupe by alias |
| `enable-key` | Yes (state=no-op) | n/a | `describe-key` + check state |
| `disable-key` | Yes (state=no-op) | n/a | `describe-key` + check state |
| `schedule-key-deletion` | Yes (already scheduled=success) | n/a | `describe-key` + check state |
| `cancel-key-deletion` | Yes | n/a | `describe-key` + check state |
| `create-grant` | Yes (by key+principal+ops) | n/a | `list-grants` + dedupe |
| `revoke-grant` | Yes (already revoked=success) | n/a | n/a |
| `import-key-material` | No (BYOK import is one-time per key) | n/a | Verify key state is `ENABLED` and origin is `EXTERNAL` |
| `create-datakey` | No (always generates new DEK) | n/a | n/a |
| `decrypt-data-key` | Yes (deterministic for same ciphertext) | n/a | n/a |

## Non-Idempotent Caveats

- **`create-key` with duplicate alias**: Returns the existing key's metadata, not a new key.
  **But**: if the existing key belongs to a different project/entity, you may unknowingly
  use the wrong key for encryption.
- **`import-key-material`**: One-time per key. Calling twice returns `KeyMaterialAlreadyImported`.
- **`create-datakey`**: Always generates a new DEK. Calling twice produces two different DEKs.
  Store both if needed; never call twice for the same data.
