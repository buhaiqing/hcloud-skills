# 智能运维（AIOps）

> **JSON paths**: Centralized below — inline `jq` calls in scripts use these.
> `.total` — total event count (ListEvents / ListVulnerabilities top-level)
> `.data_list[]` — HSS list API result array (host / event / vuln)
> `.data_list[].host_id` / `.data_list[].host_name` — host identifiers
> `.data_list[].severity` — severity per item
> `.data_list[].vul_id` / `.data_list[].vul_name` — vulnerability ID / name
> `.data_list[].alarm_id` — alarm identifier
> `.data_list | length` — array size
> `.host_num` — host count from summary APIs
> `.risk_host_num` — at-risk host count

## 异常检测模式

### 模式 1：安全告警风暴检测

**场景**：某主机突然产生大量安全告警，可能正在遭受攻击。

**检测指标**：`ListEvents` 返回的告警数量在时间窗口内的激增。

```bash
#!/bin/bash
# 告警风暴检测

check_alert_storm() {
    local THRESHOLD=${1:-50}  # 单日告警阈值

    TODAY=$(hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
      --begin_time=$(( $(date +%s) * 1000 - 86400000 )) --end_time=$(( $(date +%s) * 1000 )) | jq '.total_num // 0')
    YESTERDAY=$(hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
      --begin_time=$(( $(date +%s) * 1000 - 172800000 )) --end_time=$(( $(date +%s) * 1000 - 86400000 )) | jq '.total_num // 0')

    echo "[AIOPS] Today: $TODAY alerts, Yesterday: $YESTERDAY alerts"
    
    if [ "$YESTERDAY" -gt 0 ]; then
        CHANGE_RATE=$(( (TODAY - YESTERDAY) * 100 / YESTERDAY ))
        if [ "$CHANGE_RATE" -gt 200 ]; then
            echo "[ALERT] Alert storm detected: ${CHANGE_RATE}% increase!"
            echo "[ACTION] Check ListEvents for new attack patterns"
        fi
    fi
    
    if [ "$TODAY" -gt "$THRESHOLD" ]; then
        echo "[ALERT] Alert count exceeds threshold ($TODAY > $THRESHOLD)"
    fi
}

check_alert_storm
```

### 模式 2：未处理告警堆积检测

**检测指标**：未处理告警数量持续增长，超过基线值。

```bash
#!/bin/bash
# 未处理告警堆积检测

# 收集一周的未处理告警数据
for day in $(seq 0 6); do
    DAY=$(date -v-${day}d +%Y-%m-%d 2>/dev/null || date -d "-${day} day" +%Y-%m-%d)
    BEGIN_TS=$(date -j -f "%Y-%m-%d" "$DAY" "+%s" 2>/dev/null || date -d "$DAY" +%s)
    END_TS=$((BEGIN_TS + 86400))
    
    COUNT=$(hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
      --handle_status="unhandled" --begin_time=$((BEGIN_TS * 1000)) --end_time=$((END_TS * 1000)) | jq '.total_num // 0')
    echo "$DAY: $COUNT unhandled alerts"
done

# 如果未处理告警连续 3 天增长，触发告警
# 建议：设置自动处理流程或通知安全运维人员
```

### 模式 3：漏洞修复进度监控

**检测指标**：漏洞修复率低于阈值或关键漏洞修复超时。

```bash
#!/bin/bash
# 漏洞修复进度监控

TOTAL_VUL=$(hcloud HSS ListVulnerabilities --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq '.total_num // 0')
UNHANDLED_VUL=$(hcloud HSS ListVulnerabilities --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --handle_status="unhandled" | jq '.total_num // 0')
CRITICAL_UNHANDLED=$(hcloud HSS ListVulnerabilities --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --severity="critical" --handle_status="unhandled" | jq '.total_num // 0')

echo "=== Vulnerability Metrics ==="
echo "Total vulnerabilities: $TOTAL_VUL"
echo "Unhandled: $UNHANDLED_VUL"
echo "Critical unhandled: $CRITICAL_UNHANDLED"

if [ "$TOTAL_VUL" -gt 0 ]; then
    FIX_RATE=$(( (TOTAL_VUL - UNHANDLED_VUL) * 100 / TOTAL_VUL ))
    echo "Fix rate: ${FIX_RATE}%"
    if [ "$FIX_RATE" -lt 80 ]; then
        echo "[ALERT] Vulnerability fix rate below 80%!"
    fi
fi

if [ "$CRITICAL_UNHANDLED" -gt 0 ]; then
    echo "[ALERT] Critical vulnerabilities unhandled — SLA at risk!"
fi
```

### 模式 4：主机健康度评分

**检测指标**：综合 Agent 状态、告警数量、漏洞数量、基线合规情况。

```bash
#!/bin/bash
# 主机健康度评分

HOST_ID=$1

if [ -z "$HOST_ID" ]; then
    echo "Usage: $0 <host_id>"
    exit 1
fi

HOST_INFO=$(hcloud HSS ShowHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --host_id="$HOST_ID")
SCORE=100

# 1. Agent 状态（-20 分）
AGENT_STATUS=$(echo "$HOST_INFO" | jq -r '.agent_status')
if [ "$AGENT_STATUS" != "online" ]; then
    SCORE=$((SCORE - 20))
    echo "[WARN] Agent offline (-20)"
fi

# 2. 防护状态（-20 分）
PROTECT_STATUS=$(echo "$HOST_INFO" | jq -r '.protect_status')
if [ "$PROTECT_STATUS" = "not_protected" ]; then
    SCORE=$((SCORE - 20))
    echo "[WARN] Host not protected (-20)"
fi

# 3. 未处理告警（-15 分）
ALERT_COUNT=$(hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --host_name="$(echo "$HOST_INFO" | jq -r '.host_name')" --handle_status="unhandled" | jq '.total_num // 0')
if [ "$ALERT_COUNT" -gt 10 ]; then
    SCORE=$((SCORE - 15))
    echo "[WARN] $ALERT_COUNT unhandled alerts (-15)"
elif [ "$ALERT_COUNT" -gt 0 ]; then
    SCORE=$((SCORE - 5))
    echo "[INFO] $ALERT_COUNT unhandled alerts (-5)"
fi

# 4. 未修复漏洞（-15 分）
VUL_COUNT=$(hcloud HSS ListVulnerabilities --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --host_id="$HOST_ID" --handle_status="unhandled" | jq '.total_num // 0')
if [ "$VUL_COUNT" -gt 20 ]; then
    SCORE=$((SCORE - 15))
    echo "[WARN] $VUL_COUNT unpatched vulnerabilities (-15)"
elif [ "$VUL_COUNT" -gt 0 ]; then
    SCORE=$((SCORE - 5))
    echo "[INFO] $VUL_COUNT unpatched vulnerabilities (-5)"
fi

SCORE=$((SCORE > 0 ? SCORE : 0))

echo ""
echo "========== Host Health Score =========="
echo "Host: $(echo "$HOST_INFO" | jq -r '.host_name')"
echo "Score: $SCORE/100"
if [ "$SCORE" -ge 80 ]; then
    echo "Rating: ✅ Good"
elif [ "$SCORE" -ge 50 ]; then
    echo "Rating: ⚠️ Needs Attention"
else
    echo "Rating: 🔴 Critical"
fi
echo "======================================="
```

## 自动修复模式

### 自动隔离恶意文件

```bash
#!/bin/bash
# 自动隔离高危恶意文件

# 检测条件：连续 3 个相同的 malware 告警来自同一主机
EVENTS=$(hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --event_types="[\"malware\"]" --handle_status="unhandled")

echo "$EVENTS" | jq -c '.data_list[]' | while read event; do
    EVENT_ID=$(echo "$event" | jq -r '.event_id')
    HOST_NAME=$(echo "$event" | jq -r '.host_name')
    FILE_PATH=$(echo "$event" | jq -r '.file_path // ""')
    
    if [ -n "$FILE_PATH" ] && [ "$FILE_PATH" != "null" ]; then
        echo "[AUTOFIX] Isolating malware on $HOST_NAME at $FILE_PATH"
        
        # 获取必要参数
        AGENT_ID=$(echo "$event" | jq -r '.agent_id')
        FILE_HASH=$(echo "$event" | jq -r '.file_hash // ""')
        PROCESS_PID=$(echo "$event" | jq -r '.process_pid // 0')
        
        # 执行隔离
        hcloud HSS OperateEvent --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
          --body="{\"operate_type\":\"isolate_and_kill\",\"event_id_list\":[\"$EVENT_ID\"],\"operate_detail_list\":[{\"agent_id\":\"$AGENT_ID\",\"file_hash\":\"$FILE_HASH\",\"file_path\":\"$FILE_PATH\",\"process_pid\":$PROCESS_PID}]}"
        
        echo "[AUTOFIX] Isolation completed for $EVENT_ID"
    fi
done
```

### 自动封锁暴力破解 IP

```bash
#!/bin/bash
# 自动封锁暴力破解 IP

# 检测：登录失败类告警来自同一 IP 超过阈值
ALERTS=$(hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --event_types="[\"login_fail\"]" --handle_status="unhandled")

# 按源 IP 分组统计
echo "$ALERTS" | jq -c '[.data_list[] | .src_ip // empty] | unique[]' 2>/dev/null | while read ip; do
    IP=$(echo "$ip" | tr -d '"')
    COUNT=$(echo "$ALERTS" | jq --arg ip "$IP" '[.data_list[] | select(.src_ip == $ip)] | length')
    
    if [ "$COUNT" -ge 10 ]; then
        echo "[AUTOFIX] Blocking brute-force IP: $IP ($COUNT attempts)"
        hcloud HSS ChangeBlockedIp --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
          --body="{\"data_list\":[{\"block_ip\":\"$IP\",\"host_id\":\"\"}],\"operate_type\":\"block\"}"
    fi
done
```

## 趋势分析与报告

```bash
#!/bin/bash
# 安全趋势周报

echo "=== HSS Weekly Security Report ==="
echo "Report period: $(date -v-7d +%Y-%m-%d 2>/dev/null || date -d "-7 day" +%Y-%m-%d) to $(date +%Y-%m-%d)"
echo ""

# 告警趋势
echo "--- Alert Trends ---"
for event_type in malware ransomware login_fail; do
    COUNT=$(hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
      --event_types="[\"$event_type\"]" | jq '.total_num // 0')
    echo "  $event_type: $COUNT"
done

# 漏洞统计
echo ""
echo "--- Vulnerability Summary ---"
for severity in critical high medium low; do
    COUNT=$(hcloud HSS ListVulnerabilities --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
      --severity="$severity" | jq '.total_num // 0')
    echo "  $severity: $COUNT"
done

echo ""
echo "--- Top 5 Hosts by Alert Count ---"
hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq -r '.data_list // [] | group_by(.host_name) | map({host: .[0].host_name, count: length}) | sort_by(.count) | reverse | .[0:5][] | "\(.host): \(.count) alerts"'
```
