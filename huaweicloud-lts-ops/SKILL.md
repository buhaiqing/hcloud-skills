---
name: huaweicloud-lts-ops
description: >-
  Use when the user needs to configure, query, transfer, or troubleshoot Huawei
  Cloud LTS (Log Tank Service / 云日志服务) — log group/stream lifecycle, log
  search/query, log transfer to OBS/DMS, structured parsing, dashboard
  management, and log-based alarm configuration. User mentions LTS, log tank,
  log group, log stream, 云日志, 日志组, 日志流, 日志转储, or describes
  logging scenarios (e.g., log search too slow, log not collected, log transfer
  failure, cannot create log group) even without naming the product directly.
  Not for billing, IAM, or related products that have their own ops skills.
license: MIT
compatibility: >-
  Official Huawei Cloud CLI (`hcloud`), Go 1.21+ runtime
  (for JIT SDK fallback via huaweicloud-sdk-go-v3), valid API credentials,
  network access to Huawei Cloud endpoints.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-05-21"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "LTS v2 — https://support.huaweicloud.com/api-lts/"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    LTS operations available via `hcloud LTS <operation>` where operation is
    the API Explorer name: CreateLogGroup, ListLogGroups, DeleteLogGroup,
    CreateLogStream, ListLogStreams, DeleteLogStream, ListLogs,
    CreateLogDumpObs, CreateTransfer, ListTransfers, DeleteTransfer,
    CreateDashboard, ListDashboards.
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
  gcl:
    enabled: true
    required: false
    rubric_version: "v1"
    max_iter: 3
    rubric_ref: "references/rubric.md"
    prompts_ref: "references/prompt-templates.md"
    trace_dir: "./audit-results/"
    changelog:
      - version: "1.1.0"
        date: "2026-06-04"
        change: "GCL Phase 3 rollout: added references/rubric.md (v1, 5-dim, S1–S9 LTS-specific Safety rules, including log-group-delete-without-confirmation / log-loss-without-backup / transfer-dangling / log-retention-change-without-notice / credential-leak guards) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud LTS (Log Tank Service) Operations Skill

## Overview

Huawei Cloud LTS provides centralized log collection, storage, search, analysis, and transfer. This skill covers log groups, log streams, log search/query, structured parsing, log transfer, and dashboard management as an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official CLI and JIT Go SDK fallback), response validation, and failure recovery.

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** KooCLI supports LTS via API Explorer operations. In each execution flow below, document **both** the SDK step **and** the CLI step.

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions with precise triggers and delegation rules |
| 2 | **Structured I/O** | `{{env.*}}` / `{{user.*}}` / `{{output.*}}` placeholders with typed sources |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute (CLI + SDK) → Validate → Recover |
| 4 | **Complete Failure Strategies** | ≥ 12 LTS error codes with HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | One product (LTS); cross-product delegation to CES, OBS, DMS, CTS, CCE, ECS documented |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

| Pillar | Integration |
|--------|-------------|
| **FinOps** | Log storage billing (TTL vs long-term OBS), transfer cost comparison, index volume cost control |
| **SecOps** | IAM permissions (lts:*), log encryption, access control via CTS audit, credential masking |
| **AIOps** | ≥ 4 anomaly patterns (log volume spike, no new logs, search latency, transfer failure), cross-skill delegation to CES/CTO/OBS |

## SHOULD Use (Trigger Conditions)

Trigger when the user's intent matches any of:

- **Log Group**: create, list, update (TTL), delete log groups
- **Log Stream**: create, list, delete log streams within a log group
- **Log Search & Query**: search logs by keyword, time range, labels; pagination through results
- **Log Transfer**: configure log transfer to OBS or DMS
- **Dashboard**: create, list, update, delete LTS dashboards
- **Structured Parsing**: configure log structuring rules for search optimization
- **Quick Search (Saved Search)**: create, list, delete saved search queries
- **Troubleshooting**: "log not showing up", "search too slow", "transfer failed", "cannot create log group/stream", "ICAgent not collecting"
- **Log Retention**: adjust `ttl_in_days` on log groups

## SHOULD NOT Use

- **IAM/Role management** → delegate to `huaweicloud-iam-ops`
- **Cloud resource monitoring/alarms (CES)** → delegate to `huaweicloud-ces-ops`
- **OBS bucket/object lifecycle** → delegate to `huaweicloud-obs-ops`
- **DMS message queue management** → delegate to `huaweicloud-dms-ops`
- **CTS audit trail management** → delegate to `huaweicloud-cts-ops`
- **K8s pod/container log investigation** → delegate to `huaweicloud-cce-ops`
- **ECS instance/disk troubleshooting** → delegate to `huaweicloud-ecs-ops`

## Operational Flows

### Common Variables

| Variable | Source | Description |
|----------|--------|-------------|
| `{{env.HW_ACCESS_KEY_ID}}` | environment | Huawei Cloud AK |
| `{{env.HW_SECRET_ACCESS_KEY}}` | environment | Huawei Cloud SK |
| `{{env.HW_REGION_ID}}` | environment | Region (e.g., cn-north-4) |
| `{{env.HW_PROJECT_ID}}` | environment | Project ID |
| `{{user.log_group_name}}` | user input | Log group name |
| `{{user.log_stream_name}}` | user input | Log stream name |
| `{{output.log_group_id}}` | API response | Log group UUID |
| `{{output.log_stream_id}}` | API response | Log stream UUID |
| `{{user.ttl_in_days}}` | user input | Log retention in days (1–365) |
| `{{user.keywords}}` | user input | Search keywords |
| `{{user.start_time}}` | user input | Search start epoch (ms) |
| `{{user.end_time}}` | user input | Search end epoch (ms) |

### Flow 1: Create Log Group

**Pre-flight:**
1. Verify `{{env.HW_ACCESS_KEY_ID}}`, `{{env.HW_SECRET_ACCESS_KEY}}`, `{{env.HW_REGION_ID}}`, `{{env.HW_PROJECT_ID}}` are set.
2. Validate `{{user.log_group_name}}` — 1–64 chars, only letters/digits/underscores/hyphens.
3. Validate `{{user.ttl_in_days}}` — integer, 1–365.

**Execute (CLI):**
```bash
hcloud LTS CreateLogGroup \
  --cli-region="{{env.HW_REGION_ID}}" \
  --project_id="{{env.HW_PROJECT_ID}}" \
  --log_group_name="{{user.log_group_name}}" \
  --ttl_in_days={{user.ttl_in_days}}
```

**Execute (SDK):**
```go
import (
    lts "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/lts/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/lts/v2/model"
)

client := lts.NewLtsClient(lts.LtsClientBuilder().
    WithRegion(region.ValueOf(os.Getenv("HW_REGION_ID"))).
    WithCredential(auth).
    Build())

request := &model.CreateLogGroupRequest{
    Body: &model.CreateLogGroupParams{
        LogGroupName: "{{user.log_group_name}}",
        TtlInDays:    int32({{user.ttl_in_days}}),
    },
}
response, err := client.CreateLogGroup(request)
```

**Validate:**
- CLI: Confirm response contains `log_group_id` (36-char UUID).
- SDK: Check `err == nil` and `response.LogGroupId != ""`.

**Recover:**
- `LTS.0001` (invalid parameter) → check name format and TTL range.
- `LTS.0101` (quota exceeded) → list existing groups, consider deletion or raise quota.
- `LTS.0102` (name conflict) → use a different name.

### Flow 2: List Log Groups

**Pre-flight:** Verify credentials as in Flow 1.

**Execute (CLI):**
```bash
hcloud LTS ListLogGroups \
  --cli-region="{{env.HW_REGION_ID}}" \
  --project_id="{{env.HW_PROJECT_ID}}"
```

**Execute (SDK):**
```go
request := &model.ListLogGroupsRequest{}
response, err := client.ListLogGroups(request)
```

**Validate:**
- Confirm `log_groups` array is returned (may be empty).
- Each entry has `log_group_id`, `log_group_name`, `ttl_in_days`, `creation_time`.

**Recover:**
- `LTS.0201` (auth failure) → check AK/SK validity.
- Empty list → no groups exist in this project+region.

### Flow 3: Delete Log Group

**Safety Gate (MANDATORY):** Confirm with user: `Are you sure you want to delete log group "{{user.log_group_name}}" ({{output.log_group_id}})? This will also delete all log streams and logs within. [yes/NO]`

**Pre-flight:**
1. Verify credentials.
2. Resolve `{{user.log_group_name}}` → `{{output.log_group_id}}` via ListLogGroups.

**Execute (CLI):**
```bash
hcloud LTS DeleteLogGroup \
  --cli-region="{{env.HW_REGION_ID}}" \
  --project_id="{{env.HW_PROJECT_ID}}" \
  --log_group_id="{{output.log_group_id}}"
```

**Execute (SDK):**
```go
request := &model.DeleteLogGroupRequest{
    LogGroupId: "{{output.log_group_id}}",
}
response, err := client.DeleteLogGroup(request)
```

**Validate:**
- CLI: Confirm HTTP 200 with no body/empty response.
- SDK: Confirm `err == nil`.

**Recover:**
- `LTS.0401` (group not found) → re-verify group ID; it may have been deleted already.
- `LTS.0402` (group has active transfer) → delete transfer rules first; use `ListTransfers` to find active rules.

### Flow 4: Create Log Stream

**Pre-flight:**
1. Verify credentials.
2. Resolve `{{user.log_group_name}}` → `{{output.log_group_id}}` via ListLogGroups.
3. Validate `{{user.log_stream_name}}` — 1–64 chars, only letters/digits/underscores/hyphens.

**Execute (CLI):**
```bash
hcloud LTS CreateLogStream \
  --cli-region="{{env.HW_REGION_ID}}" \
  --project_id="{{env.HW_PROJECT_ID}}" \
  --log_group_id="{{output.log_group_id}}" \
  --log_stream_name="{{user.log_stream_name}}"
```

**Execute (SDK):**
```go
request := &model.CreateLogStreamRequest{
    LogGroupId: "{{output.log_group_id}}",
    Body: &model.CreateLogStreamParams{
        LogStreamName: "{{user.log_stream_name}}",
    },
}
response, err := client.CreateLogStream(request)
```

**Validate:**
- Confirm `log_stream_id` is returned (36-char UUID).

**Recover:**
- `LTS.0301` (group not found) → group may have been deleted; re-list and retry.
- `LTS.0302` (stream name conflict) → use a unique stream name.
- `LTS.0101` (stream quota exceeded) → max 200 streams per group; delete unused streams.

### Flow 5: Search Logs

**Pre-flight:**
1. Verify credentials.
2. Resolve `{{user.log_group_name}}` → `{{output.log_group_id}}`.
3. Resolve `{{user.log_stream_name}}` → `{{output.log_stream_id}}` via ListLogStreams.
4. Convert `{{user.start_time}}` and `{{user.end_time}}` to epoch milliseconds.

**Execute (CLI):**
```bash
hcloud LTS ListLogs \
  --cli-region="{{env.HW_REGION_ID}}" \
  --project_id="{{env.HW_PROJECT_ID}}" \
  --log_group_id="{{output.log_group_id}}" \
  --log_stream_id="{{output.log_stream_id}}" \
  --start_time={{user.start_time}} \
  --end_time={{user.end_time}} \
  --keywords="{{user.keywords}}" \
  --limit=100
```

**Execute (SDK):**
```go
request := &model.ListLogsRequest{
    LogGroupId:  "{{output.log_group_id}}",
    LogStreamId: "{{output.log_stream_id}}",
    Body: &model.ListLogsParams{
        StartTime: {{user.start_time}},
        EndTime:   {{user.end_time}},
        Keywords:  "{{user.keywords}}",
        Limit:     int32(100),
        IsCount:   bool(true),
    },
}
response, err := client.ListLogs(request)
```

**Validate:**
- Confirm `count` or array entries returned.
- For pagination: use `line_num`, `is_desc`, and `search_type` from response.

**Recover:**
- `LTS.0501` (invalid time range) → ensure start < end; both in epoch milliseconds.
- `LTS.0502` (keywords too long) → limit to 2048 chars.
- `LTS.0503` (index not configured) → suggest configuring structured indexing for the log stream.
- Empty results → widen time range or simplify keywords.

### Flow 6: Create Log Transfer (to OBS)

**Pre-flight:**
1. Verify credentials, resolve group/stream IDs.
2. Verify OBS bucket exists via `huaweicloud-obs-ops` or confirm with user.
3. Determine transfer period (log file aggregation interval in seconds: 30, 60, 300, 3600).

**Safety Gate:** Confirm with user: `This will transfer logs from "{{user.log_stream_name}}" to OBS bucket "{{user.obs_bucket_name}}". Confirm transfer frequency and OBS path. [yes/NO]`

**Execute (CLI):**
```bash
hcloud LTS CreateTransfer \
  --cli-region="{{env.HW_REGION_ID}}" \
  --project_id="{{env.HW_PROJECT_ID}}" \
  --log_group_id="{{output.log_group_id}}" \
  --log_stream_ids="[{{output.log_stream_id}}]" \
  --obs_bucket_name="{{user.obs_bucket_name}}" \
  --obs_period=300 \
  --obs_dir_prefix="lts-transfer/"
```

**Execute (SDK):** Use `CreateTransfer` from `model` package with `CreateTransferRequestBody`.

**Validate:**
- Confirm `log_transfer_id` is returned.
- Verify transfer status via `ListTransfers`.

**Recover:**
- `LTS.0601` (bucket not found) → verify OBS bucket exists and permissions.
- `LTS.0602` (transfer already exists) → update or delete existing transfer rule.
- `LTS.0603` (invalid bucket policy) → check OBS bucket policy allows LTS write access.

### Flow 7: Configure Log Retention (Update Log Group TTL)

**Pre-flight:** Resolve `{{user.log_group_name}}` → `{{output.log_group_id}}`.

**Execute (CLI):**
```bash
hcloud LTS UpdateLogGroup \
  --cli-region="{{env.HW_REGION_ID}}" \
  --project_id="{{env.HW_PROJECT_ID}}" \
  --log_group_id="{{output.log_group_id}}" \
  --ttl_in_days={{user.ttl_in_days}}
```

**Validate:**
- Confirm HTTP 200 response.
- Re-query group and verify `ttl_in_days` updated.

**Recover:**
- `LTS.0701` (TTL out of range) → must be 1–365.

## Error Taxonomy

| Code | Agent Action | UX Feedback |
|------|--------------|-------------|
| `LTS.0001` | HALT — Validate inputs (name, TTL, etc.) | Show validation error |
| `LTS.0101` | HALT — Suggest deleting unused resources | Show quota limits and usage |
| `LTS.0102` | HALT — Suggest unique name | Show conflict details |
| `LTS.0201` | HALT — Check AK/SK validity | Show auth error |
| `LTS.0202` | HALT — Suggest IAM policy review | Show missing permissions |
| `LTS.0301` | Re-list groups and verify ID (retry 1×, 2s) | Show group resolution |
| `LTS.0302` | Re-list streams and verify ID (retry 1×, 2s) | Show stream resolution |
| `LTS.0401` | HALT — Confirm already deleted | Show not-found message |
| `LTS.0402` | HALT — List and delete transfers first | Show active transfers |
| `LTS.0501` | HALT — Ensure start < end, correct format | Show time format |
| `LTS.0502` | Trim to 2048 chars (retry 1×, 1s) | Show limit info |
| `LTS.0503` | HALT — Guide user to configure indexing | Show indexing guidance |
| `LTS.0601` | Verify OBS bucket (retry 1×, 3s) | Show bucket error |
| `LTS.0602` | HALT — Offer to update existing rule | Show transfer conflict |
| `LTS.0603` | HALT — Check bucket ACL/permissions | Show policy error |
| `LTS.0701` | HALT — Confirm 1–365 | Show accepted range |
| `LTS.0801` | Retry with backoff 3× (5s, 10s, 20s) | Show retry status |
| `LTS.0802` | Retry with backoff 3× (10s, 30s, 60s) | Show service status |

## IAM Minimum Permissions

| Operation | Required IAM Policy |
|-----------|-------------------|
| List log groups | `lts:logGroup:listLogGroup` |
| Create log group | `lts:logGroup:createLogGroup` |
| Delete log group | `lts:logGroup:deleteLogGroup` |
| Update log group (TTL) | `lts:logGroup:updateLogGroup` |
| List log streams | `lts:logStream:listLogStream` |
| Create log stream | `lts:logStream:createLogStream` |
| Delete log stream | `lts:logStream:deleteLogStream` |
| Search logs | `lts:logs:listLogs` |
| Create transfer | `lts:transfer:createTransfer` |
| List transfers | `lts:transfer:listTransfers` |
| Delete transfer | `lts:transfer:deleteTransfer` |
| Manage dashboards | `lts:dashboard:*` |

> **Minimum**: `LTS ReadOnlyAccess` for read operations. `LTS FullAccess` for write/delete operations. For transfers, also `OBS OperateAccess` on the target bucket.

## Quality Gate (GCL)

This skill is **GCL-recommended** (per `AGENTS.md` §8). Every LTS mutating operation — log group create / delete, log stream create, log transfer create / delete, retention (TTL) update — runs through the **Generator-Critic-Loop** before its result is returned. Read-only list / search operations are GCL-**exempt**.

| Field | Value |
|-------|-------|
| Rubric version | v1 (Phase 3, 2026-06-04) |
| `max_iter` | **3** |
| Rubric instance | [`references/rubric.md`](references/rubric.md) |
| Prompt templates | [`references/prompt-templates.md`](references/prompt-templates.md) |
| Trace path | `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` |
| Independence | Generator and Critic in **isolated** sub-agent / session contexts |

### Five-Dimension Rubric (summary)

| # | Dimension | Threshold | Notes |
|---|-----------|-----------|-------|
| 1 | Correctness | ≥ 0.5 | `ShowLogGroup` / `ShowTransfer` post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S9 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Pre-check before create |
| 4 | Traceability | ≥ 0.5 | Credential MUST be `<masked>` |
| 5 | Spec Compliance | ≥ 0.5 | Retention period / transfer target / quota limits |

### Per-Operation Safety Anchors (binding)

- **S1** — `delete-log-group` without explicit user confirmation quoting the group ID
- **S2** — `delete-log-group` that still contains active log streams (potential log loss)
- **S3** — `delete-log-group` without offering to transfer logs to OBS first
- **S4** — `create-log-transfer` targeting a non-existent or inaccessible OBS bucket
- **S5** — `delete-log-transfer` while log retention is set to "never expire" (permanent log loss)
- **S6** — `update-retention` (TTL) shorter than existing log age without warning about data loss
- **S7** — `create-log-group` without checking quota (max groups per account)
- **S8** — any trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / password plaintext
- **S9** — `create-log-stream` under a group that has already reached max stream quota

### Termination Contract (per `AGENTS.md` §5)

| Condition | Status | Returned |
|-----------|--------|----------|
| All dimensions pass | **PASS** | Generator result + scores + trace path |
| `iter == max_iter` (3) and any dim < threshold | **MAX_ITER** | best-so-far + unresolved rubric items |
| `Safety == 0` | **SAFETY_FAIL** | violated S-rule id; **never** return partial |

### Trace Persistence (mandatory)

Every GCL run writes `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` (schema in `references/prompt-templates.md` §3). Trace is **append-only**; sanitize secrets before write. The path `./audit-results/` is in root `.gitignore`.

### See also

- [`references/rubric.md`](references/rubric.md) — full rubric, S1–S9 rules, per-op thresholds
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## Reference Directory

- [Core Concepts](references/core-concepts.md)
- [API & SDK Usage](references/api-sdk-usage.md)
- [CLI Usage](references/cli-usage.md)
- [Troubleshooting Guide](references/troubleshooting.md)
- [Monitoring & Alerts](references/monitoring.md)
- [Integration](references/integration.md)
- [Well-Architected Assessment](references/well-architected-assessment.md)
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 5-dim, S1–S9 LTS-specific Safety rules)
- [GCL Prompt Templates](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons

## Delegation to Other Skills

| Scenario | Skill to Delegate |
|----------|------------------|
| Cloud resource monitoring, alarm rules | `huaweicloud-ces-ops` |
| OBS bucket/object management for transfer target | `huaweicloud-obs-ops` |
| DMS queue/topic as transfer target | `huaweicloud-dms-ops` |
| CTS audit trail for LTS API calls | `huaweicloud-cts-ops` |
| IAM user/role/policy management | `huaweicloud-iam-ops` |
| ECS + ICAgent installation root cause | `huaweicloud-ecs-ops` |
| K8s container log collection | `huaweicloud-cce-ops` |

## Well-Architected Assessment

Refer to `references/well-architected-assessment.md` for the five-pillar assessment including FinOps, SecOps, and AIOps integration.
