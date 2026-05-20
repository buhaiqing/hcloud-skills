# 成本优化（FinOps）

## HSS 版本与价格

| 版本 | 说明 | 核心功能 | 适用场景 |
|------|------|---------|---------|
| **基础版** | 免费 | 基础资产信息采集 | 测试环境/个人 |
| **企业版** | 按需/包月 | 入侵检测、漏洞管理、基线检查 | 生产环境标准防护 |
| **旗舰版** | 按需/包月 | 企业版 + 文件完整性校验 + 应用防护 | 等保合规/重要系统 |
| **网页防篡改版** | 包月 | 旗舰版 + 网页防篡改 | Web 服务器 |

```bash
# 查看当前版本和配额使用
hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq '.data_list | group_by(.version) | map({version: .[0].version, count: length})'

hcloud HSS ListQuotas --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

## 成本优化策略

### 1. 按需 vs 包年包月

| 场景 | 推荐模式 | 节省比例 |
|------|---------|---------|
| 短期项目（< 1 月） | 按需 | 灵活，无预付费 |
| 长期稳定业务（≥ 1 年） | 包年包月 | 约节省 20%-30% |
| Web 服务器 | 网页防篡改版 + 包年包月 | 最优惠 |

### 2. 主机分级防护

不同类型的主机使用不同版本的 HSS，避免统一购买高版本：

```bash
#!/bin/bash
# 主机分级防护建议

echo "=== HSS Protection Tier Analysis ==="
hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq -r '
    .data_list[] |
    "\(.host_name)\t\(.version // "not_protected")"
' | while IFS=$'\t' read -r host version; do
    echo "Host: $host → Current: $version"
    # 建议策略（可根据主机类型调整）
    echo "  Suggested: enterprise (¥xx/month)"
done
```

### 3. 闲置资源检测

```bash
# 查找 Agent 离线超过 30 天的主机（可以降级或移除防护）
hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq '.data_list[] | select(.agent_status == "offline") | "⚠️ \(.host_name) — Agent offline since \(.last_scan_time // "unknown")"'

# 查找未受防护的主机（可能存在安全风险）
hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq '.data_list[] | select(.protect_status == "not_protected") | "⚠️ \(.host_name) — Not protected"'
```

### 4. 安全事件减少 = 潜在成本降低

```bash
# 监控安全事件趋势
# 如果经过基线加固和漏洞修复后，告警数量显著下降
# 可考虑降级到较低版本以降低成本

CURRENT_WEEK=$(hcloud HSS ListEvents --begin_time=$(( $(date +%s) * 1000 - 604800000 )) --end_time=$(( $(date +%s) * 1000 )) | jq '.total_num // 0')
LAST_WEEK=$(hcloud HSS ListEvents --begin_time=$(( $(date +%s) * 1000 - 1209600000 )) --end_time=$(( $(date +%s) * 1000 - 604800000 )) | jq '.total_num // 0')
echo "This week events: $CURRENT_WEEK, Last week: $LAST_WEEK"
```

## 成本监控脚本

```bash
#!/bin/bash
# HSS 月度成本估算

HOST_COUNT=$(hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq '.total_num // 0')

echo "========== HSS Cost Estimation =========="
echo "Total hosts: $HOST_COUNT"

# 基于企业版按需价格估算
echo "Estimated monthly cost (enterprise on-demand): ¥$(echo "$HOST_COUNT * 50" | bc)/month"
echo "Estimated yearly cost (enterprise annual): ¥$(echo "$HOST_COUNT * 50 * 12 * 0.75" | bc)/year (25% saving)"

echo "========================================="
```

## 预算规划

| 规模 | 主机数 | 推荐版本 | 月费估算 |
|------|--------|---------|---------|
| 小型 | 1-10 | 企业版按需 | ¥50 × 主机数 |
| 中型 | 10-50 | 企业版包年 | ¥45 × 主机数 |
| 大型 | 50-200 | 旗舰版包年 | ¥80 × 主机数 |
| Web 服务 | 按需 | 网页防篡改版 | ¥150 × 主机数 |
