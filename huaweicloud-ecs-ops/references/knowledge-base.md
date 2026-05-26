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
| **ML特征** | `cpu_util` > 95, `load1/vCPU` > 1.0, 持续时间 > 600s |
| **训练窗口** | 10min sliding window |
| **异常分数阈值** | 0.85 (基于历史CPU分布) |
| **自动修复触发** | cpu_util > 98% AND duration > 5min → CloudShell: top -b -n 1 → kill STOP可疑进程 |

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
| **ML特征** | `diskUsage_percent` > 90, `write_iops` delta < -50%, `fill_rate` > baseline |
| **训练窗口** | 1h sliding window with rate acceleration detection |
| **异常分数阈值** | 0.90 (disk_usage + fill_rate_acceleration weighted) |
| **自动修复触发** | diskUsage > 95% AND fill_acceleration > 0 → CloudShell: rm -rf /var/log/*.old |

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
| **ML特征** | `mem_usedPercent` slope > 0.5%/min, `mem_monotonic_increase` count > 30 |
| **训练窗口** | 30min sliding window with ARIMA forecasting |
| **异常分数阈值** | 0.88 (slope_coefficient × time_weighted) |
| **自动修复触发** | mem_slope > 1.0%/min AND forecast(48h) > 95% → Alert + auto_heap_dump |

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
| **ML特征** | `metrics_missing` duration > 5min, `instance_state` == ACTIVE |
| **训练窗口** | Event correlation window (CTS audit + CES metric gap) |
| **异常分数阈值** | 0.92 (config_change_event + connectivity_loss correlation) |
| **自动修复触发** | CTS: security_group_modify AND CES: metrics_missing → Alert ops team |

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
| **ML特征** | `elb_connection_count` delta < -80%, `ecs_cpu_util` multi_instance ↓, `correlation_score` |
| **训练窗口** | Multi-service correlation window (ELB + ECS + RDS metrics) |
| **异常分数阈值** | 0.90 (cross_service_anomaly_aggregation) |
| **自动修复触发** | elb_health_check_fail AND ecs_sg_modified → Auto_restore_sg_rules |

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
| **ML特征** | `instance_type` == spot, `price_trend` rising, `market_price` > bid_price |
| **训练窗口** | Price prediction window (7d historical + market trend) |
| **异常分数阈值** | 0.95 (price_forecast_model confidence) |
| **自动修复触发** | spot_price_forecast > threshold → Pre-spin on-demand replacement |
