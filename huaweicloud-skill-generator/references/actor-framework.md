# Actor Framework — L5 Autonomous Operations

> **Purpose**: Safe action execution framework for L5 autonomous operations. Executes action plans from the Decider with verification and rollback.
> **Extends**: `action-catalog.md` + `decider-design.md` + `enhanced-self-healing-framework.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Overview

The Actor is the execution engine of L5 autonomous operations. It receives action plans from the Decider and executes them safely with verification and rollback support.

```
┌─────────────────┐     ┌──────────────┐     ┌─────────────────┐
│   Action Plan   │ ──▶ │    Actor     │ ──▶ │  Execution      │
│   (from Decider)│     │              │     │  Result         │
└─────────────────┘     │  Execute     │     └─────────────────┘
                        │  Verify      │              │
                        │  Rollback    │              ▼
                        └──────────────┘     ┌─────────────────┐
                                              │  Knowledge Base │
                                              │  (feedback)     │
                                              └─────────────────┘
```

### 1.1 Execution Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `dry-run` | Simulate execution, return expected result | Pre-execution validation |
| `execute` | Actual execution with verification | Normal autonomous operation |
| `verify-only` | Verify previous execution state | Post-incident verification |

---

## 2. Execution Contract

### 2.1 Input Contract

```yaml
actor_input:
  action_plan: ActionPlan           # From Decider
  execution_mode: string            # "dry-run" | "execute" | "verify-only"
  actor_context:
    request_id: string              # For tracing
    actor_id: string                # "actor-l5-{region}"
    timestamp: string
```

### 2.2 Output Contract

```yaml
execution_result:
  plan_id: string
  action_id: string
  status: string                    # "success" | "failed" | "rolled_back" | "dry_run_success"
  execution_time_ms: int
  output: dict                      # Command output / SDK response
  verification:
    verified: bool
    verification_method: string
    verification_output: dict
  rollback_performed: bool
  rollback_result: dict | null
  error: dict | null                # {code, message, recoverable}
  next_state: string                # "complete" | "pending_approval" | "escalated"
```

---

## 3. Execution Engine

### 3.1 Core Execution Loop

```python
def execute_action_plan(actor_input):
    plan = actor_input.action_plan
    context = actor_input.actor_context
    result = ExecutionResult(plan_id=plan.plan_id)

    # Select best action from plan
    action = select_best_action(plan.actions)

    # Pre-execution: dry-run if supported
    if action.dry_run_supported and actor_input.execution_mode == "execute":
        dry_run_result = execute_dry_run(action, context)
        if not dry_run_result.success:
            return result.fail(
                error=f"Dry-run failed: {dry_run_result.error}",
                next_state="escalated"
            )

    # Pre-execution: backup state
    backup = create_backup(action, context)

    # Execute
    if actor_input.execution_mode == "dry-run":
        result.status = "dry_run_success"
        result.next_state = "pending_approval"
        return result

    # Actual execution
    exec_result = execute_with_retry(
        action,
        context,
        max_retries=3,
        retry_interval=5
    )

    if exec_result.success:
        # Verify execution
        verified = verify_execution(action, exec_result, context)
        if verified:
            result.status = "success"
            result.next_state = "complete"
            result.verification = verified
        else:
            # Verification failed → rollback
            rollback_result = execute_rollback(backup, action, context)
            result.status = "rolled_back"
            result.rollback_performed = True
            result.rollback_result = rollback_result
            result.next_state = "escalated"
    else:
        # Execution failed
        if exec_result.recoverable:
            # Retry with backoff
            exec_result = execute_with_retry(...)
        else:
            result.status = "failed"
            result.error = exec_result.error
            result.next_state = "escalated"

    return result
```

### 3.2 CLI Execution Path

```python
def execute_cli_action(action, context):
    """
    Execute action via hcloud CLI.
    Returns: (exit_code, stdout, stderr)
    """
    cmd = build_cli_command(action, context)

    # Log command (without secrets)
    log.debug(f"Executing: {mask_secrets(cmd)}")

    # Execute with timeout
    proc = subprocess.run(
        cmd,
        shell=True,
        capture_output=True,
        timeout=action.timeout_seconds or 300
    )

    return (proc.returncode, proc.stdout, proc.stderr)
```

### 3.3 SDK Execution Path

```python
def execute_sdk_action(action, context):
    """
    Execute action via Go SDK fallback.
    Returns: (success, response, error)
    """
    # Build SDK request
    sdk_request = build_sdk_request(action, context)

    # Log request (without secrets)
    log.debug(f"SDK Request: {action.action_type}")

    # Execute
    try:
        response = sdk_client.call(
            service=action.skill,
            operation=action.action_type,
            params=sdk_request,
            timeout=action.timeout_seconds or 300
        )
        return (True, response, None)
    except SDKError as e:
        return (False, None, e)
```

---

## 4. Verification Logic

### 4.1 Verification Methods

| Method | Description | Use For |
|--------|-------------|---------|
| `health_check` | Call health check API | Instance/ service operations |
| `metric_check` | Query CES metrics within time window | Resource health |
| `state_check` | Query resource state directly | Configuration changes |
| `log_check` | Query LTS for error patterns | Process/ application issues |
| `compare_output` | Compare expected vs actual output | Data operations |

### 4.2 Verification Workflow

```python
def verify_execution(action, exec_result, context):
    verification = {
        "verified": False,
        "verification_method": None,
        "verification_output": {}
    }

    for method in action.verification_methods:
        if method == "health_check":
            result = verify_health_check(action, context)
        elif method == "metric_check":
            result = verify_metric_check(action, exec_result, context)
        elif method == "state_check":
            result = verify_state_check(action, context)
        elif method == "log_check":
            result = verify_log_check(action, context)

        if result.passed:
            verification["verified"] = True
            verification["verification_method"] = method
            verification["verification_output"] = result.output
            break

    return verification
```

### 4.3 Verification Time Windows

| Action Type | Verification Window | Polling Interval |
|-------------|--------------------|--------------------|
| Instance start/stop | 60s | 5s |
| Scale up/down | 120s | 10s |
| Configuration change | 30s | 5s |
| Network change | 60s | 5s |
| Storage expand | 120s | 10s |

---

## 5. Rollback Mechanism

### 5.1 Rollback Trigger Conditions

| Condition | Rollback Action |
|-----------|----------------|
| Verification failed after execution | Automatic rollback |
| Execution timeout | Automatic rollback |
| Execution error (recoverable) | Retry 3x, then rollback |
| Execution error (unrecoverable) | Immediate rollback + escalate |
| Human approval timeout | No execution (already prevented) |

### 5.2 Rollback Execution

```python
def execute_rollback(backup, action, context):
    """
    Rollback execution using pre-action backup.
    Returns: rollback result
    """
    if not backup:
        return RollbackResult(success=False, error="No backup available")

    rollback_steps = build_rollback_steps(action, backup)

    for step in rollback_steps:
        try:
            result = execute_rollback_step(step, context)
            if not result.success:
                # Rollback of rollback failed → escalate
                return RollbackResult(
                    success=False,
                    error=f"Rollback failed at step {step}: {result.error}",
                    escalated=True
                )
        except Exception as e:
            return RollbackResult(success=False, error=str(e), escalated=True)

    return RollbackResult(success=True, steps_executed=len(rollback_steps))
```

### 5.3 Rollback Strategy by Action Type

| Action | Rollback Strategy |
|--------|-------------------|
| Scale up | Remove scaled instance |
| Scale down | Re-add instance |
| Config change | Restore previous config |
| Security group rule | Remove added rule |
| Instance reboot | Power on (if stopped) |
| Delete | Recreate from backup (if available) |
| Irreversible | Cannot rollback — must escalate |

---

## 6. Idempotency Guarantees

### 6.1 Idempotency Keys

Every action execution includes an idempotency key to prevent duplicate execution:

```yaml
idempotency_key: "{action_id}-{resource_id}-{timestamp}"
# Example: "ECS-A02-ecs-12345-2026-07-18T10:00:00Z"
```

### 6.2 Idempotency Check Flow

```python
def check_idempotency(idempotency_key):
    """
    Check if action was already executed.
    Returns: (already_executed, previous_result)
    """
    cached = redis.get(f"idempotency:{idempotency_key}")
    if cached:
        return (True, json.loads(cached))
    return (False, None)

def set_idempotency(idempotency_key, result, ttl=3600):
    """
    Cache execution result for idempotency.
    """
    redis.setex(
        f"idempotency:{idempotency_key}",
        ttl,
        json.dumps(result)
    )
```

### 6.3 Idempotency by Action Type

| Action Category | Idempotency Strategy |
|-----------------|---------------------|
| Create resource | Check if exists → skip if exists |
| Delete resource | Check if exists → skip if not exists |
| Modify config | Store previous config → restore on rollback |
| Start/Stop | State check before action |
| Scale | Delta-based scaling |

---

## 7. Execution Logging

### 7.1 Execution Log Entry

```yaml
execution_log:
  plan_id: string
  action_id: string
  actor_id: string
  execution_mode: string
  command_executed: string          # Masked
  execution_start: timestamp
  execution_end: timestamp
  execution_time_ms: int
  status: string
  verification_method: string
  verification_result: bool
  rollback_performed: bool
  rollback_result: dict | null
  error: dict | null
  next_state: string
```

---

## 8. Error Handling

### 8.1 Error Classification

| Error Type | Recovery Strategy |
|------------|-------------------|
| `NETWORK_ERROR` | Retry 3x with exponential backoff |
| `TIMEOUT` | Retry 1x with extended timeout |
| `AUTH_ERROR` | Refresh credentials, retry |
| `QUOTA_EXCEEDED` | Escalate to human |
| `RESOURCE_NOT_FOUND` | Skip, mark as success (already gone) |
| `PERMISSION_DENIED` | Escalate to human |
| `UNKNOWN_ERROR` | Log, escalate to human |

### 8.2 Degradation Path

```
[Execution Failed]
    │
    ├── Retry successful → Continue
    │
    ├── Retry exhausted
    │   ├── Rollback successful → Mark as rolled_back
    │   └── Rollback failed → Escalate to human
    │
    └── Non-retryable error
        └── Immediate escalation
```

---

## 9. CLI vs SDK Selection

### 9.1 CLI-First Path

```python
def execute_with_cli_first(action, context):
    """
    Try CLI first, fall back to SDK on unsupported operation.
    """
    # Try CLI
    try:
        exit_code, stdout, stderr = execute_cli_action(action, context)
        if exit_code == 0:
            return ExecutionResult(success=True, output=stdout)
        else:
            # Check if CLI doesn't support this operation
            if is_unsupported_cli_operation(stderr):
                # Fall back to SDK
                return execute_sdk_action(action, context)
            else:
                return ExecutionResult(success=False, error=stderr)
    except Exception as e:
        # Fall back to SDK on any CLI error
        return execute_sdk_action(action, context)
```

### 9.2 SDK-Only Operations

Some operations only support SDK:

| Operation | Reason | SDK Package |
|-----------|--------|-------------|
| Batch operations | CLI doesn't support | `huaweicloud-sdk-go-v3` |
| Fine-grained IAM | SDK only | `iam` service |
| BSS billing queries | SDK only | `bss` service |

---

## 10. Compliance Checklist

- [ ] Supports hcloud CLI and Go SDK execution paths
- [ ] Dry-run mode implemented and verified
- [ ] Execution verification logic covers all action types
- [ ] Rollback mechanism documented with trigger conditions
- [ ] Idempotency guarantees implemented
- [ ] Error classification and recovery strategies defined
- [ ] Degradation path documented
- [ ] Execution logging format specified
- [ ] CLI-first with SDK fallback pattern implemented
