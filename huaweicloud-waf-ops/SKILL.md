---
name: huaweicloud-waf-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud WAF (Web Application Firewall) — policies, rules, domains, certificates,
  attack events, and reference tables. User mentions WAF, Web Application Firewall,
  网站防护, 防火墙, 攻击拦截, CC攻击, SQL注入, XSS, or describes web security
  scenarios (e.g., website under attack, DDoS protection needed, rule tuning,
  certificate upload, attack event analysis) even without naming the product
  directly. Not for billing, IAM, or related products that have their own ops skills.
license: MIT
compatibility: >-
  Official Huawei Cloud CLI (`hcloud`), Go 1.21+ runtime
  (for JIT SDK fallback via huaweicloud-sdk-go-v3), valid API credentials,
  network access to Huawei Cloud WAF endpoints.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-05-21"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "WAF v1 — https://support.huaweicloud.com/api-waf/"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    WAF operations available via `hcloud WAF <operation>` where
    operation matches the API Explorer name: ListPolicies, CreatePolicy,
    ListDomains, CreateDomain, ListRules, CreateRule, ListCertificates,
    CreateCertificate, ListEvents, ListReferenceTables, etc.
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
        change: "GCL Phase 2 rollout: added references/rubric.md (v1, 5-dim, S1–S17 WAF-specific Safety rules, including policy downgrade / proxy disable / rule bypass / system rule / private key leak guards) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
      - version: "1.0.0"
        date: "2026-05-21"
        change: "Initial skill release."
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud WAF (Web Application Firewall) Operations Skill

## Overview

Web Application Firewall (WAF) protects web applications from common web exploits (SQL injection, XSS, CC attacks, bots) and provides custom rule configuration for precise access control. This skill is an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official `hcloud` CLI and JIT Go SDK fallback), response validation, and failure recovery.

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** Official CLI supports WAF. In each execution flow below, document **both** the SDK step **and** the CLI step.

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions below with delegation rules |
| 2 | **Structured I/O** | `{{env.*}}` for credentials, `{{user.*}}` for interactive input, `{{output.*}}` for response capture |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute (CLI + SDK) → Validate → Recover |
| 4 | **Complete Failure Strategies** | 12 WAF error codes documented; HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | One product (WAF); cross-product delegation to other skills |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **FinOps** | Policy/domain count monitoring, rule optimization for efficiency | `references/advanced/cost-optimization.md` |
| **SecOps** | Rule tuning, certificate management, attack response SOP | `references/advanced/security-best-practices.md` |
| **AIOps** | 4 anomaly patterns (attack surge, rule bypass, false positives, certificate expiry) | `references/advanced/aiops-patterns.md` |

### Well-Architected Framework Integration (卓越架构)

| Pillar | Skill Integration |
|--------|-------------------|
| **安全 (Security)** | Core WAF function — web attack protection, custom rules, SSL/TLS |
| **稳定 (Stability)** | Multi-rule redundancy, domain backup configuration |
| **成本 (Cost)** | Rule optimization, event storage management |
| **效率 (Efficiency)** | Batch rule creation, template reuse, CI/CD integration |
| **性能 (Performance)** | Rule evaluation latency, domain routing optimization |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud WAF" / "Web Application Firewall" / "网站防护" / "应用防火墙"
- Task involves policy management (create, list, update, delete policies)
- Task involves domain configuration (add/remove domains, configure source servers, certificates)
- Task involves rule configuration (CC attack, blacklist, whitelist, geo-blocking, anti-crawler)
- Task involves attack event analysis (view events, extract attack patterns, tune rules)
- Task involves certificate management (upload, update, delete SSL/TLS certificates)
- Task keywords: **WAF**, **policy**, **rule**, **domain**, **certificate**, **attack**, **CC**, **SQL注入**, **XSS**, **防护策略**, **域名接入**
- User describes symptoms: website attack, DDoS, bot traffic, SQL injection attempts, XSS attempts

### SHOULD NOT Use This Skill When

- Task is purely billing / account management → delegate to: billing skill (when present)
- Task is IAM / permission model only → delegate to: `huaweicloud-iam-ops`
- Task is about VPC/subnet/security group → delegate to: `huaweicloud-vpc-ops`
- Task is about SSL certificate from SCM → delegate to: `huaweicloud-scm-ops` (when present)
- Task is about monitoring/alarm rules → delegate to: `huaweicloud-ces-ops`
- Task is about DDoS protection (Anti-DDoS) → delegate to: `huaweicloud-antiddos-ops` (when present)

### Delegation Rules

- Domain requires valid certificate → verify certificate exists via `ListCertificates` before domain creation
- Policy requires domain binding → create domain first, then bind to policy
- For SecOps questions: use this skill's security section; delegate account-level IAM to `huaweicloud-iam-ops`
- For attack event investigation: use WAF events; delegate host-level security to `huaweicloud-hss-ops`

## Variables

| Variable | Source | Description | Example |
|----------|--------|-------------|---------|
| `{{env.HW_ACCESS_KEY_ID}}` | Environment | Huawei Cloud AK | `AKIA...` |
| `{{env.HW_SECRET_ACCESS_KEY}}` | Environment | Huawei Cloud SK | `***` (masked) |
| `{{env.HW_REGION_ID}}` | Environment | Region code | `ap-southeast-1` |
| `{{env.HW_PROJECT_ID}}` | Environment | Project ID | `a1b2c3d4...` |
| `{{user.policy_id}}` | User | WAF policy UUID | `policy-abc123` |
| `{{user.policy_name}}` | User | WAF policy name | `prod-waf-policy` |
| `{{user.domain_id}}` | User | Domain UUID | `domain-xyz789` |
| `{{user.domain_name}}` | User | Domain name | `example.com` |
| `{{user.certificate_id}}` | User | Certificate UUID | `cert-def456` |
| `{{user.rule_id}}` | User | Rule UUID | `rule-ghi012` |
| `{{output.policy_id}}` | API Response | Created policy ID | from `CreatePolicy` |
| `{{output.domain_id}}` | API Response | Created domain ID | from `CreateDomain` |

> **Security Warning:** NEVER log or expose `{{env.HW_SECRET_ACCESS_KEY}}` or any credential values.

---

## 场景

### 防护策略管理
- 创建/查询/更新/删除 WAF 防护策略
- 绑定/解绑防护域名到策略
- 调整策略配置（如防护模式、BOT攻击防护）

### 域名与网站接入
- 添加/查询/更新/删除云模式防护域名
- 添加/查询/更新/删除独享模式防护域名
- 为域名配置源站、证书、TLS 版本

### 规则配置
- CC 攻击防护规则（频率限制、人机验证）
- 精准访问防护规则（自定义条件匹配）
- 黑白名单规则（IP/地址组/IPv6）
- 地理位置访问控制规则（国家/地区级别）
- 网页防篡改规则
- 信息防泄漏规则
- 数据防泄露响应规则（隐私屏蔽）
- 反爬虫规则（JavaScript 挑战/基于 Session 的防护）
- 已知攻击源规则
- 全局白名单（误报屏蔽）规则

### 攻击事件分析
- 查看攻击事件列表与详情
- 按时间、域名、攻击类型筛选事件
- 从事件详情中提取攻击特征以优化规则

### 证书管理
- 上传/查询/更新/删除 SSL/TLS 证书
- 将证书应用到防护域名

### 引用表管理
- 创建/查询/更新/删除引用表（Value List）
- 在规则中使用引用表作为匹配条件

## 前置条件

### CLI 模式
```bash
# 已安装并配置 hcloud CLI
hcloud configure
# 可用 region（可覆盖）
export HW_CLOUD_REGION=ap-southeast-1
# 目标项目 ID
export HW_PROJECT_ID={{env.HW_PROJECT_ID}}
```

### Go SDK 模式（JIT 回退）
```bash
go get github.com/huaweicloud/huaweicloud-sdk-go-v3
```

## 操作手册

### 防护策略管理

#### 列出所有防护策略
```bash
hcloud WAF ListPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 创建防护策略
```bash
hcloud WAF CreatePolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --body="{\"name\":\"prod-waf-policy\"}"
```

参数说明：
- `name` — 策略名称，必填，1~64 字符
- `level` — 防护等级（1:宽松 2:中等 3:严格），可选，默认 2
- `full_detection` — 是否开启全检测（true/false），可选

#### 查询防护策略详情
```bash
hcloud WAF ShowPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>"
```

#### 更新防护策略
```bash
hcloud WAF UpdatePolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --body="{\"name\":\"prod-waf-policy-v2\",\"level\":1}"
```

#### 删除防护策略
```bash
hcloud WAF DeletePolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>"
```

### 域名与网站接入（云模式）

#### 列出所有防护域名
```bash
hcloud WAF ListHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

参数说明：
- `enterprise_project_id` — 企业项目 ID，可选

#### 添加云模式防护域名
```bash
hcloud WAF CreateHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --body="{\"hostname\":\"www.example.com\",\"policyid\":\"<policyId>\",\"server\":[{\"front_protocol\":\"HTTPS\",\"back_protocol\":\"HTTP\",\"address\":\"192.168.1.100\",\"port\":8080,\"type\":\"ipaddr\"}],\"certificateid\":\"<certId>\",\"certificatename\":\"my-cert\"}"
```

#### 查询防护域名详情
```bash
hcloud WAF ShowHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --host_id="<hostId>"
```

#### 更新防护域名
```bash
hcloud WAF UpdateHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --host_id="<hostId>" --body="{\"server\":[{\"front_protocol\":\"HTTPS\",\"back_protocol\":\"HTTP\",\"address\":\"10.0.0.1\",\"port\":8443,\"type\":\"ipaddr\"}]}"
```

#### 删除防护域名
```bash
hcloud WAF DeleteHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --host_id="<hostId>"
```

### CC 攻击防护规则

#### 列出 CC 防护规则
```bash
hcloud WAF ListCcRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>"
```

#### 创建 CC 防护规则
```bash
hcloud WAF CreateCcRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --body="{\"url\":\"/api/v1/*\",\"limit_num\":100,\"limit_period\":60,\"lock_time\":300,\"tag_type\":\"ip\",\"action\":{\"category\":\"block\"}}"
```

参数说明：
- `url` — 保护的 URL 路径，必填
- `limit_num` — 访问次数阈值，必填
- `limit_period` — 统计周期（秒），必填
- `lock_time` — 锁定时间（秒），必填
- `tag_type` — 限速依据（ip/cookie/header/params），必填
- `action.category` — 动作类型（block/log/captcha）

#### 更新 CC 防护规则
```bash
hcloud WAF UpdateCcRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>" --body="{\"limit_num\":200,\"limit_period\":60,\"lock_time\":600,\"action\":{\"category\":\"captcha\"}}"
```

#### 删除 CC 防护规则
```bash
hcloud WAF DeleteCcRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>"
```

### 精准访问防护规则（自定义规则）

#### 列出精准防护规则
```bash
hcloud WAF ListCustomRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>"
```

#### 创建精准防护规则
```bash
hcloud WAF CreateCustomRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --body="{\"name\":\"block-sql-injection\",\"time\":true,\"start\":0,\"terminal\":86399,\"conditions\":[{\"category\":\"url\",\"logic_operation\":\"contain\",\"contents\":[\"/admin\"]}],\"action\":{\"category\":\"block\"},\"priority\":100}"
```

参数说明：
- `name` — 规则名称，必填
- `conditions` — 匹配条件列表，必填
- `action.category` — 动作类型（block/log/redirect）
- `priority` — 优先级（1~65535），值越小优先级越高

#### 更新精准防护规则
```bash
hcloud WAF UpdateCustomRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>" --body="{\"name\":\"block-admin-access-v2\",\"priority\":50}"
```

#### 删除精准防护规则
```bash
hcloud WAF DeleteCustomRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>"
```

### 黑白名单规则

#### 列出黑白名单规则
```bash
hcloud WAF ListWhiteBlackIpRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>"
```

#### 创建黑白名单规则
```bash
hcloud WAF CreateWhiteBlackIpRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --body="{\"name\":\"block-attacker-ip\",\"addr\":\"10.0.0.0/24\",\"white\":0}"
```

参数说明：
- `name` — 规则名称，必填
- `addr` — IP 地址/地址段，支持 IPv4/IPv6，必填
- `white` — 类型（0:黑名单 1:白名单），必填
- `address_group_id` — 引用地址组 ID（与 addr 二选一）

#### 更新黑白名单规则
```bash
hcloud WAF UpdateWhiteBlackIpRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>" --body="{\"name\":\"block-attacker-ip-v2\",\"white\":0}"
```

#### 删除黑白名单规则
```bash
hcloud WAF DeleteWhiteBlackIpRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>"
```

### 地理位置访问控制规则

#### 列出地理位置规则
```bash
hcloud WAF ListGeoIpRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>"
```

#### 创建地理位置规则
```bash
hcloud WAF CreateGeoIpRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --body="{\"name\":\"block-unusual-regions\",\"geoip\":\"RU|KP|IR\",\"white\":0}"
```

参数说明：
- `name` — 规则名称，必填
- `geoip` — 地理区域编码（ISO 3166，多个用 | 分隔），必填
- `white` — 类型（0:拦截 1:放行），必填

#### 更新地理位置规则
```bash
hcloud WAF UpdateGeoIpRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>" --body="{\"name\":\"block-unusual-regions-v2\",\"geoip\":\"RU|KP\",\"white\":0}"
```

#### 删除地理位置规则
```bash
hcloud WAF DeleteGeoIpRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>"
```

### 网页防篡改规则

#### 列出防篡改规则
```bash
hcloud WAF ListAntiTamperRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>"
```

#### 创建防篡改规则
```bash
hcloud WAF CreateAntiTamperRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --body="{\"url\":\"/about\",\"category\":\"url\"}"
```

#### 删除防篡改规则
```bash
hcloud WAF DeleteAntiTamperRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>"
```

### 信息防泄漏规则

#### 列出信息防泄漏规则
```bash
hcloud WAF ListAntileakageRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>"
```

#### 创建信息防泄漏规则
```bash
hcloud WAF CreateAntileakageRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --body="{\"url\":\"/api/user/*\",\"category\":\"all\",\"contents\":[\"phone\",\"id_card\",\"email\"]}"
```

参数说明：
- `url` — 保护 URL 路径，必填
- `category` — 类型，必填（`all` 或 `sensitive`）
- `contents` — 防护内容（phone/id_card/email），必填

#### 更新信息防泄漏规则
```bash
hcloud WAF UpdateAntileakageRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>" --body="{\"url\":\"/api/user/*\",\"category\":\"sensitive\",\"contents\":[\"id_card\"]}"
```

#### 删除信息防泄漏规则
```bash
hcloud WAF DeleteAntileakageRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>"
```

### 反爬虫规则

#### 列出反爬虫规则
```bash
hcloud WAF ListAnticrawlerRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>"
```

#### 创建反爬虫规则
```bash
hcloud WAF CreateAnticrawlerRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --body="{\"name\":\"protect-login-api\",\"url\":\"/api/login\",\"logic_operation\":\"equal\",\"description\":\"protect login endpoint from crawlers\"}"
```

#### 更新反爬虫规则
```bash
hcloud WAF UpdateAnticrawlerRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>" --body="{\"name\":\"protect-login-api-v2\",\"url\":\"/api/login\",\"logic_operation\":\"equal\"}"
```

#### 删除反爬虫规则
```bash
hcloud WAF DeleteAnticrawlerRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>"
```

### 已知攻击源规则

#### 列出已知攻击源规则
```bash
hcloud WAF ListAttackMitigationRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>"
```

#### 更新已知攻击源规则（开启/关闭）
```bash
hcloud WAF UpdateAttackMitigationRule --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" --rule_id="<ruleId>" --body="{\"action\":\"block\"}"
```

### 攻击事件分析

#### 列出攻击事件
```bash
hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --recent="true"
```

按域名和时间范围筛选：
```bash
hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --hosts="www.example.com" --from="<recentTimestampMs>" --to="<nowTimestampMs>"
```

#### 查看事件详情
```bash
hcloud WAF ShowEvent --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --eventid="<eventId>"
```

### 证书管理

#### 列出所有证书
```bash
hcloud WAF ListCertificates --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 上传证书
```bash
hcloud WAF CreateCertificate --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --body="{\"name\":\"my-cert\",\"content\":\"<base64-cert>\",\"key\":\"<base64-key>\"}"
```

#### 查询证书详情
```bash
hcloud WAF ShowCertificate --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --certificate_id="<certId>"
```

#### 更新证书
```bash
hcloud WAF UpdateCertificate --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --certificate_id="<certId>" --body="{\"name\":\"my-renewed-cert\",\"content\":\"<base64-cert>\",\"key\":\"<base64-key>\"}"
```

#### 删除证书
```bash
hcloud WAF DeleteCertificate --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --certificate_id="<certId>"
```

> ⚠️ **安全门**：删除证书前须确认其未被任何域名绑定。如有绑定，先解绑或替换证书。

### 引用表管理

#### 列出引用表
```bash
hcloud WAF ListValueList --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

#### 创建引用表
```bash
hcloud WAF CreateValueList --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --body="{\"name\":\"admin-whitelist-ips\",\"type\":\"ip\",\"values\":[\"10.0.0.0/8\",\"172.16.0.0/12\"]}"
```

参数说明：
- `name` — 引用表名称，必填
- `type` — 类型（ip/url/params/cookie/header），必填
- `values` — 引用表值列表，必填

#### 查询引用表详情
```bash
hcloud WAF ShowValueList --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --valuelistid="<valueListId>"
```

#### 更新引用表
```bash
hcloud WAF UpdateValueList --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --valuelistid="<valueListId>" --body="{\"name\":\"admin-whitelist-ips-v2\",\"values\":[\"10.0.0.0/8\"]}"
```

#### 删除引用表
```bash
hcloud WAF DeleteValueList --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --valuelistid="<valueListId>"
```

> ⚠️ **安全门**：删除引用表前须确认其未被任何规则引用。已引用的引用表删除将导致相关规则失效。

### Go SDK 快速参考

```go
package main

import (
    "fmt"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/waf/v1"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/waf/v1/model"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/waf/v1/region"
)

func main() {
    auth, _ := basic.NewCredentialsBuilder().
        WithAk("{{env.HW_ACCESS_KEY}}").
        WithSk("{{env.HW_SECRET_KEY}}").
        WithProjectId("{{env.HW_PROJECT_ID}}").
        Build()

    client := waf.NewWafClient(
        waf.WafClientBuilder().
            WithRegion(region.ValueOf("{{env.HW_CLOUD_REGION}}")).
            WithCredential(auth).
            Build())

    // 列出所有策略
    resp, err := client.ListPolicy(&model.ListPolicyRequest{})
    if err != nil {
        fmt.Printf("[ERROR] %v", err)
        return
    }
    for _, p := range respp.Items {
        fmt.Printf("Policy: %s (%s)\n", p.Name, p.Id)
    }
}
```

## 质量门 (GCL)

本 skill 强制 GCL(参见 `AGENTS.md` §8)。所有 WAF 变更操作(策略创建/更新/删除、防护域名创建/更新/删除、规则创建/更新/删除/禁用、证书删除)均需经过 **Generator-Critic-Loop** 后才能返回结果。只读操作为 GCL-豁免。

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
| 1 | Correctness | ≥ 0.5(`delete-policy` / `delete-host` / `delete-rule` / `disable-rule` 要求 = 1) | `ShowPolicy` / `ShowHost` / `ShowRule` post-state |
| 2 | Safety | **= 1**(任一 S-rule 命中 → ABORT) | rubric §2 中 S1–S17 |
| 3 | Idempotency | ≥ 0.5 | 创建前先检查 |
| 4 | Traceability | ≥ 0.5 | `password` / 证书私钥必须为 `<masked>` |
| 5 | Spec Compliance | ≥ 0.5 | 防护等级 (1/2/3) / 规则动作 / 域名协议 / 证书格式 |

### 每操作 Safety 锚点(强制)

- **S1 / S2 / S3** — `delete-policy` 确认 / 仍有防护域名引用 / 是默认或最后一条策略
- **S4 / S5** — `delete-host` 确认 / 生产域名二次确认
- **S6 / S7** — `delete-rule` / `disable-rule` 二次确认
- **S8** — `update-policy` 防护等级降到 1(宽松)
- **S9** — `update-host` 关闭 proxy (禁用 WAF 隧道)
- **S10** — `delete-certificate` 仍有域名引用
- **S11** — `create-host` 私网地址 + 关闭 proxy 错配
- **S12** — `create-rule` 动作 pass(绕过检查)
- **S13** — `create/update-policy` 关闭全检测
- **S14** — trace 包含私钥 (`BEGIN PRIVATE KEY`)
- **S16** — 删除 `sys_` 前缀系统规则
- **S17** — `update-host` 改 `policyid` 未二次确认

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
- 仓库根 [`AGENTS.md`](../AGENTS.md) §3、§5、§7、§8 — GCL 规范

## 参考文档

| 文档 | 说明 |
|------|------|
| [API 导航](references/api-navigation.md) | WAF v1 API 完整接口列表 |
| [CLI 语法参考](references/cli-syntax-reference.md) | KooCLI 命令参数速查 |
| [常见故障处理](references/common-faults.md) | 常见错误及处理方案 |
| [成本优化](references/advanced/cost-optimization.md) | FinOps 费用优化指南 |
| [安全最佳实践](references/advanced/security-best-practices.md) | SecOps 安全加固方案 |
| [智能运维](references/advanced/aiops-patterns.md) | AIOps 异常检测与自愈 |
| [安全门](references/advanced/safety-gates.md) | 高危操作审批流程 |
| [错误处理](references/error-handling.md) | 标准化错误诊断流程 |
| [GCL Rubric](references/rubric.md) | 对抗式质量门 (v1, 5 维, S1–S17 WAF 特定 Safety 规则) |
| [GCL Prompt 模板](references/prompt-templates.md) | Generator / Critic / Orchestrator 模板 |
