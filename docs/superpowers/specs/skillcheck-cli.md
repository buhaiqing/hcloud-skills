# Spec: skillcheck — 跨平台单二进制 Skill 校验 CLI

> Status: ✅ FINAL_SPEC (self-critique 3 轮闭环)
> Last updated: 2026-07-18
> Author: orchestrator (per user request)

## 1. 背景与目标 (Why)

当前 hcloud-skills 仓库的质量校验体系已从 Python 迁移至 Go。原 Python 脚本（约 5000 行 + 测试 3650 行）已全部删除，由 `skillcheck` Go 二进制替代。
它面向外部用户：**从 Git 下载即用的单文件二进制**，无需 Python 解释器 / 构建环境，
即可校验自己的 hcloud-skill 仓库。

**目标产品**：`skillcheck` —— 一个用 Go 编写、GitHub Action 编译多平台制品、对外分发的单二进制 CLI。
外部用户下载后放入 PATH，执行 `skillcheck validate --root ./my-skills` 即可完成校验。

### 1.1 验收成功标准 (Definition of Done)
- [ ] 提供 `skillcheck` 单二进制，支持 `darwin-arm64` / `linux-amd64` / `windows-amd64` 三平台。
- [ ] 外部用户下载二进制后，**无需任何 Python/解释器/构建环境**，仅依赖二进制本身即可运行。
- [ ] 所有 schema / fixtures 通过 Go `embed` 内嵌进二进制，零外部文件依赖。
- [ ] 覆盖 Spec §4 定义的 A 类通用校验全部子命令，行为与现有 Python 脚本**结构化等价**（同一输入同退出码、同失败项集合；路径/时间戳差异不计入）。
- [ ] GitHub Action 在 tag push 时自动编译并发布多平台 release artifact。
- [ ] `skillcheck validate --root <外部仓库>` 能正确校验一个不含本仓库硬编码结构的独立 skill 仓库。

## 2. 范围 (Scope)

### 2.1 IN — A 类：通用校验（对外分发核心，约 18 个脚本）
这些脚本接受 `--root` 参数、逻辑与"被校验仓库路径"解耦，外部用户仓库可直接复用：

| 子命令 | 功能 |
|---|---|
| `validate schema trace` | 校验 GCL trace JSON schema |
| `validate schema summary` | 校验 GCL quality summary schema |
| `validate schema alarm-plan` | 校验 CES alarm plan schema |
| `validate eval-queries` | 校验 assets/eval_queries.json |
| `validate frontmatter` | 校验 SKILL.md YAML frontmatter |
| `validate product-assessment` | 校验 well-architected worker JSON |
| `check example-config` | 校验 assets/example-config.yaml |
| `check markdown-links` | 校验本地 markdown 链接 |
| `check references-links` | 校验 references/ 深链健康 |
| `check advanced-coverage` | 校验 TE-7 advanced/ 覆盖 |
| `aggregate trace` | 聚合 trace → quality summary |
| `scan secret trace` | 扫描 trace 凭据泄露 |
| `scan secret summary` | 扫描 summary 凭据泄露 |
| `scan secret alarm-plan` | 扫描 alarm plan 凭据泄露 |
| `scan secret shared` | 共享凭据扫描器 |
| `validate`（总入口，默认） | 一键跑全部 A 类检查；`--root` 默认 = 当前工作目录 |

### 2.2 OUT — B 类：本仓库专属逻辑（部分待迁移）

以下脚本硬编码了本仓库结构（20 个 skill 名列表、`huaweicloud-ces-ops` 路径、dual-copy drift 等），
外部用户仓库无对应内容。**其中 6 个已规划迁移到 skillcheck**（见 B 类迁移 spec），其余保留 Python：

| 脚本 | 状态 | 迁移计划 |
|------|------|---------|
| `check_gcl_conformance.py` | 待迁移 | 可做 `validate gcl-conformance` |
| `check_gcl_alarm_wire_contract.py` | 待迁移 | 可做 `validate alarm-wire-contract` |
| `check_safety_class_enum.py` | 待迁移 | 可做 `validate safety-class` |
| `check_resource_scope_pii.py` | 待迁移 | 可做 `validate resource-scope` |
| `check_generator_contract.py` | 待迁移 | 可做 `validate generator-contract` |
| `check_audit_results_guard.py` | 待迁移 | 可做 `check audit-results` |
| `check_skill_generator_drift.py` | 保留 Python | 仓库维护工具，不适合 Go |
| `check_py310_compat.py` | 保留 Python | Python 运行时检查，Go 无意义 |
| `gcl_runner.py` | 保留 Python | 运行时 GCL 循环，需执行 hcloud CLI |
| `gcl_alarm_wire.py` | 保留 Python | 运行时 CES 告警联动 |

### 2.3 OUT — 运行时 GCL 循环（不在本 Spec）
`gcl_runner.py` / `gcl_alarm_wire.py` 属 agent 运行时调用（调 `hcloud`/`git`），本 Spec 聚焦"静态校验分发"，
不在 Phase 1 范围。如后续需要，单独 Spec。

### 2.4 ruff lint 处理
`run_ruff.sh`（Python lint）改用 Go 侧 `golangci-lint` 类思路替代（语义不完全等价，作为独立子命令
`lint go`，不阻断 A 类校验）。Python 文件的 ruff 检查不在二进制职责内。

## 3. 架构 (Architecture)

```
skillcheck/                      # 新 Go module (module path 待 Plan 阶段按实际 repo 确定, 如 github.com/<owner>/hcloud-skills/skillcheck)
├── go.mod
├── main.go                      # 子命令路由: root → validate / check / scan / aggregate / lint
├── internal/
│   ├── embed/                   # //go:embed fixtures/ schemas/ 内嵌资源
│   │   ├── fixtures/            # 从 scripts/fixtures 迁移: gcl-alarm-plan-healthy.json,
│   │   │                         #   gcl-quality-summary-healthy.json, gcl-trace-healthy.json
│   │   │                         #   (仅迁移 A 类扫描/聚合用到的健康样例, 详见 Plan §embed 清单)
│   │   └── schemas/             # JSON schema 定义 (trace/summary/alarm-plan/eval-queries)
│   ├── schema/                  # JSON schema subset 校验器 (替代 json_schema_subset.py)
│   ├── security/                # 共享凭据扫描器 (替代 gcl_security_scan.py)
│   ├── yaml/                    # YAML frontmatter/config 解析 (gopkg.in/yaml.v3)
│   ├── markdown/                # 链接/深链健康检查
│   └── coverage/                # TE-7 advanced coverage 检查
└── cmd/                         # 各子命令实现
    ├── validate.go
    ├── check.go
    ├── scan.go
    ├── aggregate.go
    └── lint.go
```

### 3.1 唯一外部依赖
- `gopkg.in/yaml.v3`：解析 YAML frontmatter / example-config（替代原脚本手写正则）。
- 其余全用 Go 标准库（`encoding/json`、`regexp`、`os`、`path/filepath`、`embed`）。

### 3.2 分发形态
- GitHub Action（`build-release.yml`）：tag push 触发，matrix `GOOS/GOARCH` 编译，
  产物 `skillcheck-<os>-<arch>` 上传至 GitHub Release。
- 外部用户：`curl`/Release 页下载 → `chmod +x` → 放入 PATH → 可用。
- fixtures/schemas 经 `//go:embed` 编译进二进制，运行时无外部文件读取。

## 4. 功能契约 (Functional Contract)

每个子命令的输入/输出契约对齐现有 Python 脚本（以 `--root` 指向被校验仓库）：
- **默认入口**：`skillcheck validate --root <dir>` 为默认总入口，依次跑全部 A 类检查并汇总。
  细粒度子命令（`check`/`scan`/`aggregate`）供调试与 CI 单步使用。
- **`--root` 默认值**：当前工作目录（cwd）。外部用户二进制不在仓库内，不再默认 `parents[1]`。
- **退出码**：0 = 全部通过；非 0 = 存在失败项（与 Python 脚本一致）。
- **输出格式**：人类可读的 `[OK]` / `[FAIL]` 行 + 末尾汇总；`--json` 输出机器可读 JSON（对齐现有 `--json` 参数）。
- **凭据遮蔽**：扫描类命令输出中凭据一律 `***` / `<masked>`（继承 AGENTS.md 安全规则）。
- **aggregate 无输入**：当 `--root` 下无 `audit-results/gcl-trace-*.json` 时，aggregate 子命令 warn 跳过，不 fail（trace 由运行时 runner 产生，外部用户可能未跑）。
- **自校验**：扫描类命令提供 `--self-check` 开关，对 embed 内 fixture 执行扫描，验证二进制自身健康。

## 5. 异常与边界 (Edge Cases)

| 场景 | 行为 |
|---|---|
| `--root` 指向不存在路径 | 报错退出，码 2，提示路径无效 |
| 被校验仓库无 `huaweicloud-*-ops` 目录 | 跳过相关检查，warn 而非 fail |
| YAML 解析失败 | 报告具体文件+行号，fail |
| 大仓库（100+ skill） | 顺序遍历，O(n)，无 N+1；内存复用 |
| Windows 路径分隔符 | 用 `path/filepath` 跨平台处理 |
| embed 资源缺失（编译期） | `go:embed` 编译失败，CI 阻断 |

## 6. 验收测试 (Acceptance Tests)

1. **等价性测试**：对同一 fixtures 输入，Python 脚本 vs `skillcheck` 的**退出码与失败项集合**一致（路径/时间戳差异归一化后比对，不要求逐字节）。
2. **零依赖测试**：干净容器（无 python3）中运行 `skillcheck`，全部子命令可用。
3. **跨平台测试**：三平台二进制在对应 CI runner 上 smoke 跑通。
4. **外部仓库测试**：用 `skillcheck validate --root <独立 skill 仓库>` 校验非本仓库结构，正确通过。
5. **自校验测试**：`skillcheck scan secret <x> --self-check` 对 embed fixture 扫描，无凭据泄露报告。
