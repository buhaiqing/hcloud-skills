# WAF v1 API 导航

> API 参考: https://support.huaweicloud.com/api-waf/index.html
> Go SDK: `github.com/huaweicloud/huaweicloud-sdk-go-v3/services/waf/v1`

## 策略管理

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出策略 | GET /v1/{project_id}/waf/policy | `hcloud WAF ListPolicy` | `ListPolicy()` |
| 创建策略 | POST /v1/{project_id}/waf/policy | `hcloud WAF CreatePolicy` | `CreatePolicy()` |
| 查询策略详情 | GET /v1/{project_id}/waf/policy/{policy_id} | `hcloud WAF ShowPolicy` | `ShowPolicy()` |
| 更新策略 | PUT /v1/{project_id}/waf/policy/{policy_id} | `hcloud WAF UpdatePolicy` | `UpdatePolicy()` |
| 删除策略 | DELETE /v1/{project_id}/waf/policy/{policy_id} | `hcloud WAF DeletePolicy` | `DeletePolicy()` |

## 域名管理（云模式）

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出域名 | GET /v1/{project_id}/waf/instance | `hcloud WAF ListHost` | `ListHost()` |
| 添加域名 | POST /v1/{project_id}/waf/instance | `hcloud WAF CreateHost` | `CreateHost()` |
| 查询域名详情 | GET /v1/{project_id}/waf/instance/{instance_id} | `hcloud WAF ShowHost` | `ShowHost()` |
| 更新域名 | PATCH /v1/{project_id}/waf/instance/{instance_id} | `hcloud WAF UpdateHost` | `UpdateHost()` |
| 删除域名 | DELETE /v1/{project_id}/waf/instance/{instance_id} | `hcloud WAF DeleteHost` | `DeleteHost()` |

## 域名管理（独享模式）

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出域名 | GET /v1/{project_id}/premium-waf/host | `hcloud WAF ListPremiumHost` | `ListPremiumHost()` |
| 添加域名 | POST /v1/{project_id}/premium-waf/host | `hcloud WAF CreatePremiumHost` | `CreatePremiumHost()` |
| 查询域名详情 | GET /v1/{project_id}/premium-waf/host/{host_id} | `hcloud WAF ShowPremiumHost` | `ShowPremiumHost()` |
| 更新域名 | PUT /v1/{project_id}/premium-waf/host/{host_id} | `hcloud WAF UpdatePremiumHost` | `UpdatePremiumHost()` |
| 删除域名 | DELETE /v1/{project_id}/premium-waf/host/{host_id} | `hcloud WAF DeletePremiumHost` | `DeletePremiumHost()` |

## CC 攻击防护规则

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出规则 | GET /v1/{project_id}/waf/policy/{policy_id}/cc | `hcloud WAF ListCcRules` | `ListCcRules()` |
| 创建规则 | POST /v1/{project_id}/waf/policy/{policy_id}/cc | `hcloud WAF CreateCcRule` | `CreateCcRule()` |
| 更新规则 | PUT /v1/{project_id}/waf/policy/{policy_id}/cc/{rule_id} | `hcloud WAF UpdateCcRule` | `UpdateCcRule()` |
| 删除规则 | DELETE /v1/{project_id}/waf/policy/{policy_id}/cc/{rule_id} | `hcloud WAF DeleteCcRule` | `DeleteCcRule()` |

## 精准访问防护规则（自定义规则）

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出规则 | GET /v1/{project_id}/waf/policy/{policy_id}/custom | `hcloud WAF ListCustomRules` | `ListCustomRules()` |
| 创建规则 | POST /v1/{project_id}/waf/policy/{policy_id}/custom | `hcloud WAF CreateCustomRule` | `CreateCustomRule()` |
| 更新规则 | PUT /v1/{project_id}/waf/policy/{policy_id}/custom/{rule_id} | `hcloud WAF UpdateCustomRule` | `UpdateCustomRule()` |
| 删除规则 | DELETE /v1/{project_id}/waf/policy/{policy_id}/custom/{rule_id} | `hcloud WAF DeleteCustomRule` | `DeleteCustomRule()` |

## 黑白名单规则

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出规则 | GET /v1/{project_id}/waf/policy/{policy_id}/whiteblackip | `hcloud WAF ListWhiteBlackIpRules` | `ListWhiteBlackIpRules()` |
| 创建规则 | POST /v1/{project_id}/waf/policy/{policy_id}/whiteblackip | `hcloud WAF CreateWhiteBlackIpRule` | `CreateWhiteBlackIpRule()` |
| 更新规则 | PUT /v1/{project_id}/waf/policy/{policy_id}/whiteblackip/{rule_id} | `hcloud WAF UpdateWhiteBlackIpRule` | `UpdateWhiteBlackIpRule()` |
| 删除规则 | DELETE /v1/{project_id}/waf/policy/{policy_id}/whiteblackip/{rule_id} | `hcloud WAF DeleteWhiteBlackIpRule` | `DeleteWhiteBlackIpRule()` |

## 地理位置访问控制规则

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出规则 | GET /v1/{project_id}/waf/policy/{policy_id}/geoip | `hcloud WAF ListGeoIpRules` | `ListGeoIpRules()` |
| 创建规则 | POST /v1/{project_id}/waf/policy/{policy_id}/geoip | `hcloud WAF CreateGeoIpRule` | `CreateGeoIpRule()` |
| 更新规则 | PUT /v1/{project_id}/waf/policy/{policy_id}/geoip/{rule_id} | `hcloud WAF UpdateGeoIpRule` | `UpdateGeoIpRule()` |
| 删除规则 | DELETE /v1/{project_id}/waf/policy/{policy_id}/geoip/{rule_id} | `hcloud WAF DeleteGeoIpRule` | `DeleteGeoIpRule()` |

## 网页防篡改规则

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出规则 | GET /v1/{project_id}/waf/policy/{policy_id}/antitamper | `hcloud WAF ListAntiTamperRules` | `ListAntiTamperRules()` |
| 创建规则 | POST /v1/{project_id}/waf/policy/{policy_id}/antitamper | `hcloud WAF CreateAntiTamperRule` | `CreateAntiTamperRule()` |
| 删除规则 | DELETE /v1/{project_id}/waf/policy/{policy_id}/antitamper/{rule_id} | `hcloud WAF DeleteAntiTamperRule` | `DeleteAntiTamperRule()` |

## 信息防泄漏规则

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出规则 | GET /v1/{project_id}/waf/policy/{policy_id}/antileakage | `hcloud WAF ListAntileakageRules` | `ListAntileakageRules()` |
| 创建规则 | POST /v1/{project_id}/waf/policy/{policy_id}/antileakage | `hcloud WAF CreateAntileakageRule` | `CreateAntileakageRule()` |
| 更新规则 | PUT /v1/{project_id}/waf/policy/{policy_id}/antileakage/{rule_id} | `hcloud WAF UpdateAntileakageRule` | `UpdateAntileakageRule()` |
| 删除规则 | DELETE /v1/{project_id}/waf/policy/{policy_id}/antileakage/{rule_id} | `hcloud WAF DeleteAntileakageRule` | `DeleteAntileakageRule()` |

## 数据防泄露响应规则（隐私屏蔽）

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出规则 | GET /v1/{project_id}/waf/policy/{policy_id}/privacy | `hcloud WAF ListPrivacyResponseRules` | `ListPrivacyResponseRules()` |
| 创建规则 | POST /v1/{project_id}/waf/policy/{policy_id}/privacy | `hcloud WAF CreatePrivacyResponseRule` | `CreatePrivacyResponseRule()` |
| 删除规则 | DELETE /v1/{project_id}/waf/policy/{policy_id}/privacy/{rule_id} | `hcloud WAF DeletePrivacyResponseRule` | `DeletePrivacyResponseRule()` |

## 反爬虫规则

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出规则 | GET /v1/{project_id}/waf/policy/{policy_id}/anticrawler | `hcloud WAF ListAnticrawlerRules` | `ListAnticrawlerRules()` |
| 创建规则 | POST /v1/{project_id}/waf/policy/{policy_id}/anticrawler | `hcloud WAF CreateAnticrawlerRule` | `CreateAnticrawlerRule()` |
| 更新规则 | PUT /v1/{project_id}/waf/policy/{policy_id}/anticrawler/{rule_id} | `hcloud WAF UpdateAnticrawlerRule` | `UpdateAnticrawlerRule()` |
| 删除规则 | DELETE /v1/{project_id}/waf/policy/{policy_id}/anticrawler/{rule_id} | `hcloud WAF DeleteAnticrawlerRule` | `DeleteAnticrawlerRule()` |

## 已知攻击源规则

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出规则 | GET /v1/{project_id}/waf/policy/{policy_id}/punishment | `hcloud WAF ListAttackMitigationRules` | `ListAttackMitigationRules()` |
| 更新规则 | PUT /v1/{project_id}/waf/policy/{policy_id}/punishment/{rule_id} | `hcloud WAF UpdateAttackMitigationRule` | `UpdateAttackMitigationRule()` |

## 全局白名单规则（误报屏蔽）

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出规则 | GET /v1/{project_id}/waf/policy/{policy_id}/ignore | `hcloud WAF ListIgnoreRule` | `ListIgnoreRule()` |
| 创建规则 | POST /v1/{project_id}/waf/policy/{policy_id}/ignore | `hcloud WAF CreateIgnoreRule` | `CreateIgnoreRule()` |
| 删除规则 | DELETE /v1/{project_id}/waf/policy/{policy_id}/ignore/{rule_id} | `hcloud WAF DeleteIgnoreRule` | `DeleteIgnoreRule()` |

## 攻击事件

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出事件 | GET /v1/{project_id}/waf/event | `hcloud WAF ListEvents` | `ListEvents()` |
| 查询事件详情 | GET /v1/{project_id}/waf/event/{eventid} | `hcloud WAF ShowEvent` | `ShowEvent()` |

## 证书管理

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出证书 | GET /v1/{project_id}/waf/certificate | `hcloud WAF ListCertificates` | `ListCertificates()` |
| 创建证书 | POST /v1/{project_id}/waf/certificate | `hcloud WAF CreateCertificate` | `CreateCertificate()` |
| 查询证书详情 | GET /v1/{project_id}/waf/certificate/{certificate_id} | `hcloud WAF ShowCertificate` | `ShowCertificate()` |
| 更新证书 | PUT /v1/{project_id}/waf/certificate/{certificate_id} | `hcloud WAF UpdateCertificate` | `UpdateCertificate()` |
| 删除证书 | DELETE /v1/{project_id}/waf/certificate/{certificate_id} | `hcloud WAF DeleteCertificate` | `DeleteCertificate()` |

## 引用表管理

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出引用表 | GET /v1/{project_id}/waf/valuelist | `hcloud WAF ListValueList` | `ListValueList()` |
| 创建引用表 | POST /v1/{project_id}/waf/valuelist | `hcloud WAF CreateValueList` | `CreateValueList()` |
| 查询引用表详情 | GET /v1/{project_id}/waf/valuelist/{valuelistid} | `hcloud WAF ShowValueList` | `ShowValueList()` |
| 更新引用表 | PUT /v1/{project_id}/waf/valuelist/{valuelistid} | `hcloud WAF UpdateValueList` | `UpdateValueList()` |
| 删除引用表 | DELETE /v1/{project_id}/waf/valuelist/{valuelistid} | `hcloud WAF DeleteValueList` | `DeleteValueList()` |

## 地址组管理

| 操作 | API | hcloud CLI | Go SDK |
|------|-----|-----------|--------|
| 列出地址组 | GET /v1/{project_id}/waf/ip-group | `hcloud WAF ListIpGroup` | `ListIpGroup()` |
| 创建地址组 | POST /v1/{project_id}/waf/ip-group | `hcloud WAF CreateIpGroup` | `CreateIpGroup()` |
| 查询地址组 | GET /v1/{project_id}/waf/ip-group/{id} | `hcloud WAF ShowIpGroup` | `ShowIpGroup()` |
| 更新地址组 | PUT /v1/{project_id}/waf/ip-group/{id} | `hcloud WAF UpdateIpGroup` | `UpdateIpGroup()` |
| 删除地址组 | DELETE /v1/{project_id}/waf/ip-group/{id} | `hcloud WAF DeleteIpGroup` | `DeleteIpGroup()` |
