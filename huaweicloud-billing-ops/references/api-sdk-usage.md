# API & SDK Usage — BSS

## Operation-to-API Mapping

| Op | Operation Name | BSS API | Method | Pagination |
|----|---------------|---------|--------|------------|
| 1 | Account Balance | `ShowCustomerAccountInfo` | GET | N/A |
| 2 | Monthly Bill | `ListCustomerselfResourceRecords` | GET | limit/offset |
| 3 | Bill Detail | `ListCustomerselfResourceRecords` | GET (DETAIL) | limit/offset |
| 4 | Resource Usage | `ListCustomerResourceUsage` | GET | limit/offset |
| 5 | Monthly Summary | `ListMonthlyExpenditures` | GET | limit/offset |
| 6 | Budget Alert | `CreateBudget` / `UpdateBudget` / `DeleteBudget` | POST/PUT/DELETE | N/A |
| 7 | Resource Package | `ListResourcePackages` / `RefundResourcePackage` | GET/POST | limit/offset |
| 8 | Cost Analysis | `ListCosts` | POST | offset |
| 9 | Unit Economics | Compute-only (Op 8 + division) | N/A | N/A |
| 10 | TCO Comparison | Compute-only (Op 4 + pricing API) | N/A | N/A |
| 11 | Anomaly Detection | Compute-only (Op 8 historical compare) | N/A | N/A |
| 12 | Optimization Mining | Compute-only (Op 3 + usage analysis) | N/A | N/A |
| 13 | Reserved Sizing | Compute-only (Op 4 + Op 8) | N/A | N/A |
| 14 | Closed-Loop Tracker | File-based (JSONL log at ~/.hcloud) | N/A | N/A |
| 15 | Maturity Assessment | Compute-only (checklist evaluation) | N/A | N/A |

## Required Fields

### ShowCustomerAccountInfo

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| — | — | — | No required parameters (single response per account) |

### ListCustomerselfResourceRecords

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `cycle` | string | Yes | Billing cycle: YYYY-MM |
| `method` | string | Yes | SUMMARY or DETAIL |
| `limit` | int | No | Page size (default 100) |
| `offset` | int | No | Pagination offset |

### CreateBudget

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Budget name (unique per account) |
| `amount` | decimal | Yes | Budget amount in CNY |
| `thresholds` | array | Yes | Alert thresholds: [80, 90, 100] |
| `notify_by` | array | Yes | Notification methods: ["email", "sms"] |

### ListResourcePackages

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | No | active / expired / refunded |

### RefundResourcePackage

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `package_id` | string | Yes | Resource package ID |

## Pagination Pattern

For APIs supporting pagination:

```
Page 1: GET ...?limit=100&offset=0
Page 2: GET ...?limit=100&offset=100
```

Response includes:
- `count` — total records
- `records` — current page data
- `offset` — for next page

## Request/Response Examples

### Balance Query

**Request:**
```bash
hcloud BSS ShowCustomerAccountInfo
```

**Response:**
```json
{
  "account_balances": [
    {
      "amount": "1250.50",
      "currency": "CNY",
      "credit_amount": "0.00",
      "designated_amount": "0.00"
    }
  ]
}
```

### Bill Query (Summary)

**Request:**
```bash
hcloud BSS ListCustomerselfResourceRecords --cycle="2026-05" --method="SUMMARY"
```

**Response:**
```json
{
  "records": [
    {
      "cycle": "2026-05",
      "bill_type": 1,
      "customer_id": "custo...",
      "currency": "CNY",
      "consumption": "8520.00",
      "cash_amount": "8520.00",
      "debt_amount": "0.00"
    }
  ]
}
```

### Bill Query (Detail)

**Request:**
```bash
hcloud BSS ListCustomerselfResourceRecords --cycle="2026-05" --method="DETAIL" --limit=100
```

**Response:**
```json
{
  "records": [
    {
      "cycle": "2026-05",
      "bill_type": 1,
      "resource_id": "res-abc123",
      "resource_name": "ecs-web-01",
      "service_type": "ECS",
      "region": "cn-north-4",
      "amount": 352.50,
      "usage_type": "vCPU-hour",
      "usage": 744.0,
      "unit": "小时",
      "unit_price": 0.47
    }
  ],
  "count": 1
}
```

## Authentication

BSS API uses Huawei Cloud standard AK/SK authentication via `hcloud` CLI. No additional auth setup required beyond `{{env.HW_ACCESS_KEY_ID}}` and `{{env.HW_SECRET_ACCESS_KEY}}`.

## Error Handling

| Error Code | Message | Cause | Recovery |
|------------|---------|-------|----------|
| BSS.0001 | Invalid credential | Wrong or expired AK/SK | Verify credentials |
| BSS.0002 | Auth failure | AK lacks BSS permission | Grant `bss:*` policy |
| BSS.0003 | Invalid parameter | Wrong cycle format | Use YYYY-MM |
| BSS.0100 | Internal error | BSS system error | Retry with backoff |
| BSS.0101 | No data found | No records for cycle | Inform user |
| BSS.0201 | Budget limit reached | Max 50 budgets | Delete unused budgets |
| BSS.0202 | Package not refundable | Package policy | Inform user |
| BSS.0901 | Throttled | Rate limit exceeded | Retry after 3s |