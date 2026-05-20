# Troubleshooting Guide — Huawei Cloud IAM

> **Purpose:** Error codes, diagnostics, and recovery strategies for IAM operations.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [Error Code Taxonomy](#1-error-code-taxonomy)
2. [Common Issues & Solutions](#2-common-issues--solutions)
3. [Diagnostic Workflow](#3-diagnostic-workflow)
4. [HALT vs Retry Decision Matrix](#4-halt-vs-retry-decision-matrix)
5. [Escalation Paths](#5-escalation-paths)

---

## 1. Error Code Taxonomy

### 1.1 IAM Error Codes (≥15 Required)

| Error Code | HTTP Status | Description | Severity | Recovery Action |
|------------|-------------|-------------|----------|-----------------|
| IAM.0001 | 400 | Invalid parameter value | High | Fix parameter based on error message; retry |
| IAM.0002 | 409 | Resource already exists | Medium | Use different name or manage existing |
| IAM.0003 | 400 | Quota exceeded | Critical | HALT; request quota increase via console |
| IAM.0004 | 404 | Resource not found | Medium | Verify resource ID; check if deleted |
| IAM.0005 | 403 | Permission denied | Critical | HALT; check IAM policy for caller |
| IAM.0006 | 401 | Authentication failed | Critical | Verify credentials; check AK/SK validity |
| IAM.0007 | 400 | Invalid policy document | High | Fix policy JSON syntax; validate against schema |
| IAM.0008 | 409 | Resource in use | High | Wait for operation; check dependent resources |
| IAM.0009 | 400 | MFA required | High | Provide MFA code; re-authenticate |
| IAM.0010 | 429 | Rate limit exceeded | High | Back off 60s; retry with exponential backoff |
| IAM.0011 | 500 | Internal server error | Medium | Retry with exponential backoff (2s, 4s, 8s) |
| IAM.0012 | 403 | Domain mismatch | Critical | Verify domain ID matches the target domain |
| IAM.0013 | 400 | Invalid credential type | High | Use correct credential type for the operation |
| IAM.0014 | 400 | Password policy violation | High | Fix password to meet policy requirements |
| IAM.0015 | 403 | Account locked | Critical | Wait lockout duration or admin unlock |

### 1.2 Permission Error Patterns

| Error Pattern | Symptoms | Diagnosis | Fix |
|---------------|----------|-----------|-----|
| 403 on resource operation | User cannot access resource | Check user's policies and role assignments | Add appropriate policy/role |
| 403 on IAM operation | Admin operation blocked | Check if caller has `iam:*` permissions | Grant IAM management policy |
| 403 cross-account | Agency access denied | Check agency policies and trust | Update agency delegation |
| 401 token expired | API calls fail after time | Token has expired | Re-authenticate to get new token |

### 1.3 Network Error Patterns

| Error Pattern | Symptoms | Diagnosis | Fix |
|---------------|----------|-----------|-----|
| Connection refused | Cannot reach IAM endpoint | Check network connectivity | Verify access to iam.myhuaweicloud.com |
| DNS resolution failed | Host not found | Check DNS configuration | Use IP or fix DNS |
| SSL handshake failed | TLS error | Check TLS version | Ensure TLS 1.2+ support |

---

## 2. Common Issues & Solutions

### 2.1 "Permission Denied" Diagnosis Flow

```
[403 Permission Denied Error]
        │
        ▼
Step 1: Identify the denied action
  ├─► What API/resource was being accessed?
  │   └─► Extract from error message
  │
        ▼
Step 2: Check caller identity
  ├─► Which user/AK/SK made the request?
  │   └─► List user policies → any matching action?
  │   └─► List group memberships → any group with access?
  │
        ▼
Step 3: Check policy scope
  ├─► Domain-level or project-level?
  │   └─► Policy may allow action but in wrong project
  │
        ▼
Step 4: Check for explicit deny
  ├─► Any policy with "Effect": "Deny" for this action?
  │   └─► Deny always overrides Allow
  │
        ▼
Step 5: Check resource conditions
  ├─► Policy has conditions (IP, time, project)?
  │   └─► Request must meet all conditions
  │
        ▼
Step 6: Fix and verify
  ├─► Add/modify policy to grant required action
  │   └─► Verify with test request
  │
[Resolved]
```

### 2.2 "User Cannot Login" Diagnosis Flow

```
[Login Failure]
        │
        ▼
Step 1: Check user status
  ├─► ShowUser → enabled = true?
  │   └─► Disabled → Re-enable user
  │
        ▼
Step 2: Check account lockout
  ├─► Too many failed login attempts?
  │   └─► Yes → Wait lockout period or admin unlock
  │
        ▼
Step 3: Check password
  ├─► Password correct?
  │   └─► Wrong → Reset password
  │
        ▼
Step 4: Check MFA requirement
  ├─► MFA required but not configured?
  │   └─► Configure MFA device
  │
        ▼
Step 5: Check federation (if SSO)
  ├─► Identity provider configured correctly?
  │   └─► No → Fix SAML/OIDC configuration
  │
[Resolved]
```

### 2.3 "AK/SK Not Working" Diagnosis Flow

```
[AK/SK Authentication Failure]
        │
        ▼
Step 1: Check key status
  ├─► ListAccessKeys → status = active?
  │   └─► Inactive → Re-enable or create new
  │
        ▼
Step 2: Check key ownership
  ├─► AK/SK belongs to the correct user?
  │   └─► Wrong user → Use correct user's AK/SK
  │
        ▼
Step 3: Check key age
  ├─► Created > 90 days ago?
  │   └─► Yes → Rotate AK/SK
  │
        ▼
Step 4: Check user permissions
  ├─► User has required policies?
  │   └─► No → Attach appropriate policy
  │
        ▼
Step 5: Check domain scope
  ├─► AK/SK domain matches target domain?
  │   └─► No → Use correct domain's credentials
  │
[Resolved]
```

### 2.4 "Policy Not Taking Effect" Diagnosis Flow

```
[Policy Not Working]
        │
        ▼
Step 1: Verify policy attachment
  ├─► ListUserPolicies or ListGroupPolicies → policy attached?
  │   └─► Not attached → Attach policy
  │
        ▼
Step 2: Check policy scope
  ├─► Domain-level policy used for project-scoped resource?
  │   └─► Wrong scope → Create project-scoped assignment
  │
        ▼
Step 3: Check for deny override
  ├─► Any other policy explicitly denying the action?
  │   └─► Yes → Remove or modify the deny policy
  │
        ▼
Step 4: Check policy syntax
  ├─► Action names correct? (e.g., `ecs:servers:list` not `ecs:list`)
  │   └─► Wrong action → Fix policy document
  │
        ▼
Step 5: Check conditions
  ├─► Condition block restricting access?
  │   └─► Yes → Verify condition is met
  │
[Resolved]
```

---

## 3. Diagnostic Workflow

### Round 1: Initial Diagnosis

1. **Collect error details**
   - HTTP status code
   - Error code (e.g., IAM.0005)
   - Error message
   - Request ID

2. **Check identity context**
   - Which user/AK/SK made the request?
   - What policies are assigned?
   - What groups is the user in?

3. **Output initial hypothesis**

### Round 2: Critical Reflection

1. **Challenge assumptions**
   - Is the 403 really a permission issue, or could it be domain mismatch?
   - Is the AK/SK actually disabled, or is there a network issue?
   - Are there multiple conflicting policies?

2. **Expand investigation**
   - Check CTS audit trail for recent changes
   - Check if other users in the same group have the same issue
   - Verify policy evaluation order

3. **Output revised hypothesis**

### Round 3: Deep Review (if needed)

1. **Check policy evaluation**
   - List ALL policies assigned to user (direct + group-inherited)
   - Check for deny statements
   - Check condition blocks
   - Verify resource scope

2. **Check change history**
   - Any recent policy modifications?
   - Any recent group membership changes?
   - Any recent credential rotations?

3. **Output final root cause with confidence**

---

## 4. HALT vs Retry Decision Matrix

| Error Category | Action | Reason |
|----------------|--------|--------|
| `IAM.0001` Invalid Parameter | HALT + Fix | Bad input; will not succeed on retry |
| `IAM.0002` Already Exists | HALT + Ask | Resource exists; need user decision |
| `IAM.0003` Quota Exceeded | HALT | Cannot auto-fix; needs quota increase |
| `IAM.0004` Not Found | HALT + Verify | Resource may be deleted or wrong ID |
| `IAM.0005` Permission Denied | HALT + Delegate | Need IAM admin to fix |
| `IAM.0006` Auth Failed | HALT + Fix | Credentials are invalid |
| `IAM.0007` Invalid Policy | HALT + Fix | Policy syntax must be corrected |
| `IAM.0008` In Use | Retry (wait) | Wait for conflicting operation |
| `IAM.0009` MFA Required | HALT + Prompt | Need MFA code from user |
| `IAM.0010` Rate Limited | Retry (backoff) | Exponential: 1s, 2s, 4s |
| `IAM.0011` Internal Error | Retry (backoff) | Exponential: 2s, 4s, 8s |
| `IAM.0012` Domain Mismatch | HALT + Fix | Wrong domain ID |
| `IAM.0014` Password Policy | HALT + Fix | Password must meet policy |
| `IAM.0015` Account Locked | HALT + Wait | Wait lockout period |

---

## 5. Escalation Paths

### 5.1 Escalation Matrix

| Issue Type | L1 (Agent) | L2 (IAM Admin) | L3 (Huawei Support) |
|-----------|------------|----------------|---------------------|
| Permission denied | Check policies | Modify policy | N/A |
| AK/SK compromise | Disable key | Create new key + audit | Report incident |
| Account lockout | Check attempts | Unlock account | N/A |
| Quota exceeded | List usage | Request increase | Submit ticket |
| Internal error | Retry 3x | Check status page | Open support ticket |
| Federation failure | Check config | Fix SAML/OIDC | Contact IdP vendor |
| MFA issues | Check device | Reset MFA | N/A |

### 5.2 Emergency Contacts

| Emergency | Action | Contact |
|-----------|--------|---------|
| Compromised credentials | Immediately disable all AK/SK | Account admin + security team |
| Unauthorized access | Revoke permissions + enable MFA | Security team |
| Account takeover | Lock account + contact Huawei | Huawei Cloud support |

---

*This document defines troubleshooting patterns for IAM operations. Update with new error codes and patterns as discovered.*
