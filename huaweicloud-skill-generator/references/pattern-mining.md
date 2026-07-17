# Pattern Mining — L5 Self-Learning

> **Purpose**: Discovery of actionable patterns from operational data for predictive and autonomous operations.
> **Extends**: `self-learning-framework.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Pattern Types

### 1.1 Co-occurrence Patterns

**What**: Which alarms frequently occur together?

**Use Case**: Identify if alarm A reliably predicts alarm B

**Algorithm**: Association Rule Mining (Apriori)

```python
def mine_cooccurrence_patterns(alarm_sequences, min_support=0.05, min_confidence=0.7):
    """
    Find alarms that frequently occur together.
    """
    # Convert to transaction format
    transactions = [
        set(alarm_sequence.alarms)
        for alarm_sequence in alarm_sequences
    ]

    # Find frequent itemsets
    frequent_itemsets = apriori(transactions, min_support=min_support)

    # Generate association rules
    rules = []
    for itemset in frequent_itemsets:
        for alarm in itemset:
            consequent = {alarm}
            antecedent = itemset - consequent

            if antecedent:
                confidence = support(itemset) / support(antecedent)
                if confidence >= min_confidence:
                    rules.append(AssociationRule(
                        antecedent=antecedent,
                        consequent=consequent,
                        support=support(itemset),
                        confidence=confidence,
                        lift=calculate_lift(itemset, antecedent, consequent)
                    ))

    return sorted(rules, key=lambda r: r.lift, reverse=True)
```

### 1.2 Causal Patterns

**What**: Which alarm causes another?

**Use Case**: Root cause identification, cascade prediction

**Algorithm**: Granger Causality + Time-lagged Correlation

```python
def mine_causal_patterns(metric_timeseries, max_lag_minutes=30):
    """
    Find causal relationships between metrics/alarms.
    """
    causal_rules = []

    metrics = list(metric_timeseries.keys())

    for source in metrics:
        for target in metrics:
            if source == target:
                continue

            # Test Granger causality
            for lag in range(1, max_lag_minutes + 1):
                correlation = lagged_correlation(
                    metric_timeseries[source],
                    metric_timeseries[target],
                    lag=lag
                )

                if correlation > CAUSALITY_THRESHOLD:
                    causal_rules.append(CausalRule(
                        source=source,
                        target=target,
                        lag_minutes=lag,
                        correlation=correlation,
                        direction="source_causes_target"
                    ))

    return causal_rules
```

### 1.3 Time Patterns

**What**: When do certain alarms peak?

**Use Case**: Predictable load spikes, maintenance windows

**Algorithm**: Time Series Decomposition

```python
def mine_time_patterns(alarm_counts, period="daily"):
    """
    Find periodic patterns in alarm occurrence.
    """
    patterns = []

    # Decompose time series
    decomposition = seasonal_decompose(
        alarm_counts,
        model='additive',
        period=get_period_size(period)
    )

    # Extract seasonal component
    seasonal = decomposition.seasonal

    # Find peak times
    peak_indices = find_peaks(seasonal)[0]

    for peak_idx in peak_indices:
        patterns.append(TimePattern(
            alarm_type=alarm_counts.name,
            peak_time=index_to_time(peak_idx, period),
            intensity=seasonal[peak_idx],
            period=period
        ))

    return patterns
```

### 1.4 Resolution Patterns

**What**: Which actions resolve which fault patterns?

**Use Case**: Improve action selection in Decider

```python
def mine_resolution_patterns(feedback_records):
    """
    Find which actions successfully resolve which fault patterns.
    """
    resolution_patterns = {}

    for feedback in feedback_records:
        if not feedback.outcome.resolved:
            continue

        fault_signature = extract_fault_signature(feedback.diagnosis)
        action_id = feedback.action.action_id

        key = (fault_signature, action_id)
        if key not in resolution_patterns:
            resolution_patterns[key] = {"successes": 0, "total": 0}

        resolution_patterns[key]["total"] += 1
        resolution_patterns[key]["successes"] += 1

    # Calculate success rate per pattern
    patterns = []
    for (fault, action), stats in resolution_patterns.items():
        success_rate = stats["successes"] / stats["total"]
        if stats["total"] >= MIN_OCCURRENCES:
            patterns.append(ResolutionPattern(
                fault_signature=fault,
                action_id=action,
                success_rate=success_rate,
                occurrence_count=stats["total"]
            ))

    return sorted(patterns, key=lambda p: p.success_rate, reverse=True)
```

---

## 2. Pattern Quality Metrics

### 2.1 Pattern Validity

| Metric | Formula | Threshold |
|--------|---------|-----------|
| Support | occurrences / total | > 5% |
| Confidence | successes / attempts | > 70% |
| Lift | actual_conf / expected_conf | > 1.5 |
| Stability | 1 - (variance / mean) | > 0.7 |

### 2.2 Pattern Decay

Patterns can become stale. Track decay:

```python
def calculate_pattern_decay(pattern, current_time):
    """
    Calculate how much a pattern has decayed over time.
    """
    age_days = (current_time - pattern.last_observed).days

    if age_days < 7:
        return 1.0  # No decay

    # Exponential decay
    decay_rate = 0.05  # 5% per week
    decay = exp(-decay_rate * (age_days / 7))

    return max(0.3, decay)  # Minimum 30% validity
```

---

## 3. Pattern Storage

### 3.1 Pattern Schema

```yaml
learned_pattern:
  pattern_id: string
  pattern_type: string             # cooccurrence / causal / time / resolution
  created_at: timestamp
  last_updated: timestamp
  last_observed: timestamp
  occurrence_count: int
  confidence: float                # Overall confidence score

  # Pattern-specific data
  pattern_data:
    # For cooccurrence:
    alarms: list[string]
    support: float
    confidence: float
    lift: float

    # For causal:
    source: string
    target: string
    lag_minutes: int
    correlation: float

    # For time:
    alarm_type: string
    peak_times: list[time]
    period: string

    # For resolution:
    fault_signature: string
    action_id: string
    success_rate: float

  validity:
    current_validity: float        # Decay-adjusted
    is_active: bool                # validity > 0.5
    stale_reason: string | null
```

---

## 4. Pattern Update Cycle

| Phase | Frequency | Action |
|-------|-----------|--------|
| Real-time | Per incident | Update occurrence counts |
| Daily | Daily | Validate patterns, flag stale |
| Weekly | Weekly | Deep pattern mining |
| Monthly | Monthly | Full pattern refresh |

---

## 5. Integration with Decider

Patterns feed into the Decider for better action selection:

```python
def decider_uses_patterns(diagnosis, action_catalog):
    # Check for known resolution patterns
    fault_sig = extract_fault_signature(diagnosis)

    matching_patterns = [
        p for p in resolution_patterns
        if p.fault_signature == fault_sig
        and p.is_active
    ]

    if matching_patterns:
        # Boost confidence for actions with high success rate
        for pattern in matching_patterns:
            action = find_action(action_catalog, pattern.action_id)
            action.confidence_boost = pattern.success_rate * 0.2

    return action_catalog
```

---

## 6. LTS Log Query Integration

### 6.1 Log Pattern Mining

```python
def mine_log_patterns(log_group, log_streams, time_window):
    """
    Mine patterns from LTS logs using regex clustering.
    """
    logs = lts_client.query_logs(
        log_group=log_group,
        log_streams=log_streams,
        start_time=time_window.start,
        end_time=time_window.end
    )

    # Extract error patterns using regex
    error_patterns = cluster_errors(logs)

    # Find common error sequences
    sequences = find_common_sequences(logs)

    return LogPatterns(
        error_types=error_patterns,
        sequences=sequences
    )
```

---

## 7. Compliance Checklist

- [ ] All 4 pattern types implemented (cooccurrence, causal, time, resolution)
- [ ] Quality metrics (support, confidence, lift) calculated
- [ ] Pattern decay mechanism implemented
- [ ] Pattern storage schema defined
- [ ] Update cycle documented (real-time, daily, weekly, monthly)
- [ ] Decider integration for pattern-based action boosting
- [ ] LTS log pattern mining with regex clustering
