# Prompts — Huawei Cloud ELB

> **Purpose:** Structured prompts for ELB AIOps operations. Derived from `prompt-handbook-template.md`.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Backend Health Diagnosis
```
Diagnose ELB backend health for loadbalancer {{loadbalancer_id}}:
- Backend node status: {{backend_status}} (normal/abnormal)
- Health check failures: {{health_check_failures}} in past {{time_window}}
- Current connections: {{active_connections}}, throughput {{throughput}}Mbps
- Related alerts: {{alert_count}} active alerts
Determine root cause and recommend recovery actions.

Applicable CES metrics: SYS.ELB.backendIngressPortStatus, SYS.ELB.backendServerConnections, SYS.ELB.active_connections
```

### 1.2 SSL/TLS Error Analysis
```
Analyze SSL/TLS errors on ELB listener {{listener_id}}:
- SSL error rate: {{ssl_error_rate}}% (baseline: {{baseline_error_rate}}%)
- Certificate expiry: {{cert_expiry_days}} days remaining
- TLS version: {{tls_version}} (accepted: {{accepted_tls_versions}})
- Client报错: {{client_error_message}}
Identify root cause and remediation steps.

ELB SSL error patterns: cert expired, incomplete cert chain, domain mismatch, protocol mismatch
```

### 1.3 Latency Root Cause Analysis
```
Analyze ELB latency issue for {{loadbalancer_id}}:
- Backend request latency: {{backend_latency}}ms (SLO: {{slo_target}}ms)
- Frontend request latency: {{frontend_latency}}ms
- Network latency: {{network_latency}}ms
- Backend health: {{backend_health_status}}
- Current throughput: {{throughput}}Mbps
Correlate metrics to identify bottleneck location.

ELB latency metrics: SYS.ELB.backendRequestLatency, SYS.ELB.frontendRequestLatency, SYS.ELB.backendServerConnections
```

### 1.4 Throughput Bottleneck Diagnosis
```
Diagnose throughput bottleneck for ELB {{loadbalancer_id}}:
- Current throughput: {{current_throughput}}Mbps (limit: {{throughput_limit}}Mbps)
- Bandwidth usage: {{bandwidth_usage}}%
- Active connections: {{active_connections}}
- Packet rate: {{packet_rate}}pps
- ELB specification: {{elb_spec}}
Determine if bottleneck is bandwidth, connection limit, or instance spec.

ELB throughput limits: shared bandwidth vs dedicated bandwidth, per-spec connection limits
```

### 1.5 Connection Issue Diagnosis
```
Diagnose ELB connection issues for {{loadbalancer_id}}:
- New connections/sec: {{new_conn_per_sec}} (limit: {{conn_limit}}/sec)
- Active connections: {{active_connections}} (limit: {{conn_limit}})
- Connection timeout rate: {{connection_timeout_rate}}%
- Rejected connections: {{rejected_connections}}
- Backend response time: {{backend_response_time}}ms
Identify if issue is ELB spec, backend capacity, or network.

Common causes: ELB规格超限, 后端连接池耗尽, 健康检查失败导致流量不均
```

---

## 2. Root Cause Analysis Prompts

### 2.1 502 Error Root Cause
```
Perform root cause analysis for 502 errors on ELB {{loadbalancer_id}}:
- 502 error rate: {{error_rate}}% (threshold: 1%)
- Affected backend: {{affected_backends}}
- Health check status: {{health_check_status}}
- Backend response time: {{backend_response_time}}ms
- Recent changes: {{recent_changes}}
Rank possible causes with confidence scores.

Common 502 causes: 后端实例不健康, 后端响应超时, 安全组阻止健康检查, 后端应用崩溃
```

### 2.2 Intermittent Connection Failures
```
Analyze intermittent connection failures:
- Failure pattern: {{failure_pattern}} (time-based/percentage-based)
- Affected nodes: {{affected_nodes}}
- Health check history: {{health_check_history}}
- Backend resource utilization: CPU {{cpu}}%, Memory {{mem}}%
- Network: {{network_in}}Mbps in, {{network_out}}Mbps out
Identify if cause is network, backend instability, or configuration issue.

Common causes: 健康检查参数不当(间隔太短), 后端资源偶尔瓶颈, 网络抖动
```

### 2.3 SSL Handshake Failure RCA
```
Root cause analysis for SSL handshake failures:
- Failure rate: {{failure_rate}}%
- Failed TLS versions: {{failed_tls_versions}}
- Client hello versions: {{client_hello_versions}}
- Certificate subject: {{cert_subject}}
- Certificate chain: {{cert_chain_complete}} (complete/incomplete)
- Backend status: {{backend_status}}
Determine if cause is certificate, protocol compatibility, or configuration.

SSL handshake failure patterns: 证书过期, 证书链不完整, TLS版本不兼容, SNI配置错误
```

### 2.4 Performance Degradation Analysis
```
Analyze ELB performance degradation:
- Latency increase: {{latency_increase}}ms (baseline: {{baseline_latency}}ms)
- Throughput change: {{throughput_change}}% (was {{old_throughput}}, now {{new_throughput}})
- Error rate change: {{error_rate_change}}%
- Backend count: {{backend_count}}, healthy: {{healthy_backend_count}}
- Resource utilization trend: {{resource_trend}}
Identify primary bottleneck and recommend actions.

ELB性能降级常见原因: 后端负载不均, 带宽触顶, 连接数接近上限, 后端响应变慢
```

### 2.5 Cascading Failure Analysis
```
Analyze potential cascading failure from ELB {{loadbalancer_id}}:
- Primary symptom: {{primary_symptom}}
- Downstream services: {{downstream_services}} (ECS, RDS, Redis)
- Backend health trend: {{backend_health_trend}}
- Connection pool status: {{connection_pool_status}}
- Recent scaling events: {{scaling_events}}
Assess failure propagation risk and suggest containment actions.

级联故障路径: ELB后端全灭 → 流量切到备用 → 备用被打垮 → 整体不可用
```

---

## 3. Capacity Prompts

### 3.1 Bandwidth Capacity Planning
```
Evaluate bandwidth capacity for ELB {{loadbalancer_id}}:
- Current throughput: {{current_throughput}}Mbps (peak: {{peak_throughput}}Mbps)
- Bandwidth limit: {{bandwidth_limit}}Mbps
- Usage trend: {{usage_trend}}% weekly growth
- Projected exhaustion: {{exhaustion_date}} (at current growth rate)
- Available bandwidth tiers: {{available_tiers}}
Recommend scaling timeline and bandwidth tier.

华为云ELB带宽规格: 基础型(10Mbps-1Gbps), 增强型(1Gbps-10Gbps), 超级型(10Gbps+)
```

### 3.2 Connection Count Capacity
```
Assess connection capacity for ELB {{loadbalancer_id}}:
- Current active connections: {{active_connections}} (limit: {{conn_limit}})
- New connections/sec: {{new_conn_per_sec}} (limit: {{new_conn_limit}}/sec)
- Connection growth: {{conn_growth}}% weekly
- Peak projection: {{peak_connections}} at {{forecast_date}}
- Alternative: {{alternative_spec}}
Recommend ELB spec upgrade or architecture optimization.

连接数规格参考: 基础型1万, 增强型10万, 超级型100万
```

### 3.3 Backend Pool Sizing
```
Evaluate backend pool sizing for ELB {{loadbalancer_id}}:
- Current backend count: {{backend_count}}
- Average backend utilization: CPU {{avg_cpu}}%, Memory {{avg_mem}}%
- Request distribution: {{request_distribution}} (even/uneven)
- Failure impact: {{failure_impact}} backends down = {{impact_percent}}% capacity loss
- AZ distribution: {{az_distribution}}
Recommend optimal backend count and AZ distribution.

后端池容量规划: 单AZ故障时其余AZ需承受全部流量, 建议保留 1/N 余量
```

### 3.4 Listener Capacity Audit
```
Audit listener capacity for ELB {{loadbalancer_id}}:
- Protocol: {{protocol}} (HTTP/HTTPS/TCP/UDP)
- Current connections: {{connections}} (limit: {{limit}})
- Throughput: {{throughput}}Mbps (limit: {{throughput_limit}}Mbps)
- Cipher suite: {{cipher_suite}}
- TLS version support: {{tls_versions}}
Assess if current configuration meets capacity requirements.

监听器规格: 每ELB最多50个监听器, 每监听器对应一个端口和协议
```

### 3.5 Cost Optimization Scan
```
Scan ELB {{loadbalancer_id}} for cost optimization:
- Current cost: {{monthly_cost}}/month (type: {{pricing_type}})
- Utilization: {{avg_utilization}}% average over 30 days
- Idle time: {{idle_hours}} hours/day with < 5% utilization
- Alternative pricing: {{alternative_pricing}} (shared vs dedicated)
- Right-sizing candidate: {{right_size_candidate}}
Provide cost reduction recommendations with trade-offs.

ELB计费模式: 按规格计费(包月), 按流量计费, 按带宽计费
```

---

## 4. Availability Prompts

### 4.1 HA Configuration Review
```
Review HA configuration for ELB {{loadbalancer_id}}:
- AZ distribution: {{az_distribution}} ({{primary_az}} primary)
- Backend AZ coverage: {{backend_az_coverage}}
- Health check configuration: interval {{health_check_interval}}s, timeout {{health_check_timeout}}s
- Last AZ failure: {{last_az_failure}} (never/{{failure_date}})
- Failover time: {{failover_time}}s (target: < 60s)
Assess HA readiness and identify single points of failure.

ELB高可用架构: 多AZ部署, 健康检查自动摘除, 跨AZ流量均衡
```

### 4.2 Disaster Recovery Drill
```
Execute DR drill for ELB deployment:
- Primary: {{primary_region}}/{{primary_az}}
- DR: {{dr_region}}/{{dr_az}}
- RTO target: {{rto_target}} minutes
- RPO target: {{rpo_target}} minutes
- Test scenario: {{test_scenario}}
Validate failover procedures and document gaps.

DR验证要点: DNS切换时间, 后端注册速度, 健康检查恢复, 数据一致性
```

### 4.3 Maintenance Window Planning
```
Plan maintenance window for ELB {{loadbalancer_id}}:
- Maintenance type: {{maintenance_type}} (backend upgrade/config change)
- Impact duration: {{impact_duration}} minutes
- Traffic during window: {{traffic_during_maintenance}}% of normal
- Backend count: {{backend_count}}, healthy: {{healthy_count}}
- Rollback plan: {{rollback_plan}}
Assess impact and prepare traffic shift plan.

维护窗口策略: 先摘除要维护的后端, 验证流量切换, 执行维护, 逐步恢复
```

### 4.4 Failure Impact Assessment
```
Assess failure impact for ELB {{loadbalancer_id}}:
- Affected services: {{affected_services}}
- Current request rate: {{requests_per_second}} req/s
- Error tolerance: {{error_tolerance}}%
- User impact: {{user_impact}} (internal/external, {{user_count}} users)
- Revenue impact: {{revenue_impact}}/hour
- Mitigation options: {{mitigation_options}}
Provide priority ranking for incident response.

ELB故障影响评估: 按请求量 × 用户影响 × 时长计算业务损失
```

### 4.5 Resilience Score Calculation
```
Calculate resilience score for ELB {{loadbalancer_id}}:
- AZ redundancy: {{az_score}}/25 (multi-AZ: 25, single-AZ: 0)
- Backend distribution: {{backend_score}}/25 (spread across AZs: 25)
- Health check coverage: {{hc_score}}/25 (comprehensive: 25)
- Failure isolation: {{isolation_score}}/25 (auto-failover: 25)
Total score: {{total_score}}/100
Identify weakest link in availability chain.

ELB韧性评估: 100分满分, >=75分良好, 50-75分需改进, <50分高风险
```

---

## 5. Inspection Prompts

### 5.1 Routine Health Inspection
```
Perform routine inspection on ELB {{loadbalancer_id}}:
- Backend health summary: {{healthy_count}}/{{total_count}} healthy
- Active connections: {{active_connections}} (max: {{max_connections}})
- Throughput: {{throughput}}Mbps (limit: {{bandwidth_limit}}Mbps)
- SSL certificate: {{cert_expiry_days}} days until expiry
- Recent alerts: {{alert_count}} in past 24 hours
- Configuration drift: {{drift_detected}} (yes/no)
Report findings in structured format.

Inspection checklist: 健康状态, 带宽/连接数使用率, 证书过期, 配置变更, 告警历史
```

### 5.2 Security Compliance Check
```
Audit ELB security compliance for {{loadbalancer_id}}:
- Security groups: {{sg_attached}} (least privilege: {{sg_compliant}})
- HTTPS enforcement: {{https_enforced}} (redirect HTTP to HTTPS: {{http_redirect}})
- TLS version: {{tls_version}} (minimum acceptable: TLS 1.2)
- Cipher suite: {{cipher_suite}} (weak ciphers: {{weak_ciphers}})
- Access logs: {{access_logs_enabled}} (retention: {{log_retention}} days)
- WAF status: {{waf_enabled}}
Report compliance status and remediation priorities.

ELB安全基线: HTTPS强制, TLS 1.2+, 无弱加密套件, 访问日志开启, WAF防护
```

### 5.3 Configuration Audit
```
Audit ELB configuration for {{loadbalancer_id}}:
- Listener config: {{listener_count}} listeners, protocols: {{protocols}}
- Backend pool: {{backend_pool_id}}, {{backend_count}} instances
- Health check: {{health_check_enabled}}, interval {{hc_interval}}s, timeout {{hc_timeout}}s
- Session persistence: {{session_persistence}} (type: {{persistence_type}})
- Connection draining: {{connection_draining}} (timeout: {{draining_timeout}}s)
- Access control: {{access_control}} (whitelist/blackitelist/none)
Identify configuration issues and optimization opportunities.

ELB配置最佳实践: 健康检查间隔30s超时10s, 连接耗尽超时300s, X-Forwarded-For记录客户端IP
```

### 5.4 Cost and Usage Report
```
Generate cost and usage report for ELB {{loadbalancer_id}}:
- Billing model: {{billing_model}} ({{monthly_fee}} + {{traffic_fee}})
- Bandwidth usage: {{bandwidth_usage}}% average, {{peak_usage}}% peak
- Data transfer: {{data_transfer}}GB/month (in: {{data_in}}GB, out: {{data_out}}GB)
- Connection usage: {{connection_usage}}% of {{conn_limit}}
- Associated costs: ECS {{ecs_cost}}, RDS {{rds_cost}}, total {{total_cost}}
Compare actual usage vs contracted capacity.

ELB费用优化: 共享带宽包, 预留实例, 业务低谷期释放资源
```

### 5.5 Monitoring Coverage Check
```
Verify monitoring coverage for ELB {{loadbalancer_id}}:
- CES metrics enabled: {{ces_metrics_enabled}} (namespace: SYS.ELB)
- Metrics covered: {{metrics_covered}}/{{total_metrics}}
- Alarm rules: {{alarm_rules_count}} active
- Alarm targets: {{alarm_targets}} (SMS/Email/Webhook)
- Log shipping: {{log_shipping_enabled}} (to LTS: {{lts_enabled}})
- Monitoring agent: {{agent_status}} on {{agent_count}} backends
Identify gaps in observability coverage.

ELB必监控指标: backendIngressPortStatus, active_connections, throughput, backendRequestLatency, ssl_error_rate
```

---

## Appendix: ELB-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{loadbalancer_id}}` | ELB load balancer ID | `5b11e6bc-3a1c-4e3e-9f8a-1c3d5e7f9a2b` |
| `{{listener_id}}` | ELB listener ID | `0a8b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d` |
| `{{backend_pool_id}}` | Backend pool ID | `0b9e8f7a-6b5c-4d3e-2f1a-0b2c3d4e5f6a` |
| `{{az}}` | Availability zone | `cn-north-4a` |
| `{{elb_spec}}` | ELB specification type | `enhanced` (基础型/增强型/超级型) |
| `{{throughput_limit}}` | Bandwidth limit in Mbps | `1000` |
| `{{conn_limit}}` | Connection count limit | `100000` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance (Prompt Handbook P1-3)*
