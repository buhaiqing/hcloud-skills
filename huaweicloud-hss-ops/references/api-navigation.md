# HSS v5 API 导航

> API 参考: https://support.huaweicloud.com/intl/en-us/api-hss2.0/
> Go SDK: `github.com/huaweicloud/huaweicloud-sdk-go-v3/services/hss/v5`

## 主机管理

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 查询主机列表 | GET /v5/{project_id}/host-management/hosts | `hcloud HSS ListHosts` | `ListHosts()` |
| 查询主机详情 | GET /v5/{project_id}/host-management/hosts/{host_id} | `hcloud HSS ShowHost` | `ShowHost()` |
| 切换防护状态 | POST /v5/{project_id}/host-management/protection | `hcloud HSS SwitchHostsProtectStatus` | `SwitchHostsProtectStatus()` |
| 查询主机组 | GET /v5/{project_id}/host-management/groups | `hcloud HSS ListHostGroups` | `ListHostGroups()` |
| 创建主机组 | POST /v5/{project_id}/host-management/groups | `hcloud HSS CreateHostGroup` | `CreateHostGroup()` |
| 删除主机组 | DELETE /v5/{project_id}/host-management/groups/{group_id} | `hcloud HSS DeleteHostGroup` | `DeleteHostGroup()` |

## 资产管理

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 查询资产统计 | GET /v5/{project_id}/asset/statistic | `hcloud HSS ShowAssetStatistic` | `ShowAssetStatistic()` |
| 查询账户列表 | GET /v5/{project_id}/asset/user/statistics | `hcloud HSS ListAccounts` | `ListAccounts()` |
| 查询端口统计 | GET /v5/{project_id}/asset/port/statistics | `hcloud HSS ListPorts` | `ListPorts()` |
| 查询进程列表 | GET /v5/{project_id}/asset/process/statistics | `hcloud HSS ListProcesses` | `ListProcesses()` |
| 查询软件列表 | GET /v5/{project_id}/asset/app/statistics | `hcloud HSS ListApps` | `ListApps()` |
| 查询自启动项 | GET /v5/{project_id}/asset/auto-launch/statistics | `hcloud HSS ListAutoLaunchs` | `ListAutoLaunchs()` |
| 查询中间件 | GET /v5/{project_id}/asset/middleware/statistics | `hcloud HSS ListMiddlewares` | `ListMiddlewares()` |
| 查询主机账户列表 | GET /v5/{project_id}/asset/users | `hcloud HSS ListHostAccounts` | `ListHostAccounts()` |
| 查询主机端口列表 | GET /v5/{project_id}/asset/ports | `hcloud HSS ListHostPorts` | `ListHostPorts()` |
| 查询主机进程列表 | GET /v5/{project_id}/asset/processes | `hcloud HSS ListHostProcesses` | `ListHostProcesses()` |
| 查询主机软件列表 | GET /v5/{project_id}/asset/apps | `hcloud HSS ListHostApps` | `ListHostApps()` |
| 获取资产变更历史 | GET /v5/{project_id}/asset/changes | `hcloud HSS ListAssetChanges` | `ListAssetChanges()` |

## 入侵检测与告警事件

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 查询告警事件 | GET /v5/{project_id}/event/events | `hcloud HSS ListEvents` | `ListEvents()` |
| 处理告警事件 | POST /v5/{project_id}/event/operate | `hcloud HSS OperateEvent` | `OperateEvent()` |
| 查询关联告警 | GET /v5/{project_id}/event/related-events | `hcloud HSS ListRelatedEvents` | `ListRelatedEvents()` |
| 查询事件类型统计 | GET /v5/{project_id}/event/type-statistics | `hcloud HSS ListEventType` | `ListEventType()` |
| 查询ATT&CK阶段统计 | GET /v5/{project_id}/event/attack-stage | `hcloud HSS ListEventAttCk` | `ListEventAttCk()` |
| 查询已封锁IP | GET /v5/{project_id}/blocked-ip | `hcloud HSS ListBlockedIp` | `ListBlockedIp()` |
| 解封IP | POST /v5/{project_id}/blocked-ip | `hcloud HSS ChangeBlockedIp` | `ChangeBlockedIp()` |
| 查询已隔离文件 | GET /v5/{project_id}/isolated-file | `hcloud HSS ListIsolatedFile` | `ListIsolatedFile()` |
| 恢复隔离文件 | POST /v5/{project_id}/isolated-file | `hcloud HSS ChangeIsolatedFile` | `ChangeIsolatedFile()` |
| 删除隔离文件 | DELETE /v5/{project_id}/isolated-file | `hcloud HSS DeleteIsolatedFile` | `DeleteIsolatedFile()` |
| 事件取证调查 | POST /v5/{project_id}/event/forensics | `hcloud HSS ListEventForensic` | `ListEventForensic()` |

## 漏洞管理

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 查询漏洞列表 | GET /v5/{project_id}/vulnerability/vulnerabilities | `hcloud HSS ListVulnerabilities` | `ListVulnerabilities()` |
| 查询漏洞详情 | GET /v5/{project_id}/vulnerability/vulnerability/{vul_id} | `hcloud HSS ShowVulnerability` | `ShowVulnerability()` |
| 创建漏洞扫描 | POST /v5/{project_id}/vulnerability/scan-task | `hcloud HSS CreateScanTask` | `CreateScanTask()` |
| 处理漏洞 | PUT /v5/{project_id}/vulnerability/vulnerabilities | `hcloud HSS ChangeVulStatus` | `ChangeVulStatus()` |
| 导出漏洞报告 | POST /v5/{project_id}/vulnerability/export | `hcloud HSS ExportVulnerabilities` | `ExportVulnerabilities()` |

## 基线检查

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 查询基线检查结果 | GET /v5/{project_id}/baseline/results | `hcloud HSS ListBaselineCheckResults` | `ListBaselineCheckResults()` |
| 查询基线策略 | GET /v5/{project_id}/baseline/policies | `hcloud HSS ListBaselinePolicies` | `ListBaselinePolicies()` |
| 查询基线规则 | GET /v5/{project_id}/baseline/rules | `hcloud HSS ListBaselineRules` | `ListBaselineRules()` |
| 创建基线检查任务 | POST /v5/{project_id}/baseline/scan-task | `hcloud HSS CreateBaselineCheckTask` | `CreateBaselineCheckTask()` |
| 处理基线检查结果 | PUT /v5/{project_id}/baseline/results | `hcloud HSS ChangeBaselineResult` | `ChangeBaselineResult()` |

## 防护策略管理

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 查询策略组 | GET /v5/{project_id}/policy/groups | `hcloud HSS ListPolicyGroups` | `ListPolicyGroups()` |
| 更新策略组 | PUT /v5/{project_id}/policy/groups/{group_id} | `hcloud HSS UpdatePolicyGroup` | `UpdatePolicyGroup()` |
| 查询策略详情 | GET /v5/{project_id}/policy/groups/{group_id} | `hcloud HSS ShowPolicyGroup` | `ShowPolicyGroup()` |

## 网页防篡改

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 查询防篡改保护 | GET /v5/{project_id}/webtamper/hosts | `hcloud HSS ListWtpProtection` | `ListWtpProtection()` |
| 创建防篡改保护 | POST /v5/{project_id}/webtamper/host | `hcloud HSS CreateWtpProtection` | `CreateWtpProtection()` |
| 删除防篡改保护 | DELETE /v5/{project_id}/webtamper/host/{host_id} | `hcloud HSS DeleteWtpProtection` | `DeleteWtpProtection()` |

## 配额管理

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 查询配额列表 | GET /v5/{project_id}/billing/quotas | `hcloud HSS ListQuotas` | `ListQuotas()` |
| 创建订单 | POST /v5/{project_id}/billing/orders | `hcloud HSS CreateOrder` | `CreateOrder()` |

## Agent 管理

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 查询 Agent 列表 | GET /v5/{project_id}/agent/agents | `hcloud HSS ListAgents` | `ListAgents()` |
| 安装 Agent | POST /v5/{project_id}/agent/install | `hcloud HSS InstallAgent` | `InstallAgent()` |
| 升级 Agent | PUT /v5/{project_id}/agent/upgrade | `hcloud HSS UpgradeAgent` | `UpgradeAgent()` |

## 容器管理

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 查询容器节点 | GET /v5/{project_id}/container/nodes | `hcloud HSS ListContainerNodes` | `ListContainerNodes()` |
| 查询容器镜像 | GET /v5/{project_id}/container/images | `hcloud HSS ListContainerImages` | `ListContainerImages()` |
| 查询容器 Pod | GET /v5/{project_id}/container/pods | `hcloud HSS ListContainerPods` | `ListContainerPods()` |
| 查询容器告警 | GET /v5/{project_id}/container/alarms | `hcloud HSS ListContainerAlarms` | `ListContainerAlarms()` |

## 白名单管理

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 查询告警白名单 | GET /v5/{project_id}/whitelist/alarm | `hcloud HSS ListAlarmWhitelists` | `ListAlarmWhitelists()` |
| 查询登录白名单 | GET /v5/{project_id}/whitelist/login | `hcloud HSS ListLoginWhitelists` | `ListLoginWhitelists()` |

## 仪表盘与报告

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 查询安全概览 | GET /v5/{project_id}/dashboard | `hcloud HSS ShowDashboard` | `ShowDashboard()` |
| 导出安全报告 | POST /v5/{project_id}/report/export | `hcloud HSS ExportSecurityReport` | `ExportSecurityReport()` |
| 查询报告列表 | GET /v5/{project_id}/report/reports | `hcloud HSS ListSecurityReports` | `ListSecurityReports()` |
