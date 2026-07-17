# hcloud-skills

> **[English](README.md)** | **中文**

华为云（Huawei Cloud）运维 Agent Skills 集合。

## 目录

- [概述](#概述)
- [核心价值](#核心价值)
- [项目结构](#项目结构)
- [三支柱运维体系](#三支柱运维体系)
  - [FinOps（财务运营）](#finops财务运营)
  - [SecOps（安全运营）](#secops安全运营)
  - [AIOps（智能运营）](#aiops智能运营)
- [快速开始](#快速开始)
  - [1. 安装 Huawei Cloud CLI](#1-安装-huawei-cloud-cli)
  - [2. 配置凭证](#2-配置凭证)
  - [3. 使用现有 Skills](#3-使用现有-skills)
  - [4. 生成新 Skill](#4-生成新-skill)
- [可用 Skills](#可用-skills)
- [华为云服务映射](#华为云服务映射)
- [生成质量门](#生成质量门)
- [参考资源](#参考资源)

## 概述

本项目是华为云运维 Agent Skills 的生成器与集合。提供云产品的自动化运维、监控、成本管理、安全治理和智能诊断能力。

> **Skills Farm 是一套 Meta Skill（元技能）体系**——将运维知识转化为结构化的、AI Agent 可解析、可执行、可验证的声明式规范。

## 核心价值

| 特性 | 说明 |
|------|------|
| **三支柱集成** | FinOps（财务运营）+ SecOps（安全运营）+ AIOps（智能运营）内建于每个 Skill |
| **占位符机制** | `{{env.*}}`（环境变量）、`{{user.*}}`（用户输入）、`{{output.*}}`（输出捕获），实现人机双通道 |
| **职责委托** | `SHOULD/SHOULD NOT Use` 定义边界，跨产品操作自动委派 |
| **生成器** | 基于 OpenAPI 规范自动生成 Skill 框架模板，支持人工审核和完善 |
| **CLI-first 执行** | 优先使用 `hcloud` CLI，不支持时 JIT 构建 Go SDK 脚本 |
| **安全机制** | 凭证隔离（`{{env.*}}` 不暴露）、操作安全门（删除/恢复需确认） |
| **卓越架构** | 五支柱（安全、稳定、成本、效率、性能）+ FinOps + SecOps + AIOps 全覆盖 |

## 项目结构

```
hcloud-skills/
├── README.md
├── LICENSE
├── huaweicloud-billing-ops/              # 费用中心 Skill（FinOps）
│   ├── SKILL.md                          # 主文件：账单查询、费用分析、预算告警、优化挖掘、闭环跟踪
│   ├── references/
│   │   ├── core-concepts.md              # BSS 架构与计费模型
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # BSS 监控与闭环跟踪
│   │   ├── integration.md                # 跨技能 FinOps 委派
│   │   ├── knowledge-base.md             # 费用故障模式知识库
│   │   └── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
├── huaweicloud-skill-generator/          # Skill 生成器（Meta Skill）
│   ├── SKILL.md                          # 生成器主文件
│   ├── assets/
│   │   ├── eval_queries.json             # 触发准确率评估查询
│   │   └── example-config.yaml           # 配置示例
│   └── references/
│       ├── huaweicloud-skill-template.md # SKILL.md 模板
│       ├── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│       ├── aiops-best-practices.md        # AIOps 最佳实践
│       ├── governance-and-adversarial-review.md # 治理与对抗审查
│       ├── enhanced-self-healing-framework.md   # 自愈框架
│       ├── execution-environment.md       # 执行环境配置
│       ├── cli-behavior.md               # CLI 行为参考
│       ├── user-experience-spec.md        # UX 规范
│       ├── optimization-analysis.md       # 优化分析框架
│       └── prompt-library.md             # 提示词手册
├── huaweicloud-ces-ops/                  # 云监控服务 Skill
│   ├── SKILL.md                          # 主文件：告警规则、指标查询、仪表盘
│   ├── references/
│   │   ├── core-concepts.md              # CES 架构与核心概念
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # CES 自监控模式
│   │   ├── integration.md                # JIT SDK 集成
│   │   ├── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   │   └── idempotency-checklist.md      # 幂等性检查清单
│   └── assets/
│       └── eval_queries.json             # 触发准确率评估查询
├── huaweicloud-vpc-ops/                  # 虚拟私有云 Skill
│   ├── SKILL.md                          # 主文件：VPC、子网、安全组、EIP、NAT
│   ├── references/
│   │   ├── core-concepts.md              # VPC 架构与网络概念
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # 网络监控模式
│   │   ├── integration.md                # JIT SDK 集成
│   │   ├── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   │   └── idempotency-checklist.md      # 幂等性检查清单
│   └── assets/
│       └── eval_queries.json             # 触发准确率评估查询
├── huaweicloud-iam-ops/                  # 身份与访问管理 Skill
│   ├── SKILL.md                          # 主文件：用户、用户组、策略、委托、AK/SK、MFA
│   ├── references/
│   │   ├── core-concepts.md              # IAM 架构与身份模型
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # CTS 事件监控模式
│   │   ├── integration.md                # 跨技能委托与 SDK 集成
│   │   └── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
├── huaweicloud-dcs-ops/                  # 分布式缓存 Skill
│   ├── SKILL.md                          # 主文件：实例生命周期、备份/恢复、扩容、密码重置、白名单
│   ├── references/
│   │   ├── core-concepts.md              # DCS 架构与核心概念
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # DCS 监控模式
│   │   ├── integration.md                # JIT SDK 集成与跨技能委托
│   │   ├── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   │   └── knowledge-base.md             # 故障模式知识库
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
├── huaweicloud-obs-ops/                  # 对象存储 Skill
│   ├── SKILL.md                          # 主文件：桶/对象生命周期、ACL、版本控制、CDN集成、静态网站
│   ├── references/
│   │   ├── core-concepts.md              # OBS 架构与存储类别
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI/obsutil 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # OBS 监控模式
│   │   ├── integration.md                # JIT SDK 集成与跨技能委托
│   │   ├── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   │   └── knowledge-base.md             # 故障模式知识库
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
├── huaweicloud-ecs-ops/                  # 弹性云服务器 Skill
│   ├── SKILL.md                          # 主文件：实例生命周期、磁盘、快照、CloudShell远程执行
│   ├── references/
│   │   ├── core-concepts.md              # ECS 架构与核心概念
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # ECS 监控模式
│   │   ├── observability.md              # 可观测性集成
│   │   ├── integration.md                # JIT SDK 集成与跨技能委托
│   │   ├── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   │   └── knowledge-base.md             # 故障模式知识库
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
├── huaweicloud-rds-ops/                  # 云数据库 RDS Skill
│   ├── SKILL.md                          # 主文件：实例生命周期、备份/恢复、参数管理、性能监控
│   ├── references/
│   │   ├── core-concepts.md              # RDS 架构与核心概念
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # RDS 监控模式
│   │   ├── observability.md              # 可观测性集成
│   │   ├── integration.md                # JIT SDK 集成与跨技能委托
│   │   ├── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   │   └── idempotency-checklist.md      # 幂等性检查清单
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
├── huaweicloud-elb-ops/                  # 弹性负载均衡 Skill
│   ├── SKILL.md                          # 主文件：负载均衡器、监听器、后端池、健康检查
│   ├── references/
│   │   ├── core-concepts.md              # ELB 架构与核心概念
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # ELB 监控模式
│   │   ├── integration.md                # JIT SDK 集成与跨技能委托
│   │   └── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
├── huaweicloud-cce-ops/                  # 云容器引擎 Skill
│   ├── SKILL.md                          # 主文件：集群、节点、节点池、插件管理
│   ├── references/
│   │   ├── core-concepts.md              # CCE 架构与核心概念
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # CCE 监控模式
│   │   ├── observability.md              # 可观测性集成
│   │   ├── integration.md                # JIT SDK 集成与跨技能委托
│   │   ├── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   │   └── knowledge-base.md             # 故障模式知识库
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
├── huaweicloud-cts-ops/                  # 云审计服务 Skill
│   ├── SKILL.md                          # 主文件：审计追踪、事件收集、追踪查询、诊断分析
│   ├── references/
│   │   ├── core-concepts.md              # CTS 架构与核心概念
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # CTS 监控模式
│   │   ├── integration.md                # JIT SDK 集成与跨技能委托
│   │   ├── aiops-best-practices.md        # AIOps 最佳实践
│   │   └── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
├── huaweicloud-css-ops/                  # 云搜索服务 Skill
│   ├── SKILL.md                          # 主文件：集群生命周期、快照管理、词典管理、配置管理
│   ├── references/
│   │   ├── core-concepts.md              # CSS 架构与核心概念
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # CSS 监控模式
│   │   ├── integration.md                # JIT SDK 集成与跨技能委托
│   │   ├── knowledge-base.md             # 故障模式知识库
│   │   ├── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   │   ├── user-experience-spec.md        # UX 规范
│   │   └── references/advanced/           # AIOps/SecOps/FinOps 深度分析
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
├── huaweicloud-dms-ops/                  # 分布式消息服务 Skill
│   ├── SKILL.md                          # 主文件：Kafka/RabbitMQ实例生命周期、Topic/Queue管理、消费组、消息查询
│   ├── references/
│   │   ├── core-concepts.md              # DMS 架构与核心概念
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # DMS 监控模式
│   │   ├── integration.md                # JIT SDK 集成与跨技能委托
│   │   ├── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   │   └── idempotency-checklist.md      # 幂等性检查清单
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
├── huaweicloud-cbr-ops/                  # 云备份 Skill
│   ├── SKILL.md                          # 主文件：备份存储库、备份策略、备份执行/恢复、跨区域复制
│   ├── references/
│   │   ├── core-concepts.md              # CBR 架构与核心概念
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # CBR 监控模式
│   │   ├── integration.md                # JIT SDK 集成与跨技能委托
│   │   ├── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   │   └── idempotency-checklist.md      # 幂等性检查清单
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
├── huaweicloud-swr-ops/                  # 容器镜像服务 Skill
│   ├── SKILL.md                          # 主文件：组织管理、仓库管理、镜像标签管理、保留策略、跨区域同步
│   ├── references/
│   │   ├── core-concepts.md              # SWR 架构与核心概念
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # SWR 监控模式
│   │   ├── integration.md                # JIT SDK 集成与跨技能委托
│   │   ├── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   │   └── idempotency-checklist.md      # 幂等性检查清单
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
├── huaweicloud-gaussdb-ops/              # GaussDB Skill
│   ├── SKILL.md                          # 主文件：实例生命周期、备份/恢复、参数模板、数据库/账户管理
│   ├── references/
│   │   ├── api-navigation.md             # API 导航与调用规范
│   │   ├── cli-syntax-reference.md       # CLI 语法参考
│   │   ├── cost-optimization.md          # 成本优化指南
│   │   ├── security-best-practices.md    # 安全最佳实践
│   │   ├── aiops-patterns.md             # AIOps 模式
│   │   ├── common-faults.md              # 常见故障处理
│   │   ├── error-handling.md             # 错误处理规范
│   │   └── safety-gates.md               # 安全门控
│   └── assets/
│       ├── example-config.yaml           # 配置示例
│       └── example-output.json           # 输出示例
├── huaweicloud-hss-ops/                  # 主机安全服务 Skill
│   ├── SKILL.md                          # 主文件：主机管理、资产采集、告警事件、漏洞管理、基线检查
│   ├── references/
│   │   ├── api-navigation.md             # API 导航与调用规范
│   │   ├── cli-syntax-reference.md       # CLI 语法参考
│   │   ├── cost-optimization.md          # 成本优化指南
│   │   ├── security-best-practices.md    # 安全最佳实践
│   │   ├── aiops-patterns.md             # AIOps 模式
│   │   ├── common-faults.md              # 常见故障处理
│   │   ├── error-handling.md             # 错误处理规范
│   │   └── safety-gates.md               # 安全门控
│   └── assets/
│       ├── example-config.yaml           # 配置示例
│       └── example-output.json           # 输出示例
├── huaweicloud-waf-ops/                  # Web应用防火墙 Skill
│   ├── SKILL.md                          # 主文件：策略、规则、域名、证书、攻击事件、引用表
│   ├── references/
│   │   ├── api-navigation.md             # API 导航与调用规范
│   │   ├── cli-syntax-reference.md       # CLI 语法参考
│   │   ├── cost-optimization.md          # 成本优化指南
│   │   ├── security-best-practices.md    # 安全最佳实践
│   │   ├── aiops-patterns.md             # AIOps 模式
│   │   ├── common-faults.md              # 常见故障处理
│   │   ├── error-handling.md             # 错误处理规范
│   │   └── safety-gates.md               # 安全门控
│   └── assets/
│       ├── example-config.yaml           # 配置示例
│       └── example-output.json           # 输出示例
├── huaweicloud-lts-ops/                  # 云日志服务 Skill
│   ├── SKILL.md                          # 主文件：日志组/日志流生命周期、日志搜索/查询、日志转储、结构化解析
│   ├── references/
│   │   ├── core-concepts.md              # LTS 架构与核心概念
│   │   ├── api-sdk-usage.md              # API 与 SDK 使用
│   │   ├── cli-usage.md                  # CLI 命令映射
│   │   ├── troubleshooting.md            # 故障排查指南
│   │   ├── monitoring.md                 # LTS 监控模式
│   │   ├── integration.md                # JIT SDK 集成与跨技能委托
│   │   ├── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
│   │   └── idempotency-checklist.md      # 幂等性检查清单
│   └── assets/
│       ├── eval_queries.json             # 触发准确率评估查询
│       └── example-config.yaml           # 配置示例
└── huaweicloud-functiongraph-ops/        # 函数工作流 Skill
    ├── SKILL.md                          # 主文件：函数生命周期、触发器、版本管理、诊断
    ├── references/
    │   ├── core-concepts.md              # FunctionGraph 架构与核心概念
    │   ├── api-sdk-usage.md              # API 与 SDK 使用
    │   ├── troubleshooting.md            # 故障排查指南
    │   ├── monitoring.md                 # FunctionGraph 监控模式
    │   ├── integration.md                # JIT SDK 集成与跨技能委托
    │   └── well-architected-assessment.md # 五支柱 + FinOps + SecOps + AIOps
    └── assets/
        ├── eval_queries.json             # 触发准确率评估查询
        └── example-config.yaml           # 配置示例
```

## 三支柱运维体系

### FinOps（财务运营）

| 能力 | 说明 |
|------|------|
| 成本可见性 | 计费模式对比（按需 vs 包年包月 vs 竞价）、成本标签策略、费用中心集成 |
| 成本优化 | 闲置资源检测、适配矩阵（利用率→推荐）、生命周期成本管理 |
| 成本问责 | 预算告警（80%/90%/100%阈值）、成本中心分摊 |

### SecOps（安全运营）

| 能力 | 说明 |
|------|------|
| 身份安全 | IAM 最小权限、AK/SK 轮换（90天）、MFA 强制、凭证委托 |
| 网络安全 | VPC Endpoint 隔离、安全组最佳实践、DDoS 防护 |
| 数据安全 | KMS 加密、TDE 透明加密、审计日志（≥180天）、数据泄露防护 |
| 威胁检测 | HSS 主机安全集成、WAF 联动、漏洞扫描 |

### AIOps（智能运营）

| 能力 | 说明 |
|------|------|
| 多指标关联分析 | ≥ 4 种异常模式（资源压力、趋势、突变、关联） |
| 跨技能诊断委托 | 命名空间 → 主/次诊断 Skill 路由矩阵 |
| 知识库 | ≥ 3 产品故障模式 + ≥ 2 跨产品级联故障 |
| 告警风暴处理 | 频率检测、聚合抑制、根资源识别 |
| 主动巡检 | 发现→采集→检测→诊断→报告 闭环 |
| 自愈框架 | 预检→智能下载→安装→验证，多级降级路径 |

## 快速开始

### 1. 安装 Huawei Cloud CLI

**One-click install (Linux):**
```bash
curl -sSL https://cn-north-4.myhuaweicloud.com/cli/latest/hcloud_install.sh -o ./hcloud_install.sh && bash ./hcloud_install.sh -y
```

**macOS:**
```bash
curl -sSL https://ap-southeast-3-hwcloudcli.obs.ap-southeast-3.myhuaweicloud.com/cli/latest/hcloud_install.sh -o ./hcloud_install.sh && bash ./hcloud_install.sh -y
```

**Verify:**
```bash
hcloud version
# Current KooCLI version: 4.1.6
```

### 2. 配置凭证

```bash
export HW_ACCESS_KEY_ID="your-access-key-id"
export HW_SECRET_ACCESS_KEY="your-secret-access-key"
export HW_REGION_ID="cn-north-4"
export HW_PROJECT_ID="your-project-id"
```

### 3. 使用现有 Skills

**CES 云监控操作** — 引用 `huaweicloud-ces-ops`：
```
"创建告警规则：当ECS实例CPU使用率超过80%时告警，通过短信通知"
"查询实例i-abc123最近1小时的CPU使用率"
"列出cn-north-4区域的所有告警规则"
```

**BSS 费用中心操作** — 引用 `huaweicloud-billing-ops`：
```
"查询本月的账单总览和按产品汇总"
"分析过去30天内是否存在闲置ECS实例（CPU使用率<5%）"
"创建预算告警，当本月支出超过预算80%时通知"
"给出下个月云资源采购建议——哪些按需实例应该转包年包月"
```

**VPC 网络操作** — 引用 `huaweicloud-vpc-ops`：
```
"创建VPC，CIDR为10.0.0.0/16，并创建生产子网"
"添加安全组规则：允许10.0.0.0/8访问端口22"
"创建弹性公网IP并绑定到ECS实例"
"配置NAT网关使私有子网可以访问外网"
```

### 4. 生成新 Skill

在 Agent Runtime 中引用生成器，然后提供提示词：

> "生成 huaweicloud-ecs-ops Skill，核心功能：实例生命周期管理、磁盘、快照，包含 FinOps 成本优化和 SecOps 安全治理"

## 可用 Skills

> 以下 Skills 可通过 Agent Runtime 引用，用于华为云指定产品的运维操作。

| Skill 名称 | 产品 | 核心功能 | 状态 |
|-----------|------|---------|------|
| `huaweicloud-billing-ops` | 费用中心 (BSS) | 账单查询、费用分析、预算告警、优化挖掘（闲置/Reserved/Sizing）、闭环跟踪、成熟度自评 | ✅ 已生成 |
| `huaweicloud-ces-ops` | 云监控服务 (CES) | 告警规则、指标查询、仪表盘、事件监控 | ✅ 已生成 |
| `huaweicloud-vpc-ops` | 虚拟私有云 (VPC) | VPC/子网/安全组/EIP/带宽/NAT网关/VPC对等连接 | ✅ 已生成 |
| `huaweicloud-ecs-ops` | 弹性云服务器 (ECS) | 实例生命周期、磁盘、快照、CloudShell远程执行 | ✅ 已生成 |
| `huaweicloud-rds-ops` | 云数据库 RDS | 实例、备份、恢复、参数管理、性能监控 | ✅ 已生成 |
| `huaweicloud-elb-ops` | 弹性负载均衡 (ELB) | 监听器、后端池、健康检查 | ✅ 已生成 |
| `huaweicloud-cce-ops` | 云容器引擎 (CCE) | 集群、节点、节点池、插件管理 | ✅ 已生成 |
| `huaweicloud-dcs-ops` | 分布式缓存 (DCS) | 实例生命周期、备份/恢复、扩容、密码重置、白名单 | ✅ 已生成 |
| `huaweicloud-cts-ops` | 云审计服务 (CTS) | 审计追踪、事件收集、追踪查询、诊断分析 | ✅ 已生成 |
| `huaweicloud-css-ops` | 云搜索服务 (CSS) | Elasticsearch/OpenSearch 集群生命周期、快照管理、词典管理、配置管理 | ✅ 已生成 |
| `huaweicloud-functiongraph-ops` | 函数工作流 (FunctionGraph) | 函数生命周期、触发器、版本管理、诊断 | ✅ 已生成 |
| `huaweicloud-iam-ops` | 身份与访问管理 (IAM) | 用户、用户组、策略、委托、AK/SK、MFA | ✅ 已生成 |
| `huaweicloud-obs-ops` | 对象存储 (OBS) | 桶/对象生命周期、ACL、版本控制、生命周期规则、CDN集成、静态网站 | ✅ 已生成 |
| `huaweicloud-dms-ops` | 分布式消息服务 (DMS) | Kafka/RabbitMQ实例生命周期、Topic/Queue管理、消费组、消息查询、备份恢复 | ✅ 已生成 |
| `huaweicloud-cbr-ops` | 云备份 (CBR) | 备份存储库、备份策略、备份执行/恢复、跨区域复制 | ✅ 已生成 |
| `huaweicloud-swr-ops` | 容器镜像服务 (SWR) | 组织管理、仓库管理、镜像标签管理、保留策略、跨区域同步、Docker集成 | ✅ 已生成 |
| `huaweicloud-hss-ops` | 主机安全服务 (HSS) | 主机管理、资产采集、告警事件、漏洞管理、基线检查、网页防篡改、容器安全 | ✅ 已生成 |
| `huaweicloud-waf-ops` | Web应用防火墙 (WAF) | 策略、规则、域名、证书、攻击事件、引用表 | ✅ 已生成 |
| `huaweicloud-lts-ops` | 云日志服务 (LTS) | 日志组/日志流生命周期、日志搜索/查询、日志转储(OBS/DMS)、结构化解析、仪表盘管理、日志告警配置 | ✅ 已生成 |
| `huaweicloud-gaussdb-ops` | GaussDB | 实例生命周期、备份/恢复、参数模板、数据库/账户管理、标签管理、企业项目管理、回收站 | ✅ 已生成 |

## 华为云服务映射

| 华为云服务 | 缩写 | Go SDK Package | 主要操作 |
|-----------|------|---------------|---------|
| 弹性云服务器 | ECS | `services/ecs/v2` | Create, Delete, Describe, Resize |
| 云数据库 RDS | RDS | `services/rds/v3` | Instance, Backup, Restore |
| 云监控服务 | CES | `services/ces/v1` | Alarm, Metric, Dashboard |
| 虚拟私有云 | VPC | `services/vpc/v3` | VPC, Subnet, SecurityGroup |
| 弹性负载均衡 | ELB | `services/elb/v3` | Listener, Pool, Health |
| 云容器引擎 | CCE | `services/cce/v3` | Cluster, Node, Addon |
| 分布式缓存 | DCS | `services/dcs/v2` | Instance, Backup, Resize |
| 主机安全服务 | HSS | `services/hss/v5` | Host, Vulnerability, Event |
| Web应用防火墙 | WAF | `services/waf/v1` | Policy, Rule, Domain |
| 云日志服务 | LTS | `services/lts/v2` | Log Group, Stream, Search |
| 对象存储 | OBS | `services/obs` | Bucket, Object, ACL |
| 身份与访问管理 | IAM | `services/iam/v3` | User, Group, Policy, Agency, Credential |
| 分布式消息服务 | DMS | `services/dms/v2` | Instance, Topic, Queue, Consumer Group, Backup |
| 云备份 | CBR | `services/cbr/v3` | Vault, Policy, Backup, Restore, Replication |
| 云搜索服务 | CSS | `services/css/v2` | Cluster, Snapshot, Dictionary, Config |
| 容器镜像服务 | SWR | `services/swr/v2` | Organization, Repository, Image, Tag, Retention |
| GaussDB for openGauss | GaussDB | `services/gaussdb/v3` | Instance, Backup, Template, Database/User, Quota, RecycleBin |

## 生成质量门

每个生成的 Skill 必须通过 **P0 强制检查清单**：

### 基础标准
- [ ] SHOULD/SHOULD NOT Use 触发条件完整
- [ ] 预检→执行→验证→恢复 流程完备
- [ ] ≥ 10 个产品错误码及恢复策略
- [ ] 破坏性操作安全门
- [ ] 凭证掩蔽（`***`）

### FinOps 检查
- [ ] 计费模式对比表
- [ ] 闲置资源检测模式
- [ ] 适配矩阵（利用率→推荐操作）
- [ ] 成本标签策略

### SecOps 检查
- [ ] IAM 最小权限表
- [ ] VPC/安全组隔离指导
- [ ] 加密（静态+传输中）文档
- [ ] HSS/WAF 威胁检测触发条件

### AIOps 检查
- [ ] ≥ 4 种异常模式及检测逻辑
- [ ] 跨技能委托矩阵
- [ ] 故障模式知识库
- [ ] 告警风暴聚合处理

## 参考资源

- [Huawei Cloud Go SDK](https://github.com/huaweicloud/huaweicloud-sdk-go-v3)
- [Huawei Cloud API Docs](https://support.huaweicloud.com/api/)
- [Huawei Cloud CLI](https://support.huaweicloud.com/hcli/index.html)
- [Huawei Cloud Well-Architected Framework](https://support.huaweicloud.com/topic/68733-1-I)
- [Agent Skills Open Specification](https://agentskills.io/specification)
- [AGENTS.md](AGENTS.md) — 仓库 Agent 与贡献者约定
- [README.md](README.md) — English README（精简目录树与 GCL / 本地校验说明）
