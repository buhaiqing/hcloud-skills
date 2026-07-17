# Autonomous Loop — L5 Closed-Loop Operations

> **Purpose**: Complete closed-loop implementation for L5 autonomous operations — Detect → Diagnose → Decide → Act → Verify → Learn.
> **Extends**: `decider-design.md` + `actor-framework.md` + `action-catalog.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Loop Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     L5 Autonomous Loop                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────┐    ┌───────────┐    ┌─────────┐    ┌─────────┐   │
│  │ Detect  │───▶│ Diagnose  │───▶│ Decide  │───▶│   Act   │   │
│  └─────────┘    └───────────┘    └─────────┘    └─────────┘   │
│       │                                     │                  │
│       │              ┌─────────┐            │                  │
│       │              │ Verify  │◀───────────┘                  │
│       │              └─────────┘                               │
│       │                  │                                     │
│       │              ┌─────────┐                               │
│       └─────────────▶│  Learn  │                               │
│                      └─────────┘                               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 1.1 Component Responsibilities

| Component | Responsibility | Input | Output |
|-----------|---------------|-------|--------|
| **Detect** | Alarm ingestion + anomaly detection | CES alarm | Anomaly event |
| **Diagnose** | Root cause analysis + confidence scoring | Anomaly event | Diagnosis result |
| **Decide** | Risk assessment + action selection | Diagnosis result | Action plan |
| **Act** | Safe execution with verification | Action plan | Execution result |
| **Verify** | Post-execution validation | Execution result | Verification result |
| **Learn** | Feedback collection + pattern update | Execution + Verification | Learned patterns |

---

## 2. State Machine

### 2.1 Loop States

```yaml
loop_state:
  DETECTING      # Collecting alarm data
  DIAGNOSING     # Running diagnosis analysis
  DECIDING       # Selecting action plan
  ACTING         # Executing action
  VERIFYING      # Validating execution result
  LEARNING       # Updating knowledge base
  COMPLETE       # Loop finished successfully
  ESCALATED      # Human intervention required
  FAILED         # Loop failed
```

### 2.2 State Transitions

```
DETECTING ──▶ DIAGNOSING ──▶ DECIDING ──▶ ACTING ──▶ VERIFYING ──▶ LEARNING ──▶ COMPLETE
    │             │              │            │            │             │
    │             │              │            │            │             │
    ▼             ▼              ▼            ▼            ▼             │
 ESCALATED    ESCALATED      ESCALATED    ESCALATED    ESCALATED         │
                                                                   │
                                                                   ▼
                                                              COMPLETE
```

### 2.3 State Transition Rules

| Current State | Condition | Next State |
|---------------|-----------|------------|
| DETECTING | Alarm data collected | DIAGNOSING |
| DETECTING | No data / timeout | ESCALATED |
| DIAGNOSING | Diagnosis complete | DECIDING |
| DIAGNOSING | Confidence < 0.2 | ESCALATED |
| DECIDING | Action plan ready | ACTING |
| DECIDING | No matching action | ESCALATED |
| ACTING | Execution complete | VERIFYING |
| ACTING | Execution failed + rollback OK | LEARNING |
| ACTING | Execution failed + rollback fail | ESCALATED |
| VERIFYING | Verification passed | LEARNING |
| VERIFYING | Verification failed | ESCALATED |
| LEARNING | Patterns updated | COMPLETE |
| LEARNING | Update failed | COMPLETE (with warning) |

---

## 3. Loop Execution Flow

### 3.1 Main Loop Pseudocode

```python
def run_autonomous_loop(alarm_event):
    loop_context = LoopContext(
        loop_id=generate_uuid(),
        alarm_id=alarm_event.alarm_id,
        resource_id=alarm_event.resource_id,
        skill=alarm_event.skill,
        start_time=current_timestamp()
    )

    state = "DETECTING"
    max_iterations = 10
    iteration = 0

    while state not in ("COMPLETE", "ESCALATED", "FAILED") and iteration < max_iterations:
        iteration += 1

        if state == "DETECTING":
            state = detect(loop_context, alarm_event)

        elif state == "DIAGNOSING":
            state = diagnose(loop_context)

        elif state == "DECIDING":
            state = decide(loop_context)

        elif state == "ACTING":
            state = act(loop_context)

        elif state == "VERIFYING":
            state = verify(loop_context)

        elif state == "LEARNING":
            state = learn(loop_context)

    # Final state handling
    if state == "COMPLETE":
        log.info(f"Loop {loop_context.loop_id} completed successfully")
    elif state == "ESCALATED":
        escalate_to_human(loop_context)
    else:
        log.error(f"Loop {loop_context.loop_id} failed after {iteration} iterations")

    return LoopResult(
        loop_id=loop_context.loop_id,
        final_state=state,
        iterations=iteration,
        execution_time_ms=elapsed_ms(loop_context.start_time)
    )
```

### 3.2 Component Implementations

#### Detect
```python
def detect(context, alarm_event):
    # Collect alarm details
    alarm_data = fetch_alarm_details(alarm_event.alarm_id)

    # Enrich with resource info
    resource_data = fetch_resource_details(alarm_event.resource_id)

    # Check for correlated alarms (alarm storm detection)
    correlated = detect_correlated_alarms(alarm_event, window_minutes=30)

    context.alarm_data = alarm_data
    context.resource_data = resource_data
    context.correlated_alarms = correlated

    if alarm_data.is_suppressed:
        return "COMPLETE"  # No action needed

    return "DIAGNOSING"
```

#### Diagnose
```python
def diagnose(context):
    # Run diagnosis with confidence scoring
    diagnosis = diagnose_with_confidence(context.alarm_data)

    # Check if confidence meets minimum threshold
    min_confidence = 0.2
    if diagnosis.confidence < min_confidence:
        log.warning(f"Diagnosis confidence {diagnosis.confidence} below threshold")
        return "ESCALATED"

    context.diagnosis = diagnosis
    return "DECIDING"
```

#### Decide
```python
def decide(context):
    # Get action catalog
    catalog = load_action_catalog()

    # Run decider
    action_plan = decide(context.diagnosis, catalog)

    if action_plan.requires_approval:
        # Send for human approval
        send_approval_request(action_plan)
        return "ESCALATED"  # Wait for approval (async)

    context.action_plan = action_plan
    return "ACTING"
```

#### Act
```python
def act(context):
    # Execute action plan via Actor
    result = execute_action_plan(context.action_plan)

    context.execution_result = result

    if result.status == "failed":
        if result.rollback_performed:
            return "LEARNING"
        return "ESCALATED"

    return "VERIFYING"
```

#### Verify
```python
def verify(context):
    result = context.execution_result

    if result.status == "success":
        # Check if verification passes
        if result.verification.verified:
            return "LEARNING"
        else:
            # Verification failed - rollback
            execute_rollback(context.action_plan)
            return "ESCALATED"

    return "LEARNING"  # Already handled in act()
```

#### Learn
```python
def learn(context):
    # Collect feedback
    feedback = Feedback(
        loop_id=context.loop_id,
        diagnosis=context.diagnosis,
        action_plan=context.action_plan,
        execution_result=context.execution_result,
        outcome=determine_outcome(context.execution_result),
        timestamp=current_timestamp()
    )

    # Update knowledge base
    update_knowledge_base(feedback)

    # Update confidence weights if needed
    update_confidence_weights(feedback)

    # Update pattern database
    update_patterns(feedback)

    return "COMPLETE"
```

---

## 4. Concurrency Control

### 4.1 Resource Locking

To prevent conflicting actions on the same resource:

```python
def acquire_resource_lock(resource_id, action, timeout_seconds=300):
    lock_key = f"resource_lock:{resource_id}"
    lock_value = f"{action}-{current_timestamp()}"

    acquired = redis.set(lock_key, lock_value, nx=True, ex=timeout_seconds)
    if not acquired:
        existing = redis.get(lock_key)
        if existing and existing != lock_value:
            return False  # Resource already locked
    return True

def release_resource_lock(resource_id, lock_value):
    lock_key = f"resource_lock:{resource_id}"
    current = redis.get(lock_key)
    if current == lock_value:
        redis.delete(lock_key)
```

### 4.2 Loop Mutex

Only one loop instance per alarm:

```python
def acquire_alarm_lock(alarm_id):
    lock_key = f"alarm_lock:{alarm_id}"
    return redis.set(lock_key, "1", nx=True, ex=3600)

def release_alarm_lock(alarm_id):
    redis.delete(f"alarm_lock:{alarm_id}")
```

---

## 5. Error Handling & Recovery

### 5.1 Loop-Level Error Handling

| Error | Handling |
|-------|----------|
| Detect timeout | Escalate immediately |
| Diagnose failure | Escalate with partial diagnosis |
| Decide failure | Fall back to manual |
| Act failure | Retry 3x, then rollback + escalate |
| Verify failure | Automatic rollback |
| Learn failure | Log error, complete loop with warning |

### 5.2 Recovery from Incomplete Loop

If loop crashes mid-execution:

```python
def recover_incomplete_loops():
    """
    Check for loops in non-terminal state and resume or clean up.
    """
    active_loops = redis.keys("loop_context:*")

    for loop_id in active_loops:
        context = redis.get(loop_id)
        if context.state in ("COMPLETE", "ESCALATED", "FAILED"):
            continue  # Already in terminal state

        # Check if loop has been stuck too long
        if context.elapsed_time > MAX_LOOP_DURATION:
            # Attempt recovery or escalate
            if context.state == "ACTING":
                # Check if action actually executed
                if not action_completed(context.action_id):
                    release_resource_lock(context.resource_id)
            mark_loop_recovered(loop_id)
```

---

## 6. Monitoring & Alerting

### 6.1 Loop Metrics

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| loop_duration_ms | Total loop execution time | > 600s |
| loop_success_rate | % of loops completing successfully | < 95% |
| loop_escalation_rate | % of loops requiring human intervention | > 20% |
| loop_failure_rate | % of loops failing completely | > 5% |
| action_success_rate | % of actions executing successfully | < 90% |
| verification_pass_rate | % of verifications passing | < 85% |

### 6.2 Alert Rules

```yaml
alerts:
  - name: loop_duration_high
    condition: loop_duration_ms > 600000  # 10 min
    severity: warning
    channel: slack

  - name: loop_stuck
    condition: loop_state not in terminal for > 30 min
    severity: critical
    channel: pagerduty

  - name: escalation_rate_high
    condition: loop_escalation_rate > 0.2
    severity: warning
    channel: slack

  - name: loop_failure
    condition: loop_failure_rate > 0.05
    severity: critical
    channel: pagerduty
```

---

## 7. Integration Points

### 7.1 External System Integration

| System | Integration | Method |
|--------|-------------|--------|
| CES | Alarm ingestion | CES API / webhook |
| LTS | Log analysis | LTS query API |
| CTS | Change correlation | CTS API |
| Action Catalog | Action lookup | Local file |
| Knowledge Base | Pattern storage | PostgreSQL / Neo4j |
| Approval System | Human approval | Webhook / Slack |
| Metrics | Monitoring | CES custom metrics |

### 7.2 Data Flow

```
CES Alarm ──▶ Webhook ──▶ Loop Trigger ──▶ Detect ──▶ Diagnose ──▶ Decide ──▶ Act ──▶ Verify ──▶ Learn
                    │                                                                    │
                    └────────────────────────────────────────────────────────────────────┘
                                                      Feedback Loop
```

---

## 8. Compliance Checklist

- [ ] Complete state machine with all transitions
- [ ] All 6 components implemented (Detect, Diagnose, Decide, Act, Verify, Learn)
- [ ] Concurrency control (resource locking)
- [ ] Error handling and recovery
- [ ] Loop metrics and alerting
- [ ] External system integration defined
- [ ] Max iteration limit enforced
