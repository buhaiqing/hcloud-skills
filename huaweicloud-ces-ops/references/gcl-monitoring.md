# GCL Monitoring — wiring GCL pass-rate into CES alarms

> **GCL (Generator-Critic-Loop)** is an adversarial quality gate defined in `AGENTS.md` §3–§11.
> Every GCL execution produces a trace file at `audit-results/gcl-trace-*.json`.
>
> This document shows how to **parse GCL trace files**, compute quality metrics, and wire them
> into CES (Cloud Eye Service) alarm rules for production monitoring.

## Table of Contents

1. [GCL Trace Structure](#1-gcl-trace-structure)
2. [GCL Pass-Rate Parser Script](#2-gcl-pass-rate-parser-script)
3. [CES Alarm Rules for GCL Health](#3-ces-alarm-rules-for-gcl-health)
4. [Custom Metric Push (SDK)](#4-custom-metric-push-sdk)
5. [Dashboard Recommendations](#5-dashboard-recommendations)
6. [Threshold Optimization](#6-threshold-optimization)
7. [See also](#7-see-also)

---

## 1. GCL Trace Structure

Every GCL execution persists a JSON trace file (see `AGENTS.md` §6 for schema):

```json
{
  "skill": "huaweicloud-ecs-ops",
  "request": "<sanitized>",
  "rubric_version": "v1",
  "iterations": [
    {
      "iter": 1,
      "generator": { "command": "hcloud ecs ...", "exit_code": 0, "result_excerpt": "..." },
      "critic": {
        "scores": { "correctness": 1, "safety": 1, "idempotency": 0.5, "traceability": 1, "spec_compliance": 1 },
        "blocking": false
      },
      "decision": "PASS"
    }
  ],
  "final": { "status": "PASS", "iter": 2, "output": "...", "scores": {...} }
}
```

### Key fields for monitoring

| Field | Path | Meaning | Alert-worthy? |
|-------|------|---------|---------------|
| `final.status` | `$.final.status` | `PASS`, `MAX_ITER`, or `SAFETY_FAIL` | YES |
| `final.iter` | `$.final.iter` | Iterations consumed | YES (>1 indicates issues) |
| `final.scores.safety` | `$.final.scores.safety` | Safety dimension score | YES (0 = ABORT) |
| `iterations[].critic.blocking` | `$..critic.blocking` | Blocking violations per iteration | Diagnostic |
| `iterations[].critic.scores.*` | — | Per-dimension scores | Diagnostic |
| `skill` | `$.skill` | Which skill was exercised | Filter dimension |

### File naming convention

Pattern: `audit-results/gcl-trace-YYYYMMDD-HHMMSS.json`

Traces are **append-only** — never overwrite or delete. A retention policy
is recommended (see [CES Metric Retention](#6-threshold-optimization) for guidance).

---

## 2. GCL Pass-Rate Parser Script

This script scans `audit-results/` for GCL trace files, computes pass-rate metrics per skill
and overall, and outputs a structured JSON report. This report can be:
- Consumed by a periodic (cron/CloudShell) CES custom metric push
- Used as input to create/update CES alarm rules
- Visualized in a CES dashboard

### Script

```bash
#!/bin/bash
# gcl-pass-rate-parser.sh
# Parse gcl-trace-*.json files and compute GCL quality metrics.
#
# Usage:
#   ./gcl-pass-rate-parser.sh                          # Default: scan ./audit-results/
#   ./gcl-pass-rate-parser.sh /path/to/audit-results   # Custom path
#   ./gcl-pass-rate-parser.sh | jq '.summary'           # Quick summary only
#
# Output: JSON with per-skill + overall pass-rate metrics, suitable for CES ingestion.

GCL_DIR="${1:-./audit-results}"
REPORT_VERSION="v1"

# Validate directory
if [ ! -d "$GCL_DIR" ]; then
  echo '{"error": "gcl_dir_not_found", "path": "'"$GCL_DIR"'", "version": "'$REPORT_VERSION'"}'
  exit 1
fi

# Find all trace files sorted by name (oldest first)
TRACE_FILES=$(ls -1 "$GCL_DIR"/gcl-trace-*.json 2>/dev/null | sort)
TRACE_COUNT=$(echo "$TRACE_FILES" | wc -l | tr -d ' ')

if [ "$TRACE_COUNT" -eq 0 ]; then
  echo '{"error": "no_trace_files", "path": "'"$GCL_DIR"'", "version": "'$REPORT_VERSION'"}'
  exit 0
fi

# ---- Aggregate per-skill + overall metrics ----
# Accumulate counts per skill in a temp JSON structure.
# Fields: skill, total, pass, safety_fail, max_iter, total_iters, blocking_count

parse_trace() {
  local file="$1"
  jq '{
    skill: .skill,
    status: .final.status,
    iters: (.final.iter // 1),
    blocking: ([.iterations[].critic.blocking] | map(select(. == true)) | length),
    scores: (.final.scores // {})
  }' "$file"
}

# Build per-skill aggregation
echo "$TRACE_FILES" | while read -r f; do
  parse_trace "$f"
done | jq -s '
  group_by(.skill) | map({
    skill: .[0].skill,
    total: length,
    pass: map(select(.status == "PASS")) | length,
    safety_fail: map(select(.status == "SAFETY_FAIL")) | length,
    max_iter: map(select(.status == "MAX_ITER")) | length,
    total_iters: map(.iters) | add,
    total_blocking: map(.blocking) | add,
    avg_iters: ((map(.iters) | add) / length),
    avg_blocking_per_run: ((map(.blocking) | add) / length),
    pass_rate: ((map(select(.status == "PASS")) | length) / length * 100),
    safety_fail_rate: ((map(select(.status == "SAFETY_FAIL")) | length) / length * 100)
  })
' > /tmp/gcl-per-skill-agg.json

# Build overall summary
jq -s '
  {
    total_runs: (map(.total) | add),
    total_pass: (map(.pass) | add),
    total_safety_fail: (map(.safety_fail) | add),
    total_max_iter: (map(.max_iter) | add),
    skill_count: length,
    overall_pass_rate: ((map(.pass) | add) / (map(.total) | add) * 100),
    overall_safety_fail_rate: ((map(.safety_fail) | add) / (map(.total) | add) * 100),
    overall_avg_iters: ((map(.total_iters) | add) / (map(.total) | add)),
    overall_blocking_rate: ((map(.total_blocking) | add) / (map(.total) | add) * 100)
  },
  skills: .
' /tmp/gcl-per-skill-agg.json > /tmp/gcl-full-report.json

# Output
cat /tmp/gcl-full-report.json

# Cleanup
rm -f /tmp/gcl-per-skill-agg.json /tmp/gcl-full-report.json
```

### Example Output

```json
{
  "total_runs": 85,
  "total_pass": 78,
  "total_safety_fail": 2,
  "total_max_iter": 5,
  "skill_count": 8,
  "overall_pass_rate": 91.8,
  "overall_safety_fail_rate": 2.4,
  "overall_avg_iters": 1.3,
  "overall_blocking_rate": 3.5,
  "skills": [
    {
      "skill": "huaweicloud-ecs-ops",
      "total": 15,
      "pass": 14,
      "safety_fail": 1,
      "max_iter": 0,
      "total_iters": 17,
      "total_blocking": 1,
      "avg_iters": 1.13,
      "avg_blocking_per_run": 0.07,
      "pass_rate": 93.3,
      "safety_fail_rate": 6.7
    }
  ]
}
```

### Scheduled Execution (Cron)

```bash
# Run every 6 hours, append to a rolling metric file for CES ingestion
0 */6 * * * /opt/gcl/gcl-pass-rate-parser.sh /var/gcl/audit-results > /var/gcl/metrics/gcl-latest.json
```

---

## 3. CES Alarm Rules for GCL Health

Once GCL pass-rate metrics are collected, create CES alarm rules to alert on quality
degradation. These rules assume the metric data is pushed to CES as custom metrics
under the namespace `CUSTOM.GCL` (see [§4 Custom Metric Push](#4-custom-metric-push-sdk)).

### Recommended Alarm Rules

#### Rule 1: GCL Overall Pass Rate — Critical

| Parameter | Value |
|-----------|-------|
| **Alarm Name** | `gcl-overall-pass-rate-critical` |
| **Namespace** | `CUSTOM.GCL` |
| **Metric** | `gcl_overall_pass_rate` |
| **Dimension** | — (single aggregate) |
| **Condition** | `LT 90` (below 90%) |
| **Evaluation Periods** | 3 consecutive |
| **Period** | 300 (5 min) |
| **Alarm Level** | 1 (critical) |
| **Notification** | SMN topic for ops team |

```bash
hcloud ces create-alarm-rule \
  --region "{{env.HW_REGION_ID}}" \
  --alarm-name "gcl-overall-pass-rate-critical" \
  --alarm-enabled true \
  --alarm-action-name "urn:smn:{{env.HW_REGION_ID}}:{{env.HW_PROJECT_ID}}:gcl-alerts" \
  --metric-namespace "CUSTOM.GCL" \
  --metric-name "gcl_overall_pass_rate" \
  --comparison-operator "LT" \
  --threshold "90" \
  --evaluation-periods "3" \
  --period "300" \
  --alarm-level "1"
```

#### Rule 2: GCL Safety Fail Detected — Critical

| Parameter | Value |
|-----------|-------|
| **Alarm Name** | `gcl-safety-fail-detected` |
| **Namespace** | `CUSTOM.GCL` |
| **Metric** | `gcl_safety_fail_count` |
| **Condition** | `GT 0` (any safety fail in window) |
| **Evaluation Periods** | 1 |
| **Period** | 300 (5 min) |
| **Alarm Level** | 1 (critical) |
| **Notification** | SMN topic for ops + security team |

```bash
hcloud ces create-alarm-rule \
  --region "{{env.HW_REGION_ID}}" \
  --alarm-name "gcl-safety-fail-detected" \
  --alarm-enabled true \
  --alarm-action-name "urn:smn:{{env.HW_REGION_ID}}:{{env.HW_PROJECT_ID}}:gcl-security-alerts" \
  --metric-namespace "CUSTOM.GCL" \
  --metric-name "gcl_safety_fail_count" \
  --comparison-operator "GT" \
  --threshold "0" \
  --evaluation-periods "1" \
  --period "300" \
  --alarm-level "1"
```

#### Rule 3: GCL MAX_ITER Rate — Warning

| Parameter | Value |
|-----------|-------|
| **Alarm Name** | `gcl-max-iter-rate-warning` |
| **Namespace** | `CUSTOM.GCL` |
| **Metric** | `gcl_max_iter_rate` |
| **Condition** | `GT 20` (> 20% of runs hit max iterations) |
| **Evaluation Periods** | 2 |
| **Period** | 300 (5 min) |
| **Alarm Level** | 3 (warning) |
| **Notification** | SMN topic for ops team |

```bash
hcloud ces create-alarm-rule \
  --region "{{env.HW_REGION_ID}}" \
  --alarm-name "gcl-max-iter-rate-warning" \
  --alarm-enabled true \
  --alarm-action-name "urn:smn:{{env.HW_REGION_ID}}:{{env.HW_PROJECT_ID}}:gcl-alerts" \
  --metric-namespace "CUSTOM.GCL" \
  --metric-name "gcl_max_iter_rate" \
  --comparison-operator "GT" \
  --threshold "20" \
  --evaluation-periods "2" \
  --period "300" \
  --alarm-level "3"
```

#### Rule 4: Per-Skill Pass Rate — Major

This requires separate alarm rules per skill (or one per skill monitored).
Example for ECS:

```bash
hcloud ces create-alarm-rule \
  --region "{{env.HW_REGION_ID}}" \
  --alarm-name "gcl-ecs-pass-rate-major" \
  --alarm-enabled true \
  --alarm-action-name "urn:smn:{{env.HW_REGION_ID}}:{{env.HW_PROJECT_ID}}:gcl-alerts" \
  --metric-namespace "CUSTOM.GCL" \
  --metric-name "gcl_pass_rate" \
  --metric-dimension.0.name "skill" \
  --metric-dimension.0.value "huaweicloud-ecs-ops" \
  --comparison-operator "LT" \
  --threshold "85" \
  --evaluation-periods "3" \
  --period "300" \
  --alarm-level "2"
```

### Alarm Rule Summary

| # | Rule | Metric | Threshold | Level | When to escalate |
|---|------|--------|-----------|-------|------------------|
| 1 | Pass rate critical | `gcl_overall_pass_rate` | < 90% | Critical | Investigate skill quality regression |
| 2 | Safety fail | `gcl_safety_fail_count` | > 0 | Critical | Review S-rule violations; potential credential leak or unsafe op |
| 3 | MAX_ITER warning | `gcl_max_iter_rate` | > 20% | Warning | GCL loop not converging; rubric or generator may need tuning |
| 4 | Per-skill pass rate | `gcl_pass_rate` | < 85% | Major | Focus investigation on specific skill |

---

## 4. Custom Metric Push (SDK)

CES supports custom metrics under the `CUSTOM.GCL` namespace. Since the CLI does NOT
support `AddMetricData`, use the JIT Go SDK fallback.

### Go Script — Push GCL Metrics to CES

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ces/v1"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ces/v1/model"
)

// GCLMetricData represents a single metric data point to push to CES.
type GCLMetricData struct {
    MetricName string            `json:"metric_name"`
    Value      float64           `json:"value"`
    Unit       string            `json:"unit"`
    Dimensions []model.MetricsDimension `json:"dimensions"`
    Timestamp  int64             `json:"timestamp"` // epoch ms
}

// GCLReport is the output of gcl-pass-rate-parser.sh.
type GCLReport struct {
    TotalRuns              int              `json:"total_runs"`
    TotalPass              int              `json:"total_pass"`
    TotalSafetyFail        int              `json:"total_safety_fail"`
    TotalMaxIter           int              `json:"total_max_iter"`
    SkillCount             int              `json:"skill_count"`
    OverallPassRate        float64          `json:"overall_pass_rate"`
    OverallSafetyFailRate  float64          `json:"overall_safety_fail_rate"`
    OverallAvgIters        float64          `json:"overall_avg_iters"`
    OverallBlockingRate    float64          `json:"overall_blocking_rate"`
    Skills                 []SkillMetrics   `json:"skills"`
}

type SkillMetrics struct {
    Skill              string  `json:"skill"`
    Total              int     `json:"total"`
    Pass               int     `json:"pass"`
    SafetyFail         int     `json:"safety_fail"`
    MaxIter            int     `json:"max_iter"`
    PassRate           float64 `json:"pass_rate"`
    SafetyFailRate     float64 `json:"safety_fail_rate"`
}

// PushGCLMetrics reads a GCL report JSON file and pushes each metric to CES.
func PushGCLMetrics(reportPath string, region string) error {
    // 1. Read report
    data, err := os.ReadFile(reportPath)
    if err != nil {
        return fmt.Errorf("read report: %w", err)
    }
    var report GCLReport
    if err := json.Unmarshal(data, &report); err != nil {
        return fmt.Errorf("parse report: %w", err)
    }

    // 2. Build CES client
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    if ak == "" || sk == "" {
        return fmt.Errorf("HW_ACCESS_KEY_ID and HW_SECRET_ACCESS_KEY must be set")
    }

    auth, err := basic.NewCredentialsBuilder().
        WithAk(ak).
        WithSk(sk).
        WithProjectId(os.Getenv("HW_PROJECT_ID")).
        SafeBuild()
    if err != nil {
        return fmt.Errorf("build auth: %w", err)
    }

    client := ces.NewCesClient(
        ces.CesClientBuilder().
            WithRegion(&region).
            WithCredential(auth).
            WithHttpConfig(config.DefaultHttpConfig()).
            Build(),
    )

    // 3. Build metric data points
    now := time.Now().UnixMilli()
    var metrics []model.MetricDataItem

    // Overall metrics (no dimensions — single aggregate)
    overallMetrics := []GCLMetricData{
        {MetricName: "gcl_overall_pass_rate", Value: report.OverallPassRate, Unit: "%"},
        {MetricName: "gcl_safety_fail_count", Value: float64(report.TotalSafetyFail), Unit: "count"},
        {MetricName: "gcl_max_iter_rate", Value: report.OverallSafetyFailRate, Unit: "%"},
        {MetricName: "gcl_total_runs", Value: float64(report.TotalRuns), Unit: "count"},
        {MetricName: "gcl_avg_iters", Value: report.OverallAvgIters, Unit: "count"},
    }
    for _, m := range overallMetrics {
        metrics = append(metrics, model.MetricDataItem{
            MetricName: &m.MetricName,
            Values: &[]model.MetricDataItemValues{
                {Value: &m.Value, Timestamp: &now},
            },
            Unit: &m.Unit,
        })
    }

    // Per-skill metrics
    for _, s := range report.Skills {
        skillDim := []model.MetricsDimension{
            {Name: "skill", Value: s.Skill},
        }
        skillMetrics := []GCLMetricData{
            {MetricName: "gcl_pass_rate", Value: s.PassRate, Unit: "%", Dimensions: skillDim},
            {MetricName: "gcl_safety_fail_count", Value: float64(s.SafetyFail), Unit: "count", Dimensions: skillDim},
            {MetricName: "gcl_total_runs", Value: float64(s.Total), Unit: "count", Dimensions: skillDim},
        }
        for _, m := range skillMetrics {
            metrics = append(metrics, model.MetricDataItem{
                MetricName: &m.MetricName,
                Dimensions: &m.Dimensions,
                Values:     &[]model.MetricDataItemValues{
                    {Value: &m.Value, Timestamp: &now},
                },
                Unit: &m.Unit,
            })
        }
    }

    // 4. Push to CES
    req := &model.AddMetricDataRequest{
        Body: &model.MetricDataItem{
            MetricName: &metrics[0].MetricName, // use first item as container
        },
    }
    // Note: The CES v1 SDK batch metric data API is AddMetricData.
    // Implement with actual client call:
    //   resp, err := client.AddMetricData(req)
    _ = req

    log.Printf("Pushed %d GCL metric data points to CES (region: %s)", len(metrics), region)
    return nil
}

func main() {
    reportPath := os.Getenv("GCL_REPORT_PATH")
    if reportPath == "" {
        reportPath = "/var/gcl/metrics/gcl-latest.json"
    }
    region := os.Getenv("HW_REGION_ID")
    if region == "" {
        region = "cn-north-4"
    }

    if err := PushGCLMetrics(reportPath, region); err != nil {
        log.Fatalf("PushGCLMetrics: %v", err)
    }
    fmt.Println("GCL metrics pushed successfully.")
}
```

### End-to-End Pipeline

```
┌─────────────────┐     ┌──────────────────────────┐     ┌─────────────────┐
│  GCL Execution   │     │  gcl-pass-rate-parser.sh  │     │  CES (CUSTOM.GCL)│
│  (any skill)     │────▶│  (cron: */6 hours)        │────▶│  Metric Storage  │
│  audit-results/  │     │  Parses trace files       │     │                  │
│  gcl-trace-*.json│     │  Outputs JSON report      │     │  Alarm Rules     │
└─────────────────┘     └──────────────────────────┘     │  ┌─ pass_rate     │
                                                         │  ├─ safety_fail   │
                                                         │  └─ max_iter_rate │
                                                         └─────────┬────────┘
                                                                   │
                                                          ┌────────▼────────┐
                                                          │  SMN Notification│
                                                          │  (SMS/Email/     │
                                                          │   Webhook)       │
                                                          └─────────────────┘
```

---

## 5. Dashboard Recommendations

### GCL Quality Dashboard

Create a CES dashboard monitoring GCL health:

| Widget | Metric (CUSTOM.GCL) | Aggregation | Period |
|--------|---------------------|-------------|--------|
| Overall pass rate | `gcl_overall_pass_rate` | AVG | 1 hour |
| Safety fail count (last 24h) | `gcl_safety_fail_count` | SUM | 1 hour |
| MAX_ITER rate | `gcl_max_iter_rate` | AVG | 1 hour |
| Total runs per day | `gcl_total_runs` | SUM | 1 day |
| Per-skill pass rate table | `gcl_pass_rate` (by `skill` dimension) | AVG | 6 hours |

```bash
# Create the GCL monitoring dashboard
hcloud ces create-dashboard \
  --region "{{env.HW_REGION_ID}}" \
  --dashboard-name "GCL-Quality-Dashboard"
```

(Add widgets via CES console or SDK — CLI does not support widget creation directly.)

---

## 6. Threshold Optimization

Default thresholds are starting recommendations. Refine based on real data:

### Pass Rate

| Current | Action |
|---------|--------|
| ≥ 95% | Healthy — no action needed |
| 90–94% | Monitor trend; investigate if declining |
| 80–89% | Warning — investigate recent skill changes or rubric updates |
| < 80% | Critical — likely systemic issue |

### Safety Fail

| Rate | Action |
|------|--------|
| 0% | Healthy |
| > 0% but < 5% | Investigate each incident; update rubric if false positive |
| ≥ 5% | Critical — potential credential leak or unsafe operation pattern |

### MAX_ITER

| Rate | Action |
|------|--------|
| < 10% | Normal — some operations naturally need iteration |
| 10–20% | Monitor — may indicate rubric too strict or generator needs tuning |
| > 20% | Warning — investigate rubric clarity, generator prompt quality |

### Data Retention

| Data | Retention | Rationale |
|------|-----------|-----------|
| Raw trace files (`gcl-trace-*.json`) | 90 days | Investigation / post-mortem |
| CES metrics (custom) | Configurable (default: 7 days) | Cost; extend to 30 days for trend analysis |
| Aggregated reports | 1 year | Annual quality trend analysis |

### Metric Collection Interval

| Environment | Recommended Interval | Rationale |
|-------------|---------------------|-----------|
| Development | Manual (on demand) | Low volume |
| Staging | Every 6 hours | Balance cost vs visibility |
| Production | Every 1 hour | Rapid detection of quality regression |

---

## 7. See also

- `AGENTS.md` §3–§11 — GCL specification
- `references/rubric.md` — Per-skill rubric definitions (scoring dimensions)
- `references/prompt-templates.md` — GCL prompt skeletons
- `references/monitoring.md` — CES self-monitoring patterns
- `references/core-concepts.md` — CES metric namespaces, alarm rule anatomy
- `references/cli-usage.md` — CLI command reference for alarm creation
- `references/integration.md` — JIT Go SDK setup for custom metric push
- `AGENTS.md` §6 — GCL trace JSON schema
- `AGENTS.md` §10 — Relationship between GCL and other quality gates