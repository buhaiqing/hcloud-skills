# Well-Architected Assessment — Huawei Cloud IAM

> **Purpose:** Five pillars + FinOps + SecOps + AIOps assessment for IAM operations.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [Framework Overview](#1-framework-overview)
2. [Five Pillar Assessment](#2-five-pillar-assessment)
3. [FinOps Integration](#3-finops-)
4. [SecOps Integration](#4-secops-)
5. [AIOps Integration](#5-aiops-integration)
6. [Compliance Checklists](#6-compliance-checklists)

---

## 1. Framework Overview

Huawei Cloud Well-Architected Framework (卓越架构) for IAM with integrated FinOps, SecOps, and AIOps.

> **Security is the MOST IMPORTANT pillar for IAM.** IAM is the security foundation for all cloud operations.

| Pillar | IAM Relevance | Key Operations |
|--------|---------------|-----------------|
| **安全 (Security)** | MOST CRITICAL | MFA, least privilege, credential rotation, audit |
| **稳定 (Stability)** | High | Credential backup, group-based access, delegation |
| **成本 (Cost)** | Low | IAM is free; indirect cost from misconfig |
| **效率 (Efficiency)** | Medium | Policy templates, group-based management |
| **性能 (Performance)** | Low | API rate limits, pagination |

---

## 2. Five Pillar Assessment

### 2.1 安全 (Security) — MOST CRITICAL

#### Zero Trust Architecture

| Principle | IAM Implementation | Enforcement |
|-----------|-------------------|-------------|
| Never trust, always verify | MFA for all human users | Policy enforcement |
| Least privilege | Fine-grained custom policies | No wildcard actions in production |
| Assume breach | Credential rotation, short-lived tokens | 90-day AK/SK rotation |
| Explicit deny | Deny overrides Allow in policy evaluation | Policy design pattern |
| Audit everything | CTS enabled for all IAM events | Continuous monitoring |

#### IAM Minimum Permissions for Managing IAM

| Operation | IAM Action | Resource Scope |
|-----------|-----------|----------------|
| List Users | iam:users:list | * |
| Create User | iam:users:create | * |
| Delete User | iam:users:delete | * |
| Create Policy | iam:policies:create | * |
| Attach Policy | iam:policies:attach | * |
| Create AK/SK | iam:credentials:create | * |
| Create Agency | iam:agencies:create | * |
| List Roles | iam:roles:list | * |

#### IAM Policy Example (for IAM Administrators)

```json
{
  "Version": "1.1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "iam:users:list",
        "iam:users:get",
        "iam:users:create",
        "iam:users:update",
        "iam:users:delete",
        "iam:groups:list",
        "iam:groups:create",
        "iam:policies:list",
        "iam:policies:create",
        "iam:policies:attach",
        "iam:policies:detach",
        "iam:credentials:list",
        "iam:credentials:create",
        "iam:credentials:delete",
        "iam:agencies:list",
        "iam:agencies:create",
        "iam:roles:list",
        "iam:roles:assign"
      ],
      "Resource": ["*"]
    },
    {
      "Effect": "Deny",
      "Action": [
        "iam:users:delete"
      ],
      "Resource": ["*"],
      "Condition": {
        "StringNotEquals": {
          "hw:UserName": ["service-admin"]
        }
      }
    }
  ]
}
```

#### MFA Enforcement

| User Type | MFA Requirement | Enforcement |
|-----------|----------------|-------------|
| Account admin | MANDATORY | Policy enforcement |
| Human users | MANDATORY | Policy enforcement |
| Service accounts | Not applicable | AK/SK only |
| Federated users | Via IdP | IdP policy |

#### Credential Security

| Data State | Encryption Method | Implementation |
|------------|-------------------|----------------|
| AK/SK in transit | TLS 1.2+ | All API calls use HTTPS |
| AK/SK at rest | KMS encryption | Application-level KMS |
| Password at rest | Hashed (bcrypt) | IAM internal |
| Token in memory | Secure storage | Environment variables, not files |

### 2.2 稳定 (Stability)

#### High Availability

| Component | Risk | Mitigation |
|-----------|------|------------|
| Single admin account | CRITICAL | Multiple admin users with MFA |
| No credential backup | HIGH | Documented credential rotation process |
| Direct user policies | MEDIUM | Group-based policy management |
| Hardcoded credentials | HIGH | Environment variable injection |

#### Credential Recovery Runbook

```markdown
## Phase 1: Credential Verification
1. Confirm AK/SK status via ListAccessKeys
2. Identify last usage time via CTS
3. Determine if credential is compromised or just expired

## Phase 2: Credential Rotation
1. Create new AK/SK
2. Update application configuration
3. Verify new credential works
4. Delete old AK/SK

## Phase 3: Post-Rotation Verification
1. Verify application connectivity
2. Monitor CTS for unauthorized usage of old key
3. Update credential inventory

## Phase 4: Post-Mortem
1. Document rotation reason
2. Identify if process can be automated
3. Update rotation schedule
```

### 2.3 成本 (Cost)

#### IAM Cost Model

| Resource | Cost | Notes |
|----------|------|-------|
| IAM service | **Free** | No direct billing |
| Users | Free | Up to quota |
| Policies | Free | Up to quota |
| AK/SK | Free | Up to 2 per user |
| MFA | Free | Virtual MFA |

#### Indirect Cost Impact

| Misconfiguration | Indirect Cost | Detection |
|------------------|---------------|-----------|
| Over-provisioned permissions | Unintended resource creation (e.g., expensive instances) | Policy audit |
| Orphaned resources | Resources no one is responsible for | IAM-Resource correlation |
| Unrestricted agency | Cross-account resource consumption | Agency audit |

### 2.4 效率 (Efficiency)

#### Policy Template Patterns

| Template | Description | Use Case |
|----------|-------------|----------|
| ReadOnly | Read-only access to a service | Auditors, monitoring |
| Operator | Read + Create + Delete | DevOps team |
| Admin | Full access to a service | Service administrators |
| ScopedOperator | Operator access to specific project | Project-scoped operations |

#### Batch Operations

```bash
# Batch add users to group
for user_id in <user_id_list>; do
  hcloud iam add-user-to-group --group-id <group-id> --user-id "$user_id"
done

# Batch create service accounts
for svc in deployment monitoring logging; do
  hcloud iam create-user --domain-id {{env.HW_DOMAIN_ID}} --name "svc-$svc"
done
```

### 2.5 性能 (Performance)

#### API Rate Limits

| Operation | Rate Limit | Optimization |
|-----------|------------|--------------|
| ListUsers | 100/min | Cache results; use pagination |
| CreateUser | 10/min | Batch creation not supported |
| ListPolicies | 100/min | Cache policy list |
| CreateAccessKey | 5/min | Plan ahead; don't create on-demand |

#### Pagination Best Practices

```go
// Efficient pagination for large accounts
func listAllUsersEfficient(client *iam.IamClient, domainId string) {
    limit := 500 // Use maximum page size
    offset := 0
    
    for {
        request := &iam_model.ListUsersRequest{
            DomainId: domainId,
            Limit:    &limit,
            Offset:   &offset,
        }
        
        response, err := client.ListUsers(request)
        if err != nil {
            break
        }
        
        // Process users
        for _, user := range response.Users {
            processUser(user)
        }
        
        if len(response.Users) < limit {
            break
        }
        offset += limit
    }
}
```

---

## 3. FinOps (财务运营)

### 3.1 Cost Visibility

IAM itself is free, but IAM configuration affects resource costs.

| Tag Key | Description | Example |
|---------|-------------|---------|
| CostCenter | Cost center attribution | CC-001 |
| Owner | Resource owner | admin@example.com |
| Environment | Environment type | prod, staging, dev |
| ManagedBy | Management method | terraform, manual |

### 3.2 Cost Optimization

| Action | Trigger | Expected Savings |
|--------|---------|------------------|
| Remove unused permissions | Permission audit | Prevent unintended resource creation |
| Delete orphaned AK/SK | Stale key detection | Prevent unauthorized usage |
| Consolidate groups | Group audit | Management overhead reduction |

### 3.3 Cost Accountability

| Practice | Description | Implementation |
|----------|-------------|----------------|
| Permission review | Quarterly review of all user permissions | Automated report |
| Credential inventory | Track all AK/SK creation and usage | CTS-based tracking |
| Access certification | Annual certification of access rights | Manager approval workflow |

---

## 4. SecOps (安全运营)

### 4.1 Zero Trust Implementation

| Zero Trust Principle | IAM Control | Verification |
|---------------------|-------------|--------------|
| Verify explicitly | MFA for all human users | MFA enrollment audit |
| Use least privilege | Custom policies with specific actions | Policy audit report |
| Assume breach | 90-day AK/SK rotation + CTS monitoring | Rotation compliance check |

### 4.2 Incident Response

| Incident Type | Detection | Response | Recovery |
|--------------|-----------|----------|----------|
| Compromised AK/SK | CTS: unusual API calls | Disable AK/SK immediately | Create new AK/SK + update apps |
| Unauthorized access | CTS: access from unknown IP | Revoke permissions | Review + re-authorize |
| Permission escalation | CTS: admin policy attached | Remove excessive permissions | Audit all policy changes |
| MFA bypass attempt | CTS: MFA disable event | Re-enable MFA + investigate | Reset credentials |
| Account takeover | Multiple failed logins + config changes | Lock account + admin review | Full credential reset |

### 4.3 Compliance Automation

| Compliance Check | Automation Method | Frequency |
|-----------------|-------------------|-----------|
| MFA enabled for all users | Script: list users → check MFA | Daily |
| No admin policies on direct users | Script: list direct policy assignments | Weekly |
| AK/SK rotation compliance | Script: check key age via CTS | Daily |
| No wildcard actions in custom policies | Script: parse policy documents | Weekly |
| Federated IdP configuration | Script: check provider metadata | Monthly |

### 4.4 Supply Chain Security

| Control | Implementation | Verification |
|---------|---------------|--------------|
| Service account isolation | Separate AK/SK per service | AK/SK inventory |
| Scoped agency permissions | Minimal delegation policies | Agency audit |
| Credential rotation | 90-day rotation policy | Rotation compliance |
| Access review | Quarterly permission review | Review completion |

### 4.5 Key Lifecycle Management

| Phase | Action | Automation |
|-------|--------|------------|
| Creation | Generate AK/SK with description | CLI/API with audit |
| Distribution | Store in KMS/secrets manager | Encrypted storage |
| Usage | Monitor via CTS | Usage alerts |
| Rotation | Create new → Update apps → Delete old | Semi-automated |
| Revocation | Disable → Delete | Immediate on compromise |

---

## 5. AIOps Integration

### 5.1 Multi-Metric Correlation

| Pattern | Data Sources | Detection Logic | Severity |
|---------|-------------|-----------------|----------|
| Permission Creep | CTS attachPolicy events | Policy count growth > 20%/30d | Warning |
| Unused Credentials | CTS AK/SK usage | No usage > 90 days | Warning |
| Stale Accounts | CTS login events | No login > 180 days | Warning |
| Brute Force | CTS loginFailure events | > 5 failures/5min from same IP | Critical |
| Privilege Escalation | CTS attachPolicy (admin) | Admin policy without change request | Critical |

### 5.2 Cross-Skill Diagnosis

| Alert | Primary Skill | Secondary Skill | Diagnosis |
|-------|---------------|-----------------|-----------|
| 403 Permission Denied | huaweicloud-iam-ops | Resource skill | Check IAM policy first |
| Credential Compromise | huaweicloud-iam-ops | huaweicloud-cts-ops | Audit CTS for actions taken |
| MFA Disabled | huaweicloud-iam-ops | huaweicloud-hss-ops | Security incident response |
| Mass Permission Change | huaweicloud-iam-ops | — | Investigate authorization |

### 5.3 Knowledge Base

#### Fault Pattern: IAM-01 — Permission Creep

| Field | Content |
|-------|---------|
| Trigger | User policy count increasing over time |
| Symptoms | Users have more permissions than their role requires |
| Correlated Events | CTS: multiple attachPolicy events for same user |
| Root Cause | Ad-hoc permission grants without review process |
| Diagnosis | List user policies → compare against role template |
| Fix | Replace direct assignments with group membership; remove excess policies |
| Prevention | Group-based permission management; quarterly review |

#### Fault Pattern: IAM-02 — Compromised AK/SK

| Field | Content |
|-------|---------|
| Trigger | Unusual API calls detected via CTS |
| Symptoms | API calls from unknown IP, at unusual times, for unauthorized resources |
| Correlated Events | CTS: API calls with AK/SK from unusual source |
| Root Cause | AK/SK leaked via code repository, config file, or phishing |
| Diagnosis | Check CTS for usage pattern; compare with baseline |
| Fix | Immediately disable compromised key → create new key → update applications |
| Prevention | Never commit AK/SK to code; use environment variables; rotate regularly |

---

## 6. Compliance Checklists

### P0 — Must Pass

#### Security (CRITICAL for IAM)
- [ ] MFA enabled for all human users
- [ ] No admin policies attached directly to users (use groups)
- [ ] AK/SK rotation policy defined (90 days)
- [ ] Credential masking enforced in all outputs
- [ ] No wildcard actions in production custom policies
- [ ] CTS enabled for IAM events
- [ ] Agency permissions scoped to minimum required

#### Stability
- [ ] Multiple admin users with MFA configured
- [ ] Credential rotation process documented
- [ ] Group-based permission management implemented
- [ ] Emergency access procedure documented

#### Cost
- [ ] IAM indirect cost impact documented
- [ ] Permission audit prevents unintended resource creation

#### Performance
- [ ] API rate limits documented
- [ ] Pagination implemented for large account operations

### P1 — Should Pass

- [ ] Quarterly permission review process
- [ ] Automated MFA compliance check
- [ ] Stale AK/SK detection automation
- [ ] Federation configuration for enterprise SSO
- [ ] Policy-as-code implementation
- [ ] Proactive inspection workflow
- [ ] Incident response runbook

### Compliance Standards Mapping

| Standard | IAM Control | Implementation |
|----------|------------|----------------|
| CIS Benchmark | MFA for all users | Policy enforcement |
| ISO 27001 | Access control policy | Custom policies + audit |
| GDPR | Data access control | Scoped policies + CTS audit |
| SOC 2 | Logical access security | MFA + rotation + audit |

---

*This document defines the Well-Architected assessment for IAM operations. Refer to official Huawei Cloud documentation for the latest specifications.*
