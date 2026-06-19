# FunctionGraph Safety Gates — High-Risk Operation Controls

> Advanced safety controls for FunctionGraph serverless.
> Load when deleting functions, rotating triggers, or deploying new versions.

## 1. Destructive Operation Catalogue

| Operation | Risk class | Default gate |
|-----------|-----------|--------------|
| `DeleteFunction` | irreversible (code + alias) | publish new version first + confirmation |
| `DeleteFunctionTrigger` | event-source disconnect | route events to alternate alias |
| `UpdateFunctionCode` | live replacement | canary via alias + rollback plan |
| `DeleteEvent` | log loss | archive to LTS first |
| `AsyncInvokeFail` | DLQ injection | dry-run + retry budget |

## 2. Safety Gate Workflow

1. **Snapshot**: `PublishVersion` to capture current code + config
2. **Alias routing**: shift traffic via alias to staged version
3. **Canary**: run new version at ≤ 10% traffic for 10 min
4. **Confirm**: collect `{{user.confirm_destructive}}` per trigger / version
5. **Execute**: apply mutation, monitor error rate
6. **Rollback**: shift alias back to snapshot version on alarm

## 3. Cross-Skill Delegation

- `huaweicloud-functiongraph-ops → huaweicloud-obs-ops` for code package storage
- `huaweicloud-functiongraph-ops → huaweicloud-swr-ops` for container images
- `huaweicloud-functiongraph-ops → huaweicloud-lts-ops` for log aggregation

> **Security-Sensitive**: function deletion, trigger deletion, and code
> mutation MUST pass the Safety Gate. Always publish a new version before
> mutating the live alias.