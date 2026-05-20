# Core Concepts — Huawei Cloud IAM

> **Purpose:** Defines IAM architecture, identity model, permission model, quotas, and resource relationships.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [Service Overview](#1-service-overview)
2. [Architecture](#2-architecture)
3. [Core Concepts](#3-core-concepts)
4. [Permission Models](#4-permission-models)
5. [Authentication Methods](#5-authentication-methods)
6. [Regions and Global Behavior](#6-regions-and-global-behavior)
7. [Quotas and Limits](#7-quotas-and-limits)
8. [Resource Relationships](#8-resource-relationships)
9. [SPOF Analysis](#9-spof-analysis)

---

## 1. Service Overview

Huawei Cloud Identity and Access Management (IAM) provides identity authentication, permission control, and secure access to cloud resources and services. IAM is the **security foundation** for all Huawei Cloud services.

### Key Features

| Feature | Description |
|---------|-------------|
| Identity Management | Users, groups, and domains for identity lifecycle |
| Access Control | Policies, roles, and permissions for fine-grained access |
| Credential Management | AK/SK, passwords, and tokens for authentication |
| Agency | Cross-account delegation for secure resource sharing |
| Federation | SAML/OIDC federation for enterprise identity integration |
| MFA | Multi-factor authentication for enhanced security |

---

## 2. Architecture

### 2.1 IAM Global Service Plane

```
┌─────────────────────────────────────────────────────────┐
│                   IAM Global Service                     │
│              (iam.myhuaweicloud.com)                     │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐    │
│  │  Domain   │  │  Users   │  │  Credentials      │    │
│  │ (Account) │  │  Groups  │  │  (AK/SK, Password)│    │
│  └──────────┘  └──────────┘  └───────────────────┘    │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐    │
│  │ Policies │  │  Roles   │  │  Agencies          │    │
│  │ (Custom) │  │ (System) │  │  (Delegation)      │    │
│  └──────────┘  └──────────┘  └───────────────────┘    │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐    │
│  │ Projects │  │ Federation│  │  MFA Devices      │    │
│  │ (Region) │  │ (SAML)   │  │  (Virtual)        │    │
│  └──────────┘  └──────────┘  └───────────────────┘    │
└─────────────────────────────────────────────────────────┘
          │                    │
          ▼                    ▼
┌─────────────────┐  ┌─────────────────┐
│  Regional        │  │  Regional        │
│  Services (ECS,  │  │  Services (RDS,  │
│  VPC, CES...)    │  │  ELB, OBS...)    │
│  cn-north-4      │  │  cn-east-2       │
└─────────────────┘  └─────────────────┘
```

### 2.2 Identity Hierarchy

```
Domain (Account)
    │
    ├─► IAM User 1
    │   ├─► Credentials (AK/SK, Password, MFA)
    │   ├─► Group Memberships
    │   └─► Direct Policy Assignments
    │
    ├─► IAM User 2
    │   └─► ...
    │
    ├─► User Group A
    │   ├─► Members: [User 1, User 2]
    │   └─► Policies: [Policy X, Policy Y]
    │
    ├─► User Group B
    │   └─► ...
    │
    ├─► Agency (Cross-Account)
    │   └─► Delegated Policies
    │
    └─► Projects (Region-scoped)
        ├─► cn-north-4: Project A
        └─► cn-east-2: Project B
```

---

## 3. Core Concepts

### 3.1 Domain (Account)

| Concept | Description |
|---------|-------------|
| Domain | The top-level account entity; owns all resources |
| Domain ID | Unique identifier for the account (`HW_DOMAIN_ID`) |
| Domain Name | Account name used for login |

### 3.2 User

| Concept | Description |
|---------|-------------|
| IAM User | Identity within a domain; can be human or service |
| User ID | Unique identifier for the user |
| User Name | Display name; must be unique within domain |
| User Type | Normal user or federated user |

### 3.3 Group

| Concept | Description |
|---------|-------------|
| User Group | Collection of users with shared permissions |
| Group ID | Unique identifier for the group |
| Group Name | Display name; must be unique within domain |

### 3.4 Policy

| Concept | Description |
|---------|-------------|
| System Policy | Pre-defined by Huawei Cloud (e.g., `ECS FullAccess`) |
| Custom Policy | User-defined with fine-grained permission rules |
| Policy Document | JSON document defining Allow/Deny actions on resources |

### 3.5 Role

| Concept | Description |
|---------|-------------|
| Role | Legacy permission model; system-defined roles |
| Role Assignment | Assigning roles to users/groups on projects |

### 3.6 Agency

| Concept | Description |
|---------|-------------|
| Agency | Cross-account delegation mechanism |
| Trust Domain | The domain that delegates access |
| Trusted Domain | The domain that receives delegated access |
| Duration | Validity period of the delegation |

### 3.7 Credential

| Credential Type | Description | Use Case |
|-----------------|-------------|----------|
| AK/SK (Access Key) | Permanent programmatic access | API/SDK authentication |
| Password | Console login credential | Human user login |
| Token | Temporary access token | Session-based authentication |
| MFA Device | Multi-factor authentication | Enhanced login security |

### 3.8 Federation

| Federation Type | Description | Protocol |
|-----------------|-------------|----------|
| SAML 2.0 | Enterprise SSO integration | SAML |
| OIDC | OpenID Connect identity provider | OIDC |

---

## 4. Permission Models

### 4.1 RBAC (Role-Based Access Control)

| Aspect | Description |
|--------|-------------|
| Model | Users → Groups → Roles/Permissions |
| Granularity | Service-level (e.g., ECS FullAccess) |
| Assignment | Assign system roles to groups/users |
| Scope | Project-level or domain-level |

### 4.2 PBAC (Policy-Based Access Control)

| Aspect | Description |
|--------|-------------|
| Model | Users/Groups → Custom Policies → Actions |
| Granularity | API-level (e.g., `ecs:servers:create`) |
| Assignment | Create custom policies with specific actions |
| Scope | Project-level or domain-level |
| Conditions | Support IP condition, time condition |

### 4.3 ABAC (Attribute-Based Access Control)

| Aspect | Description |
|--------|-------------|
| Model | Conditions based on resource tags, request attributes |
| Granularity | Resource tag-based (e.g., `environment:prod`) |
| Status | Partially supported; evolving |

### Policy Evaluation Logic

```
1. Explicit Deny → ALWAYS deny (overrides everything)
2. Explicit Allow → Allow (if no deny)
3. Default → Deny (no matching policy = deny)
```

---

## 5. Authentication Methods

| Method | Credential | Use Case | Token Validity |
|--------|-----------|----------|----------------|
| AK/SK | Access Key ID + Secret Access Key | API/SDK calls | Per-request signature |
| Password + MFA | Username + Password + MFA code | Console login | Session-based |
| Token | X-Auth-Token header | API calls | 24 hours (default) |
| Agency | Domain + Agency name | Cross-account access | Configurable |

---

## 6. Regions and Global Behavior

### Global Service

| Aspect | Behavior |
|--------|----------|
| Endpoint | `https://iam.myhuaweicloud.com` (fixed, NOT region-specific) |
| Users | Global — visible across all regions |
| Groups | Global — visible across all regions |
| Policies | Global — apply across all regions |
| Projects | Region-scoped — one default project per region |
| Roles | Project-scoped — assigned per project |

### Project Scope

| Scope | Description | Example |
|-------|-------------|---------|
| Domain-level | Applies to all projects | `iam:users:list` |
| Project-level | Applies to specific project | `ecs:servers:create` in `cn-north-4` |

---

## 7. Quotas and Limits

### 7.1 User Quotas

| Resource | Default Quota | Maximum |
|----------|---------------|---------|
| IAM Users per domain | 200 | 5,000 (request increase) |
| User Groups per domain | 100 | 500 |
| Custom Policies per domain | 200 | 1,000 |
| AK/SK per user | 2 | 2 (hard limit) |
| Agencies per domain | 100 | 500 |
| MFA Devices per user | 1 | 1 (virtual or physical) |
| Projects per region | 1 (default) | 10 |

### 7.2 Policy Size Limits

| Limit | Value |
|-------|-------|
| Policy document size | Max 4,096 characters |
| Statements per policy | Max 20 |
| Actions per statement | Max 100 |
| Resources per statement | Max 100 |
| Conditions per statement | Max 10 |

### 7.3 API Rate Limits

| Operation | Rate Limit |
|-----------|------------|
| CreateUser | 10/minute |
| ListUsers | 100/minute |
| CreatePolicy | 10/minute |
| CreateAccessKey | 5/minute |
| Login (Token) | 20/minute |

---

## 8. Resource Relationships

### 8.1 Dependency Graph

```
Domain (Account)
    │
    ├─► User ──► AK/SK (Credentials)
    │        ├──► Password
    │        └──► MFA Device
    │
    ├─► Group ──► User Membership
    │         └──► Policy Assignment
    │
    ├─► Policy ──► Policy Document (JSON)
    │          └──► Assigned to Users/Groups
    │
    ├─► Agency ──► Trust Domain
    │          └──► Delegated Policies
    │
    └─► Project ──► Region-scoped
               └──► Role Assignments
```

### 8.2 Creation Order

1. **Domain** — Automatically created with account registration
2. **Groups** — Create before users for organized permission management
3. **Policies** — Create custom policies before assigning
4. **Users** — Create users and add to groups
5. **Credentials** — Create AK/SK or configure MFA
6. **Agencies** — Create for cross-account access

### 8.3 Deletion Order

1. **AK/SK** — Delete access keys first (prevent access)
2. **Agency Delegation** — Remove agency permissions
3. **Group Membership** — Remove user from groups
4. **Policy Assignments** — Detach policies from user/group
5. **User** — Delete user (cascade: removes memberships, credentials)
6. **Group** — Delete group (must be empty)
7. **Policy** — Delete custom policy (must be unattached)

---

## 9. SPOF Analysis

### 9.1 Single Points of Failure

| Component | Risk Level | Mitigation |
|-----------|------------|------------|
| Single admin account | **CRITICAL** | Create multiple admin users with MFA |
| No MFA on admin | **CRITICAL** | Enforce MFA for all admin users |
| Single AK/SK for services | **HIGH** | Use separate AK/SK per service; rotate regularly |
| Over-permissioned group | **HIGH** | Apply least privilege; regular audits |
| No credential rotation | **HIGH** | Enforce 90-day rotation policy |

### 9.2 Security Recommendations

| Scenario | Recommendation |
|----------|----------------|
| Production account | Multiple admin users + MFA + credential rotation |
| Service accounts | Dedicated users + least privilege policies + AK/SK rotation |
| Development | Separate domain/project from production |
| Cross-account access | Agency with scoped permissions + audit |

### 9.3 Disaster Recovery

| DR Level | RTO | Implementation |
|----------|-----|----------------|
| User deletion | Immediate | Re-create user + reassign policies |
| Credential compromise | Minutes | Disable AK/SK + create new |
| Policy deletion | Minutes | Re-create from backup policy document |
| Account lockout | Minutes | Admin reset or console recovery |

---

*This document defines the core architectural concepts for IAM operations. Refer to official Huawei Cloud documentation for the latest limits and specifications.*
