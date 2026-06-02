# Troubleshooting — BSS (费用中心)

## Error Code Taxonomy

| Code | Message | Cause | Recovery Action | Retryable |
|------|---------|-------|----------------|-----------|
| BSS.0001 | Invalid credential | AK/SK expired or invalid | Regenerate AK/SK in IAM console | No |
| BSS.0002 | Auth failure | AK lacks BSS permission | Grant `bss:*` or `bss:bill:view` policy | No |
| BSS.0003 | Invalid parameter | Wrong input format | Verify parameter types and formats | No |
| BSS.0004 | Missing parameter | Required field empty | Provide all required fields | No |
| BSS.0010 | Token expired | Session token expired | Refresh IAM token | Yes |
| BSS.0100 | Internal error | BSS server-side error | Wait 5s and retry | Yes (3x, exponential) |
| BSS.0101 | No data found | No records for given cycle | Try a different billing cycle | No |
| BSS.0102 | Cycle not closed | Current month still in progress | Wait until month-end for full data | No |
| BSS.0201 | Budget limit | Max 50 budgets reached | Delete unused budgets | No |
| BSS.0202 | Duplicate budget | Budget name already exists | Use a different name | No |
| BSS.0203 | Invalid amount | Budget amount ≤ 0 | Provide amount > 0 | No |
| BSS.0301 | Package not found | Invalid package ID | Verify package ID | No |
| BSS.0302 | Package not refundable | Package type does not support refund | Check package terms | No |
| BSS.0303 | Refund limit | Exceeded monthly refund count | Wait for next month | No |
| BSS.0901 | Throttled | Rate limit exceeded (20 req/s) | Wait 3-5s and retry | Yes |
| BSS.0902 | Timeout | Request timeout > 30s | Reduce query range, retry | Yes |
| BSS.0999 | Unknown error | Undefined system error | Contact Huawei Cloud support | No |

## Diagnostic Flows

### Flow 1: "Bill data not found"

```
User: "帮我查上个月的账单"
   ↓
Get billing cycle (default: last month)
   ↓
Call ListCustomerselfResourceRecords
   ↓
ERROR BSS.0101 (no data)?
   ├─ YES → Check cycle is correct → Try previous month
   │        → Check account permissions (may not be billing account)
   │        → Inform user
   └─ NO  → Return records
```

### Flow 2: "Budget creation failed"

```
User: "设置本月预算¥10,000"
   ↓
Call CreateBudget
   ↓
ERROR BSS.0201 (budget limit)?
   ├─ YES → List existing budgets → Suggest delete unused → Retry
   └─ NO  → 
ERROR BSS.0002 (auth)?
   ├─ YES → Check IAM policy → Need bss:budget:create
   └─ NO  → Return error details
```

### Flow 3: "Resource package refund failed"

```
User: "退订资源包 xxx"
   ↓
Safety gate: confirm with user
   ↓
Call RefundResourcePackage
   ↓
ERROR BSS.0302 (not refundable)?
   ├─ YES → Inform user: package type cannot be refunded
   └─ NO  →
ERROR BSS.0303 (refund limit)?
   ├─ YES → Inform user: monthly refund count exceeded
   └─ NO  → Return error details
```

### Flow 4: "Cost analysis returns empty"

```
User: "分析最近3个月的成本"
   ↓
Call ListCosts with time-range
   ↓
Empty result?
   ├─ YES → Check if enterprise project filter is too narrow
   │        → Check if time range includes current month (may have delay)
   │        → Widen filter and retry
   └─ NO  → Return analysis
```

## Common Issues

### Issue: "Can't see any billing data"

**Likely causes:**
1. IAM sub-account without `bss:bill:view` permission
2. Wrong region endpoint
3. Account is not the main billing account

**Diagnosis:**
```bash
# Check account type
hcloud BSS ShowCustomerAccountInfo

# Check endpoint
echo $HW_BSS_ENDPOINT
```

### Issue: "Costs seem too high"

**Diagnosis:**
```bash
# Compare with previous month
hcloud BSS ListCosts --time-range="LAST_60_DAYS" --interval="MONTHLY"

# Check if new resources were deployed
hcloud BSS ListCustomerselfResourceRecords --cycle="YYYY-MM" --method="DETAIL"

# Run anomaly detection (Op 11)
```

### Issue: "API calls fail intermittently"

**Diagnosis:**
- Check rate limit: throttle at 20 req/s
- Implement backoff: start with 2s, double on each retry, max 3 retries
- Reduce batch size: query one service at a time