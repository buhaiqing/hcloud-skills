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

#### 3.1.2 单位经济学 (Unit Economics)

Every generated skill SHOULD define unit cost metrics that map resource cost to business output:

```markdown
## FinOps Assessment — Unit Economics

### 单位成本指标模板
| 指标 | 计算公式 | 数据来源 | 优化方向 |
|------|---------|---------|---------|
| 每vCPU成本 | 月费用 / vCPU数 | BSS + 产品API | 选择最优规格组合 |
| 每GB存储成本 | 存储费用 / GB数 | BSS + EVS/OBS API | 评估存储分层 |
| 每请求成本 | 总费用 / 请求量 | BSS + CES QPS指标 | 提升单请求效率 |
| 每用户成本 | 总费用 / 活跃用户数 | BSS + 业务指标 | 架构降本 |
| 每交易成本 | 总费用 / 交易笔数 | BSS + 业务指标 | 消除冗余链路 |

### 成本效率基线
- 建立每产品的成本效率基线 (Cost Efficiency Baseline)
- 环比监控: 周环比 / 月环比偏差 > 15% 触发分析
- 同比监控: 与去年同期对比，识别季节性模式
- 建议: 在 `monitoring.md` 中增加 Cost per Unit 指标面板
```

#### 3.1.3 成本异常检测 (Cost Anomaly Detection)

```markdown
## FinOps Assessment — Cost Anomaly Detection

### 异常检测规则
| 异常类型 | 检测逻辑 | 严重等级 | 响应动作 |
|---------|---------|---------|---------|
| 成本突增 | 日成本 > 7日均值 × 1.5 | Critical | 立即通知 + 根因分析 |
| 成本突降 | 日成本 < 7日均值 × 0.5 | Warning | 检查服务降级或资源释放 |
| 预算偏差 | 实际支出 > 预算 × 110% (月度) | Warning | 调整预算或优化资源 |
| 资源突增 | 新增资源数 > 7日均值 × 2 | Warning | 确认是否为计划内扩容 |
| 闲置浪费反弹 | 闲置率环比上升 > 10% | Info | 触发新一轮清理 |

### CLI 集成
```bash
# 查询日成本趋势
hcloud bss query-bill --bill_cycle=$(date +%Y-%m) --output json | \
  jq '[.[] | {date: .bill_date, amount: .amount}]'

# 对比周环比
hcloud bss query-bill --bill_cycle=$(date -d '7 days ago' +%Y-%m)
```
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

#### 3.2.2 预留容量策略 (Reserved Capacity Strategy)

```markdown
## FinOps Assessment — Reserved Capacity

### 包年包月/预留覆盖率分析
| 覆盖率区间 | 状态评估 | 建议动作 |
|-----------|---------|---------|
| > 85% | 优化 | 检查是否存在过度预留、闲置的包年包月资源 |
| 60-85% | 健康 | 持续监控，对新增稳定负载考虑预留 |
| 40-60% | 待优化 | 识别稳定负载，制定预留转换计划 |
| < 40% | 风险 | 大量按需支出，优先转换长期运行资源 |

### 盈亏平衡点计算
- 公式: 盈亏月数 = 包年包月总价 / (按需月价 - 包年包月月均价)
- 示例: 某ECS实例包年包月月均 ¥300，按需月价 ¥900，年付总价 ¥2400
  - 盈亏月数 = 2400 / (900 - 300) = 4 个月
  - 结论: 运行超过4个月即回本

### 预留优化检查清单
- [ ] 连续运行 > 4个月的按需实例 → 评估转包年包月
- [ ] 包年包月到期前30天 → 评估续约/降配/释放
- [ ] 包年包月实例利用率 < 30% → 评估降配或释放
- [ ] 竞价实例中断率 > 5% → 评估切换到按需/包年包月
```

#### 3.2.3 浪费消除 (Waste Elimination)

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

#### 3.2.4 TCO 总拥有成本模型 (Total Cost of Ownership)

```markdown
## FinOps Assessment — TCO Model

### TCO 构成要素
| 成本类别 | 包含项目 | 数据来源 | 占比参考 |
|---------|---------|---------|---------|
| 计算成本 | 实例规格、License | BSS账单 | 40-60% |
| 存储成本 | 磁盘、备份、OBS | BSS账单 | 20-30% |
| 网络成本 | EIP、带宽、CDN | BSS账单 | 10-15% |
| 运维成本 | 监控、日志、安全 | BSS账单 | 5-10% |
| 人力成本 | 管理工时 × 人均成本 | 估算 | 变动较大 |

### 成本驱动因素分析
1. 识别 Top-5 成本驱动资源 (按费用降序)
2. 分析每项驱动的可优化空间 (规格调整 / 计费模式 / 架构优化)
3. 量化优化收益: 每项改进的月度节省金额
4. 优先级排序: 节省金额 × 实施难度 (高收益低难度优先)
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

#### 4.4.2 安全事件响应 Runbook

```markdown
## SecOps Assessment — Incident Response Runbook

### 安全事件分级
| 等级 | 定义 | 响应时间 | 示例 |
|------|------|---------|------|
| P0-Critical | 数据泄露/服务被控 | ≤ 15min | 入侵确认、数据外泄 |
| P1-High | 漏洞被利用风险 | ≤ 1h | 0-day漏洞暴露、AK/SK泄露 |
| P2-Medium | 安全策略违规 | ≤ 4h | 安全组误配置、未加密存储 |
| P3-Low | 合规偏差 | ≤ 24h | 审计日志缺失、标签不合规 |

### 响应流程 (P0/P1)
1. **隔离 (Contain)**: 安全组阻断非必要流量，保留取证环境
2. **取证 (Preserve)**: CTS审计日志导出、内存快照、磁盘快照
3. **根除 (Eradicate)**: 清除后门、修复漏洞、轮换凭证
4. **恢复 (Recover)**: 验证系统完整性、逐步恢复流量
5. **复盘 (Post-Mortem)**: 72h内完成事故报告、改进措施落地

### 取证保留要求
- CTS操作日志: 保留 ≥ 180天
- 安全事件日志: 保留 ≥ 365天
- 内存/磁盘快照: 事件关闭后保留 ≥ 90天
```

### 4.5 零信任架构对齐 (Zero Trust Architecture)

```markdown
## SecOps Assessment — Zero Trust Alignment

### 零信任核心原则
| 原则 | Skill实现 | 验证方式 |
|------|----------|---------|
| 永不信任，始终验证 | 每次API调用验证凭证有效性 | AK/SK + STS临时凭证 |
| 最小权限 | IAM策略精确到操作+资源 | 权限审计脚本 |
| 最小暴露面 | VPC Endpoint + 安全组白名单 | 网络拓扑审计 |
| 假设已被入侵 | 凭证轮换 + 异常行为检测 | HSS + CES告警联动 |
| 持续监控 | 全操作审计 + 实时告警 | CTS + LTS集成 |

### 持续验证模式
- API调用: 每次验证AK/SK有效性和权限范围
- 跨服务访问: 使用IAM Agency (委托) 而非嵌入凭证
- 临时授权: STS Token 有时效限制 (最长1小时)
- 异常行为检测: 同一AK短时间跨Region操作 → 触发告警
```

### 4.6 安全合规自动化 (Compliance Automation)

```markdown
## SecOps Assessment — Compliance Automation

### 等保2.0 / GDPR 自动检查清单
| 检查项 | 自动化检测方式 | 合规基线 | 不合规动作 |
|-------|--------------|---------|-----------|
| 网络隔离 | VPC + 安全组配置检查 | 生产VPC独立 | 自动告警+建议修复 |
| 数据加密 | KMS加密状态检查 | 所有存储启用KMS | 自动告警+启用引导 |
| 访问控制 | IAM策略审计 | 最小权限原则 | 生成过度权限报告 |
| 审计日志 | CTS启用状态检查 | 全操作审计 | 自动启用CTS |
| 备份策略 | 备份频率+保留检查 | 日备+7天保留 | 告警+创建备份计划 |
| 漏洞修复 | HSS漏洞扫描 | Critical ≤ 24h修复 | 生成修复工单 |

### CSPM 集成 (云安全态势管理)
- 对接华为云 Config (配置审计) 服务
- 资源配置变更实时检测
- 合规规则引擎: 预定义 + 自定义规则
- 不合规资源自动标记 + 修复建议
```

### 4.7 供应链安全 (Supply Chain Security)

```markdown
## SecOps Assessment — Supply Chain Security

### SDK 依赖安全
| 检查项 | 方法 | 频率 |
|-------|------|------|
| Go SDK 版本漏洞 | `govulncheck ./...` | 每次JIT执行前 |
| 依赖CVE扫描 | `nancy` / `snyk` | CI/CD集成 |
| 许可证合规 | `go-licenses` | 版本发布时 |
| 依赖锁定 | `go.sum` 完整性校验 | 每次构建 |

### JIT脚本安全准则
- [ ] 所有JIT Go脚本使用 `go.sum` 校验依赖完整性
- [ ] 禁止 `replace` 指向非官方仓库
- [ ] SDK来源限定: `github.com/huaweicloud/huaweicloud-sdk-go-v3` 官方仓库
- [ ] Go模块代理: 使用 `GOPROXY=https://goproxy.cn,direct` 防止中间人

### SBOM (软件物料清单)
- 每个生成的Skill SHOULD在 `assets/sbom.json` 中维护依赖清单
- 包含: 依赖名称、版本、许可证、CVE状态
- 更新时机: SDK版本升级时同步更新
```

### 4.8 密钥生命周期管理 (Key Lifecycle Management)

```markdown
## SecOps Assessment — Key Lifecycle

### KMS密钥管理策略
| 阶段 | 操作 | 自动化 |
|------|------|--------|
| 创建 | KMS CreateKey，指定算法(AES-256/RSA-2048) | 按需自动 |
| 启用 | KMS EnableKey | 创建后自动 |
| 轮换 | KMS RotateKey，周期: 365天(默认)/自定义 | 定时触发 |
| 禁用 | KMS DisableKey，密钥疑似泄露时 | 安全事件触发 |
| 计划删除 | KMS ScheduleKeyDeletion，延迟7-1096天 | 密钥替换后 |
| 审计 | CTS记录所有密钥操作 | 持续 |

### 密钥权限分离
- 密钥管理员: CreateKey, RotateKey, DisableKey (运维角色)
- 密钥使用着: Encrypt, Decrypt, GenerateDataKey (应用角色)
- 审计员: GetKeyPolicy, ListKeyDetail (安全角色)
- 原则: 管理、使用、审计三权分立

### 凭证泄露应急
1. 立即禁用泄露的AK/SK (IAM DeleteAccessKey)
2. 轮换所有相关KMS密钥 (KMS RotateKey)
3. 审查CTS日志确认泄露时间窗口
4. 评估影响范围并生成报告
5. 更新所有引用该凭证的配置
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

### 7.1 Maturity Levels

| Level | Name | Characteristics | Target |
|-------|------|-----------------|--------|
| L1 | **Compliant** | Skill includes all eight pillars checklists | All generated skills |
| L2 | **Actionable** | Skills include CLI commands for each assessment | P0 product skills |
| L3 | **Automated** | Skills auto-detect and report violations | Core P0 (ECS, RDS, CCE) |
| L4 | **Predictive** | Skills forecast risks before manifestation | Future target |
| L5 | **Self-Optimizing** | Auto-remediate gaps with user approval | Future target |

**Target:** All generated skills MUST achieve **L1** minimum. P0 skills SHOULD achieve **L2**.

### 7.2 Quantified Maturity Scorecard (量化评分卡)

Every generated skill MUST self-assess using this scorecard. Score each dimension 0-10:

```markdown
## Well-Architected Maturity Scorecard — [Product]

| 维度 | L1 (0-4) | L2 (5-6) | L3 (7-8) | L4 (9-10) | 自评分数 |
|------|----------|----------|----------|-----------|---------|
| **Security** | IAM表+凭证遮盖 | 安全组+加密 | 零信任+合规自动检查 | 威胁预测+自修复 | ___/10 |
| **Stability** | 备份文档+确认门 | RTO/RPO+DR Runbook | 自动故障切换+混沌验证 | 预测性容灾 | ___/10 |
| **Cost** | 计费模式对比 | 闲置检测+Right-Sizing | TCO模型+预留覆盖分析 | 成本异常预测+自优化 | ___/10 |
| **Efficiency** | CLI+SDK双路径 | 批量操作+CI/CD集成 | IaC模板+GitOps | 全自动Pipeline | ___/10 |
| **Performance** | 指标+阈值 | 性能基线+扩缩容 | 容量预测+SLO体系 | 自适应性能优化 | ___/10 |
| **FinOps** | 成本可见性 | 优化建议+浪费消除 | 单位经济学+预留策略 | 成本异常自检测 | ___/10 |
| **SecOps** | IAM最小权限 | 网络隔离+加密 | 零信任+事件响应Runbook | 安全态势自评估 | ___/10 |
| **AIOps** | ≥4异常模式 | 委托矩阵+知识库 | SLO+变更关联+容量预测 | 混沌工程+韧性评分 | ___/10 |
| **Sustainability** | 资源利用率文档 | 碳效率指标 | 绿色计算建议 | 碳足迹优化 | ___/10 |

**综合评分**: 总分 / 90 = ___% 
**达标要求**: L1 ≥ 40%, L2 ≥ 55%, L3 ≥ 70%, L4 ≥ 85%
```

### 7.3 Gap Analysis Template

```markdown
## Gap Analysis — [Product]

| 维度 | 当前分数 | 目标等级 | 差距项 | 优先修复 | 预计工时 |
|------|---------|---------|--------|---------|---------|
| Security | 5/10 | L2(6) | 缺少合规自动检查 | P0 | 2h |
| Cost | 4/10 | L2(5) | 缺少闲置检测模式 | P1 | 1h |
| ... | ... | ... | ... | ... | ... |
```

---

## 8. Cross-Pillar Trade-off Matrix (跨支柱冲突权衡矩阵)

### 8.1 已识别的支柱冲突

| 冲突场景 | 冲突支柱 | 具体表现 | 推荐权衡策略 | 决策原则 |
|---------|---------|---------|-------------|---------|
| 全审计日志 vs 成本 | Security ↔ Cost | CTS全量审计增加存储成本和API调用量 | 关键操作必审计, 只读操作抽样审计 | 安全优先, 成本可控 |
| Spot实例 vs 稳定性 | Cost ↔ Stability | 竞价实例可回收, 影响服务连续性 | 无状态/批处理用Spot, 有状态用包年包月 | 稳定性优先 |
| 最大加密 vs 性能 | Security ↔ Performance | TDE/全盘加密增加CPU/IO开销 | 性能敏感场景评估加密范围 | 敏感数据必加密, 非敏感可选择性加密 |
| 强隔离 vs 效率 | Security ↔ Efficiency | VPC隔离+安全组增加网络管理复杂度 | 自动化安全组管理, IaC模板化 | 安全基线不降, 管理复杂度靠自动化 |
| 高可用 vs 成本 | Stability ↔ Cost | 多AZ部署增加2×资源成本 | 生产双AZ, 测试单AZ | 生产稳定性优先 |
| 全监控 vs 成本 | Performance ↔ Cost | 高频指标采集增加CES调用量 | 关键指标1min, 一般指标5min | SLO相关指标高频, 其他标准频率 |
| 严格权限 vs 效率 | Security ↔ Efficiency | 最小权限增加权限申请流程 | 预定义角色模板, 自动权限申请 | 最小权限不妥协, 但流程可自动化 |

### 8.2 Trade-off Decision Template

```markdown
## Trade-off Decision — [Scenario]

### 冲突描述
[描述两个支柱之间的冲突]

### 影响分析
| 选项 | 对支柱A影响 | 对支柱B影响 | 成本影响 | 风险 |
|------|-----------|-----------|---------|------|
| 选项1: 优先A | +2 | -1 | — | ... |
| 选项2: 优先B | -1 | +2 | — | ... |
| 选项3: 折中 | +1 | +1 | +10% | ... |

### 决策
选择: [选项N]
理由: [记录决策理由]
ADR编号: [ADR-XXX]
```

---

## 9. Architecture Decision Records (架构决策记录 ADR)

### 9.1 ADR 模板

Every generated skill SHOULD maintain an ADR log for significant architectural decisions:

```markdown
# ADR-[N]: [Decision Title]

## Status
[Proposed | Accepted | Deprecated | Superseded by ADR-XXX]

## Context
[What is the issue that we're seeing that is motivating this decision?]

## Decision
[What is the change that we're proposing/making?]

## Consequences
### Positive
- [Benefit 1]
- [Benefit 2]

### Negative
- [Trade-off 1]
- [Trade-off 2]

### Neutral
- [Side effect 1]

## Related Pillars
- [Security/Stability/Cost/Efficiency/Performance] — [Impact description]

## Compliance
- [ ] Trade-off documented per Section 8
- [ ] No pillar reduced below L1 minimum
```

### 9.2 Common ADRs for Generated Skills

| ADR | Typical Decision | Affected Pillars |
|-----|-----------------|-----------------|
| ADR-1 | CLI-first vs SDK-only execution path | Efficiency, Performance |
| ADR-2 | Multi-AZ requirement level (required vs recommended) | Stability, Cost |
| ADR-3 | Encryption scope (all data vs sensitive only) | Security, Performance, Cost |
| ADR-4 | Monitoring frequency (1min vs 5min) | Performance, Cost |
| ADR-5 | Spot instance eligibility | Cost, Stability |

---

## 10. Efficiency Pillar Enhancement (效率支柱增强)

### 10.1 IaC (Infrastructure as Code) 集成

```markdown
## Efficiency — IaC Integration

### Terraform / Ansible 集成模板
| 操作 | CLI命令 | Terraform等价 | Ansible Module |
|------|---------|--------------|----------------|
| 创建实例 | `hcloud [product] create` | `huaweicloud_[product]_instance` | `huaweicloud.[product]` |
| 查询实例 | `hcloud [product] describe` | `data "huaweicloud_[product]"` | — |
| 修改配置 | `hcloud [product] modify` | Resource update | `huaweicloud.[product]_config` |
| 删除实例 | `hcloud [product] delete` | `resource "..." { count = 0 }` | — |

### GitOps 工作流
1. 代码提交 → PR → Review → Merge
2. Merge触发CI/CD Pipeline
3. Terraform Plan → 审批 → Apply
4. CTS记录变更 → 自动验证 → 巡检报告
```

### 10.2 CI/CD Pipeline 模板

```yaml
# .gitlab-ci.yml 示例
stages:
  - validate
  - plan
  - apply
  - verify

validate:
  stage: validate
  script:
    - hcloud [product] describe --region $HW_REGION_ID --output json | jq '.status'

plan:
  stage: plan
  script:
    - echo "Dry-run: hcloud [product] create --dry-run"

apply:
  stage: apply
  script:
    - hcloud [product] create --region $HW_REGION_ID
  when: manual  # 需要人工确认

verify:
  stage: verify
  script:
    - hcloud [product] describe --region $HW_REGION_ID --output json | jq '.status == "ACTIVE"'
```

---

## 11. Sustainability (可持续性 / 绿色计算)

### 11.1 碳效率考量

```markdown
## Sustainability Assessment

### 资源碳效率指标
| 指标 | 计算方式 | 优化方向 |
|------|---------|---------|
| 碳强度 | 碳排放量 / 业务产出 | 选择低碳Region, 提升资源利用率 |
| 资源利用率 | 平均CPU利用率 / 峰值CPU利用率 | 闲置资源释放, 弹性伸缩 |
| PUE (数据中心) | 数据中心总能耗 / IT设备能耗 | 选择华为云绿色数据中心 |
| 服务器寿命 | 服务器运行时长 / 设计寿命 | 优化替换周期, 减少电子废物 |

### 绿色计算建议
- 选择低碳区域: 华为云乌兰察布数据中心 (PUE < 1.2)
- 右-sizing: 过度配置 = 能源浪费, 闲置资源及时释放
- 弹性调度: 非高峰时段自动缩容, 减少无效能耗
- 存储分层: 冷数据迁移到低频存储, 减少高性能存储能耗
- 批处理优化: 将分散任务集中执行, 减少空转
- 生命周期管理: 自动清理临时资源, 避免长期闲置消耗能源
```

### 11.2 Sustainability Checklist (P2 — Nice to Have)

- [ ] 碳效率指标纳入监控面板
- [ ] 低碳Region选择建议
- [ ] 闲置资源自动释放策略 (节能)
- [ ] 存储分层建议 (热/温/冷)
- [ ] 弹性调度建议 (非高峰缩容)

---

## 12. Compliance Checklists

### 12.1 P0 — Must Pass

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
- [ ] **FinOps — Unit Economics:** At least 1 unit cost metric defined (cost/request or cost/vCPU)
- [ ] **FinOps — Anomaly Detection:** Cost anomaly detection rule documented

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

#### Well-Architected
- [ ] **Maturity Scorecard:** Self-assessment scorecard completed
- [ ] **Cross-Pillar Conflicts:** Trade-off matrix reviewed for known conflicts

### 12.2 P1 — Should Pass

- [ ] **FinOps — Budget:** Budget alert integration documented
- [ ] **FinOps — Reserved Coverage:** RI/包年包月覆盖率 analysis template
- [ ] **FinOps — TCO Model:** Total Cost of Ownership model documented
- [ ] **SecOps — MFA:** MFA requirement for interactive operations documented
- [ ] **SecOps — Compliance:** Alignment with 等保2.0 / GDPR where applicable
- [ ] **SecOps — Zero Trust:** Zero Trust Architecture alignment guidance (upgraded from P2)
- [ ] **SecOps — Incident Response:** Security incident response runbook defined
- [ ] **SecOps — Supply Chain:** SDK dependency security and SBOM guidance
- [ ] **SecOps — Key Lifecycle:** KMS key lifecycle management strategy
- [ ] **Five Pillars — Multi-AZ:** Cross-AZ deployment recommendation
- [ ] **Five Pillars — DR Runbook:** Phase 1/2/3 structure
- [ ] **Five Pillars — Auto-Scaling:** Scaling trigger thresholds documented
- [ ] **AIOps — Proactive:** Scheduled巡检 workflow defined
- [ ] **AIOps — Self-Healing:** Automated recovery patterns documented
- [ ] **AIOps — SLO/SLI:** At least 1 SLO with Error Budget and burn rate alerting
- [ ] **AIOps — Change Correlation:** CTS-based change-anomaly correlation workflow
- [ ] **AIOps — Capacity Forecast:** 30-day capacity prediction methodology
- [ ] **Efficiency — IaC:** Terraform/Ansible integration template documented
- [ ] **Architecture — ADR:** Architecture Decision Records for key decisions

### 12.3 P2 — Nice to Have

- [ ] **FinOps — Showback:** Cost report generation template
- [ ] **FinOps — Unit Economics Advanced:** Full unit economics dashboard (≥ 3 metrics)
- [ ] **SecOps — Audit:** Audit log integration (CTS/Cloud Trace Service)
- [ ] **SecOps — CSPM:** Cloud Security Posture Management integration
- [ ] **AIOps — ML Prediction:** ML-based anomaly prediction integration
- [ ] **AIOps — Chaos Engineering:** Fault injection experiment design documented
- [ ] **AIOps — Resilience Score:** Product-specific resilience scoring model
- [ ] **AIOps — Diagnosis Confidence:** Confidence score model with uncertainty declaration
- [ ] **Auto-Remediation:** Skills auto-suggest fixes for common violations
- [ ] **Sustainability — Carbon Efficiency:** Carbon intensity metrics and green computing guidance
- [ ] **Sustainability — Green Region:** Low-carbon Region selection recommendations
- [ ] **Efficiency — GitOps:** Full GitOps workflow with CI/CD pipeline template

---

*This assessment specification is mandatory. All generated skills MUST pass P0 checklists for their applicable pillars. Integration depth depends on the skill's primary purpose.*
