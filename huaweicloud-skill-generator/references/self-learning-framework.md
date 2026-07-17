# Self-Learning Framework — L5 Autonomous Operations

> **Purpose**: Framework for continuous learning from operation outcomes to improve future decisions.
> **Extends**: `decider-design.md` + `action-catalog.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Learning Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Self-Learning System                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐   ┌──────────────┐   ┌──────────────┐    │
│  │  Feedback   │──▶│   Learner    │──▶│  Pattern     │    │
│  │  Collector  │   │   Engine     │   │  Database    │    │
│  └─────────────┘   └──────────────┘   └──────────────┘    │
│         │                                      │            │
│         ▼                                      ▼            │
│  ┌─────────────┐                       ┌──────────────┐    │
│  │  Metrics    │                       │  Action      │    │
│  │  Store      │                       │  Catalog     │    │
│  └─────────────┘                       │  Updates     │    │
│                                        └──────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Learning Data Sources

### 2.1 Feedback Types

| Data Source | Description | Frequency | Usage |
|-------------|-------------|-----------|-------|
| Action outcomes | Success/failure of executed actions | Per action | Confidence weight tuning |
| Diagnosis accuracy | Did diagnosis match reality? | Per loop | Confidence scoring |
| Verification results | Did verification pass/fail? | Per action | Verification logic tuning |
| Human corrections | Human overrides to AI decisions | On-event | Pattern updates |
| Timeout events | Action timeouts | On-event | Threshold adjustment |
| Escalation reasons | Why human intervention needed | On-event | Decision logic improvement |

### 2.2 Feedback Schema

```yaml
feedback_record:
  feedback_id: string              # UUID
  loop_id: string
  timestamp: timestamp

  # What happened
  diagnosis:
    predicted_cause: string
    confidence: float
    actual_cause: string           # Filled after resolution

  action:
    action_id: string
    action_type: string
    risk_level: string
    executed: bool
    success: bool

  verification:
    method: string
    passed: bool
    attempts: int

  outcome:
    resolved: bool
    resolution_time_ms: int
    escalated: bool
    escalated_reason: string | null

  # Learning signals
  learning_signals:
    - signal_type: string          # "confidence_accuracy" | "action_success" | etc
      expected: any
      actual: any
      delta: float
```

---

## 3. Learning Algorithms

### 3.1 Confidence Weight Adjustment

```python
def update_confidence_weights(feedback_batch):
    """
    Adjust evidence weights based on historical accuracy.
    Uses exponential moving average.
    """
    current_weights = load_current_weights()

    for evidence_type in feedback_batch.evidence_types:
        # Calculate prediction error
        predictions = [f.predicted for f in feedback_batch if f.evidence_type == evidence_type]
        actuals = [f.actual for f in feedback_batch if f.evidence_type == evidence_type]

        errors = [abs(p - a) for p, a in zip(predictions, actuals)]
        avg_error = sum(errors) / len(errors)

        # Adjust weight: lower error → higher weight
        # Weight range: 0.05 - 0.50
        new_weight = current_weights[evidence_type] * (1 - LEARNING_RATE) + (1 - avg_error) * LEARNING_RATE
        new_weight = max(0.05, min(0.50, new_weight))

        current_weights[evidence_type] = new_weight

    save_weights(current_weights)
    return current_weights
```

### 3.2 Action Success Rate Tracking

```python
def update_action_success_rates(feedback_batch):
    """
    Track success rate per action type for action selection.
    """
    action_stats = load_action_stats()

    for feedback in feedback_batch:
        key = f"{feedback.skill}:{feedback.action_id}"

        if key not in action_stats:
            action_stats[key] = {"successes": 0, "failures": 0, "total": 0}

        action_stats[key]["total"] += 1
        if feedback.outcome.resolved:
            action_stats[key]["successes"] += 1
        else:
            action_stats[key]["failures"] += 1

        # Calculate success rate with confidence interval
        total = action_stats[key]["total"]
        successes = action_stats[key]["successes"]
        action_stats[key]["success_rate"] = successes / total
        action_stats[key]["confidence"] = 1.96 * sqrt(success_rate * (1 - success_rate) / total)

    save_action_stats(action_stats)
```

### 3.3 Threshold Optimization

```python
def optimize_thresholds(feedback_batch):
    """
    Adjust action thresholds based on outcomes.
    """
    for action_type in feedback_batch.unique_action_types:
        action_feedback = [f for f in feedback_batch if f.action_id == action_type]

        # Find optimal confidence threshold
        # Higher threshold = fewer false positives but more missed opportunities
        for threshold in [0.5, 0.6, 0.7, 0.8, 0.9]:
            true_positives = sum(1 for f in action_feedback if f.confidence >= threshold and f.outcome.resolved)
            false_positives = sum(1 for f in action_feedback if f.confidence >= threshold and not f.outcome.resolved)
            false_negatives = sum(1 for f in action_feedback if f.confidence < threshold and f.outcome.resolved)

            precision = true_positives / (true_positives + false_positives) if (true_positives + false_positives) > 0 else 0
            recall = true_positives / (true_positives + false_negatives) if (true_positives + false_negatives) > 0 else 0
            f1 = 2 * precision * recall / (precision + recall) if (precision + recall) > 0 else 0

            if f1 > best_f1:
                best_threshold = threshold
                best_f1 = f1

        update_threshold(action_type, best_threshold)
```

---

## 4. Learning Cycles

### 4.1 Real-Time Learning

Triggered after each loop completion:

```python
def real_time_learn(loop_feedback):
    """
    Immediate learning from single loop outcome.
    """
    # Update confidence weights incrementally
    update_confidence_weights([loop_feedback])

    # Update action success rates
    update_action_success_rates([loop_feedback])

    # If unexpected outcome, flag for analysis
    if loop_feedback.outcome.escalated:
        queue_for_deep_analysis(loop_feedback)
```

### 4.2 Batch Learning

Triggered weekly or when积累了足够的feedback:

```python
def batch_learn(feedback_batch):
    """
    Deep learning from accumulated feedback.
    """
    # Full confidence weight recalibration
    current_weights = load_current_weights()
    new_weights = compute_optimal_weights(feedback_batch)
    save_weights(new_weights)

    # Threshold optimization
    optimize_thresholds(feedback_batch)

    # Pattern mining for new correlations
    new_patterns = mine_patterns(feedback_batch)
    if new_patterns:
        update_pattern_database(new_patterns)

    # Generate learning report
    report = generate_learning_report(feedback_batch, new_weights, new_patterns)
    return report
```

### 4.3 Learning Triggers

| Trigger | Learning Type | Frequency |
|---------|---------------|-----------|
| Loop complete | Real-time | Per loop |
| 100 feedback items | Batch learning | Variable |
| Weekly cron | Batch learning | Weekly |
| Manual trigger | Batch learning | On-demand |

---

## 5. Knowledge Base Updates

### 5.1 Pattern Storage

```yaml
learned_pattern:
  pattern_id: string
  pattern_type: string             # "fault_sequence" | "action_outcome" | "metric_correlation"
  first_observed: timestamp
  last_observed: timestamp
  occurrence_count: int
  confidence: float                # How confident we are in this pattern
  description: string
  evidence:
    - type: string
      data: dict
  related_patterns: list[string]
  suggested_actions: list[string]
```

### 5.2 Knowledge Base Update Flow

```
Loop Completion
       │
       ├── Extract patterns
       │       │
       │       ├── Fault sequences (alarm → cause → action → resolution)
       │       ├── Action outcomes (which actions resolve which faults)
       │       └── Metric correlations (which metrics predict which issues)
       │
       ├── Compare with existing patterns
       │       │
       │       ├── Existing match → Increment occurrence count
       │       └── New pattern → Store with initial confidence
       │
       └── Validate pattern
               │
               └── If pattern accuracy < threshold after N occurrences → Deprecate
```

---

## 6. Learning Metrics

| Metric | Description | Target |
|--------|-------------|--------|
| diagnosis_accuracy | % of diagnoses matching actual cause | > 85% |
| action_success_rate | % of actions resolving issue | > 90% |
| false_positive_rate | % of auto-executions that didn't help | < 10% |
| escalation_rate | % of loops requiring human intervention | < 15% |
| learning_cycle_time | Time from feedback to pattern update | < 1 hour |
| pattern_accuracy | % of patterns that remain valid after 30 days | > 80% |

---

## 7. Compliance Checklist

- [ ] Feedback collection schema defined
- [ ] Real-time learning implemented
- [ ] Batch learning with weekly cadence
- [ ] Confidence weight adjustment algorithm
- [ ] Action success rate tracking
- [ ] Threshold optimization
- [ ] Pattern mining and storage
- [ ] Knowledge base update flow
- [ ] Learning metrics tracked
