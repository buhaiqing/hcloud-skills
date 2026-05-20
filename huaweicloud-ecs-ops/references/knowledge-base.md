# Knowledge Base — Huawei Cloud ECS Fault Patterns

## Pattern: ECS-001 — CPU 满载导致服务不可用

| Attribute | Content |
|-----------|---------|
| 触发指标 | `cpu_util` > 95% 持续 10min |
| 典型特征 | 连接超时，SSH 响应极慢，CES 无内存异常 |
| 关联指标 | `load1` >> vCPU count, `mem_usedPercent` 正常 |
| 根因 | 1. 僵尸进程/死循环 2. Cron 任务爆炸 3. 恶意挖矿进程 |
| 诊断步骤 | 1. CloudShell: `top -c` 识别高占用进程 2. `ps -ef` 3. `cat /etc/crontab` 4. 检查陌生进程 |
| 修复方案 | 1. 立即: kill 异常进程 2. 长期: 限制用户CPU配额，HSS入侵检测 |
| 预防措施 | HSS主机安全基线检查，进程白名单限制 |

## Pattern: ECS-002 — 磁盘空间耗尽

| Attribute | Content |
|-----------|---------|
| 触发指标 | `diskUsage_percent` > 90% |
| 典型特征 | 应用无法写入日志，数据库报错，SSH可能断开 |
| 关联指标 | `write_iops` 突降后为0, `cpu_util` 正常 |
| 根因 | 1. 日志轮转未配置 2. 临时文件未清理 3. Core dump文件堆积 |
| 诊断步骤 | 1. `df -h` 2. `du -sh /var/log/*` 3. `find / -size +1G` 4. CloudShell清理 |
| 修复方案 | 1. 临时: 删除大文件 2. 长期: 配置logrotate, 日志输出到LTS |
| 预防措施 | 配置磁盘使用率告警 (80%预警)，LTS日志服务，core文件大小限制 |

## Pattern: ECS-003 — 内存泄漏趋势

| Attribute | Content |
|-----------|---------|
| 触发指标 | `mem_usedPercent` 单调上升，斜率 > 0.5%/min |
| 典型特征 | OOM Killer触发，Java应用heap持续增长 |
| 关联指标 | `cpu_util` 可能在GC时突增，swap使用率上升 |
| 根因 | 1. 应用内存泄漏 2. 数据库连接池未释放 3. 系统cache积累 |
| 诊断步骤 | 1. `free -m` 2. JVM: `jmap -heap <pid>` 3. `cat /proc/meminfo` |
| 修复方案 | 1. 临时: 重启应用释放内存 2. 长期: 修复代码泄漏，设置JVM -Xmx |
| 预防措施 | JVM堆配置最佳实践，heap dump自动触发，内存泄漏检测APM |

## Pattern: ECS-004 — 安全组误配导致实例不可达

| Attribute | Content |
|-----------|---------|
| 触发指标 | CES metrics消失(无数据上报), 无法SSH |
| 典型特征 | 实例状态ACTIVE但无法任何网络连接 |
| 关联指标 | 所有网络指标无数据, CPU正常 |
| 根因 | 1. 安全组规则修改/误删 2. 新实例绑定到错误安全组 3. NACL规则阻挡 |
| 诊断步骤 | 1. `hcloud vpc describe-security-group <sg_id>` 2. 检查入站规则 |
| 修复方案 | 1. 恢复安全组规则(SSH 22端口) 2. 验证应用端口规则 |
| 预防措施 | 安全组变更审批流, 默认安全组不直接修改 |

## Pattern: ECS-005 — 级联故障：ECS→ELB→应用

| Attribute | Content |
|-----------|---------|
| 触发指标 | ELB active_connection_count ↓ 同时 ECS cpu_util ↓ |
| 典型特征 | 多个ECS实例同时下线或负载异常 |
| 关联指标 | ELB 504错误率 ↑, ECS 502/503 ↑, RDS connections ↓ |
| 根因 | 1. 安全组阻止了ELB健康检查 2. 应用崩溃 3. 网络分区 |
| 诊断步骤 | 1. 检查ELB后端健康 2. 逐个检查ECS状态 3. 网络traceroute |
| 修复方案 | 1. 修复安全组ELB健康检查规则 2. 重启/修复应用 |
| 预防措施 | 安全组自动化测试(ELB→ECS健康检查放行), 多可用区部署 |

## Pattern: ECS-006 — 竞价实例被回收

| Attribute | Content |
|-----------|---------|
| 触发指标 | ECS 状态变为 `TERMINATED` 未主动操作 |
| 典型特征 | 实例突然消失，ELB健康检查失败 |
| 关联指标 | CES metric断崖式消失(无数据)，费用账单显示竞价实例 |
| 根因 | 1. 竞价价格超过市场价 2. 华为云资源回收 |
| 诊断步骤 | 1. 检查操作审计CTS事件 2. 检查竞价实例回收通知 |
| 修复方案 | 1. 立即: 使用按需实例替代 2. 长期: 配置竞价实例回收告警和自动替换 |
| 预防措施 | AS配置混合付费类型，竞价实例仅用于无状态可中断任务 |
