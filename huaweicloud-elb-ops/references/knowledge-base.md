# Knowledge Base — Huawei Cloud ELB Fault Patterns

> ML feature definitions and anomaly scoring belong in
> [`huaweicloud-ces-ops/references/advanced/aiops-patterns.md`](../../huaweicloud-ces-ops/references/advanced/aiops-patterns.md).
> This file focuses on ELB-specific diagnosis and remediation.

## Pattern: ELB-001 — 后端实例不健康导致 502

| Attribute | Content |
|-----------|---------|
| 触发指标 | `backendIngressPortStatus` = `abnormal` OR `ELB.502_error_rate` > 1% 持续 5min |
| 典型特征 | 客户端收到 502 Bad Gateway，ELB 健康检查状态显示后端实例 down |
| 关联指标 | `ELB.backendServerConnections` 下降, `ELB.active_connections` 可能正常但处理失败 |
| 根因 | 1. 后端应用崩溃/无响应 2. 安全组未放行 ELB 健康检查 IP 3. 后端端口配置错误 4. 后端实例负载过高 |
| 诊断步骤 | 1. `hcloud elb list backend-nodes --loadbalancer-id <lb_id>` 检查后端状态 2. 检查后端安全组规则(允许 100.125.0.0/16 健康检查) 3. `hcloud elb healthcheck --loadbalancer-id <lb_id>` 查看健康检查配置 4. 登录后端实例检查应用日志 |
| 修复方案 | 1. 立即: 重启异常后端应用或摘除故障节点 2. 修复安全组规则放行健康检查 3. 调整健康检查参数(超时/间隔) |
| 预防措施 | 配置后端实例多可用区部署，健康检查失败自动报警，设置健康检查阈值告警 |
| 自动修复触发 | `backendIngressPortStatus = abnormal` AND `duration > 3min` → 自动摘除故障节点 + 触发后端告警 |

## Pattern: ELB-002 — SSL证书过期/配置错误

| Attribute | Content |
|-----------|---------|
| 触发指标 | `ELB.ssl_error_rate` > 0.5% OR SSL证书剩余有效期 < 30天 |
| 典型特征 | 客户端报 SSL handshake failed，浏览器显示证书错误，安全连接无法建立 |
| 关联指标 | `ELB.backendIngressPortStatus` 正常但 HTTPS 请求失败, `ELB.ssl_connections` 下降 |
| 根因 | 1. 证书过期 2. 证书链不完整 3. 域名与证书不匹配 4. TLS 版本不兼容(客户端不支持) |
| 诊断步骤 | 1. `hcloud elb show-certificate --listener-id <listener_id>` 查看证书信息 2. 检查证书有效期和域名匹配 3. `openssl s_client -connect <elb_fip>:443` 测试 SSL 握手 4. 检查监听器 TLS 版本配置 |
| 修复方案 | 1. 立即: 更新过期证书 2. 补充完整证书链 3. 调整 TLS 版本兼容性(建议 TLS 1.2+) |
| 预防措施 | 证书到期前 30/15/7 天自动提醒，证书监控指标告警，使用 ACM 自动管理证书 |
| 自动修复触发 | `cert_expiry_days < 30` → 发送证书更新提醒; `ssl_error_rate > 1%` → 检查证书状态 |

## Pattern: ELB-003 — 连接超时（后端响应慢）

| Attribute | Content |
|-----------|---------|
| 触发指标 | `ELB.backendRequestLatency` > 30000ms OR `ELB.connection_timeout` > 100 持续 5min |
| 典型特征 | 客户端请求等待超时，后端实例 CPU/内存正常但响应缓慢，ELB 等待后端响应 |
| 关联指标 | `ELB.backendServerConnections` 积压, `ELB.active_connections` 正常但吞吐量下降 |
| 根因 | 1. 后端应用 SQL 慢查询 2. 后端连接池耗尽 3. 后端资源(CPU/磁盘)瓶颈 4. ELB 到后端网络延迟 |
| 诊断步骤 | 1. `hcloud ces metric-data query --namespace SYS.ELB --metric-name backendRequestLatency` 查看后端延迟 2. 检查后端数据库连接池状态 3. `hcloud elb list-backend-nodes --loadbalancer-id <lb_id>` 检查后端负载 4. 查看后端应用慢查询日志 |
| 修复方案 | 1. 立即: 扩容后端或重启应用释放连接池 2. 优化应用慢查询 3. 调整 ELB 调度算法(Leastconn) 4. 配置后端超时时间 |
| 预防措施 | 后端响应时间 SLO 告警，数据库连接池监控，慢查询定期优化，后端性能基线监控 |
| 自动修复触发 | `backendRequestLatency > 60s` AND `duration > 10min` → 触发后端性能告警 + 记录诊断信息 |

## Pattern: ELB-004 — 带宽瓶颈导致限流

| Attribute | Content |
|-----------|---------|
| 触发指标 | `ELB.throughput` 达到套餐上限 OR `ELB.bandwidth_usage` > 90% 持续 5min |
| 典型特征 | 带宽上限触发，新增连接被丢弃或延迟，大量请求排队，应用性能下降 |
| 关联指标 | `ELB.active_connections` 正常但 `ELB.throughput` 触顶, `ELB.connection_limit_rejected` > 0 |
| 根因 | 1. 业务流量超出 ELB 带宽套餐 2. 攻击流量(DDoS/CC) 3. 配置了共享带宽但超出限额 |
| 诊断步骤 | 1. `hcloud elb show-loadbalancer --id <lb_id>` 查看带宽配置 2. `hcloud ces metric-data query --namespace SYS.ELB --metric-name throughput` 对比带宽上限 3. 检查是否有异常流量来源 4. 分析流量成分(内网 vs 公网) |
| 修复方案 | 1. 立即: 临时升级带宽套餐或开启限速 2. 排查异常流量 3. 配置 CC 防护/DDoS 基础防护 4. 长期: 业务拆分到多个 ELB |
| 预防措施 | 带宽使用率 > 80% 告警，设置带宽自动扩展(企业级 ELB)，配置流量监控和异常检测 |
| 自动修复触发 | `bandwidth_usage > 90%` → 自动升级带宽或触发告警; `throughput > limit` → 启动限流保护 |

## Pattern: ELB-005 — 健康检查配置不当导致频繁摘挂

| Attribute | Content |
|-----------|---------|
| 触发指标 | `ELB.backendIngressPortStatus` 在 normal/abnormal 间频繁切换，切换周期 < 10min |
| 典型特征 | 后端实例状态不稳定，频繁上下线，部分请求失败，客户感知断续 |
| 关联指标 | `ELB.backendServerConnections` 波动大, 健康检查失败次数突增 |
| 根因 | 1. 健康检查间隔太短，后端响应延迟波动时被误判 2. 后端服务不稳定 3. 健康检查超时时间太短 4. 后端资源偶尔瓶颈 |
| 诊断步骤 | 1. `hcloud elb healthcheck-config --loadbalancer-id <lb_id>` 查看健康检查参数 2. 检查后端实例资源使用波动 3. 查看后端应用是否有 GC 或偶发阻塞 4. 分析健康检查失败的时间分布 |
| 修复方案 | 1. 调整健康检查间隔(建议 30s) 和超时时间(建议 10s) 2. 连续几次检查失败才摘除(建议 3 次) 3. 优化后端应用稳定性 |
| 预防措施 | 健康检查参数按业务特征调优，监控健康检查状态切换频率，设置最小健康检查间隔 |
| 自动修复触发 | `health_check_flap > 3 times/hour` → 告警 + 建议调整健康检查参数 |

## Pattern: ELB-006 — ELB 规格超限导致丢包

| Attribute | Content |
|-----------|---------|
| 触发指标 | `ELB.active_connections` 接近规格上限，或 `ELB.new_connection_per_second` 超限 |
| 典型特征 | 新建连接失败或延迟，现有连接正常，请求成功率下降 |
| 关联指标 | `ELB.connection_limit_rejected` > 0, CPU/带宽 可能未触顶但连接数超限 |
| 根因 | 1. 选择的 ELB 规格(connection limit)太低 2. 短连接频繁创建销毁 3. 连接复用率低 |
| 诊断步骤 | 1. `hcloud elb show-loadbalancer --id <lb_id>` 查看规格和当前连接数 2. `hcloud ces metric-data query --namespace SYS.ELB --metric-name active_connections` 对比规格上限 3. 分析连接模式(长连接 vs 短连接) |
| 修复方案 | 1. 立即: 升级 ELB 规格 2. 优化客户端使用连接池，提高连接复用 3. 切换到更高规格的增强型 ELB |
| 预防措施 | 连接数使用率 > 70% 告警，根据业务峰值提前扩容 ELB 规格 |
| 自动修复触发 | `active_connections > 80% of limit` → 发送规格扩容建议 |
