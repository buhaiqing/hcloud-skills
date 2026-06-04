---
name: huaweicloud-hss-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud HSS (Host Security Service) — host management, asset collection, intrusion
  detection, vulnerability management, baseline checks, web tamper protection,
  container security, and alarm events. User mentions HSS, Host Security Service,
  主机安全, 服务器安全, 漏洞扫描, 入侵检测, 基线检查, 网页防篡改, or describes
  server security scenarios (e.g., server compromised, malware detected,
  vulnerability found, baseline failure, ransomware protection) even without
  naming the product directly. Not for billing, IAM, or related products that
  have their own ops skills.
license: MIT
compatibility: >-
  Official Huawei Cloud CLI (`hcloud`), Go 1.21+ runtime
  (for JIT SDK fallback via huaweicloud-sdk-go-v3), valid API credentials,
  network access to Huawei Cloud HSS endpoints.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-05-21"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "HSS v5 — https://support.huaweicloud.com/api-hss/"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    HSS operations available via `hcloud HSS <operation>` where
    operation matches the API Explorer name: ListHosts, ShowHostDetail,
    ListAlarmEvents, HandleAlarmEvent, ListVulnerabilities, HandleVulnerability,
    ListBaselineResults, ExecuteBaselineCheck, ListHostGroups, ListContainerNodes,
    etc.
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
  gcl:
    enabled: true
    required: true
    rubric_version: "v1"
    max_iter: 2
    rubric_ref: "references/rubric.md"
    prompts_ref: "references/prompt-templates.md"
    trace_dir: "./audit-results/"
    changelog:
      - version: "1.1.0"
        date: "2026-06-04"
        change: "GCL Phase 2 rollout: added references/rubric.md (v1, 5-dim, S1–S17 HSS-specific Safety rules, including version downgrade / prod process kill / system process / private IP block / false-positive-ignore guards) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains '质量门 (GCL)' chapter."
      - version: "1.0.0"
        date: "2026-05-21"
        change: "Initial skill release."
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud HSS (Host Security Service) Operations Skill

## Overview

Host Security Service (HSS) provides server security including intrusion detection, vulnerability management, baseline checks, web tamper protection, and container security. This skill is an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official `hcloud` CLI and JIT Go SDK fallback), response validation, and failure recovery.

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** Official CLI supports HSS. In each execution flow below, document **both** the SDK step **and** the CLI step.

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions below with delegation rules |
| 2 | **Structured I/O** | `{{env.*}}` for credentials, `{{user.*}}` for interactive input, `{{output.*}}` for response capture |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute (CLI + SDK) → Validate → Recover |
| 4 | **Complete Failure Strategies** | 14 HSS error codes documented; HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | One product (HSS); cross-product delegation to other skills |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **FinOps** | Version tier optimization, host grading for cost efficiency | `references/cost-optimization.md` |
| **SecOps** | Core HSS function — intrusion detection, vulnerability, baseline, WTP | `references/security-best-practices.md` |
| **AIOps** | 4 anomaly patterns (alert storm, unhandled backlog, vuln fix progress, host health) | `references/aiops-patterns.md` |

### Well-Architected Framework Integration (卓越架构)

| Pillar | Skill Integration |
|--------|-------------------|
| **安全 (Security)** | Core HSS function — host intrusion detection, vulnerability scanning, baseline checks |
| **稳定 (Stability)** | Multi-layer protection (HSS + WAF + Anti-DDoS), agent health monitoring |
| **成本 (Cost)** | Version tier selection (basic/enterprise/ultimate), host protection optimization |
| **效率 (Efficiency)** | Batch vulnerability handling, automated baseline checks, CI/CD security gates |
| **性能 (Performance)** | Agent CPU/memory footprint monitoring, scan scheduling optimization |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud HSS" / "Host Security Service" / "主机安全服务" / "服务器安全"
- Task involves host management (list hosts, show details, switch protection version, manage host groups)
- Task involves asset collection (accounts, ports, processes, software, middlewares, apps)
- Task involves intrusion detection (list/handle alarm events, blocked IPs, isolated files)
- Task involves vulnerability management (list/handle vulnerabilities, create scan tasks)
- Task involves baseline checks (list results, policies, execute checks)
- Task involves web tamper protection (WTP — list, create, delete protected directories)
- Task involves container security (nodes, images, pods, container alarms)
- Task keywords: **HSS**, **主机**, **服务器**, **漏洞**, **入侵**, **基线**, **防篡改**, **告警**, **容器安全**
- User describes symptoms: server compromised, malware detected, vulnerability found, baseline failure, ransomware detected

### SHOULD NOT Use This Skill When

- Task is purely billing / account management → delegate to: billing skill (when present)
- Task is IAM / permission model only → delegate to: `huaweicloud-iam-ops`
- Task is about WAF (web application firewall) → delegate to: `huaweicloud-waf-ops`
- Task is about Anti-DDoS → delegate to: `huaweicloud-antiddos-ops` (when present)
- Task is about ECS instance lifecycle → delegate to: `huaweicloud-ecs-ops`
- Task is about VPC/subnet/security group → delegate to: `huaweicloud-vpc-ops`

### Delegation Rules

- HSS agent requires ECS instance → verify instance exists via `huaweicloud-ecs-ops` before protection switch
- Web tamper protection may require WAF coordination → check WAF domain configuration
- Container security requires CCE cluster → verify cluster via `huaweicloud-cce-ops`
- For SecOps questions: use this skill's security section; delegate account-level IAM to `huaweicloud-iam-ops`

## Variables

| Variable | Source | Description | Example |
|----------|--------|-------------|---------|
| `{{env.HW_ACCESS_KEY_ID}}` | Environment | Huawei Cloud AK | `AKIA...` |
| `{{env.HW_SECRET_ACCESS_KEY}}` | Environment | Huawei Cloud SK | `***` (masked) |
| `{{env.HW_REGION_ID}}` | Environment | Region code | `ap-southeast-1` |
| `{{env.HW_PROJECT_ID}}` | Environment | Project ID | `a1b2c3d4...` |
| `{{user.host_id}}` | User | HSS host UUID | `host-abc123` |
| `{{user.host_name}}` | User | Host name | `prod-server-01` |
| `{{user.event_id}}` | User | Alarm event UUID | `event-xyz789` |
| `{{user.vulnerability_id}}` | User | Vulnerability UUID | `vuln-def456` |
| `{{user.handle_type}}` | User | Event/vuln handling action | `isolate`, `block_ip`, `ignore`, `mark_handled` |
| `{{output.host_id}}` | API Response | Host ID | from `ListHosts` |
| `{{output.event_id}}` | API Response | Event ID | from `ListAlarmEvents` |

> **Security Warning:** NEVER log or expose `{{env.HW_SECRET_ACCESS_KEY}}` or any credential values.

---

## 场景

### 服务器资产管理
- 查询主机列表、主机详情、服务器组
- 查询账户、端口、进程、软件、中间件等资产清单
- 查看资产变更历史

### 入侵检测与告警事件处理
- 查看安全告警事件列表与详情（文件入侵/进程入侵/Rootkit/异常Shell/恶意程序等）
- 处理告警事件：隔离/终止进程/加入白名单
- 查询已封锁IP和已隔离文件
- 解封IP、恢复隔离文件

### 漏洞管理
- 查看漏洞列表与详情（Linux/Windows/Web-CMS）
- 创建漏洞扫描任务
- 处理漏洞（修复/忽略/验证）

### 基线检查
- 查看基线检查结果（配置检查/弱口令/风险配置）
- 查看基线策略与规则

### 防护管理
- 切换主机防护版本（企业版/旗舰版/网页防篡改版）
- 管理防护策略组
- 管理网页防篡改保护

### 容器安全
- 查看容器节点、容器镜像、容器Pod
- 查看容器风险与告警

## 前置条件

### CLI 模式
```bash
# 已安装并配置 hcloud CLI
hcloud configure
# 可用 region
export HW_CLOUD_REGION=ap-southeast-1
# 目标项目 ID
export HW_PROJECT_ID={{env.HW_PROJECT_ID}}
```

### Go SDK 模式（JIT 回退）
```bash
go get github.com/huaweicloud/huaweicloud-sdk-go-v3
```

## 操作手册

### 主机管理

#### 查询主机列表
```bash
hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

参数说明：
- `enterprise_project_id` — 企业项目 ID，可选
- `host_name` — 主机名称模糊匹配，可选
- `os_type` — 操作系统类型（Linux/Windows），可选
- `host_status` — 主机状态，可选

#### 查询主机详情
```bash
hcloud HSS ShowHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --host_id="<hostId>"
```

#### 切换主机防护版本
```bash
hcloud HSS SwitchHostsProtectStatus --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"version\":\"hss.version.enterprise\",\"charging_mode\":\"on_demand\",\"resource_id\":\"<resourceId>\",\"host_id_list\":[\"<hostId>\"]}"
```

参数说明：
- `version` — 版本：`hss.version.basic`（基础版）、`hss.version.enterprise`（企业版）、`hss.version.premium`（旗舰版）、`hss.version.wtp`（网页防篡改版），必填
- `charging_mode` — 计费模式：`prePaid`（包年包月）、`on_demand`（按需），必填
- `host_id_list` — 主机 ID 列表，必填

#### 查询服务器组
```bash
hcloud HSS ListHostGroups --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

### 资产管理

#### 查询账户列表
```bash
hcloud HSS ListAccounts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 查询开放端口统计
```bash
hcloud HSS ListPorts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 查询进程列表
```bash
hcloud HSS ListProcesses --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 查询软件列表
```bash
hcloud HSS ListApps --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 查询自启动项列表
```bash
hcloud HSS ListAutoLaunchs --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 查询中间件列表
```bash
hcloud HSS ListMiddlewares --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 查询资产统计
```bash
hcloud HSS ShowAssetStatistic --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

### 入侵检测与告警事件

#### 查询告警事件列表
```bash
hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

按事件类型筛选：
```bash
hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --event_types="[\"malware\",\"ransomware\"]" --handle_status="unhandled"
```

参数说明：
- `event_types` — 事件类型数组，可选（malware/ransomware/process/file等）
- `handle_status` — 处理状态：`unhandled`/`handled`，可选
- `host_name` — 主机名筛选，可选
- `begin_time` / `end_time` — 时间范围，可选

#### 处理告警事件
```bash
# 隔离并终止进程
hcloud HSS OperateEvent --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"operate_type\":\"isolate_and_kill\",\"event_id_list\":[\"<eventId>\"],\"operate_detail_list\":[{\"agent_id\":\"<agentId>\",\"file_hash\":\"<fileHash>\",\"file_path\":\"/tmp/malware\",\"process_pid\":1234}]}"

# 加入告警白名单
hcloud HSS OperateEvent --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"operate_type\":\"add_to_alarm_whitelist\",\"event_id_list\":[\"<eventId>\"],\"operate_detail_list\":[{\"keyword\":\"<keyword>\",\"hash\":\"<hash>\"}]}"

# 加入登录白名单
hcloud HSS OperateEvent --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"operate_type\":\"add_to_login_whitelist\",\"event_id_list\":[\"<eventId>\"],\"operate_detail_list\":[{\"login_ip\":\"10.0.0.1\",\"private_ip\":\"192.168.1.1\",\"login_user_name\":\"root\"}]}"
```

操作类型（`operate_type`）说明：
| 值 | 说明 |
|------|------|
| `mark_as_handled` | 标记为已处理 |
| `mark_as_unhandled` | 标记为未处理 |
| `isolate_and_kill` | 隔离并终止进程 |
| `do_not_isolate_or_kill` | 仅告警不做隔离 |
| `add_to_alarm_whitelist` | 加入告警白名单 |
| `remove_from_alarm_whitelist` | 从告警白名单移除 |
| `add_to_login_whitelist` | 加入登录白名单 |
| `remove_from_login_whitelist` | 从登录白名单移除 |

#### 查询已封锁 IP
```bash
hcloud HSS ListBlockedIp --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 解封已封锁 IP
```bash
hcloud HSS ChangeBlockedIp --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"data_list\":[{\"block_ip\":\"x.x.x.x\",\"host_id\":\"<hostId>\"}],\"operate_type\":\"unblock\"}"
```

#### 查询已隔离文件
```bash
hcloud HSS ListIsolatedFile --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 恢复隔离文件
```bash
hcloud HSS ChangeIsolatedFile --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"data_list\":[{\"host_id\":\"<hostId>\",\"file_hash\":\"<fileHash>\"}],\"operate_type\":\"restore\"}"
```

#### 删除隔离文件记录
```bash
hcloud HSS DeleteIsolatedFile --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"data_list\":[{\"host_id\":\"<hostId>\",\"file_hash\":\"<fileHash>\"}]}"
```

### 漏洞管理

#### 查询漏洞列表
```bash
hcloud HSS ListVulnerabilities --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

按类型和严重程度筛选：
```bash
hcloud HSS ListVulnerabilities --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --type="linux" --severity="critical"
```

参数说明：
- `type` — 漏洞类型：`linux`/`windows`/`web_cms`，可选
- `severity` — 严重级别：`critical`/`high`/`medium`/`low`，可选
- `handle_status` — 处理状态：`unhandled`/`handled`，可选

#### 创建漏洞扫描任务
```bash
hcloud HSS CreateScanTask --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"task_name\":\"manual-scan-$(date +%Y%m%d)\",\"host_id_list\":[\"<hostId>\"],\"type\":\"linux\"}"
```

#### 处理漏洞
```bash
# 标记漏洞为已修复
hcloud HSS ChangeVulStatus --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"operate_type\":\"repair\",\"data\":[{\"vul_id\":\"<vulId>\",\"host_id_list\":[\"<hostId>\"]}]}"

# 忽略漏洞
hcloud HSS ChangeVulStatus --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"operate_type\":\"ignore\",\"data\":[{\"vul_id\":\"<vulId>\",\"host_id_list\":[\"<hostId>\"]}]}"
```

### 基线检查

#### 查询基线检查结果
```bash
hcloud HSS ListBaselineCheckResults --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 查询基线策略
```bash
hcloud HSS ListBaselinePolicies --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 执行基线检查
```bash
hcloud HSS CreateBaselineCheckTask --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"host_id_list\":[\"<hostId>\"],\"baseline_id_list\":[\"<baselineId>\"]}"
```

### 网页防篡改

#### 查询防篡改保护状态
```bash
hcloud HSS ListWtpProtection --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 创建防篡改保护
```bash
hcloud HSS CreateWtpProtection --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"host_id\":\"<hostId>\",\"policy_name\":\"web-protection-policy\",\"protected_directory\":\"/var/www/html\",\"backup_directory\":\"/var/backup/html\"}"
```

### Agent 管理

#### 查询 Agent 安装状态
```bash
hcloud HSS ListAgents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

### 安全报告与仪表盘

#### 查询 Dashboard 安全概览
```bash
hcloud HSS ShowDashboard --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 导出安全报告
```bash
hcloud HSS ExportSecurityReport --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"report_name\":\"security-report-$(date +%Y%m%d)\",\"report_type\":\"weekly\"}"
```

### Go SDK 快速参考

```go
package main

import (
    "fmt"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/hss/v5"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/hss/v5/model"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/hss/v5/region"
)

func main() {
    auth, _ := basic.NewCredentialsBuilder().
        WithAk("{{env.HW_ACCESS_KEY}}").
        WithSk("{{env.HW_SECRET_KEY}}").
        WithProjectId("{{env.HW_PROJECT_ID}}").
        Build()

    client := hss.NewHssClient(
        hss.HssClientBuilder().
            WithRegion(region.ValueOf("{{env.HW_CLOUD_REGION}}")).
            WithCredential(auth).
            Build())

    // 查询主机列表
    resp, err := client.ListHosts(&model.ListHostsRequest{})
    if err != nil {
        fmt.Printf("[ERROR] %v", err)
        return
    }
    for _, h := range *resp.DataList {
        fmt.Printf("Host: %s (%s) status=%s\n", *h.HostName, *h.HostId, *h.HostStatus)
    }
}
```

## 质量门 (GCL)

本 skill 强制 GCL(参见 `AGENTS.md` §8)。所有 HSS 变更操作(主机防护版本切换、告警事件处理、隔离文件恢复/删除、基线策略增删改、网页防篡改策略增删改、漏洞处理)均需经过 **Generator-Critic-Loop** 后才能返回结果。只读操作为 GCL-豁免。

| 字段 | 值 |
|------|-----|
| Rubric 版本 | v1 (Phase 2, 2026-06-04) |
| `max_iter` | **2** |
| Rubric 实例 | [`references/rubric.md`](references/rubric.md) |
| Prompt 模板 | [`references/prompt-templates.md`](references/prompt-templates.md) |
| Trace 路径 | `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` |
| 独立性 | Generator 与 Critic 必须在 **隔离** 的子代理/会话中运行 |

### 五维评分(摘要)

| # | 维度 | 阈值 | 备注 |
|---|------|------|------|
| 1 | Correctness | ≥ 0.5(`switch-protect-status` / `isolate_and_kill` / `delete-policy` 要求 = 1) | `ShowHost` / `ShowAlarmEvent` / `ShowPolicy` post-state |
| 2 | Safety | **= 1**(任一 S-rule 命中 → ABORT) | rubric §2 中 S1–S17 |
| 3 | Idempotency | ≥ 0.5 | 创建前先检查 |
| 4 | Traceability | ≥ 0.5 | `password` / 文件 hash 必须为 `<masked>` |
| 5 | Spec Compliance | ≥ 0.5 | HSS 版本 (basic/enterprise/premium/wtp) / 告警严重度 |

### 每操作 Safety 锚点(强制)

- **S1 / S2 / S3** — `switch-protect-status` 降级(premium/enterprise → basic)/ prePaid 退款 / 存在未关告警
- **S4 / S5** — `isolate_and_kill` 生产主机 / 系统进程 (systemd / kubelet / dockerd)
- **S6** — `block_ip` 私网 IP
- **S7 / S17** — 标记 critical 事件 / 低置信度检测为 ignore
- **S8 / S9** — `delete-isolated-file`(证据销毁)/ `recover-isolated-file`(恢复恶意文件)需二次确认
- **S10 / S11** — `delete-baseline-policy` / `delete-web-tamper-policy` 仍有绑定
- **S12** — `ignore-vulnerability` critical CVE 需二次确认
- **S13** — `fix-vulnerability` 生产主机重启需维护窗口
- **S16** — `update-baseline-policy` 关闭 `auto_check`

### 终止契约(参见 `AGENTS.md` §5)

| 条件 | 状态 | 返回 |
|------|------|------|
| 所有维度达标 | **PASS** | Generator 结果 + 分数 + trace 路径 |
| `iter == max_iter` (2) 且仍有维度未达标 | **MAX_ITER** | 当前最佳结果 + 未达标清单 |
| `Safety == 0` | **SAFETY_FAIL** | 违规 S-rule id;**绝不**返回部分结果 |

### Trace 持久化(强制)

每次 GCL 运行写入 `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json`(schema 见
`references/prompt-templates.md` §3)。Trace 追加写,不入 Git;落盘前做脱敏
(参见 `prompt-templates.md` §4)。

### 参见

- [`references/rubric.md`](references/rubric.md) — 完整 rubric、S1–S17 规则、按操作阈值
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator 模板
- 仓库根 [`AGENTS.md`](../../AGENTS.md) §3、§5、§7、§8 — GCL 规范

## 参考文档

| 文档 | 说明 |
|------|------|
| [API 导航](references/api-navigation.md) | HSS v5 API 完整接口列表 |
| [CLI 语法参考](references/cli-syntax-reference.md) | KooCLI 命令参数速查 |
| [常见故障处理](references/common-faults.md) | 常见错误及处理方案 |
| [成本优化](references/cost-optimization.md) | FinOps 费用优化指南 |
| [安全最佳实践](references/security-best-practices.md) | SecOps 安全加固方案 |
| [智能运维](references/aiops-patterns.md) | AIOps 异常检测与自愈 |
| [安全门](references/safety-gates.md) | 高危操作审批流程 |
| [错误处理](references/error-handling.md) | 标准化错误诊断流程 |
