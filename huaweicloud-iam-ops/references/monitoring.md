# Monitoring & Alerts — Huawei Cloud IAM

> **Purpose:** CTS events, CES metrics, alert rules, and AIOps patterns for IAM monitoring.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [Monitoring Overview](#1-monitoring-overview)
2. [CES Metrics](#2-ces-metrics)
3. [CTS Event Monitoring](#3-cts-event-monitoring)
4. [Alert Configuration](#4-alert-configuration)
5. [AIOps Patterns](#5-aiops-patterns)
6. [Cross-Skill Delegation](#6-cross-skill-delegation)
7. [Proactive Inspection](#7-proactive-inspection)

---

## 1. Monitoring Overview

IAM monitoring relies primarily on **CTS (Cloud Trace Service)** for audit events and **CES (Cloud Eye Service)** for limited metrics. Unlike resource services (ECS, RDS), IAM does not have extensive performance metrics — security event monitoring is the priority.

| Monitoring Type | Primary Tool | Secondary Tool |
|----------------|--------------|----------------|
| Security Events | CTS (Cloud Trace) | — |
| API Metrics | CES (limited) | CTS |
| Credential Status | IAM API | CTS |
| Permission Changes | CTS | — |

---

## 2. CES Metrics

### 2.1 IAM-Related CES Metrics

IAM itself has limited CES metrics. Most monitoring is done via CTS events.

| Metric | Description | Unit | Threshold |
|--------|-------------|------|-----------|
| `iam_api_calls` | IAM API call count | count/min | > 100/min: Warning |
| `iam_auth_failures` | Authentication failure count | count/min | > 10/min: Critical |
| `iam_token_issuances` | Token issuance count | count/min | — |

---

## 3. CTS Event Monitoring

### 3.1 Critical CTS Events

| Event Type | Event Name | Severity | Action |
|------------|-----------|----------|--------|
| User Created | `createUser` | Info | Verify authorization |
| User Deleted | `deleteUser` | Critical | Verify authorization; check CTS for affected apps |
| Policy Created | `createPolicy` | Warning | Review policy content for least privilege |
| Policy Attached | `attachPolicy` | Warning | Verify permission scope |
| AK/SK Created | `createAccessKey` | Critical | Verify authorized creation |
| AK/SK Deleted | `deleteAccessKey` | Critical | May indicate compromise response |
| Login Success | `loginSuccess` | Info | Monitor for unusual patterns |
| Login Failure | `loginFailure` | Warning | Check for brute force attempts |
| MFA Enabled | `enableMFA` | Info | Positive security event |
| MFA Disabled | `disableMFA` | Critical | Investigate immediately |
| Agency Created | `createAgency` | Warning | Review trust scope |
| Role Assigned | `assignRole` | Warning | Verify least privilege |
| Password Changed | `changePassword` | Info | Monitor for unauthorized changes |
| Password Reset | `resetPassword` | Warning | Verify authorization |

### 3.2 CTS Query Patterns

```bash
# Query recent IAM events via CTS
hcloud cts list-traces \
  --service-type IAM \
  --from "$(date -d '1 hour ago' +%Y-%m-%dT%H:%M:%S)" \
  --to "$(date +%Y-%m-%dT%H:%M:%S)"

# Query specific user events
hcloud cts list-traces \
  --service-type IAM \
  --resource-name "zhang-san"

# Query AK/SK creation events
hcloud cts list-traces \
  --service-type IAM \
  --resource-type credential \
  --from "$(date -d '24 hours ago' +%Y-%m-%dT%H:%M:%S)"
```

---

## 4. Alert Configuration

### 4.1 Critical Alerts

| Alert Name | CTS Event | Condition | Severity | Action |
|------------|-----------|-----------|----------|--------|
| MFA Disabled | `disableMFA` | Any occurrence | Critical | Investigate immediately |
| Unauthorized AK/SK | `createAccessKey` | Key created by non-admin | Critical | Disable key + audit |
| User Deleted | `deleteUser` | Any occurrence | Critical | Verify authorization |
| Login Brute Force | `loginFailure` | > 5 failures in 5 min | Critical | Lock account + investigate |
| Policy Over-Permission | `attachPolicy` | Admin policy attached | Warning | Review necessity |

### 4.2 Warning Alerts

| Alert Name | CTS Event | Condition | Severity | Action |
|------------|-----------|-----------|----------|--------|
| New User Created | `createUser` | Any occurrence | Warning | Verify authorization |
| Custom Policy Created | `createPolicy` | Any occurrence | Warning | Review policy content |
| Agency Created | `createAgency` | Any occurrence | Warning | Review trust scope |
| Role Assigned | `assignRole` | Admin role assigned | Warning | Review necessity |
| Stale AK/SK | Periodic check | AK/SK > 90 days old | Warning | Rotate or delete |

### 4.3 Alert Action Matrix

| Alert Type | Primary Action | Secondary Action | Escalation |
|------------|---------------|-----------------|------------|
| MFA Disabled | Alert security team | Re-enable MFA | If confirmed unauthorized |
| Unauthorized AK/SK | Disable key immediately | Audit CTS for usage | If confirmed compromise |
| User Deleted | Verify authorization | Re-create if unauthorized | If unauthorized deletion |
| Brute Force | Lock account | Notify account owner | If > 20 attempts |
| Stale AK/SK | Notify key owner | Schedule rotation | If > 180 days |

---

## 5. AIOps Patterns

### 5.1 Anomaly Detection Patterns

| Pattern ID | Pattern Name | Detection Logic | Severity |
|------------|-------------|-----------------|----------|
| IAM-P001 | Permission Creep | User policy count increases > 20% over 30 days | Warning |
| IAM-P002 | Unused Credentials | AK/SK not used for > 90 days (check via CTS) | Warning |
| IAM-P003 | Stale Accounts | No login events for > 180 days | Warning |
| IAM-P004 | Brute Force Login | > 5 login failures from same IP in 5 min | Critical |
| IAM-P005 | Privilege Escalation | User gains admin-level policy without change request | Critical |
| IAM-P06 | Mass Permission Change | > 10 policy changes in 1 hour | Critical |
| IAM-P007 | Off-Hours Access | IAM API calls outside business hours (> 80% of baseline) | Warning |

### 5.2 Pattern Detection Examples

```markdown
## Pattern: IAM-P001 — Permission Creep

### Detection Logic
1. Query: CTS ListTraces for attachPolicy events per user
2. Aggregate: Count policy attachments per user over 30 days
3. Condition: Growth rate > 20% compared to previous 30 days

### Interpretation
- User is accumulating permissions beyond original scope
- Possible "permission sprawl" from ad-hoc requests
- Security risk: violation of least privilege principle

### Root Causes
1. Ad-hoc permission grants without review
2. Missing group-based permission management
3. No periodic permission audit process

### Fix Actions
1. **Immediate**: Review user's current policies
2. **Short-term**: Replace direct policy assignments with group membership
3. **Long-term**: Implement quarterly permission review process

### Prevention
- Use group-based permission management
- Require approval for direct policy assignments
- Automated permission audit reports
```

### 5.3 Credential Lifecycle Monitoring

```go
// Check for stale AK/SK
func detectStaleCredentials(client *iam.IamClient, domainId string) ([]StaleCredential, error) {
    users, err := listAllUsers(client, domainId)
    if err != nil {
        return nil, err
    }
    
    var staleKeys []StaleCredential
    for _, user := range users {
        keys, err := listAccessKeys(client, user.Id)
        if err != nil {
            continue
        }
        for _, key := range keys {
            // Check if key is older than 90 days
            if time.Since(key.CreateTime) > 90*24*time.Hour {
                staleKeys = append(staleKeys, StaleCredential{
                    UserID:      user.Id,
                    UserName:    user.Name,
                    AccessKeyID: key.Access,
                    CreateTime:  key.CreateTime,
                    Age:         time.Since(key.CreateTime),
                })
            }
        }
    }
    return staleKeys, nil
}
```

---

## 6. Cross-Skill Delegation

### 6.1 Monitoring Delegation Matrix

| Monitoring Need | Primary Skill | Secondary Skill | Notes |
|----------------|---------------|-----------------|-------|
| CTS trace queries | huaweicloud-iam-ops | huaweicloud-cts-ops | CTS skill for complex queries |
| CES metric alerts | huaweicloud-ces-ops | huaweicloud-iam-ops | CES for alert configuration |
| Security incidents | huaweicloud-iam-ops | huaweicloud-hss-ops | HSS for host-level threats |
| Compliance audit | huaweicloud-iam-ops | — | IAM handles identity compliance |

---

## 7. Proactive Inspection

### 7.1 Inspection Workflow

```
[Identity Discovery] → [Permission Analysis] → [Credential Audit] → [Security Report] → [Remediation]
```

### 7.2 Step Details

#### Step 1: Identity Discovery
```bash
# List all IAM users
hcloud iam list-users --domain-id {{env.HW_DOMAIN_ID}} --output json

# List all groups
hcloud iam list-groups --domain-id {{env.HW_DOMAIN_ID}} --output json

# List all custom policies
hcloud iam list-policies --domain-id {{env.HW_DOMAIN_ID}} --type AX --output json
```

#### Step 2: Permission Analysis
```bash
# For each user, list assigned policies
# For each group, list assigned policies
# Identify: over-permissioned users, unused policies, policy conflicts
```

#### Step 3: Credential Audit
```bash
# List all AK/SK for each user
# Identify: stale keys (> 90 days), disabled keys, key count per user
```

#### Step 4: Security Report
```markdown
## IAM Security Posture Report

### Summary
- Total users: N
- Users with MFA: N (X%)
- Users without MFA: N (CRITICAL)
- Stale AK/SK (> 90 days): N
- Over-permissioned users: N
- Unused policies: N

### Critical Findings
| User | Issue | Risk | Recommended Action |
|------|-------|------|-------------------|
| admin-01 | No MFA | Critical | Enable MFA immediately |
| svc-deploy | Stale AK/SK | High | Rotate AK/SK |

### Recommendations
1. Enable MFA for all human users
2. Rotate AK/SK older than 90 days
3. Review permissions for over-provisioned users
4. Clean up unused custom policies
```

#### Step 5: Remediation
- Execute recommended actions with user approval
- Track remediation progress
- Re-inspect after remediation

### 7.3 Dashboard Configuration

```yaml
# IAM Security Dashboard
dashboard:
  name: "IAM Security Posture"
  widgets:
    - title: "Users without MFA"
      type: metric
      query: "count(iam_users where mfa_enabled=false)"
      threshold:
        critical: "> 0"
        
    - title: "Stale AK/SK Count"
      type: metric
      query: "count(iam_credentials where age > 90d)"
      threshold:
        warning: "> 5"
        critical: "> 20"
        
    - title: "Permission Changes (24h)"
      type: timeseries
      query: "cts_events(service=IAM, type=attachPolicy, window=24h)"
      
    - title: "Login Failures (1h)"
      type: timeseries
      query: "cts_events(service=IAM, type=loginFailure, window=1h)"
      threshold:
        warning: "> 10"
        critical: "> 50"
```

---

*This document defines monitoring and alerting patterns for IAM. Update with new metrics and patterns as discovered.*
