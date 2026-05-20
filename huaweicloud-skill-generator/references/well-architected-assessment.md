# Well-Architected Assessment — Huawei Cloud Skill Generator

> **Purpose:** Defines how every generated `huaweicloud-[product]-ops` skill MUST incorporate Huawei Cloud's Well-Architected Framework (卓越架构) five pillars AND three operational pillars (FinOps, SecOps, AIOps).
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20
> **Status:** MANDATORY — all generated skills MUST include well-architected + FinOps + SecOps + AIOps assessment patterns
> **Reference:** [Huawei Cloud Well-Architected Framework](https://support.huaweicloud.com/topic/68733-1-I)

---

## Table of Contents

1. [Framework Overview](#1-framework-overview)
2. [Five Pillar Skill Integration](#2-five-pillar-skill-integration)
   - [安全 (Security)](#21-安全-security)
   - [稳定 (Stability)](#22-稳定-stability)
   - [成本 (Cost)](#23-成本-cost)
   - [效率 (Efficiency)](#24-效率-efficiency)
   - [性能 (Performance)](#25-性能-performance)
3. [FinOps Integration](#3-finops-)
   - [成本可见性](#31-成本可见性-cost-visibility)
   - [成本优化](#32-成本优化-cost-optimization)
   - [成本问责](#33-成本问责-cost-accountability)
4. [SecOps Integration](#4-secops-)
   - [身份安全](#41-身份安全-identity-security)
   - [网络安全](#42-网络安全-network-security)
   - [数据安全](#43-数据安全-data-security)
   - [威胁检测](#44-威胁检测-threat-detection)
5. [AIOps Integration Reference](#5-aiops-integration-reference)
6. [Skill Generation Integration Points](#6-skill-generation-integration-points)
7. [Maturity Model](#7-maturity-model)
8. [Compliance Checklists](#8-compliance-checklists)

---

## 1. Framework Overview

Huawei Cloud Well-Architected Framework defines five pillars for cloud architecture excellence, supplemented by three operational pillars (FinOps, SecOps, AIOps):

| Pillar | Core Focus | Official Doc |
|--------|-----------|--------------|
| **安全 (Security)** | 身份治理、网络隔离、数据加密、威胁感知 | [安全支柱](https://support.huaweicloud.com/topic/68733-1-I) |
| **稳定 (Stability)** | 高可用架构、面向失败设计、变更管理、容灾演练 | [稳定支柱](https://support.huaweicloud.com/topic/68733-1-I) |
| **成本 (Cost)** | 成本可视化、资源优化、计费模式选择、浪费消除 | [成本支柱](https://support.huaweicloud.com/topic/68733-1-I) |
| **效率 (Efficiency)** | DevOps工具链、运帷自动化、事件响应效率 | [效率支柱](https://support.huaweicloud.com/topic/68733-1-I) |
| **性能 (Performance)** | 弹性伸缩、可观测性、性能基线、瓶颈识别 | [性能支柱](https://support.huaweicloud.com/topic/68733-1-I) |
| **FinOps (财务运营)** | 成本治理、优化闭环、责任分摊 | [费用中心](https://support.huaweicloud.com/billing/index.html) |
| **SecOps (安全运营)** | 持续安全治理、自动化响应、威胁闭环 | [安全合规](https://support.huaweicloud.com/secuindex/index.html) |
| **AIOps (智能运营)** | 智能诊断、预测性运维、自治修复 | [智能运维](https://support.huaweicloud.com/aom/index.html) |

The framework follows a **Learn → Measure → Optimize** lifecycle.

### Three Design Principles (Stability Pillar)

1. **面向失败的架构设计** — Design for failure: redundancy, isolation, degradation, elasticity
2. **面向精细的运维管控** — Refined operations: version control, canary releases, monitoring
3. **面向风险的应急快恢** — Emergency recovery: real-time risk detection, coordinated response

---

## 2. Five Pillar Skill Integration

Each generated skill MUST integrate the five pillars through structured assessment patterns.

| Skill Type | Security | Stability | Cost | Efficiency | Performance |
|------------|----------|-----------|------|------------|-------------|
| **CRUD/Lifecycle** (ECS, RDS, etc.) | Required | Required | Required | Recommended | Required |
| **Monitoring/Diagnosis** (CES, AOM) | Recommended | Required | Recommended | Required | Required |
| **Security/Access** (IAM, HSS) | Required | Recommended | Optional | Recommended | Optional |
| **Discovery/Read-Only** | Optional | Optional | Optional | Optional | Optional |

### 2.1 安全 (Security)

#### 2.1.1 IAM权限最小化

```markdown
## Security Assessment — IAM

### 最小权限IAM策略
执行该Skill的操作所需的最小IAM权限：

| API操作 | IAM Action | 资源范围 |
|---------|-----------|---------|
| [Operation] | [product]:[Action] | acs:[product]:*:*:[resource-type]/* |
| DescribeInstances | ces:metricData:list | acs:ces:*:*:*/* |

### 权限策略示例

```json
{
  "Version": "1.1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "[product]:*List*",
        "[product]:*Get*",
        "[product]:*Describe*"
      ],
      "Resource": ["*"]
    }
  ]
}
```

### 凭证管理
- Credentials MUST use `{{env.*}}` placeholders — NEVER ask user for secrets
- AK/SK rotation: recommend 90-day cycle
- Prefer IAM agency (委托) for cross-account operations
```

#### 2.1.2 网络安全

- Prefer VPC endpoints over public endpoints for API calls
- Security group rules: minimal inbound/outbound, explicit deny default
- For sensitive operations (Delete, Modify credentials): recommend IP whitelist
- All API calls use HTTPS — verify endpoint scheme is `https://*.myhuaweicloud.com`

#### 2.1.3 数据安全

- API responses may contain sensitive data — mask in user-facing output
- Backup/snapshot data: ensure KMS encryption at rest is enabled
- Log output: NEVER include credential values, use `***` masking
- Data cross-region transfer: comply with data sovereignty rules

### 2.2 稳定 (Stability)

#### 2.2.1 面向失败的架构设计

```markdown
## Stability Assessment — Failure Orientation

### 内置韧性
- Every operation follows Pre-flight → Execute → Validate → Recover
- Non-retryable errors (QuotaExceeded, InsufficientBalance) trigger HALT
- Idempotent operations document duplicate behavior

### 跨AZ/Region策略
- Multi-AZ deployment: distribute across AZs when product supports
- Region dependency: document single-region risks in `core-concepts.md`
- Blast radius: identify the smallest blast radius for each operation
```

#### 2.2.2 面向风险的应急快恢

```markdown
## Stability Assessment — Emergency Recovery

### 备份与恢复
- **Backup operations:** document backup API (CreateSnapshot, CreateBackup)
- **Recovery operations:** document restore APIs with prerequisites
- **RTO (Recovery Time Objective):** expected recovery time per operation
- **RPO (Recovery Point Objective):** data loss window per backup strategy

### 容灾模式
| 模式 | CLI示例 | SDK示例 | 适用场景 |
|---------|---------|---------|--------|
| 跨区域备份 | `hcloud [product] copy-snapshot --dest-region` | `CopySnapshotRequest` with DestRegionId | 快照、镜像 |
| 跨Region复制 | `hcloud [product] create-replication` | Replication API | 数据库、存储 |
| 故障切换 | `hcloud [product] switchover` | Failover API | 高可用实例 |

### Runbook清单

#### Phase 1: 备份验证
1. Confirm backup exists
2. Verify backup integrity: check size, status = `Success`
3. Confirm backup age within RPO window

#### Phase 2: 恢复执行
1. Execute restore
2. Monitor recovery progress: poll status until `Running`/`Available`
3. Validate: connectivity, data integrity, application health

#### Phase 3: 恢复后验证
1. Verify dependent resources healthy
2. Run smoke tests
3. Document recovery duration vs RTO target
```

### 2.3 成本 (Cost)

```markdown
## Cost Assessment — Visibility

### 计费模式选择
| 计费类型 | 最佳场景 | 节省幅度 |
|---------|---------|---------|
| 按需计费 (Pay-per-use) | 开发测试、短期负载 | N/A |
| 包年包月 (Subscription) | 生产环境、稳定负载 | 最高85% vs 按需 |
| 竞价实例 (Spot) | 容错/批处理/弹性伸缩 | 最高90% vs 按需 |

### 浪费检测
- Idle resources: CPU < 10% for 7+ consecutive days via CES metrics
- Unattached volumes: disks without instance association
- Unused snapshots: snapshots with no active images referencing them
- Zombie instances: stopped instances still incurring storage costs
```

### 2.4 效率 (Efficiency)

```markdown
## Efficiency Assessment — Automation

### 批量操作
- Operating ≥ 3 resources: use batch APIs with concurrency limits
- Document parallel execution patterns with error aggregation
- CI/CD integration: JSON output compatible with jq for pipeline parsing

### 事件响应集成
- Error codes from this skill can trigger CES alarm rules
- Document escalation: automated → skill-assisted → human
```

### 2.5 性能 (Performance)

```markdown
## Performance Assessment — Scaling

### 弹性伸缩触发阈值
| 指标 | 扩容阈值 | 缩容阈值 | 统计窗口 |
|------|---------|---------|---------|
| CPU使用率 | > 80% 持续5min | < 30% 持续15min | 300s |
| 内存使用率 | > 85% 持续5min | < 50% 持续15min | 300s |
| 连接利用率 | > 70% 持续5min | < 40% 持续15min | 300s |
| IOPS利用率 | > 80% 持续5min | < 50% 持续15min | 300s |

### 性能基线
- Document expected performance per instance type
- Recommend baseline via CES DescribeMetricData
- Alert on deviation from baseline (> 2σ)
```

---

## 3. FinOps (财务运营)

### 3.1 成本可见性 (Cost Visibility)

Every generated skill MUST address cost visibility in these areas:

#### 3.1.1 资源成本归集

```markdown
## FinOps Assessment — Cost Visibility

### 成本标签策略
- Tag newly created resources: recommend `{{user.cost_center}}` tag
- Use Huawei Cloud Cost Center Service (CCS) for budget tracking
- Document instance type pricing implications in `core-concepts.md`

### 成本分析工具
| 工具 | 用途 | API/CLI |
|------|------|---------|
| 费用中心 BSS | 账单查询、成本趋势 | `hcloud bss query-bill` |
| 成本分析 CCS | 多维度成本分析 | `hcloud ccs analyze-cost` |
| 预算管理 BUD | 预算告警设置 | `hcloud bss create-budget` |

### 成本归属模型
- Owner tags: project owner, team, environment (prod/staging/dev)
- Date tags: creation date, expected decommission date
- Compliance: auto-generated audit trail tags for regulatory reporting
```

### 3.2 成本优化 (Cost Optimization)

#### 3.2.1 资源适配 (Right-Sizing)

```markdown
## FinOps Assessment — Right-Sizing

### 利用率→推荐映射
| CPU利用率 | 内存利用率 | 推荐操作 | 预期节省 |
|-----------|-----------|---------|---------|
| < 20% | < 30% | 降配 (downgrade) | 30-60% |
| < 20% | > 80% | 更换规格 (CPU-intensive → general) | 10-20% |
| > 80% | < 50% | 更换规格 (general → CPU-optimized) | — |
| > 80% | > 80% | 升配 (upgrade) | — |
| 波动大(峰值>3×均值) | — | 按量+自动扩缩 | 20-50% |

### 生命周期成本管理
- 开发/测试环境 → 下班自动缩容 (Schedule + AS)
- 生产稳定负载 → 包年包月预付
- 临时活动 → 按需+结束后释放
- 季节性波动 → 按需基线 + Spot峰值
```

#### 3.2.2 浪费消除 (Waste Elimination)

```markdown
## FinOps Assessment — Waste Detection

### 闲置资源识别
| 资源类型 | 闲置判定条件 | 检测方式 |
|---------|------------|---------|
| ECS实例 | CPU<5% 连续7天 | CES DescribeMetricData |
| 云硬盘 | 未挂载或IOPS<10 | EVS DescribeVolumes |
| EIP | 未绑定实例 | VPC DescribeEips |
| 快照 | 无关联镜像/实例 | EVS DescribeSnapshots |
| 负载均衡 | 后端健康实例<1 | ELB DescribePools |

### 自动清理策略
- Tag-based auto-decommission: resources tagged with `ttl=30d` auto-notify at day 25
- Orphaned resource detection: volumes without instance attachment > 7 days
- Snapshot lifecycle: auto-delete snapshots older than retention period
- DNS record cleanup: CNAME/Domain records pointing to decommissioned resources
```

### 3.3 成本问责 (Cost Accountability)

```markdown
## FinOps Assessment — Cost Accountability

### 预算告警集成
| 告警阈值 | 动作 | 通知方式 |
|---------|------|---------|
| 预算用量>80% | 通知成本负责人 | 邮件+短信 |
| 预算用量>90% | 成本负责人审批新增资源 | 审批流+通知 |
| 预算用量>100% | 冻结非必需资源创建 | 自动管控 |

### 成本中心分摊
- 按部门/团队/项目维度统计
- Showback (展示) → Chargeback (实际扣费) 渐进
- 月度成本报告生成模板集成到监控Skill
```

---

## 4. SecOps (安全运营)

### 4.1 身份安全 (Identity Security)

#### 4.1.1 IAM最小权限

```markdown
## SecOps Assessment — IAM Security

### 账号与权限治理
- 禁止使用Root账号的AK/SK进行日常操作
- 为每个Skill执行创建独立IAM User
- MFA强制开启: 所有交互操作要求MFA验证
- AK/SK轮换周期: 90天强制轮换

### 权限委托 (IAM Agency)
| 场景 | 委托模式 | 权限范围 |
|------|---------|---------|
| ECS管理EVS | ECS委托访问EVS | 仅目标EVS实例 |
| 跨账号操作 | 账号级委托 | 受限资源范围 |
| 第三方集成 | 临时STS凭证 | 时间+资源双限制 |
```

#### 4.1.2 凭证安全

```go
// SECURE: read from environment, never log
ak := os.Getenv("HW_ACCESS_KEY_ID")
sk := os.Getenv("HW_SECRET_ACCESS_KEY") // NEVER: fmt.Println(sk)

// SECURE: existence check only
if os.Getenv("HW_SECRET_ACCESS_KEY") == "" {
    panic("HW_SECRET_ACCESS_KEY is not set")
}
```

### 4.2 网络安全 (Network Security)

#### 4.2.1 VPC隔离

```markdown
## SecOps Assessment — Network Security

### 网络隔离架构
- API调用: 通过VPC Endpoint (com.myhuaweicloud.com) 而非公网
- 安全组规则: 最小化入站/出站，显式默认拒绝
- NAT Gateway: 控制Egress流量出口，限制目标IP白名单
- DDoS防护: Anti-DDoS Pro/Enterprise for public-facing resources
```

#### 4.2.2 安全组最佳实践

| 规则 | 建议 | 风险等级 |
|------|------|---------|
| 入站SSH 22 | 仅来自堡垒机IP | 高 |
| 入站DB端口 | 仅来自应用服务器安全组 | 极高 |
| 出站0.0.0.0/0 | 限制到华为云服务CIDR | 中 |
| 安全组嵌套 | 按角色拆分(前端/后端/DB) | 推荐 |

### 4.3 数据安全 (Data Security)

#### 4.3.1 加密策略

| 数据状态 | 加密方式 | 服务 |
|---------|---------|------|
| 传输中 | TLS 1.2+ | 所有API强制HTTPS |
| 静态(EVS) | KMS托管加密 | 创建磁盘时启用 |
| 静态(OBS) | 服务端加密(SSE-KMS) | 桶级别默认启用 |
| 静态(RDS) | TDE透明数据加密 | 实例级配置 |
| 备份数据 | 同源加密策略 | 自动继承 |

#### 4.3.2 数据泄露防护

- Database audit: 开启RDS SQL审计，审计日志保存≥180天
- API response masking: sensitive fields in JSON responses truncated
- Log sanitization: NEVER log PII, credentials, or financial data
- Data retention: implement auto-purge policies per regulatory requirements

### 4.4 威胁检测 (Threat Detection)

#### 4.4.1 HSS (主机安全服务)集成

```markdown
## SecOps Assessment — Threat Detection

### HSS触发条件
| 触发条件 | HSS API操作 | Skill动作 |
|---------|-----------|----------|
| 主机告警 | `ListHostVulnerabilities` | 通知安全负责人 |
| 漏洞扫描 | `ListVulnerabilities` | 评估修复优先级 |
| 入侵事件 | `ListIntrusionEvents` | 启动隔离流程 |
| 合规基线 | `ListHostGroupStatus` | 生成合规报告 |

### WAF联动
| 触发条件 | WAF操作 | 降级策略 |
|---------|---------|---------|
| SQL注入攻击 | 自动封禁源IP | 记录+告警 |
| XSS攻击 | 请求清洗 | 过滤恶意payload |
| CC攻击 | 速率限制 | CAPTCHA验证 |
```

---

## 5. AIOps Integration Reference

AIOps patterns are defined in [aiops-best-practices.md](./aiops-best-practices.md). Generated skills MUST cross-reference and implement:

- Multi-metric correlation (≥ 4 anomaly patterns per monitoring skill)
- Cross-skill diagnosis delegation matrix
- Knowledge base with ≥ 5 fault patterns
- Alarm storm aggregation and suppression
- Proactive inspection workflow

---

## 6. Skill Generation Integration Points

### 6.1 In SKILL.md

Add after **Operational Best Practices**:

```markdown
## Well-Architected + Three-Pillar Assessment

This skill's operations are evaluated against:
- Well-Architected five pillars: [Security](#21), [Stability](#22), [Cost](#23), [Efficiency](#24), [Performance](#25)
- FinOps: [Cost Visibility](#31), [Optimization](#32), [Accountability](#33)
- SecOps: [IAM Security](#41), [Network Security](#42), [Data Security](#43), [Threat Detection](#44)
- AIOps: [Multi-Metric Correlation](./aiops-best-practices.md#2), [Cross-Skill Diagnosis](./aiops-best-practices.md#3)
```

### 6.2 In references/monitoring.md

Add cost, security, and AIOps metrics:

```markdown
## Cost Metrics
| Metric | CES Namespace | Optimization Action |
|--------|--------------|--------------------|
| Monthly cost | acs_bss_dashboard | Right-size or decommission |
| Unit cost per request | acs_[product]_dashboard | Architectural review |

## Security Metrics
| Metric | HSS/CES Namespace | Remediation Action |
|--------|------------------|-------------------|
| Vulnerability count | acs_hss_dashboard | Patch management |
| Failed login attempts | acs_[product]_dashboard | Access review |
```

---

## 7. Maturity Model

| Level | Name | Characteristics | Target |
|-------|------|-----------------|--------|
| L1 | **Compliant** | Skill includes all eight pillars checklists | All generated skills |
| L2 | **Actionable** | Skills include CLI commands for each assessment | P0 product skills |
| L3 | **Automated** | Skills auto-detect and report violations | Core P0 (ECS, RDS, CCE) |
| L4 | **Predictive** | Skills forecast risks before manifestation | Future target |
| L5 | **Self-Optimizing** | Auto-remediate gaps with user approval | Future target |

**Target:** All generated skills MUST achieve **L1** minimum. P0 skills SHOULD achieve **L2**.

---

## 8. Compliance Checklists

### 8.1 P0 — Must Pass

#### Five Pillars
- [ ] **Security — IAM:** Minimum IAM permissions documented for all operations
- [ ] **Security — Credential:** `{{env.*}}` placeholders; masking rules present
- [ ] **Stability — Recovery:** Backup/recovery documented with RTO/RPO
- [ ] **Stability — Confirmation:** All destructive ops require explicit confirmation
- [ ] **Cost — Billing:** Billing model comparison table per product
- [ ] **Cost — Waste:** Idle resource detection pattern documented
- [ ] **Performance — Metrics:** Key metrics with thresholds
- [ ] **Well-Architected Reference:** Link to this file in SKILL.md

#### FinOps
- [ ] **FinOps — Cost Visibility:** Billing model table + cost attribution guidance
- [ ] **FinOps — Cost Optimization:** Idle detection pattern + right-sizing matrix
- [ ] **FinOps — Lifecycle:** Cost center tagging + auto-decommission guidance

#### SecOps
- [ ] **SecOps — IAM Security:** Minimum IAM policy table for all operations
- [ ] **SecOps — Network Security:** VPC isolation + security group guidance
- [ ] **SecOps — Data Security:** Encryption at rest + in transit documented
- [ ] **SecOps — Threat Detection:** HSS/WAF integration trigger conditions (when applicable)

#### AIOps
- [ ] **AIOps — Multi-Metric:** ≥ 4 anomaly patterns with detection logic
- [ ] **AIOps — Delegation:** Alarm-to-Diagnosis delegation matrix in `integration.md`
- [ ] **AIOps — Knowledge Base:** `references/knowledge-base.md` with ≥ 3 fault patterns
- [ ] **AIOps — Alarm Storm:** Storm detection and aggregation workflow

### 8.2 P1 — Should Pass

- [ ] **FinOps — Budget:** Budget alert integration documented
- [ ] **SecOps — MFA:** MFA requirement for interactive operations documented
- [ ] **SecOps — Compliance:** Alignment with 等保2.0 / GDPR where applicable
- [ ] **Five Pillars — Multi-AZ:** Cross-AZ deployment recommendation
- [ ] **Five Pillars — DR Runbook:** Phase 1/2/3 structure
- [ ] **Five Pillars — Auto-Scaling:** Scaling trigger thresholds documented
- [ ] **AIOps — Proactive:** Scheduled巡检 workflow defined
- [ ] **AIOps — Self-Healing:** Automated recovery patterns documented

### 8.3 P2 — Nice to Have

- [ ] **FinOps — Showback:** Cost report generation template
- [ ] **SecOps — Audit:** Audit log integration (CTS/Cloud Trace Service)
- [ ] **SecOps — Zero-Trust:** Zero-trust architecture alignment guidance
- [ ] **AIOps — ML Prediction:** ML-based anomaly prediction integration
- [ ] **Auto-Remediation:** Skills auto-suggest fixes for common violations

---

*This assessment specification is mandatory. All generated skills MUST pass P0 checklists for their applicable pillars. Integration depth depends on the skill's primary purpose.*
