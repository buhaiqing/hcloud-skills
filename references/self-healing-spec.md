# Self-Healing Loop Specification (L4)

> **Purpose**: 定义所有 `huaweicloud-*-ops` skill 的自愈闭环标准 — 从被动响应到主动修复。
> **Scope**: 每个 skill 的 `../huaweicloud-ecs-ops/assets/remediation-playbooks.json` + `assets/failure_patterns.json`
> **Version**: 1.0.0

---

## 1. Architecture

```
Alarm/Anomaly ──▶ Detect ──▶ Diagnose ──▶ Match Playbook ──▶ Execute ──▶ Verify ──▶ Learn
                                              │                                    │
                                              ▼                                    ▼
                                    failure_patterns.json              Update failure_patterns.json
                                    (pre-execution risk check)         (post-execution feedback)
```

### 1.1 与 GCL 的关系

| 组件 | 职责 | 触发时机 |
|------|------|---------|
| GCL | 操作质量门（Correctness/Safety/Idempotency） | 每次操作执行时 |
| Self-Healing Loop | 故障自动修复（Detect→Fix→Verify） | 告警/异常触发时 |
| Experience Learning | 从 trace 中提取模式，更新 knowledge base | GCL 执行后 / 定期聚合 |

---

## 2. Remediation Playbook Schema

每个 skill 维护 `../huaweicloud-ecs-ops/assets/remediation-playbooks.json`：

```json
{
  "$schema": "remediation-playbooks/v1",
  "skill_id": "huaweicloud-<product>-ops",
  "playbooks": [
    {
      "id": "<PRODUCT>-R001",
      "name": "human-readable name",
      "trigger": {
        "metric": "CES metric name",
        "condition": "> 90 for 5min",
        "namespace": "SYS.<PRODUCT>"
      },
      "diagnosis": {
        "steps": ["step1 command", "step2 command"],
        "confidence_factors": ["factor1", "factor2"]
      },
      "remediation": {
        "risk_level": "low|medium|high|critical",
        "auto_execute_threshold": 0.8,
        "preconditions": ["precondition check commands"],
        "dry_run": "dry-run command",
        "execute": "execute command",
        "verification": "verification command",
        "rollback": "rollback command or null if irreversible",
        "timeout_seconds": 300
      },
      "escalation": {
        "condition": "when to escalate to human",
        "channel": "notification method"
      },
      "metadata": {
        "success_rate": 0.95,
        "avg_execution_seconds": 120,
        "last_updated": "ISO8601",
        "learned_from": ["gcl-trace-*.json references"]
      }
    }
  ]
}
```

### 2.1 Risk Level 与自治分级

| Risk Level | auto_execute_threshold | 行为 |
|-----------|----------------------|------|
| `low` | 0.7 | 自动执行 + 通知 |
| `medium` | 0.85 | 置信度达标自动执行，否则确认 |
| `high` | 0.95 | 必须人工确认 |
| `critical` | — | 永远需要人工确认（不可自动） |

### 2.2 Confidence 计算

```
confidence = base_confidence
  × diagnosis_factor    (诊断步骤完成度)
  × precondition_factor (前置条件满足度)
  × history_factor      (历史成功率)
```

- `base_confidence`: playbook 定义的基准置信度
- `diagnosis_factor`: 诊断步骤全部通过 = 1.0，部分通过按比例
- `precondition_factor`: 前置条件全部满足 = 1.0
- `history_factor`: 从 `failure_patterns.json` 读取历史成功率

---

## 3. Failure Patterns Schema (Experience Learning)

每个 skill 维护 `assets/failure_patterns.json`：

```json
{
  "$schema": "failure-patterns/v1",
  "skill_id": "huaweicloud-<product>-ops",
  "patterns": [
    {
      "id": "<PRODUCT>-FP001",
      "category": "cli_parameter|runtime|cross_skill|permission|resource_state|network",
      "signature": {
        "error_code": "Ecs.0801",
        "error_message_regex": "InsufficientResource",
        "command_pattern": "create-server"
      },
      "root_cause": "human-readable root cause",
      "fix": {
        "strategy": "retry|fallback|delegate|halt|auto_remediate",
        "action": "concrete fix command or delegation target",
        "playbook_ref": "<PRODUCT>-R001 or null"
      },
      "prevention": "how to avoid this in future",
      "stats": {
        "occurrence_count": 5,
        "first_seen": "ISO8601",
        "last_seen": "ISO8601",
        "auto_fixed_count": 3,
        "escalated_count": 2,
        "success_rate": 0.6
      },
      "learned_from": ["gcl-trace-20260717-155347.json"]
    }
  ],
  "meta": {
    "total_patterns": 10,
    "last_aggregation": "ISO8601",
    "source_traces_analyzed": 42
  }
}
```

---

## 4. Self-Healing Execution Flow

```python
def self_heal(alarm_event, skill_id):
    # 1. Load playbooks + failure patterns
    playbooks = load_playbooks(skill_id)
    patterns = load_failure_patterns(skill_id)

    # 2. Pre-execution risk check (Experience Learning integration)
    known_risks = match_known_patterns(alarm_event, patterns)
    if known_risks:
        # Apply learned fix directly if high confidence
        if known_risks.fix.strategy == "auto_remediate":
            if known_risks.stats.success_rate > 0.9:
                return execute_known_fix(known_risks)

    # 3. Match playbook
    playbook = match_playbook(alarm_event, playbooks)
    if not playbook:
        return escalate("No matching playbook", alarm_event)

    # 4. Run diagnosis
    diagnosis = run_diagnosis(playbook.diagnosis)
    confidence = compute_confidence(playbook, diagnosis, patterns)

    # 5. Decision gate
    if confidence < playbook.remediation.auto_execute_threshold:
        return escalate(f"Confidence {confidence} below threshold", alarm_event)

    # 6. Dry-run
    dry_run_result = execute(playbook.remediation.dry_run)
    if not dry_run_result.success:
        return escalate("Dry-run failed", dry_run_result)

    # 7. Execute
    result = execute(playbook.remediation.execute)

    # 8. Verify
    verified = verify(playbook.remediation.verification)
    if not verified:
        rollback(playbook.remediation.rollback)
        return escalate("Verification failed, rolled back", result)

    # 9. Learn (update failure_patterns.json)
    record_success(playbook, patterns)
    return Success(result)
```

---

## 5. Experience Learning Loop

### 5.1 触发条件

- GCL trace 写入后（`gcl_runner.py` 自动触发）
- 定期聚合（`scripts/trace_learning.py aggregate`）
- 手动触发（`scripts/trace_learning.py learn --skill <skill_id>`）

### 5.2 学习流程

```
audit-results/gcl-trace-*.json
        │
        ▼
┌─────────────────────┐
│  trace_learning.py  │
│  aggregate          │
└─────────────────────┘
        │
        ├── 1. Parse all traces since last aggregation
        ├── 2. Extract failure_pattern from each trace
        ├── 3. Cluster by (skill, category, error_code)
        ├── 4. Compute stats (occurrence, success_rate)
        ├── 5. Match against existing patterns (dedup)
        ├── 6. Update assets/failure_patterns.json
        └── 7. Generate learning report (stdout)
```

### 5.3 GCL Runner 集成

`gcl_runner.py` 在执行前查询 failure_patterns.json：

```python
# Pre-execution: check known risks
patterns = load_failure_patterns(skill_id)
risk = match_pre_execution_risk(command, patterns)
if risk:
    trace["pre_execution_risk"] = {
        "pattern_id": risk.id,
        "category": risk.category,
        "known_fix": risk.fix.action,
        "historical_success_rate": risk.stats.success_rate,
    }
```

---

## 6. Skill Generator 集成

`huaweicloud-skill-generator` 生成新 skill 时须：

1. 创建 `../huaweicloud-ecs-ops/assets/remediation-playbooks.json`（至少 3 个 playbook）
2. 创建 `assets/failure_patterns.json`（从 troubleshooting.md 提取种子）
3. 在 SKILL.md 中引用：`[Self-Healing Playbooks](../huaweicloud-ecs-ops/assets/remediation-playbooks.json)`

---

## 7. Compliance Checklist

- [ ] 每个 required GCL skill 有 `../huaweicloud-ecs-ops/assets/remediation-playbooks.json`
- [ ] 每个 required GCL skill 有 `assets/failure_patterns.json`
- [ ] Playbook 覆盖 troubleshooting.md 中 ≥ 60% 的错误码
- [ ] 每个 playbook 有 dry_run + verification + rollback
- [ ] `trace_learning.py` 可成功聚合现有 traces
- [ ] GCL runner 输出包含 `pre_execution_risk` 字段（当匹配到已知模式时）
