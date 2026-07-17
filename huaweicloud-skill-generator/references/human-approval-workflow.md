# Human Approval Workflow — L5 Autonomous Operations

> **Purpose**: Workflow for routing high-risk actions to human approvers with timeout and escalation.
> **Extends**: `decider-design.md` §4 (Human Approval Gate)
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Approval Workflow Overview

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Decider    │────▶│   Approval   │────▶│  Human      │
│  (requires  │     │   Router     │     │  Approver   │
│   approval) │     │              │     │             │
└─────────────┘     └──────────────┘     └─────────────┘
                           │                    │
                           ▼                    ▼
                    ┌──────────────┐     ┌─────────────┐
                    │  Approval    │◀────│  Decision   │
                    │  Manager     │     │  Received   │
                    └──────────────┘     └─────────────┘
                           │
                    ┌──────┴──────┐
                    ▼             ▼
               ┌─────────┐   ┌─────────┐
               │ Approved│   │Rejected │
               └─────────┘   └─────────┘
```

---

## 2. Approval Request Format

```yaml
approval_request:
  request_id: string              # UUID
  plan_id: string                 # From Decider
  alarm_id: string
  resource_id: string
  resource_type: string           # ecs / rds / cce / etc
  action:                         # What we want to do
    action_id: string
    description: string
    risk_level: string            # High / Critical
  reason: string                  # Why this action is needed
  diagnosis:
    confidence: float
    root_cause: string
    evidence_summary: string
  impact_assessment:
    blast_radius: string
    affected_resources: int
    sla_impact: string
    estimated_downtime: string
  preconditions_status: string    # All met / Partial / Failed
  estimated_duration: string      # e.g. "~5 minutes"
  rollback_available: bool
  rollback_steps: string          # How to undo
  requested_at: timestamp
  decision_required_by: timestamp # Timeout
  approval_channel: string        # slack / pagerduty / email
  direct_link: string             # Link to alarm / resource
```

---

## 3. Approval Channels

### 3.1 Slack Integration

**Channel**: `#ops-approval`

**Message Format**:
```json
{
  "blocks": [
    {
      "type": "header",
      "text": {"type": "plain_text", "text": "⚠️ Action Approval Required"}
    },
    {
      "type": "section",
      "fields": [
        {"type": "mrkdwn", "text": "*Resource:*\n<{direct_link}|{resource_id}>"},
        {"type": "mrkdwn", "text": "*Action:*\n{action.description}"},
        {"type": "mrkdwn", "text": "*Risk Level:*\n:{risk_emoji}: {risk_level}"},
        {"type": "mrkdwn", "text": "*Requested By:*\nL5 Autonomous Agent"}
      ]
    },
    {
      "type": "section",
      "text": {"type": "mrkdwn", "text": "*Reason:*\n{reason}"}
    },
    {
      "type": "actions",
      "elements": [
        {"type": "button", "text": {"type": "plain_text", "text": "✅ Approve"}, "action_id": "approve_{request_id}", "style": "primary"},
        {"type": "button", "text": {"type": "plain_text", "text": "❌ Reject"}, "action_id": "reject_{request_id}"},
        {"type": "button", "text": {"type": "plain_text", "text": "🔍 Investigate"}, "action_id": "investigate_{request_id}"}
      ]
    },
    {
      "type": "context",
      "elements": [
        {"type": "mrkdwn", "text": "⏱️ Decision required within *{timeout}* | Request ID: `{request_id}`"}
      ]
    }
  ]
}
```

### 3.2 PagerDuty Integration (Critical Only)

**Escalation**: On-call engineer → Team lead → Manager

**Payload**:
```json
{
  "routing_key": "<pagerduty_key>",
  "event_action": "trigger",
  "payload": {
    "summary": "[L5-AUTO] Action Approval Required: {action.description} on {resource_id}",
    "severity": "critical",
    "source": "l5-autonomous-loop",
    "custom_details": {
      "request_id": "{request_id}",
      "resource_id": "{resource_id}",
      "action": "{action.description}",
      "risk_level": "{risk_level}",
      "reason": "{reason}",
      "decision_required_by": "{decision_required_by}",
      "direct_link": "{direct_link}"
    }
  },
  "links": [
    {"href": "{direct_link}", "text": "View Resource"}
  ]
}
```

### 3.3 Email Integration (Fallback)

**To**: `ops-team@company.com`
**Subject**: `[L5-AUTO] Approval Required - {action.description} on {resource_id}`

---

## 4. Approval Manager

### 4.1 Approval States

```yaml
approval_state:
  PENDING     # Awaiting decision
  APPROVED    # Human approved
  REJECTED    # Human rejected
  TIMEOUT     # No response within timeout
  EXPIRED     # Past decision_required_by
```

### 4.2 Approval Manager Logic

```python
class ApprovalManager:
    def __init__(self, redis_client, notification_client):
        self.redis = redis_client
        self.notification = notification_client
        self.timeouts = {
            "High": 300,      # 5 minutes
            "Critical": 120,  # 2 minutes
        }

    def submit_approval_request(self, action_plan):
        request = self.build_request(action_plan)
        self.store_request(request)
        self.send_notification(request)
        self.schedule_timeout_check(request)
        return request.request_id

    def handle_approval_response(self, request_id, decision, approver_comments=None):
        request = self.get_request(request_id)

        if decision == "APPROVED":
            request.state = "APPROVED"
            request.approver = approver_comments.get("approver")
            request.approved_at = current_timestamp()
            self.update_request(request)
            self.notify_actor(request, decision="APPROVED")
            self.log_approval_decision(request, decision, approver_comments)

        elif decision == "REJECTED":
            request.state = "REJECTED"
            request.rejection_reason = approver_comments.get("reason")
            request.rejected_at = current_timestamp()
            self.update_request(request)
            self.notify_actor(request, decision="REJECTED")
            self.escalate_rejection(request)
            self.log_approval_decision(request, decision, approver_comments)

    def handle_timeout(self, request_id):
        request = self.get_request(request_id)
        if request.state != "PENDING":
            return  # Already decided

        request.state = "TIMEOUT"
        self.update_request(request)
        self.notify_actor(request, decision="TIMEOUT")
        self.escalate_timeout(request)
        self.log_approval_decision(request, "TIMEOUT", None)
```

---

## 5. Timeout Handling

### 5.1 Timeout Rules

| Risk Level | Timeout | Escalation | Action on Timeout |
|------------|---------|------------|-------------------|
| High | 5 min | On-call engineer | Re-assign to next on-call |
| Critical | 2 min | On-call + Manager | Immediate escalation to manager |

### 5.2 Escalation Path

```
Timeout (High)
    │
    ├── Re-assign to next on-call in rotation
    │       │
    │       └── If next on-call also times out → Escalate to Manager
    │
    └── Log incident, continue loop but flag for review

Timeout (Critical)
    │
    └── Immediate escalation to Manager
            │
            └── If no response in 1 min → Escalate to Director
```

---

## 6. Approval Audit Log

```yaml
approval_audit_log:
  request_id: string
  plan_id: string
  action_id: string
  resource_id: string
  risk_level: string
  requested_at: timestamp
  channel: string                # slack / pagerduty / email
  approver: string | null        # Who approved/rejected
  decision: string               # APPROVED / REJECTED / TIMEOUT
  decision_at: timestamp | null
  comments: string | null
  escalation_count: int
  timeout_count: int
  loop_id: string                # Reference to parent loop
```

---

## 7. Integration Points

| System | Integration Method |
|--------|-------------------|
| Slack | Webhook API (incoming webhook) |
| PagerDuty | Events API v2 |
| Email | SMTP / SendGrid |
| Redis | Approval state storage |
| Actor | Callback / message queue |

---

## 8. Compliance Checklist

- [ ] Approval request format complete
- [ ] All 3 channels implemented (Slack, PagerDuty, Email)
- [ ] Timeout handling with escalation
- [ ] Approval audit logging
- [ ] States: PENDING → APPROVED / REJECTED / TIMEOUT
- [ ] Actor notified on decision
- [ ] Escalation path defined
