# Prediction Models — L5 Predictive Maintenance

> **Purpose**: Forecasting models for predictive maintenance and capacity planning.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Supported Models

| Model | Use Case | Data Requirements | Forecast Horizon |
|-------|----------|-------------------|------------------|
| Linear Regression | Stable growth trends | 30+ days history | 7-30 days |
| Seasonal Decomposition | Periodic patterns (daily/weekly) | 2+ periods | 7-30 days |
| Exponential Smoothing | Short-term prediction | 14+ days | 1-7 days |
| Anomaly Detection | Outlier prediction | 30+ days | Real-time |

---

## 2. Model Selection Guide

```python
def select_prediction_model(resource_type, data_characteristics):
    """
    Select appropriate prediction model based on data.
    """
    if data_characteristics.has_trend and not data_characteristics.has_seasonality:
        return "linear_regression"

    elif data_characteristics.has_seasonality:
        return "seasonal_decomposition"

    elif data_characteristics.is_short_term:
        return "exponential_smoothing"

    elif data_characteristics.is_anomaly_detection:
        return "anomaly_detection"

    else:
        return "linear_regression"  # Default
```

---

## 3. Linear Regression

### 3.1 Formula

```
y = β₀ + β₁ × t + ε

Where:
  y = predicted value
  t = time index
  β₀ = intercept
  β₁ = slope (growth rate)
  ε = error term
```

### 3.2 Implementation

```python
def linear_regression_forecast(values, forecast_days=7):
    """
    Simple linear regression for trend prediction.
    """
    n = len(values)
    t = np.arange(n)
    coeffs = np.polyfit(t, values, 1)  # Linear fit

    slope, intercept = coeffs

    # Forecast
    future_t = np.arange(n, n + forecast_days)
    forecast = intercept + slope * future_t

    # Confidence interval (95%)
    residuals = values - (intercept + slope * t)
    std_err = np.std(residuals)
    confidence_interval = 1.96 * std_err * np.sqrt(1 + 1/n)

    return ForecastResult(
        values=forecast,
        lower_bound=forecast - confidence_interval,
        upper_bound=forecast + confidence_interval,
        model="linear_regression",
        slope=slope,
        r_squared=calculate_r_squared(values, intercept + slope * t)
    )
```

---

## 4. Seasonal Decomposition

### 4.1 Formula

```
Y(t) = T(t) + S(t) + R(t)

Where:
  T(t) = Trend component
  S(t) = Seasonal component
  R(t) = Residual (noise)
```

### 4.2 Implementation

```python
def seasonal_decomposition_forecast(values, period=7, forecast_days=7):
    """
    STL decomposition for seasonal data.
    """
    from statsmodels.tsa.seasonal import STL

    # Decompose
    stl = STL(values, period=period)
    result = stl.fit()

    trend = result.trend
    seasonal = result.seasonal
    residual = result.resid

    # Forecast trend using linear regression
    trend_forecast = linear_regression_forecast(trend, forecast_days)

    # Repeat seasonal pattern
    seasonal_pattern = seasonal[-period:]
    seasonal_forecast = np.tile(seasonal_pattern, forecast_days // period + 1)[:forecast_days]

    # Combine
    forecast = trend_forecast.values + seasonal_forecast
    lower_bound = trend_forecast.lower_bound + seasonal_forecast
    upper_bound = trend_forecast.upper_bound + seasonal_forecast

    return ForecastResult(
        values=forecast,
        lower_bound=lower_bound,
        upper_bound=upper_bound,
        model="seasonal_decomposition",
        trend_slope=trend_forecast.slope,
        seasonal_period=period
    )
```

---

## 5. Anomaly Detection

### 5.1 3-Sigma Rule

```python
def anomaly_detection(values, threshold=3):
    """
    Detect anomalies using 3-sigma rule.
    """
    mean = np.mean(values)
    std = np.std(values)

    lower_bound = mean - threshold * std
    upper_bound = mean + threshold * std

    anomalies = []
    for i, v in enumerate(values):
        if v < lower_bound or v > upper_bound:
            anomalies.append({
                "index": i,
                "value": v,
                "expected_range": (lower_bound, upper_bound),
                "deviation": abs(v - mean) / std
            })

    return anomalies
```

### 5.2 Prediction-based Anomaly Detection

```python
def predictive_anomaly_detection(values, model, threshold=3):
    """
    Detect anomalies by comparing actual vs predicted.
    """
    # Use last N points to train
    train_size = len(values) - 7
    train_values = values[:train_size]
    test_values = values[train_size:]

    # Fit model
    forecast = model.fit(train_values)

    # Detect anomalies
    residuals = test_values - forecast.values
    std = np.std(residuals)

    anomalies = []
    for i, (actual, predicted) in enumerate(zip(test_values, forecast.values)):
        deviation = abs(actual - predicted) / std
        if deviation > threshold:
            anomalies.append({
                "index": train_size + i,
                "actual": actual,
                "predicted": predicted,
                "deviation": deviation
            })

    return anomalies
```

---

## 6. Model Accuracy Metrics

| Metric | Formula | Target |
|--------|---------|--------|
| MAE | mean(\|actual - predicted\|) | < 10% of mean |
| MAPE | mean(\|actual - predicted\| / \|actual\|) × 100 | < 15% |
| RMSE | sqrt(mean((actual - predicted)²)) | < 15% of mean |
| R² | 1 - SS_res / SS_tot | > 0.7 |

---

## 7. Compliance Checklist

- [ ] All 4 model types implemented
- [ ] Model selection logic documented
- [ ] Accuracy metrics calculated
- [ ] Confidence intervals provided
- [ ] Forecast horizon defined per model
