# 成本优化（FinOps）

## WAF 计费模式

WAF 提供两种计费模式：

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| **包年/包月** | 预付费，价格优惠 | 长期稳定业务 |
| **按需计费** | 后付费，按使用量计费 | 临时/测试/弹性业务 |

## 成本优化策略

### 1. 选择合适的 WAF 版本

WAF 版本与价格从低到高：

| 版本 | 防护域名数 | 适用规模 | 月费参考 |
|------|-----------|----------|---------|
| 入门版 | 10 个 | 小型业务 | 低 |
| 标准版 | 10 个 | 中型业务 | 中 |
| 专业版 | 20 个 | 大型业务 | 高 |
| 铂金版 | 20+ 个 | 企业级 | 最高 |

```bash
# 查看当前 WAF 实例信息
hcloud WAF ListHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
# 统计域名数量，判断是否需要升级或降级
hcloud WAF ListHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq '.items | length'
```

### 2. 包年/包月 vs 按需选择

```bash
# 月度成本比较 (假设月请求量 1M)
# 包年/包月专业版: 约 ¥3,000/月
# 按需 (100万次): 约 ¥4,500/月
# 结论: 稳定业务选择包年/包月，节省 30%+
```

### 3. 规则精简优化

不必要的规则浪费计算资源和费用：

```bash
# 列出所有未使用的规则
for policyId in $(hcloud WAF ListPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq -r '.items[].id'); do
    ccCount=$(hcloud WAF ListCcRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="$policyId" | jq '.items | length')
    echo "Policy $policyId: CC rules=$ccCount"
done

# 标记创建超过 90 天且无命中事件的规则进行清理
```

### 4. 域名整合

```bash
# 检查域名是否绑定到正确的策略
# 建议：同类型域名尽量共用策略，减少策略数量
hcloud WAF ListHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq '.items | group_by(.policyid) | map({policyid: .[0].policyid, count: length})'
```

### 5. 攻击事件监控避免资源浪费

```bash
# 定期检查攻击事件数量，确认 WAF 的保护效果
# 如果攻击量极少（<100/天），可考虑降级到低版本
hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --recent="true" | jq '.total'
```

### 6. 证书管理成本优化

- 使用华为云 SSL 证书管理服务（SCM）统一管理证书
- 证书到期前及时续费，避免因证书过期导致业务中断
- 多个域名共用通配符证书（如 `*.example.com`）减少证书数量

## 成本监控脚本

```bash
#!/bin/bash
# WAF 月度成本估算脚本

WAF_VERSION="professional"
DOMAIN_COUNT=$(hcloud WAF ListHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq '.items | length')
EVENT_COUNT=$(hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --recent="true" | jq '.total // 0')

echo "========== WAF Cost Estimation =========="
echo "WAF Version: $WAF_VERSION"
echo "Protected Domains: $DOMAIN_COUNT"
echo "Recent Events (7d): $EVENT_COUNT"

# 粗略月度成本估算
if [ "$DOMAIN_COUNT" -le 10 ]; then
    echo "Recommended Plan: Standard (~¥1,500/month)"
elif [ "$DOMAIN_COUNT" -le 20 ]; then
    echo "Recommended Plan: Professional (~¥3,000/month)"
else
    echo "Recommended Plan: Platinum (~¥6,000/month)"
fi

echo "========================================="
```

## 年度预算规划

| 规模 | 域名数 | 建议版本 | 包年费用（估算） | 按年节省 |
|------|--------|---------|----------------|---------|
| 小型 | <10 | 标准版 | ¥16,000 | 比按月节省 ~15% |
| 中型 | 10-20 | 专业版 | ¥32,000 | 比按月节省 ~15% |
| 大型 | >20 | 铂金版 | ¥62,000 | 比按月节省 ~15% |
