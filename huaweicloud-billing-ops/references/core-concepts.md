# Core Concepts — BSS (费用中心)

## BSS Architecture

Huawei Cloud Billing & Cost Management (BSS) provides account balance management, bill query, cost analysis, and resource package services. BSS is a cross-region service — bills aggregate all region costs.

### BSS API Versions

| Version | Status | Scope | Key Difference |
|---------|--------|-------|---------------|
| BSS v1 | Active | Basic billing operations | Simple pagination, fewer fields |
| BSS v2 | Recommended | Full cost management | Enhanced filtering, enterprise project support, cost analysis API |

## Account Types

| Type | Description | BSS Access |
|------|-------------|------------|
| Main Account | Root billing account | Full access to all billing APIs |
| IAM Sub-account | Child account under main | Requires `bss:*` policy grant for billing access |
| Ephemeral | Temporary credentials | Read-only billing access possible |

## Billing Models

| Model | Description | Use Case | Cost Stability |
|-------|-------------|----------|---------------|
| Pay-per-use (按需) | Pay by actual usage, per hour/day/month | Variable workloads, testing | Low (fluctuates) |
| Monthly/Yearly Subscription (包年包月) | Pre-pay for 1-month to 3-year term | Stable production workloads | High (fixed) |
| Spot Market (竞价) | Bid-based pricing for spare capacity | Batch jobs, fault-tolerant workloads | Very Low (can be reclaimed) |

## Resource Packages

Resource packages are pre-purchased capacity bundles for specific services:

| Package Type | Applied Services | Unit |
|-------------|-----------------|------|
| Storage Package | OBS, EVS | GB |
| Data Transfer Package | EIP, NAT, CDN | GB |
| API Call Package | API Gateway | Calls |
| Compute Package | ECS, CCE | vCPU-hours |

## Enterprise Project Cost Attribution

Enterprise projects allow cost grouping by business unit, environment, or project. When resources are tagged with an enterprise project ID, costs appear under that project in billing reports.

## Regions and Endpoints

| Environment | Endpoint |
|-------------|----------|
| China Domestic | `bss.myhuaweicloud.com` |
| International | `bss-intl.myhuaweicloud.com` |

## Quotas and Limits

| Limit | Value | Notes |
|-------|-------|-------|
| API rate limit | 20 requests/second per AK | Throttling returns 429 |
| Bill query range | Max 31 days per call | Use month cycles |
| Budget count | 50 budgets per account | Soft limit, can request increase |
| Resource package refunds | 5 per month per package type | Some packages are non-refundable |
| Historical bill retention | 18 months | Bills older than 18 months unavailable |

## Dependency Graph

```
IAM (auth) ──▶ BSS (billing) ──▶ CES (budget alerts)
                  │
                  ▼
            Product Services (ECS, RDS, OBS, etc.)
            ── generate usage data → BSS bills
```

## Error Code Categories

| Category | Code Range | Examples |
|----------|-----------|---------|
| Auth errors | BSS.0001–0010 | Invalid AK/SK, expired token |
| Input errors | BSS.0100–0200 | Invalid cycle, missing parameter |
| Business errors | BSS.0201–0300 | Budget limit, package non-refundable |
| System errors | BSS.0900–0999 | Internal error, timeout |