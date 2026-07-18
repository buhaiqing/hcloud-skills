# Spec: skillcheck B 类迁移 — GCL 契约校验 + 运行时 GCL 编排

> Status: ✅ FINAL_SPEC — 3-round self-critique complete (8 findings addressed)
> Last updated: 2026-07-18
> Author: orchestrator (per user request)
> Parent: `docs/superpowers/specs/skillcheck-cli.md`

## 1. 背景与目标 (Why)

当前 hcloud-skills 仓库仍保留 8 个 Python 脚本（~1,940 行）未迁移，另有 2 个基础设施脚本。目标是**彻底消除 Python 运行时依赖**，将所有功能统一到 `skillcheck` Go 二进制中：

| 类别 | 数量 | 行数 | 操作 |
|------|------|------|------|
| B 类静态校验 | 6 | 1,118 | 迁移到 skillcheck `validate` / `check` 子命令 |
| GCL 运行时 | 2 | 822 | 迁移到 skillcheck `gcl` 子命令命名空间 |
| Python 基础设施 | 2 | 260 | 1 个删除（py310），1 个 Go 重写（install_hook） |

### 1.1 验收成功标准 (Definition of Done)

- [ ] 8 个 Python 脚本全部迁移到 skillcheck，单二进制覆盖所有功能
- [ ] 所有子命令通过 `--help` 可发现，行为与 Python 原版**结构化等价**
- [ ] CI 管线不再依赖 Python 解释器（保留 pre-commit hook 作为可选）
- [ ] `Makefile` 提供 `install-hook` target（Go 版 git hook 安装器）
- [ ] `check_py310_compat.py` 删除，`ruff.toml` / `run_ruff.sh` / `pre_commit_check.sh` 标记为可清理
- [ ] 用户手册（`docs/manual/`）更新为新命令体系
- [ ] 等价性测试覆盖所有迁移的子命令

## 2. 范围 (Scope)

### 2.1 IN — 8 个脚本全部迁移

| 脚本 | 行数 | 目标子命令 | 命名空间 |
|------|------|-----------|---------|
| `check_gcl_conformance.py` | 151 | `validate gcl-conformance` | 静态校验 |
| `check_gcl_alarm_wire_contract.py` | 220 | `validate alarm-wire-contract` | 静态校验 |
| `check_safety_class_enum.py` | 203 | `validate safety-class` | 静态校验 |
| `check_resource_scope_pii.py` | 239 | `validate resource-scope` | 静态校验 |
| `check_generator_contract.py` | 125 | `validate generator-contract` | 静态校验 |
| `check_audit_results_guard.py` | 180 | `check audit-results` | 静态校验 |
| `gcl_runner.py` | 485 | `gcl run` | GCL 运行时 |
| `gcl_alarm_wire.py` | 337 | `gcl alarm-wire` | GCL 运行时 |
| `install_git_hook.py` | 87 | `scripts/install_hook.go` + `make install-hook` | 基础设施 |

### 2.2 OUT — 直接删除

| 文件 | 行数 | 原因 |
|------|------|------|
| `check_py310_compat.py` | 173 | Python 运行时检查，Go 产物无意义 |
| `scripts/fixtures/` | — | fixture 已通过 skillcheck embed 内嵌进二进制 |

### 2.3 命令层级设计

```
skillcheck
├── validate                          # 总入口 (已有)
│   ├── schema <kind>                # 已有
│   ├── frontmatter                  # 已有
│   ├── eval-queries                 # 已有
│   ├── product-assessment           # 已有
│   ├── gcl-conformance              # 新增
│   ├── alarm-wire-contract          # 新增
│   ├── safety-class                 # 新增
│   ├── resource-scope               # 新增
│   └── generator-contract           # 新增
├── check
│   ├── example-config               # 已有
│   ├── markdown-links               # 已有
│   ├── references-links             # 已有
│   ├── advanced-coverage            # 已有
│   └── audit-results                # 新增
├── scan secret ...                  # 已有
├── aggregate trace                  # 已有
├── lint go                          # 已有
└── gcl                              # 新增命名空间
    ├── run                          # GCL 执行循环
    └── alarm-wire                   # CES 告警编排
```

## 3. 功能契约 (Functional Contract)

### 3.1 `validate gcl-conformance` — GCL 合规检查

检查每个 `huaweicloud-*-ops` skill 是否包含 GCL 必备文件：

```
skillcheck validate gcl-conformance --root <dir>
```

**检查项**：
- `references/rubric.md` — 包含 `## 1.` 到 `## 8.` 编号章节
- `references/prompt-templates.md` — 包含 `## 1.` 到 `## 7.` 编号章节 + `operation_intent` + 无裸 placeholder
- `SKILL.md` — 包含 `## Quality Gate (GCL)` 章节

**输出格式**：
```
GCL conformance: 22/24 skills conform.
  FAIL huaweicloud-dns-ops: rubric_sections=0/8, prompt_sections=0/7
```

**退出码**：0 = 全部通过；1 = 存在不达标的 skill

### 3.2 `validate alarm-wire-contract` — GCL 告警连线契约

验证 CES 的 `gcl_quality` 阈值配置在 example-config.yaml、gcl_spec.md、已存的 alarm-plan 之间一致：

```
skillcheck validate alarm-wire-contract --root <dir>
```

**检查维度**：
1. `huaweicloud-ces-ops/assets/example-config.yaml` → gcl_quality 块完整
2. `docs/gcl-spec.md` → 文档中阈值描述与配置一致
3. `audit-results/gcl-alarm-plan-*.json` → 已存计划与当前配置一致

**阈值常量**（硬编码 Go 中，不依赖 Python 模块）：
```
pass_rate_warn = 0.5
pass_rate_critical = 0.3
max_iter_warn_count = 5
safety_fail_alert = true
```

### 3.3 `validate safety-class` — 安全分类枚举契约

验证 safety_class 枚举在整个管线中一致：

```
skillcheck validate safety-class --root <dir>
```

**检查维度**：
1. **Schema gate** — `gcl-trace.schema.json` 的 safety_class enum 必须为 `read-only|mutating|destructive`
2. **Code gate** — 内联的 sanitizer（Go 版 gcl runner）拒绝非法值，接受合法值
3. **Docs gate** — `docs/gcl-spec.md` 和 `gcl-prompt-backbone.md` 中枚举值完整
4. **Traces gate** — `audit-results/gcl-trace-*.json` 中无非法枚举值

**预期枚举值**：`"read-only"`, `"mutating"`, `"destructive"`

### 3.4 `validate resource-scope` — 资源范围 PII 掩码契约

验证 `operation_intent.resource_scope` 的 PII 掩码一致性：

```
skillcheck validate resource-scope --root <dir>
```

**检查维度**：
1. **Schema gate** — schema 中 resource_scope items 的 anyOf 允许 `***` / `<masked>` / `prefix-***`
2. **Code gate** — Go 版 `maskResourceID()` 函数对已知 ID 格式正确掩码
3. **Runner gate** — `masked_fields` 包含 `operation_intent`
4. **Traces gate** — 已存 trace 中无裸 ID

### 3.5 `validate generator-contract` — Generator 模板契约

验证 `huaweicloud-skill-generator` 的 GCL 模板包含所有必需要素：

```
skillcheck validate generator-contract --root <dir>
```

**检查项**（参考 Python 原版 `check_generator_contract.py` 的 ~18 条 regex 契约——具体 regex 列表见该文件的 `CONTRACTS` 常量）：
- `huaweicloud-skill-template.md` — GCL metadata、rubric artifact、prompt templates artifact、operation_intent
- `huaweicloud-skill-generator/SKILL.md` — 引用 backbone、rubric、prompt-templates
- `gcl-prompt-backbone.md` — Generator/Critic/Orchestrator 章节、hcloud primary、Go SDK fallback、Critic read-only 约束

### 3.6 `check audit-results` — Audit Results 目录保护

验证 `audit-results/` 目录的 gitignore 和权限契约：

```
skillcheck check audit-results --root <dir>
```

**检查维度**：
1. `.gitignore` — 包含 8 个必选 pattern（audit-results/、gcl-trace-*.json、gcl-quality-summary-*.json、gcl-alarm-plan-*.json，各含 `**/` 变体）
2. **目录权限** — `audit-results/` 存在时 mode 必须为 0700
3. **Tracked files** — `git ls-files audit-results/` 为空
4. **文档** — `docs/gcl-spec.md` 包含 audit persistence 策略

**输出格式**：
```
skillcheck check audit-results --root <dir>
```
文本模式：OK / FAIL + 详细信息。
支持 `--json` 输出 JSON 报告，格式为：
```json
{"ok": true/false, "gitignore_patterns": {"found": 8, "missing": 0}, "mode_ok": true, "tracked_files": 0}
```

### 3.7 `gcl run` — GCL 执行循环

```
skillcheck gcl run \
  --skill <skill-name> \
  --request "<human-readable request>" \
  --operation-intent '<json>' \
  --command '<hcloud-command>' \
  [--max-iter <N>] \
  [--critic-json <path>] \
  [--structural-critic-only]
```

**功能**：
1. 执行 Generator 命令（`hcloud` / shell）
2. 收集 stdout/stderr/exit code
3. 调用 Critic 评分（外部 JSON 或内部结构批评器）
4. 循环修复（最多 max_iter 轮）
5. 写 trace 到 `audit-results/gcl-trace-YYYYMMDD-HHMMSS.json`

**退出码**：0 = PASS/MAX_ITER；非 0 = 执行失败（SAFETY_FAIL 也返回非 0）；TIMEOUT 返回退出码 124（与 `timeout` 命令一致）

### 3.8 `gcl alarm-wire` — CES 告警编排

```
skillcheck gcl alarm-wire plan --summary path/to/summary.json [--write-plan]
skillcheck gcl alarm-wire apply --summary path/to/summary.json [--dry-run]
```

**功能**：
1. `plan` — 只读，从 quality summary 生成 CES alarm plan（JSON）
2. `apply` — 调用 hcloud CLI 创建/更新 CES 告警规则
3. `--dry-run` — 打印计划但不执行

### 3.9 `make install-hook` — Git Hook 安装器

```
make install-hook   # 安装 git pre-commit hook
make check-hook     # 检查是否已安装
make uninstall-hook  # 移除
```

用根目录下的 Go 程序 `scripts/install_hook.go` 替代 `python3 scripts/install_git_hook.py`（注意：`scripts/install_hook.go` 是独立 Go 工具，不是 skillcheck 二进制的一部分）。

**pre-commit hook 触发条件更新**：从检测 `scripts/*.py` 改为检测 `skillcheck/**/*.go` + `skillcheck/testdata/*.py` + `scripts/*.go`。

**根 Makefile 新增 target**（见 §5.2）：
```
make install-hook    # go run scripts/install_hook.go install
make check-hook      # go run scripts/install_hook.go check
make uninstall-hook  # go run scripts/install_hook.go uninstall
```

## 4. 异常与边界 (Edge Cases)

| 场景 | 行为 |
|------|------|
| `--root` 下无 `huaweicloud-*-ops` 目录 | 跳过相关检查，warn 而非 fail |
| `audit-results/` 不存在 | 不报错（CI 上无 trace 是正常状态） |
| `gcl run` 命令超时 | 截断，标记为 TIMEOUT，保留部分输出，退出码 124 |
| `gcl run` Critic JSON 格式错误 | 报错退出（退出码 2），不执行 Generator |
| git 工作树非 git checkout | `check audit-results` 跳过 tracked_files 检查 |
| 批量 `gcl run` | 暂不支持并发执行，单次只处理一个单元 |

## 5. 架构 (Architecture)

### 5.1 新增 Go 文件

```
skillcheck/
├── cmd/
│   ├── validate.go              # 新增 validate_gcl.go (gcl-conformance, alarm-wire-contract)
│   │                            # 新增 validate_contract.go (safety-class, resource-scope, generator-contract)
│   └── check.go                # check.go + audit-results (check.go 已有，追加)
│   └── gcl_run.go              # 新增: gcl run 子命令
│   └── gcl_alarm_wire.go       # 新增: gcl alarm-wire 子命令
├── internal/
│   ├── gcl/
│   │   ├── runner.go            # GCL 循环核心逻辑
│   │   ├── runner_test.go
│   │   ├── alarm_wire.go        # CES 告警编排逻辑
│   │   ├── alarm_wire_test.go
│   │   ├── sanitizer.go         # sanitize_operation_intent + mask_resource_id
│   │   └── sanitizer_test.go
│   ├── embed/                   # 已有
│   ├── schema/                  # 已有
│   ├── security/                # 已有
│   ├── yaml/                    # 已有
│   └── coverage/                # 已有
scripts/
└── install_hook.go              # 新增: Go 版 git hook 安装器
```

### 5.2 外部依赖

- **新增依赖**：无（全部 Go 标准库）
- **子进程调用**：
  - `gcl run` → `os/exec` 执行 `hcloud` / shell 命令
  - `gcl alarm-wire apply` → `os/exec` 执行 `hcloud ces` 命令
  - `check audit-results` → `os/exec` 执行 `git ls-files`

## 6. 验收测试 (Acceptance Tests)

### 6.1 等价性测试

对每个迁移的子命令，创建对照测试用例：

| 子命令 | Python 原版 | 输入 | 验证点 |
|--------|-----------|------|--------|
| `validate gcl-conformance` | `check_gcl_conformance.py` | 仓库根目录 | 退出码 + FAIL 列表一致 |
| `validate alarm-wire-contract` | `check_gcl_alarm_wire_contract.py` | 仓库根目录 | 退出码 + 错误信息一致 |
| `validate safety-class` | `check_safety_class_enum.py` | 仓库根目录 | 退出码一致 |
| `validate resource-scope` | `check_resource_scope_pii.py` | 仓库根目录 | 退出码一致 |
| `validate generator-contract` | `check_generator_contract.py` | 仓库根目录 | 退出码一致 |
| `check audit-results` | `check_audit_results_guard.py` | 仓库根目录 | 退出码一致 |

### 6.2 单元测试 (TDD)

每个 internal 包必须：

- `sanitizer_test.go` — 覆盖合法/非法 safety_class、各 ID 格式掩码
- `runner_test.go` — Generator 执行、Critic 评分聚合、trace 写入
- `alarm_wire_test.go` — plan 生成、阈值验证

### 6.3 集成测试

- `gcl run --structural-critic-only` 在已知好命令上 PASS
- `gcl alarm-wire plan` 在 fixture 上输出有效 JSON

## 7. 迁移顺序

```
Layer 1: 共享库
  └── sanitizer.go (safety_class + mask_resource_id)
  └── runner.go (GCL 循环核心)
  └── alarm_wire.go (CES 告警编排)

Layer 2: 简单校验 (TDD)
  └── validate gcl-conformance
  └── validate generator-contract
  └── check audit-results
  └── validate alarm-wire-contract

Layer 3: 依赖共享库的校验
  └── validate safety-class (依赖 sanitizer.go)
  └── validate resource-scope (依赖 sanitizer.go)

Layer 4: 运行时命令
  └── gcl run (依赖 runner.go)
  └── gcl alarm-wire (依赖 alarm_wire.go)

Layer 5: 基础设施
  └── scripts/install_hook.go
  └── pre-commit hook 更新
  └── delete obsolete Python files
```
