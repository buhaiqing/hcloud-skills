# Causal Discovery Algorithm — L5 Root Cause Self-Discovery

> **Purpose**: Algorithms for automatically discovering causal relationships in the knowledge graph.
> **Extends**: `knowledge-graph-schema.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Causal Discovery Pipeline

```
┌─────────────────────────────────────────────────────────────────┐
│              Causal Discovery Pipeline                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐   ┌──────────────┐   ┌──────────────┐        │
│  │  Data       │──▶│  Candidate   │──▶│  Causal      │        │
│  │  Collection │   │  Generation  │   │  Testing     │        │
│  └─────────────┘   └──────────────┘   └──────────────┘        │
│                                              │                  │
│                                              ▼                  │
│                                       ┌──────────────┐         │
│                                       │  Path        │         │
│                                       │  Scoring     │         │
│                                       └──────────────┘         │
│                                              │                  │
│                                              ▼                  │
│                                       ┌──────────────┐         │
│                                       │  Knowledge   │         │
│                                       │  Graph       │         │
│                                       │  Update      │         │
│                                       └──────────────┘         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Step 1: Data Collection

### 2.1 Alarm Data Collection

```python
def collect_alarm_data(time_window_hours=24):
    """
    Collect alarm data from CES for causal analysis.
    """
    end_time = current_timestamp()
    start_time = end_time - timedelta(hours=time_window_hours)

    alarms = ces_client.getAlarms(
        start_time=start_time,
        end_time=end_time,
        alarm_status=['alarm', 'ok']
    )

    return [
        AlarmRecord(
            alarm_id=a.alarm_id,
            resource_id=a.resource_id,
            metric_name=a.metric_name,
            severity=a.severity,
            triggered_at=a.triggered_at,
            value=a.value
        )
        for a in alarms
    ]
```

### 2.2 Change Data Collection

```python
def collect_change_data(time_window_hours=24):
    """
    Collect change data from CTS.
    """
    end_time = current_timestamp()
    start_time = end_time - timedelta(hours=time_window_hours)

    changes = cts_client.getTraces(
        start_time=start_time,
        end_time=end_time,
        trace_type=['api call', 'console action']
    )

    return [
        ChangeRecord(
            change_id=c.trace_id,
            change_type=classify_change(c),
            resource_id=c.resource_id,
            executed_at=c.end_time,
            description=c.resource_name
        )
        for c in changes
    ]
```

---

## 3. Step 2: Candidate Generation

### 3.1 Time-Window Based Candidates

```python
def generate_candidate_pairs(alarms, changes, time_window_minutes=30):
    """
    Generate candidate causal pairs based on time proximity.
    """
    candidates = []

    for change in changes:
        change_time = change.executed_at

        # Find alarms within time window after change
        for alarm in alarms:
            if alarm.triggered_at > change_time:
                time_delta = (alarm.triggered_at - change_time).total_seconds() / 60

                if time_delta <= time_window_minutes:
                    candidates.append(CandidatePair(
                        potential_cause=change,
                        potential_effect=alarm,
                        time_delta_minutes=time_delta
                    ))

        # Find alarms within time window before change
        for alarm in alarms:
            if alarm.triggered_at < change_time:
                time_delta = (change_time - alarm.triggered_at).total_seconds() / 60

                if time_delta <= time_window_minutes:
                    candidates.append(CandidatePair(
                        potential_cause=alarm,
                        potential_effect=change,
                        time_delta_minutes=time_delta
                    ))

    return candidates
```

### 3.2 Correlation-Based Candidates

```python
def generate_correlation_candidates(alarms, min_correlation=0.7):
    """
    Generate candidates based on metric correlation.
    """
    # Group alarms by resource
    alarms_by_resource = group_by(alarms, 'resource_id')

    candidates = []

    for resource_id, resource_alarms in alarms_by_resource.items():
        # Check pairwise correlations
        for i, alarm1 in enumerate(resource_alarms):
            for alarm2 in resource_alarms[i+1:]:
                correlation = calculate_correlation(
                    alarm1.values,
                    alarm2.values
                )

                if correlation >= min_correlation:
                    # Determine direction based on time
                    if alarm1.triggered_at < alarm2.triggered_at:
                        candidates.append(CandidatePair(
                            potential_cause=alarm1,
                            potential_effect=alarm2,
                            correlation=correlation
                        ))
                    else:
                        candidates.append(CandidatePair(
                            potential_cause=alarm2,
                            potential_effect=alarm1,
                            correlation=correlation
                        ))

    return candidates
```

---

## 4. Step 3: Causal Testing

### 4.1 Granger Causality Test

```python
def granger_causality_test(cause_series, effect_series, max_lag=5):
    """
    Test if cause Granger-causes effect.
    Uses F-test for significance.
    """
    from statsmodels.tsa.stattools import grangercausalitytests

    # Combine into DataFrame
    data = pd.DataFrame({
        'cause': cause_series,
        'effect': effect_series
    })

    # Run Granger causality test
    results = grangercausalitytests(data, maxlag=max_lag, verbose=False)

    # Find best lag
    best_p_value = 1.0
    best_lag = 0

    for lag in range(1, max_lag + 1):
        p_value = results[lag][0]['ssr_ftest'][1]  # p-value
        if p_value < best_p_value:
            best_p_value = p_value
            best_lag = lag

    return CausalTestResult(
        is_causal=best_p_value < 0.05,
        p_value=best_p_value,
        best_lag=best_lag,
        confidence=1 - best_p_value
    )
```

### 4.2 PC Algorithm (Constraint-Based)

```python
def pc_algorithm(data, alpha=0.05):
    """
    PC algorithm for causal discovery.
    Starts with fully connected graph and removes edges based on conditional independence.
    """
    # Step 1: Start with complete undirected graph
    G = complete_graph(data.columns)

    # Step 2: Remove edges based on conditional independence
    for (X, Y) in G.edges():
        # Get neighbors of X and Y
        neighbors_X = set(G.neighbors(X)) - {Y}
        neighbors_Y = set(G.neighbors(Y)) - {X}

        # Test conditional independence for all subsets
        for subset_size in range(max(len(neighbors_X), len(neighbors_Y)) + 1):
            for subset in combinations(neighbors_X | neighbors_Y, subset_size):
                if conditional_independent(data, X, Y, list(subset), alpha):
                    G.remove_edge(X, Y)
                    break

    # Step 3: Orient edges (v-structures)
    G = orient_edges(G, data)

    return G
```

---

## 5. Step 4: Path Scoring

### 5.1 Causal Strength Score

```python
def score_causal_path(cause, effect, historical_occurrences):
    """
    Score causal relationship based on historical evidence.
    """
    occurrences = [o for o in historical_occurrences
                   if o.cause == cause and o.effect == effect]

    if not occurrences:
        return 0.0

    # Calculate metrics
    support = len(occurrences) / len(historical_occurrences)

    # Confidence: what fraction of cause events led to effect
    cause_count = sum(1 for o in historical_occurrences if o.cause == cause)
    confidence = len(occurrences) / cause_count if cause_count > 0 else 0

    # Average time delta consistency
    time_deltas = [o.time_delta_minutes for o in occurrences]
    time_consistency = 1 / (1 + np.std(time_deltas)) if len(time_deltas) > 1 else 1.0

    # Combined score
    score = (support * 0.3 + confidence * 0.5 + time_consistency * 0.2)

    return score
```

### 5.2 Top-K Root Cause Selection

```python
def select_top_k_root_causes(symptoms, k=3):
    """
    Select top-k most likely root causes for given symptoms.
    """
    candidates = []

    for symptom in symptoms:
        # Find all paths ending at symptom
        paths = find_all_paths(knowledge_graph, symptom, max_length=5)

        for path in paths:
            root_cause = path[0]
            score = calculate_path_score(path)
            candidates.append({
                'root_cause': root_cause,
                'symptom': symptom,
                'path': path,
                'score': score
            })

    # Sort by score and return top-k
    sorted_candidates = sorted(candidates, key=lambda x: x['score'], reverse=True)

    return sorted_candidates[:k]
```

---

## 6. Step 5: Knowledge Graph Update

```python
def update_knowledge_graph(discovered_causes):
    """
    Update knowledge graph with newly discovered causal relationships.
    """
    for cause, effect, score in discovered_causes:
        # Check if edge already exists
        existing_edge = find_edge(cause, effect, 'causes')

        if existing_edge:
            # Update confidence
            existing_edge.confidence = score
            existing_edge.evidence_count += 1
            existing_edge.last_observed = current_timestamp()
        else:
            # Create new edge
            create_edge(
                source=cause,
                target=effect,
                edge_type='causes',
                properties={
                    'confidence': score,
                    'evidence_count': 1,
                    'first_observed': current_timestamp(),
                    'last_observed': current_timestamp()
                }
            )
```

---

## 7. Compliance Checklist

- [ ] Data collection from CES and CTS
- [ ] Time-window based candidate generation
- [ ] Correlation-based candidate generation
- [ ] Granger causality test implementation
- [ ] PC algorithm for constraint-based discovery
- [ ] Causal strength scoring
- [ ] Top-K root cause selection
- [ ] Knowledge graph update logic
