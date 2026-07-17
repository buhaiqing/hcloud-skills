# Prediction Service — L5 Predictive Maintenance

> **Purpose**: Service architecture for running prediction models at scale.
> **Extends**: `prediction-models.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Service Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   Prediction Service                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐   ┌──────────────┐   ┌──────────────┐    │
│  │   REST API  │──▶│  Scheduler   │──▶│  Model       │    │
│  │   (Flask)   │   │  (APScheduler)│  │  Executor    │    │
│  └─────────────┘   └──────────────┘   └──────────────┘    │
│         │                                      │            │
│         ▼                                      ▼            │
│  ┌─────────────┐                       ┌──────────────┐    │
│  │  In-Memory  │                       │  PostgreSQL  │    │
│  │  Cache      │                       │  (results)   │    │
│  └─────────────┘                       └──────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. API Endpoints

### 2.1 Trigger Prediction

```
POST /api/v1/predictions
{
  "resource_id": "ecs-xxxxx",
  "resource_type": "ecs",
  "metric": "cpu_util",
  "model": "linear_regression",  # or "seasonal", "ets", "anomaly"
  "forecast_days": 7
}
```

### 2.2 Get Prediction Result

```
GET /api/v1/predictions/{prediction_id}
```

Response:
```json
{
  "prediction_id": "pred-xxxxx",
  "status": "completed",
  "resource_id": "ecs-xxxxx",
  "forecast": {
    "values": [75.2, 76.1, 77.0, ...],
    "lower_bound": [70.1, 71.0, 71.9, ...],
    "upper_bound": [80.3, 81.2, 82.1, ...],
    "forecast_end": "2026-07-25T00:00:00Z"
  },
  "model_info": {
    "model": "linear_regression",
    "r_squared": 0.85,
    "slope": 0.9
  },
  "created_at": "2026-07-18T10:00:00Z"
}
```

### 2.3 List Active Predictions

```
GET /api/v1/predictions?resource_id=ecs-xxxxx&status=running
```

---

## 3. Prediction Task Scheduling

### 3.1 Scheduled Predictions

```python
def schedule_prediction_tasks():
    """
    Schedule daily prediction tasks for all monitored resources.
    """
    scheduler = APScheduler()

    # Daily prediction for ECS metrics
    scheduler.add_job(
        func=run_ecs_predictions,
        trigger='cron',
        hour=2,  # Run at 2 AM
        minute=0
    )

    # Daily prediction for RDS metrics
    scheduler.add_job(
        func=run_rds_predictions,
        trigger='cron',
        hour=2,
        minute=30
    )

    # Weekly seasonal decomposition
    scheduler.add_job(
        func=run_seasonal_predictions,
        trigger='cron',
        day_of_week='sun',
        hour=3
    )
```

### 3.2 On-Demand Predictions

```python
@app.route('/api/v1/predictions', methods=['POST'])
def create_prediction():
    """
    Create on-demand prediction task.
    """
    data = request.json

    prediction_id = generate_uuid()

    # Queue prediction task
    prediction_queue.put({
        "prediction_id": prediction_id,
        "resource_id": data["resource_id"],
        "metric": data["metric"],
        "model": data.get("model", "linear_regression"),
        "forecast_days": data.get("forecast_days", 7)
    })

    return {"prediction_id": prediction_id}, 202
```

---

## 4. Result Storage

### 4.1 PostgreSQL Schema

```sql
CREATE TABLE prediction_results (
    prediction_id UUID PRIMARY KEY,
    resource_id VARCHAR(255) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    metric VARCHAR(100) NOT NULL,
    model VARCHAR(50) NOT NULL,
    forecast_days INTEGER NOT NULL,
    forecast_values JSONB NOT NULL,
    lower_bound JSONB NOT NULL,
    upper_bound JSONB NOT NULL,
    model_info JSONB,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX idx_prediction_resource ON prediction_results(resource_id);
CREATE INDEX idx_prediction_status ON prediction_results(status);
CREATE INDEX idx_prediction_created ON prediction_results(created_at);
```

---

## 5. Compliance Checklist

- [ ] REST API with Flask
- [ ] APScheduler for task scheduling
- [ ] PostgreSQL for result storage
- [ ] On-demand and scheduled predictions
- [ ] Prediction status tracking
