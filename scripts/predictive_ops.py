#!/usr/bin/env python3
"""Predictive Operations — trend analysis and proactive intervention.

Shifts from reactive "firefighting" to proactive "fire prevention" by analyzing
metric trends, predicting threshold breaches, and recommending preemptive actions.

Usage:
  python3 scripts/predictive_ops.py forecast --skill huaweicloud-ecs-ops --metric cpu_utilization --data <metrics.json>
  python3 scripts/predictive_ops.py scan --root . [--skill huaweicloud-ecs-ops]
  python3 scripts/predictive_ops.py recommend --forecast <forecast.json>
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

UTC = timezone.utc  # noqa: UP017

ROOT_DEFAULT = Path(__file__).resolve().parents[1]

# --- Prediction Models ---

# Threshold definitions per metric (namespace → metric → thresholds)
METRIC_THRESHOLDS: dict[str, dict[str, Any]] = {
    "cpu_utilization": {"warning": 70.0, "critical": 90.0, "unit": "%"},
    "mem_utilization": {"warning": 75.0, "critical": 92.0, "unit": "%"},
    "disk_utilization": {"warning": 80.0, "critical": 95.0, "unit": "%"},
    "network_in_rate": {"warning": 800_000_000, "critical": 950_000_000, "unit": "bps"},
    "network_out_rate": {"warning": 800_000_000, "critical": 950_000_000, "unit": "bps"},
    "iops_utilization": {"warning": 70.0, "critical": 90.0, "unit": "%"},
    "connection_utilization": {"warning": 60.0, "critical": 85.0, "unit": "%"},
    "error_rate": {"warning": 1.0, "critical": 5.0, "unit": "%"},
    "latency_p99": {"warning": 500.0, "critical": 2000.0, "unit": "ms"},
}

# Proactive intervention playbooks
INTERVENTION_PLAYBOOKS: dict[str, dict[str, Any]] = {
    "cpu_utilization": {
        "preemptive_actions": [
            "Identify top CPU consumers: hcloud ecs list-server-monitoring --metric cpu",
            "Right-size recommendation: check if instance is over/under-provisioned",
            "Schedule scale-out before predicted breach",
        ],
        "lead_time_hours": 4,
    },
    "mem_utilization": {
        "preemptive_actions": [
            "Check for memory leaks in application processes",
            "Evaluate if swap configuration is adequate",
            "Plan instance resize to higher memory tier",
        ],
        "lead_time_hours": 6,
    },
    "disk_utilization": {
        "preemptive_actions": [
            "Identify large files: du -sh /var/log/* | sort -rh | head",
            "Rotate/compress old logs",
            "Expand volume: hcloud evs extend-volume",
            "Archive cold data to OBS",
        ],
        "lead_time_hours": 24,
    },
    "connection_utilization": {
        "preemptive_actions": [
            "Check connection pool settings",
            "Identify idle connections for cleanup",
            "Scale connection limit or add read replicas",
        ],
        "lead_time_hours": 2,
    },
    "error_rate": {
        "preemptive_actions": [
            "Correlate with recent deployments (CTS event query)",
            "Check downstream dependency health",
            "Enable circuit breaker if not active",
        ],
        "lead_time_hours": 1,
    },
}


def _now_iso() -> str:
    return datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


def linear_regression(points: list[tuple[float, float]]) -> tuple[float, float]:
    """Simple least-squares linear regression. Returns (slope, intercept)."""
    n = len(points)
    if n < 2:
        return (0.0, points[0][1] if points else 0.0)
    sum_x = sum(p[0] for p in points)
    sum_y = sum(p[1] for p in points)
    sum_xy = sum(p[0] * p[1] for p in points)
    sum_x2 = sum(p[0] ** 2 for p in points)
    denom = n * sum_x2 - sum_x ** 2
    if abs(denom) < 1e-10:
        return (0.0, sum_y / n)
    slope = (n * sum_xy - sum_x * sum_y) / denom
    intercept = (sum_y - slope * sum_x) / n
    return (slope, intercept)


def exponential_smoothing(values: list[float], alpha: float = 0.3) -> list[float]:
    """Single exponential smoothing for trend detection."""
    if not values:
        return []
    smoothed = [values[0]]
    for v in values[1:]:
        smoothed.append(alpha * v + (1 - alpha) * smoothed[-1])
    return smoothed


def detect_trend(values: list[float]) -> dict[str, Any]:
    """Detect trend direction and strength from a time series."""
    if len(values) < 3:
        return {"direction": "insufficient_data", "strength": 0.0, "slope": 0.0, "relative_slope": 0.0}

    points = [(float(i), v) for i, v in enumerate(values)]
    slope, _ = linear_regression(points)

    # Normalize slope by mean to get relative trend
    mean_val = sum(values) / len(values)
    relative_slope = slope / max(abs(mean_val), 1e-10)

    if abs(relative_slope) < 0.01:
        direction = "stable"
    elif relative_slope > 0:
        direction = "increasing"
    else:
        direction = "decreasing"

    # Strength: R² approximation
    smoothed = exponential_smoothing(values)
    ss_res = sum((v - s) ** 2 for v, s in zip(values, smoothed))
    ss_tot = sum((v - mean_val) ** 2 for v in values)
    r_squared = 1 - (ss_res / max(ss_tot, 1e-10))

    return {
        "direction": direction,
        "strength": round(max(0.0, r_squared), 3),
        "slope": round(slope, 4),
        "relative_slope": round(relative_slope, 4),
    }


def predict_breach_time(
    values: list[float],
    threshold: float,
    interval_hours: float = 1.0,
) -> dict[str, Any]:
    """Predict when a metric will breach a threshold."""
    if not values:
        return {"will_breach": False, "hours_to_breach": None}

    points = [(float(i) * interval_hours, v) for i, v in enumerate(values)]
    slope, intercept = linear_regression(points)

    current = values[-1]
    if current >= threshold:
        return {"will_breach": True, "hours_to_breach": 0, "already_breached": True}

    if slope <= 0:
        return {"will_breach": False, "hours_to_breach": None, "trend": "stable_or_decreasing"}

    # Time to reach threshold: threshold = slope * t + intercept
    # Solve for t from current time
    current_time = points[-1][0]
    breach_time = (threshold - intercept) / slope
    hours_to_breach = breach_time - current_time

    if hours_to_breach < 0:
        return {"will_breach": False, "hours_to_breach": None}

    return {
        "will_breach": True,
        "hours_to_breach": round(hours_to_breach, 1),
        "predicted_value_at_breach": threshold,
        "current_value": round(current, 2),
        "slope_per_hour": round(slope, 4),
    }


def generate_forecast(
    metric: str,
    values: list[float],
    interval_hours: float = 1.0,
) -> dict[str, Any]:
    """Generate a complete forecast for a metric time series."""
    thresholds = METRIC_THRESHOLDS.get(metric, {"warning": 80.0, "critical": 95.0, "unit": "unknown"})
    trend = detect_trend(values)

    warning_breach = predict_breach_time(values, thresholds["warning"], interval_hours)
    critical_breach = predict_breach_time(values, thresholds["critical"], interval_hours)

    # Risk score: combines trend strength and proximity to threshold
    current = values[-1] if values else 0.0
    proximity = current / thresholds["critical"] if thresholds["critical"] > 0 else 0.0
    trend_factor = trend["strength"] if trend["direction"] == "increasing" else 0.0
    risk_score = round(min(1.0, proximity * 0.6 + trend_factor * 0.4), 3)

    # Urgency level
    if critical_breach.get("already_breached"):
        urgency = "critical_now"
    elif critical_breach.get("will_breach") and (critical_breach.get("hours_to_breach") or 999) < 4:
        urgency = "critical_imminent"
    elif warning_breach.get("will_breach") and (warning_breach.get("hours_to_breach") or 999) < 12:
        urgency = "warning_soon"
    elif risk_score > 0.7:
        urgency = "elevated"
    else:
        urgency = "normal"

    return {
        "metric": metric,
        "forecasted_at": _now_iso(),
        "data_points": len(values),
        "current_value": round(current, 2),
        "unit": thresholds["unit"],
        "trend": trend,
        "thresholds": thresholds,
        "warning_breach": warning_breach,
        "critical_breach": critical_breach,
        "risk_score": risk_score,
        "urgency": urgency,
    }


def get_recommendations(forecast: dict[str, Any]) -> dict[str, Any]:
    """Generate proactive recommendations based on forecast."""
    metric = forecast["metric"]
    urgency = forecast["urgency"]
    playbook = INTERVENTION_PLAYBOOKS.get(metric, {})

    recommendations: list[dict[str, Any]] = []

    if urgency in ("critical_now", "critical_imminent"):
        recommendations.append({
            "priority": "P0",
            "action": "Immediate intervention required",
            "details": playbook.get("preemptive_actions", ["Escalate to on-call engineer"])[:2],
            "lead_time_hours": 0,
        })
    elif urgency == "warning_soon":
        recommendations.append({
            "priority": "P1",
            "action": "Schedule proactive intervention",
            "details": playbook.get("preemptive_actions", ["Monitor closely"]),
            "lead_time_hours": playbook.get("lead_time_hours", 4),
        })
    elif urgency == "elevated":
        recommendations.append({
            "priority": "P2",
            "action": "Plan capacity review",
            "details": ["Review resource utilization trends", "Evaluate right-sizing options"],
            "lead_time_hours": 24,
        })

    # Always add monitoring recommendation
    if forecast["trend"]["direction"] == "increasing":
        recommendations.append({
            "priority": "P3",
            "action": "Enhance monitoring",
            "details": [
                f"Reduce alarm interval for {metric}",
                "Add derivative-based alarm (rate of change)",
            ],
            "lead_time_hours": 1,
        })

    return {
        "metric": metric,
        "urgency": urgency,
        "risk_score": forecast["risk_score"],
        "recommendations": recommendations,
        "generated_at": _now_iso(),
    }


def cmd_forecast(args: argparse.Namespace) -> int:
    """Generate forecast from metric data."""
    data_path = Path(args.data)
    if not data_path.exists():
        print(f"ERROR: data file not found: {data_path}", file=sys.stderr)
        return 1

    raw = json.loads(data_path.read_text(encoding="utf-8"))
    # Accept either {"values": [...]} or bare list
    values = raw if isinstance(raw, list) else raw.get("values", [])
    if not values:
        print("ERROR: no data points in input", file=sys.stderr)
        return 1

    values = [float(v) for v in values]
    forecast = generate_forecast(args.metric, values, args.interval)

    if args.json:
        print(json.dumps(forecast, indent=2, ensure_ascii=False))
    else:
        print(f"=== Forecast: {args.metric} ===")
        print(f"Current: {forecast['current_value']}{forecast['unit']}")
        print(f"Trend: {forecast['trend']['direction']} (strength={forecast['trend']['strength']})")
        print(f"Risk score: {forecast['risk_score']}")
        print(f"Urgency: {forecast['urgency']}")
        wb = forecast["warning_breach"]
        cb = forecast["critical_breach"]
        if wb.get("will_breach"):
            print(f"Warning breach in: {wb.get('hours_to_breach', '?')}h")
        if cb.get("will_breach"):
            print(f"Critical breach in: {cb.get('hours_to_breach', '?')}h")

    if args.output:
        Path(args.output).write_text(json.dumps(forecast, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

    return 0


def cmd_scan(args: argparse.Namespace) -> int:
    """Scan failure patterns for predictive signals."""
    root: Path = args.root
    skills = [args.skill] if args.skill else [
        d.name for d in root.iterdir()
        if d.is_dir() and d.name.startswith("huaweicloud-") and d.name.endswith("-ops")
    ]

    print("=== Predictive Scan ===")
    total_signals = 0
    for skill in sorted(skills):
        fp_path = root / skill / "assets" / "failure_patterns.json"
        if not fp_path.exists():
            continue
        data = json.loads(fp_path.read_text(encoding="utf-8"))
        patterns = data.get("patterns", [])

        # Identify patterns with increasing occurrence (predictive signal)
        rising = []
        for p in patterns:
            stats = p.get("stats", {})
            count = stats.get("occurrence_count", 0)
            if count >= 3:  # Repeated failures = predictive signal
                rising.append(p)

        if rising:
            total_signals += len(rising)
            print(f"\n  {skill}: {len(rising)} predictive signal(s)")
            for p in rising[:3]:
                stats = p.get("stats", {})
                print(f"    {p['id']} [{p['category']}] count={stats.get('occurrence_count', 0)}")

    if total_signals == 0:
        print("  No predictive signals found (insufficient occurrence data)")
    else:
        print(f"\nTotal predictive signals: {total_signals}")
    return 0


def cmd_recommend(args: argparse.Namespace) -> int:
    """Generate recommendations from a forecast file."""
    forecast_path = Path(args.forecast)
    if not forecast_path.exists():
        print(f"ERROR: forecast file not found: {forecast_path}", file=sys.stderr)
        return 1

    forecast = json.loads(forecast_path.read_text(encoding="utf-8"))
    recs = get_recommendations(forecast)

    if args.json:
        print(json.dumps(recs, indent=2, ensure_ascii=False))
    else:
        print(f"=== Recommendations: {recs['metric']} (urgency={recs['urgency']}) ===")
        for r in recs["recommendations"]:
            print(f"\n  [{r['priority']}] {r['action']} (lead time: {r['lead_time_hours']}h)")
            for d in r["details"]:
                print(f"    - {d}")

    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Predictive Operations — trend analysis and proactive intervention"
    )
    subparsers = parser.add_subparsers(dest="cmd", required=True)

    # forecast
    f = subparsers.add_parser("forecast", help="Generate metric forecast")
    f.add_argument("--metric", required=True, help="Metric name (e.g., cpu_utilization)")
    f.add_argument("--data", required=True, help="Path to JSON with metric values")
    f.add_argument("--interval", type=float, default=1.0, help="Data interval in hours")
    f.add_argument("--json", action="store_true", help="Output as JSON")
    f.add_argument("--output", default=None, help="Save forecast to file")
    f.set_defaults(func=cmd_forecast)

    # scan
    s = subparsers.add_parser("scan", help="Scan failure patterns for predictive signals")
    s.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root")
    s.add_argument("--skill", default=None, help="Specific skill to scan")
    s.set_defaults(func=cmd_scan)

    # recommend
    r = subparsers.add_parser("recommend", help="Generate recommendations from forecast")
    r.add_argument("--forecast", required=True, help="Path to forecast JSON")
    r.add_argument("--json", action="store_true", help="Output as JSON")
    r.set_defaults(func=cmd_recommend)

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
