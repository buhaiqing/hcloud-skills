# 智能运维（AIOps）

> **JSON paths**: Centralized below — inline `jq` calls in scripts use these.
> `.total`             — total event count (ListEvents top-level)
> `.items[]`           — event/item array
> `.items[].rule`      — rule ID per event
> `.items[].sip`       — source IP per event
> `.items[].attack`    — attack type per event
> `.items[].host`      — protected hostname per event
> `.items | length`    — array size
> `.server[0].front_protocol` — front-end protocol of a host
> `.certificateid`     — TLS cert ID bound to a host
> `.policyid`          — protection policy ID bound to a host
> `.name`, `.expire_time` — cert name / expiry

## 异常检测模式

### 模式 1：攻击流量突增检测

**场景**：WAF 防护域名遭遇 DDoS 或 CC 攻击，攻击量在短时间内急剧上升。

**检测指标**：`ListEvents` 返回的事件总数在时间窗口内的变化率。

```bash
#!/bin/bash
# 攻击流量突增检测脚本

check_attack_surge() {
    local POLICY_ID=$1
    local THRESHOLD=${2:-500}  # 单日攻击事件阈值

    CURRENT=$(hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --recent="true" | jq '.total // 0')
    PREVIOUS=$(hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --from=$(( $(date +%s) * 1000 - 1209600000 )) --to=$(( $(date +%s) * 1000 - 604800000 )) | jq '.total // 0')

    if [ "$PREVIOUS" -gt 0 ]; then
        CHANGE_RATE=$(( (CURRENT - PREVIOUS) * 100 / PREVIOUS ))
        echo "[AIOPS] Attack volume change rate: ${CHANGE_RATE}%"
        if [ "$CHANGE_RATE" -gt 100 ]; then
            echo "[ALERT] Attack volume surged ${CHANGE_RATE}% compared to last week!"
            echo "[ACTION] Consider tightening CC rules or enabling stricter protection level"
        fi
    fi
    echo "[INFO] Current attack events (7d): $CURRENT"
}

check_attack_surge "<policyId>"
```

### 模式 2：规则命中率异常（过低或过高）

**检测指标**：规则命中次数与基线对比偏差超过 3σ。

```bash
#!/bin/bash
# 规则命中率异常检测

POLICY_ID="<policyId>"

# 收集最近 7 天的规则命中事件
EVENTS=$(hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --recent="true")

# 按规则 ID 分组统计命中次数
echo "$EVENTS" | jq '.items | group_by(.rule) | map({rule: .[0].rule, count: length}) | sort_by(.count) | reverse' > /tmp/rule_hit_stats.json

# 警报：连续 3 天命中为 0 的规则（可能规则无效或被绕过）
echo "$EVENTS" | jq '.items | group_by(.rule) | map(select(length==0)) | .[] | "⚠️ Rule \(.rule) has zero hits"'

# 警报：命中次数超过均值 + 3σ 的规则
TOTAL=$(echo "$EVENTS" | jq '.items | length')
RULE_COUNT=$(echo "$EVENTS" | jq '.items | group_by(.rule) | length')
if [ "$RULE_COUNT" -gt 0 ]; then
    AVG=$(echo "$TOTAL / $RULE_COUNT" | bc)
    echo "[INFO] Average hits per rule: $AVG"
    # 超过均值 3 倍的规则标记为异常
    echo "$EVENTS" | jq --argjson avg "$AVG" '.items | group_by(.rule) | map(select(length > $avg * 3)) | .[] | "⚠️ Rule \(.rule) has abnormal hit count: \(length) (avg: \($avg))"'
fi
```

### 模式 3：证书到期预测

**检测指标**：证书有效期剩余天数低于阈值。

```bash
#!/bin/bash
# 证书到期预测

CERTIFICATES=$(hcloud WAF ListCertificates --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID")

echo "=== Certificate Expiration Forecast ==="
echo "$CERTIFICATES" | jq -r '.items[] | "\(.name): expires \(.expire_time)"'

# 到期前 30 天预警
echo "$CERTIFICATES" | jq -r '
    .items[] | 
    (.expire_time // "9999-12-31") as $exp |
    ($exp | strptime("%Y-%m-%d") | mktime) as $exp_ts |
    (now + 2592000) as $warning_ts |
    if $exp_ts < $warning_ts then
        "⚠️ Certificate \(.name) expires on \($exp) — renew immediately!"
    else
        empty
    end
'

# 到期前 7 天紧急告警
echo "$CERTIFICATES" | jq -r '
    .items[] | 
    (.expire_time // "9999-12-31") as $exp |
    ($exp | strptime("%Y-%m-%d") | mktime) as $exp_ts |
    (now + 604800) as $urgent_ts |
    if $exp_ts < $urgent_ts then
        "🔴 URGENT: Certificate \(.name) expires on \($exp) — will expire within 7 days!"
    else
        empty
    end
'
```

### 模式 4：防护域名健康度评分

**检测指标**：综合多个维度评估域名安全防护状态。

```bash
#!/bin/bash
# WAF 域名健康度评分

DOMAIN=$1

if [ -z "$DOMAIN" ]; then
    echo "Usage: $0 <domain>"
    exit 1
fi

HOST_INFO=$(hcloud WAF ListHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq --arg d "$DOMAIN" '.items[] | select(.hostname == $d)')
SCORE=100

# 1. HTTPS 检查（-10 分）
PROTOCOL=$(echo "$HOST_INFO" | jq -r '.server[0].front_protocol')
if [ "$PROTOCOL" != "HTTPS" ]; then
    SCORE=$((SCORE - 10))
    echo "[WARN] Domain not using HTTPS (-10)"
fi

# 2. 证书检查（-15 分）
CERT_ID=$(echo "$HOST_INFO" | jq -r '.certificateid // ""')
if [ -z "$CERT_ID" ] || [ "$CERT_ID" = "null" ]; then
    SCORE=$((SCORE - 15))
    echo "[WARN] No certificate assigned (-15)"
fi

# 3. CC 规则检查（-10 分）
POLICY_ID=$(echo "$HOST_INFO" | jq -r '.policyid')
CC_RULES=$(hcloud WAF ListCcRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="$POLICY_ID" | jq '.items | length')
if [ "$CC_RULES" -eq 0 ]; then
    SCORE=$((SCORE - 10))
    echo "[WARN] No CC protection rules (-10)"
fi

# 4. 黑白名单检查（-10 分）
IP_RULES=$(hcloud WAF ListWhiteBlackIpRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="$POLICY_ID" | jq '.items | length')
if [ "$IP_RULES" -eq 0 ]; then
    SCORE=$((SCORE - 10))
    echo "[WARN] No IP blacklist/whitelist rules (-10)"
fi

# 5. 攻击事件趋势（-15 分）
RECENT_EVENTS=$(hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --hosts="[\"$DOMAIN\"]" --recent="true" | jq '.total // 0')
if [ "$RECENT_EVENTS" -gt 10000 ]; then
    SCORE=$((SCORE - 15))
    echo "[WARN] Very high attack volume (>10000 events/7d) (-15)"
fi

# 确保分数 ≥ 0
SCORE=$((SCORE > 0 ? SCORE : 0))

echo ""
echo "========== Domain Health Score =========="
echo "Domain: $DOMAIN"
echo "Score: $SCORE/100"
if [ "$SCORE" -ge 80 ]; then
    echo "Rating: ✅ Good"
elif [ "$SCORE" -ge 50 ]; then
    echo "Rating: ⚠️ Needs Improvement"
else
    echo "Rating: 🔴 Critical"
fi
echo "========================================="
```

## 自动修复模式

### 自动缓解 CC 攻击

```bash
#!/bin/bash
# CC 攻击自动缓解

POLICY_ID=$1

# 1. 检测攻击
ATTACK_COUNT=$(hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --attacks="[\"cc\"]" --recent="true" | jq '.total // 0')

if [ "$ATTACK_COUNT" -gt 500 ]; then
    echo "[AUTOFIX] Detected CC attack surge: $ATTACK_COUNT events"

    # 2. 临时收紧 CC 规则（降低阈值 50%）
    CURRENT_CC_RULES=$(hcloud WAF ListCcRules --policy_id="$POLICY_ID" --page="1" --pagesize="50")
    echo "$CURRENT_CC_RULES" | jq -c '.items[]' | while read rule; do
        RULE_ID=$(echo "$rule" | jq -r '.id')
        CURRENT_LIMIT=$(echo "$rule" | jq -r '.limit_num')
        NEW_LIMIT=$((CURRENT_LIMIT / 2))
        
        hcloud WAF UpdateCcRule --policy_id="$POLICY_ID" --rule_id="$RULE_ID" \
          --body="{\"limit_num\":$NEW_LIMIT}"
        echo "[AUTOFIX] Reduced CC threshold for rule $RULE_ID from $CURRENT_LIMIT to $NEW_LIMIT"
    done

    # 3. 记录自动修复事件
    echo "[AUTOFIX] CC auto-mitigation applied at $(date)"
fi
```

### 自动清理僵尸规则

```bash
#!/bin/bash
# 自动清理 90 天以上无命中的规则

POLICY_ID=$1
DAYS_THRESHOLD=${2:-90}

echo "[AIOPS] Scanning for stale rules without hits..."

# 检查自定义规则列表
hcloud WAF ListCustomRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="$POLICY_ID" | \
  jq -c '.items[]' | while read rule; do
    RULE_ID=$(echo "$rule" | jq -r '.id')
    RULE_NAME=$(echo "$rule" | jq -r '.name')
    CREATE_TIME=$(echo "$rule" | jq -r '.timestamp // 0')
    NOW_TS=$(date +%s)

    # 如果规则创建超过阈值且无事件命中
    AGE_DAYS=$(( (NOW_TS - CREATE_TIME / 1000) / 86400 ))
    if [ "$AGE_DAYS" -gt "$DAYS_THRESHOLD" ]; then
        # 检查该规则最近是否有命中事件
        HIT_COUNT=$(hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --recent="true" | \
          jq --arg rid "$RULE_ID" '[.items[] | select(.rule == $rid)] | length')
        
        if [ "$HIT_COUNT" -eq 0 ]; then
            echo "[CLEANUP] Rule '$RULE_NAME' ($RULE_ID) — no hits in $AGE_DAYS days, age=$AGE_DAYS days"
            # 如需自动清理，取消下面注释
            # hcloud WAF DeleteCustomRule --policy_id="$POLICY_ID" --rule_id="$RULE_ID"
        fi
    fi
done
```

## 异常关联分析

### 攻击模式识别

```bash
#!/bin/bash
# 识别攻击模式（相同源 IP 对不同域名的攻击）

POLICY_ID=$1

# 获取最近攻击事件并按源 IP 分组
hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --recent="true" | \
  jq '.items | group_by(.sip) | map({ip: .[0].sip, attacks: [.[].attack] | unique, hosts: [.[].host | unique], count: length}) | sort_by(.count) | reverse | .[0:10]' > /tmp/top_attackers.json

echo "=== Top Attacker IPs ==="
jq -r '.[] | "\(.ip): \(.count) attacks across \(.hosts | length) hosts, types: \(.attacks | join(\", \"))"' /tmp/top_attackers.json
```

### AIOps 基线数据收集

```bash
#!/bin/bash
# 每日基线数据收集

REPORT_FILE="/tmp/waf_baseline_$(date +%Y%m%d).json"

{
    echo "{"
    echo "  \"date\": \"$(date +%Y-%m-%d)\","
    echo "  \"domain_count\": $(hcloud WAF ListHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq '.items | length'),"
    echo "  \"policy_count\": $(hcloud WAF ListPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq '.items | length'),"
    echo "  \"event_count_7d\": $(hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --recent="true" | jq '.total // 0'),"
    echo "  \"cert_count\": $(hcloud WAF ListCertificates --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq '.items | length')"
    echo "}"
} > "$REPORT_FILE"

echo "[BASELINE] Data saved to $REPORT_FILE"
```
