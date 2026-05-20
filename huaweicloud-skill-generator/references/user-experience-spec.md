# User Experience Specification — Huawei Cloud Skill Generator

> **Purpose:** Mandatory UX requirements for all generated skills. Ensures agent-generated skills are human-readable, actionable, and provide clear feedback.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20
> **Status:** MANDATORY

---

## 1. Onboarding

### 1.1 Quick Start
Every skill MUST have a Quick Start section enabling first command within 60 seconds.

### 1.2 Prerequisites
List prerequisites with verification commands. Each item should have a copy-paste command.

## 2. Interaction Budget

| Operation | Max Prompts | Examples |
|-----------|------------|---------|
| Describe/List | ≤ 1 | Requires only region or resource ID |
| Create | ≤ 2 | Name, spec/type, then auto-fill rest |
| Modify | ≤ 2 | Resource ID, new values |
| Delete | 1 + confirmation | Resource ID, explicit confirmation |

### 2.1 Smart Defaults
Document smart defaults for all optional parameters.

## 3. Feedback

### 3.1 Success Messages
```
✅ [Resource] [Operation] completed.
ID: [resource_id] | Status: [state]
Next steps: [suggestion 1], [suggestion 2]
```

### 3.2 Failure Messages (Standard Format)
```
[ERROR] code: summary
What happened: [brief explanation]
How to fix: [specific actionable fix]
Next step: [immediate next action]
```

## 4. Progress Feedback

Operations > 5 seconds MUST show:
- Current status
- Elapsed time
- ETA (if predictable)

```bash
# Example polling with progress
for i in $(seq 1 60); do
    STATUS=$(hcloud [product] describe --resource-id "$ID" | jq -r '.status')
    printf "\r⏳ Creating... [%3ds] Status: %s" $((i*5)) "$STATUS"
    [ "$STATUS" = "Active" ] && break
    sleep 5
done
```

## 5. Error Handling

All errors follow the `[ERROR]` format from Section 3.2. No raw API error dumping.

---

*This UX spec is mandatory for all generated skills.*
