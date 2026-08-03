# Hallucination Detection — GCL Extension

> **目标**: 在 Generator 输出进入 Critic 评分前，增加三层结构化幻觉检测，将"输出看起来对但实际错"的风险降到最低。

## 1. 定位

```
[0] Pre-flight      — 变量解析，选 skill，生成 operation_intent
[1] Generate (G)     — 执行 hcloud / Go SDK
[2] Hallucination Detection (H)  ← NEW — 三层验证
[3] Critique (C)    — 评分 + 建议
[4] Decide (O)       — PASS / MAX_ITER / SAFETY_FAIL
```

H 与 C 串行：H 先过滤问题，输出 `verification_result` 供 C 参考。
C 仍独立评分，但可在 `correctness` 维度叠加 H 的发现。

## 2. 三层检查

### 2.1 L1: CLI 参数存在性

**目标**: 防止 Generator 发出不存在的 `--flag`。

**检查点**:
- 解析 Generator command（从 trace 提取 `iterations[].generator.command`）
- 提取所有 `--flag` 和 `--flag=value` 形式
- 对照 `references/cli-usage.md` 中声明的 flags
- 不存在的 flag → `blocked: true`

**输出 schema**:
```json
{
  "layer": "L1",
  "blocked": false,
  "flags_checked": 5,
  "invalid_flags": [],
  "details": "..."
}
```

### 2.2 L2: JSON 结构合规

**目标**: 防止 Generator 输出不符合 OpenAPI schema。

**检查点**:
- 解析 Generator raw response（JSON body）
- 对照 skill `references/` 中的 `openapi-schema.json`（若有）
- Schema 校验：required 字段缺失、类型错误、enum 越界
- 不合规 → `blocked: true`

**输出 schema**:
```json
{
  "layer": "L2",
  "blocked": false,
  "errors": [],
  "details": "..."
}
```

### 2.3 L3: WAF 合规

**目标**: 离线安全/成本/稳定性检查。

| 子维度 | 检查项 |
|--------|--------|
| Security | 凭证泄露（AK/SK/密码明文）、危险动词（delete without guard） |
| Cost | 高成本操作（大规模实例变更）、未声明的资源范围 |
| Stability | 多资源批量变更、缺少回滚计划 |

**检查点**:
- Security: 正则扫描 raw response 中的 `<redacted>` 缺失
- Cost: 对照 `resource_context.monthly_cost_usd` 和 `utilization_pct`
- Stability: operation_intent 中 `blast_radius` > `single-resource` 且无 rollback_plan

**输出 schema**:
```json
{
  "layer": "L3",
  "blocked": false,
  "violations": [
    {"dimension": "security", "rule": "credential-exposure", "detail": "..."},
    {"dimension": "cost", "rule": "high-cost-operation", "detail": "..."},
    {"dimension": "stability", "rule": "multi-resource-mutation", "detail": "..."}
  ],
  "details": "..."
}
```

## 3. 合并结果

H 阶段输出合并为:
```json
{
  "blocked": false,
  "layers": {
    "L1": { "blocked": false, "invalid_flags": [] },
    "L2": { "blocked": false, "errors": [] },
    "L3": { "blocked": false, "violations": [] }
  },
  "summary": "3/3 layers passed"
}
```

| `blocked` | 行为 |
|-----------|------|
| `false` | 继续到 C |
| `true` (L1/L2) | SAFETY_FAIL，立即终止 |
| `true` (L3) | 记录 violations，C 评分时叠加到 `correctness` |

## 4. 实现路径

| 组件 | 文件 | 职责 |
|------|------|------|
| `HallucinationDetector` interface | `internal/gcl/hallucination.go` | H 阶段总控 |
| `CLIParamValidator` | `internal/gcl/hallucination_l1.go` | L1 检查 |
| `JSONSchemaValidator` | `internal/gcl/hallucination_l2.go` | L2 检查 |
| `WAFComplianceChecker` | `internal/gcl/hallucination_l3.go` | L3 检查 |
| 入口集成 | `internal/gcl/runner.go` | 在 C 前调用 H |

## 5. 集成方式

`runner.go` 中:

```go
// [2] Hallucination Detection
hd := NewHallucinationDetector(skillRoot)
verification, err := hd.Run(ctx, trace)
if err != nil {
    return nil, fmt.Errorf("hallucination detection failed: %w", err)
}
trace.HallucinationDetection = verification

if verification.BlockedBySafety() {
    trace.Final.Status = "SAFETY_FAIL"
    trace.Final.HallucinationBlocked = true
    return trace, nil
}

// [3] Critique (passes verification to Critic)
```

## 6. 测试策略

| 场景 | 预期 |
|------|------|
| Generator 使用不存在的 `--no-wait` flag | L1 blocked, SAFETY_FAIL |
| Response 缺少 required 字段 | L2 blocked, SAFETY_FAIL |
| Response 含明文 AK | L3 security violation, blocked by safety |
| 全部通过 | 继续 C 评分 |
