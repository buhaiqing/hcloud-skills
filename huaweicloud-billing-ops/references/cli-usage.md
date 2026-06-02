# CLI Usage — BSS (hcloud)

## CLI Command Map

| Operation | hcloud Command | Coverage |
|-----------|---------------|----------|
| Account Balance | `hcloud BSS ShowCustomerAccountInfo` | ✅ Full |
| Monthly Bill | `hcloud BSS ListCustomerselfResourceRecords --cycle=YYYY-MM --method=SUMMARY` | ✅ Full |
| Bill Detail | `hcloud BSS ListCustomerselfResourceRecords --cycle=YYYY-MM --method=DETAIL` | ✅ Full |
| Resource Usage | `hcloud BSS ListCustomerResourceUsage --cycle=YYYY-MM` | ✅ Full |
| Monthly Expenditure | `hcloud BSS ListMonthlyExpenditures --cycle=YYYY-MM` | ✅ Full |
| Create Budget | `hcloud BSS CreateBudget --name="..." --amount=... --thresholds="..."` | ⚠️ Limited (notification channels may need API) |
| List Budgets | `hcloud BSS ListBudgets` | ✅ Full |
| Delete Budget | `hcloud BSS DeleteBudget --name="..."` | ✅ Full |
| List Resource Packages | `hcloud BSS ListResourcePackages` | ✅ Full |
| Refund Resource Package | `hcloud BSS RefundResourcePackage --package-id="..."` | ✅ Full |
| Cost Analysis | `hcloud BSS ListCosts --time-range="..." --group-by="..."` | ✅ Full |

## CLI Coverage Gaps

| Operation | CLI Support | Fallback |
|-----------|-------------|----------|
| Unit Economics (Op 9) | ❌ Compute-only | Calculation in agent |
| TCO Comparison (Op 10) | ❌ Compute-only | Calculation in agent |
| Anomaly Detection (Op 11) | ❌ Compute-only | Pattern matching in agent |
| Optimization Mining (Op 12) | ❌ Compute-only | Rule engine in agent |
| Reserved Sizing (Op 13) | ❌ Compute-only | Formula calculation in agent |
| Closed-Loop Tracker (Op 14) | ❌ File-based | JSONL log management |
| Maturity Assessment (Op 15) | ❌ Checklist | Evaluation questionnaire |

## CLI Output Format

All BSS CLI responses return JSON. Parse with `--format=json`.

```bash
# Get JSON output
hcloud BSS ShowCustomerAccountInfo --format=json
```

## Common CLI Examples

```bash
# 1. Check balance
hcloud BSS ShowCustomerAccountInfo

# 2. Get monthly bill summary
hcloud BSS ListCustomerselfResourceRecords --cycle="2026-05" --method="SUMMARY"

# 3. Get detailed bill records (paginated)
hcloud BSS ListCustomerselfResourceRecords --cycle="2026-05" --method="DETAIL" --limit=100

# 4. List active resource packages
hcloud BSS ListResourcePackages --status="active"

# 5. Create budget alert
hcloud BSS CreateBudget --name="生产环境" --amount="10000" --thresholds="80,90,100"

# 6. Cost analysis by service
hcloud BSS ListCosts --time-range="LAST_30_DAYS" --group-by="service_type"

# 7. Cost analysis by enterprise project
hcloud BSS ListCosts --time-range="THIS_MONTH" --group-by="enterprise_project"
```

## Error Output

BSS CLI errors follow this format:

```json
{
  "error_code": "BSS.0002",
  "error_msg": "Invalid credential or auth failure"
}
```