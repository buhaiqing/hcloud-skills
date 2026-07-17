# Causal Chain Update — L5 Root Cause Self-Discovery

> **Purpose**: Automatic update of causal chains after incident resolution.
> **Extends**: `knowledge-graph.md` + `causal-discovery-algorithm.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Update Trigger Conditions

### 1.1 Automatic Triggers

| Trigger | Condition | Action |
|---------|-----------|--------|
| Incident Resolved | Alarm status → resolved | Update causal chain |
| Human RCA | RCA submitted | Integrate human findings |
| Pattern Matched | Known pattern detected | Update pattern confidence |
| New Correlation | Unseen correlation observed | Create new candidate edge |

### 1.2 Update Frequency

| Update Type | Frequency | Method |
|-------------|-----------|--------|
| On incident resolve | Per incident | Async job |
| Daily batch | Daily | Cron job |
| Weekly deep sync | Weekly | Scheduled job |

---

## 2. Incident Resolution Update Flow

```
Incident Resolved
       │
       ├── Extract incident data
       │       │
       │       ├── Alarm timeline
       │       ├── Changes during incident window
       │       ├── Actions taken
       │       └── Resolution details
       │
       ├── Validate causal chain
       │       │
       │       ├── Check if predicted root cause matches
       │       ├── Verify time delta consistency
       │       └── Update confidence scores
       │
       ├── Update knowledge graph
       │       │
       │       ├── Update edge confidences
       │       ├── Add new observations
       │       └── Flag stale patterns
       │
       └── Propagate to similar incidents
               │
               └── Update confidence of similar causal chains
```

---

## 3. Causal Chain Validation

### 3.1 Root Cause Validation

```python
def validate_root_cause(incident_id: str, predicted_root_cause: str) -> ValidationResult:
    """
    Validate if predicted root cause matches actual resolution.
    """
    incident = get_incident(incident_id)

    # Check if human RCA was provided
    if incident.human_rca:
        # Compare with human findings
        match_score = compare_causal_chains(
            predicted_root_cause,
            incident.human_rca.root_cause
        )

        return ValidationResult(
            validated=True,
            match_score=match_score,
            source='human_rca',
            feedback=incident.human_rca
        )

    # Infer from resolution actions
    resolution_actions = incident.resolution_actions

    # Find which action actually resolved the issue
    for action in resolution_actions:
        if action.resolved:
            # Check if action addresses predicted root cause
            addresses = action.addresses(predicted_root_cause)

            if addresses:
                return ValidationResult(
                    validated=True,
                    match_score=0.9,
                    source='action_outcome',
                    feedback=action
                )

    # No clear validation
    return ValidationResult(
        validated=False,
        match_score=0.0,
        source='unknown',
        feedback=None
    )
```

### 3.2 Time Delta Validation

```python
def validate_time_deltas(incident_id: str) -> list:
    """
    Validate if cause-effect time deltas are consistent.
    """
    incident = get_incident(incident_id)
    causal_chain = incident.causal_chain

    validations = []

    for i in range(len(causal_chain) - 1):
        cause = causal_chain[i]
        effect = causal_chain[i + 1]

        actual_delta = (effect.time - cause.time).total_seconds() / 60

        # Get expected delta from knowledge graph
        expected_delta = get_edge_property(
            cause.node_id,
            effect.node_id,
            'time_delta_ms'
        ) / 60000  # Convert to minutes

        if expected_delta:
            deviation = abs(actual_delta - expected_delta) / expected_delta

            validations.append({
                'cause': cause.node_id,
                'effect': effect.node_id,
                'actual_delta_minutes': actual_delta,
                'expected_delta_minutes': expected_delta,
                'deviation_percent': deviation * 100,
                'consistent': deviation < 0.3  # 30% tolerance
            })

    return validations
```

---

## 4. Knowledge Graph Update Operations

### 4.1 Edge Confidence Update

```python
def update_edge_confidence(cause_id: str, effect_id: str, validation_result: ValidationResult):
    """
    Update edge confidence based on validation result.
    """
    edge = get_edge(cause_id, effect_id, 'causes')

    if not edge:
        return  # Edge doesn't exist

    # Bayesian update
    prior = edge.confidence
    likelihood = validation_result.match_score

    # Posterior = (prior * likelihood) / normalization
    # Simplified: weighted average with evidence count
    n = edge.evidence_count
    posterior = (prior * n + likelihood) / (n + 1)

    # Update edge
    edge.confidence = posterior
    edge.evidence_count += 1
    edge.last_validated = current_timestamp()
    edge.validation_source = validation_result.source

    save_edge(edge)
```

### 4.2 New Observation Addition

```python
def add_causal_observation(cause_id: str, effect_id: str, observation: dict):
    """
    Add a new observation to an existing causal edge.
    """
    edge = get_or_create_edge(cause_id, effect_id, 'causes')

    # Update statistics
    edge.observation_count += 1

    # Update time delta statistics
    if 'time_delta_ms' in observation:
        edge.time_delta_avg = (
            (edge.time_delta_avg * (edge.observation_count - 1) + observation['time_delta_ms'])
            / edge.observation_count
        )

    # Update confidence
    edge.confidence = min(0.99, edge.confidence * 1.05)  # Slight increase

    edge.last_observed = current_timestamp()

    save_edge(edge)
```

### 4.3 Pattern Propagation

```python
def propagate_to_similar_incidents(incident_id: str, root_cause_id: str):
    """
    Update confidence of similar causal chains.
    """
    incident = get_incident(incident_id)

    # Find similar incidents (same symptom pattern)
    similar_incidents = find_similar_incidents(
        symptom_pattern=incident.symptom_pattern,
        exclude=[incident_id]
    )

    for similar in similar_incidents:
        # Find causal chain edges
        similar_chain = get_causal_chain(similar.incident_id)

        for edge in similar_chain:
            # Boost confidence if same root cause
            if edge.cause.node_id == root_cause_id:
                edge.confidence = min(0.99, edge.confidence * 1.1)
                edge.propagated_from = incident_id
                save_edge(edge)
```

---

## 5. Stale Pattern Detection

### 5.1 Pattern Decay

```python
def detect_stale_patterns(threshold_days=30):
    """
    Detect patterns that haven't been observed recently.
    """
    cutoff_date = current_timestamp() - timedelta(days=threshold_days)

    stale_edges = []

    for edge in get_all_causal_edges():
        if edge.last_observed < cutoff_date:
            # Calculate decay
            days_since_observed = (current_timestamp() - edge.last_observed).days
            decay_factor = exp(-0.05 * days_since_observed)  # 5% decay per week

            stale_edges.append({
                'edge': edge,
                'decay_factor': decay_factor,
                'days_since_observed': days_since_observed,
                'current_confidence': edge.confidence,
                'decayed_confidence': edge.confidence * decay_factor
            })

    return stale_edges
```

### 5.2 Deprecation Workflow

```python
def deprecate_stale_pattern(edge_id: str, reason: str):
    """
    Mark a pattern as deprecated.
    """
    edge = get_edge_by_id(edge_id)

    edge.status = 'deprecated'
    edge.deprecated_at = current_timestamp()
    edge.deprecation_reason = reason

    # Archive instead of delete
    archive_edge(edge)

    # Log for review
    log.warning(f"Deprecated causal edge {edge_id}: {reason}")
```

---

## 6. Compliance Checklist

- [ ] Update triggers defined (incident resolve, human RCA, etc.)
- [ ] Root cause validation logic
- [ ] Time delta validation
- [ ] Edge confidence update (Bayesian)
- [ ] New observation addition
- [ ] Pattern propagation to similar incidents
- [ ] Stale pattern detection
- [ ] Deprecation workflow
