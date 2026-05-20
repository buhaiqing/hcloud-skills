# API & SDK Usage — Huawei Cloud IAM

> **Purpose:** Maps IAM operations to API calls and Go SDK patterns.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [API Overview](#1-api-overview)
2. [User Operations](#2-user-operations)
3. [Group Operations](#3-group-operations)
4. [Policy Operations](#4-policy-operations)
5. [Role Operations](#5-role-operations)
6. [Agency Operations](#6-agency-operations)
7. [Project Operations](#7-project-operations)
8. [Credential Operations](#8-credential-operations)
9. [Go SDK Usage Patterns](#9-go-sdk-usage-patterns)
10. [Pagination](#10-pagination)
11. [Error Handling](#11-error-handling)

---

## 1. API Overview

### Base URL

```
https://iam.myhuaweicloud.com
```

> **Important:** IAM is a **global service** — the endpoint is NOT region-specific.

### API Version

- **Current Version:** v3 (Recommended)
- **Legacy Version:** v3.0 (Partial compatibility)

### Authentication

All API requests require:
- AK/SK signature (SDK handles automatically), OR
- `X-Auth-Token` header with valid IAM token

### Common Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Content-Type` | Yes | `application/json` |
| `X-Auth-Token` | Yes* | Domain-scoped or project-scoped token |
| `X-Domain-Id` | Yes | Domain ID for domain-level operations |

---

## 2. User Operations

### 2.1 Create User

**API Endpoint:**
```
POST /v3/users
```

**Request Body:**
```json
{
  "user": {
    "name": "app-service-user",
    "domain_id": "d78cbac1xxxxxxxxxxxxxxxx",
    "password": "SecurePassword123!",
    "email": "user@example.com",
    "phone": "+8613800138000",
    "description": "Service account for application"
  }
}
```

**Response Body:**
```json
{
  "user": {
    "id": "07609fb9xxxxxxxxxxxxxxxx",
    "name": "app-service-user",
    "domain_id": "d78cbac1xxxxxxxxxxxxxxxx",
    "email": "user@example.com",
    "phone": "+8613800138000",
    "description": "Service account for application",
    "enabled": true,
    "create_time": "2026-05-20T10:30:00.000000"
  }
}
```

### 2.2 List Users

**API Endpoint:**
```
GET /v3/users?domain_id={domain_id}
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `domain_id` | String | Yes | Domain ID |
| `limit` | Integer | No | Records per page (default: 50, max: 500) |
| `offset` | Integer | No | Offset for pagination |
| `name` | String | No | Filter by user name |
| `enabled` | Boolean | No | Filter by enabled status |

### 2.3 Show User

**API Endpoint:**
```
GET /v3/users/{user_id}
```

### 2.4 Update User

**API Endpoint:**
```
PUT /v3/users/{user_id}
```

**Request Body:**
```json
{
  "user": {
    "email": "new-email@example.com",
    "phone": "+8613800138001",
    "description": "Updated description"
  }
}
```

### 2.5 Delete User

**API Endpoint:**
```
DELETE /v3/users/{user_id}
```

---

## 3. Group Operations

### 3.1 Create Group

**API Endpoint:**
```
POST /v3/groups
```

**Request Body:**
```json
{
  "group": {
    "name": "developers",
    "domain_id": "d78cbac1xxxxxxxxxxxxxxxx",
    "description": "Development team group"
  }
}
```

### 3.2 List Groups

**API Endpoint:**
```
GET /v3/groups?domain_id={domain_id}
```

### 3.3 Add User to Group

**API Endpoint:**
```
PUT /v3/groups/{group_id}/users/{user_id}
```

### 3.4 Remove User from Group

**API Endpoint:**
```
DELETE /v3/groups/{group_id}/users/{user_id}
```

---

## 4. Policy Operations

### 4.1 Create Custom Policy

**API Endpoint:**
```
POST /v3/policies
```

**Request Body:**
```json
{
  "policy": {
    "name": "ECS-ReadOnly-Prod",
    "description": "Read-only access to ECS in production",
    "type": "AX",
    "content": {
      "Version": "1.1",
      "Statement": [
        {
          "Effect": "Allow",
          "Action": [
            "ecs:servers:get",
            "ecs:servers:list",
            "ecs:cloudservers:get"
          ],
          "Resource": ["*"],
          "Condition": {
            "StringEquals": {
              "hw:project": "cn-north-4"
            }
          }
        }
      ]
    }
  }
}
```

### 4.2 List Policies

**API Endpoint:**
```
GET /v3/policies?domain_id={domain_id}
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `domain_id` | String | Yes | Domain ID |
| `type` | String | No | Filter: `AX` (custom), `role` (system) |
| `name` | String | No | Filter by policy name |
| `limit` | Integer | No | Records per page |
| `offset` | Integer | No | Pagination offset |

### 4.3 Attach Policy to User

**API Endpoint:**
```
PUT /v3/policies/{policy_id}/users/{user_id}
```

**Request Body:**
```json
{
  "scope": {
    "domain": {
      "id": "d78cbac1xxxxxxxxxxxxxxxx"
    }
  }
}
```

### 4.4 Attach Policy to Group

**API Endpoint:**
```
PUT /v3/policies/{policy_id}/groups/{group_id}
```

### 4.5 Detach Policy from User

**API Endpoint:**
```
DELETE /v3/policies/{policy_id}/users/{user_id}?scope=domain&domain_id={domain_id}
```

---

## 5. Role Operations

### 5.1 List System Roles

**API Endpoint:**
```
GET /v3/roles?domain_id={domain_id}
```

### 5.2 Assign Role on Project

**API Endpoint:**
```
PUT /v3/projects/{project_id}/groups/{group_id}/roles/{role_id}
```

### 5.3 List User Roles

**API Endpoint:**
```
GET /v3/users/{user_id}/roles?domain_id={domain_id}
```

---

## 6. Agency Operations

### 6.1 Create Agency

**API Endpoint:**
```
POST /v3/agencies
```

**Request Body:**
```json
{
  "agency": {
    "name": "cross-account-delegation",
    "domain_id": "d78cbac1xxxxxxxxxxxxxxxx",
    "trust_domain_id": "f39a8dcexxxxxxxxxxxxxxxx",
    "description": "Delegation for cross-account access",
    "duration": "FOREVER"
  }
}
```

### 6.2 List Agencies

**API Endpoint:**
```
GET /v3/agencies?domain_id={domain_id}
```

### 6.3 Attach Policy to Agency

**API Endpoint:**
```
PUT /v3/agencies/{agency_id}/policies/{policy_id}
```

### 6.4 Delete Agency

**API Endpoint:**
```
DELETE /v3/agencies/{agency_id}
```

---

## 7. Project Operations

### 7.1 List Projects

**API Endpoint:**
```
GET /v3/projects?domain_id={domain_id}
```

### 7.2 Show Project

**API Endpoint:**
```
GET /v3/projects/{project_id}
```

---

## 8. Credential Operations

### 8.1 Create Access Key (AK/SK)

**API Endpoint:**
```
POST /v3/users/{user_id}/credentials/accesskeys
```

**Request Body:**
```json
{
  "credential": {
    "description": "AK/SK for application deployment",
    "user_id": "07609fb9xxxxxxxxxxxxxxxx"
  }
}
```

**Response Body:**
```json
{
  "credential": {
    "access": "LIXGMxxxxxxxxxxxxx",
    "secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "user_id": "07609fb9xxxxxxxxxxxxxxxx",
    "description": "AK/SK for application deployment",
    "create_time": "2026-05-20T10:30:00.000000",
    "status": "active"
  }
}
```

> **CRITICAL:** The `secret` field is only returned once during creation. It CANNOT be retrieved later.

### 8.2 List Access Keys

**API Endpoint:**
```
GET /v3/users/{user_id}/credentials/accesskeys
```

### 8.3 Delete Access Key

**API Endpoint:**
```
DELETE /v3/users/{user_id}/credentials/accesskeys/{access_key_id}
```

---

## 9. Go SDK Usage Patterns

### 9.1 Client Initialization

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    iam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
)

func initIamClient() *iam.IamClient {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    domainId := os.Getenv("HW_DOMAIN_ID")
    
    cfg := config.DefaultHttpConfig()
    client := iam.IamClientBuilder().
        WithEndpoint("https://iam.myhuaweicloud.com").
        WithCredential(basic.NewCredentialsBuilder().
            WithAk(ak).WithSk(sk).WithDomainId(domainId).Build()).
        WithHttpConfig(cfg).
        Build()
    
    return client
}
```

### 9.2 Create User Pattern

```go
func createIamUser(client *iam.IamClient, domainId string) (*iam_model.CreateUserResponse, error) {
    request := &iam_model.CreateUserRequest{
        Body: &iam_model.CreateUserRequestBody{
            User: iam_model.CreateUserOption{
                Name:        "app-service-user",
                DomainId:    domainId,
                Description: ptrString("Service account for application"),
            },
        },
    }
    
    response, err := client.CreateUser(request)
    if err != nil {
        return nil, err
    }
    return response, nil
}

func ptrString(s string) *string { return &s }
```

### 9.3 Create Policy Pattern

```go
func createCustomPolicy(client *iam.IamClient, domainId string) (*iam_model.CreatePolicyResponse, error) {
    policyContent := map[string]interface{}{
        "Version": "1.1",
        "Statement": []map[string]interface{}{
            {
                "Effect":   "Allow",
                "Action":   []string{"ecs:servers:get", "ecs:servers:list"},
                "Resource": []string{"*"},
            },
        },
    }
    contentJSON, _ := json.Marshal(policyContent)
    
    request := &iam_model.CreatePolicyRequest{
        Body: &iam_model.CreatePolicyRequestBody{
            Policy: iam_model.CreatePolicyOption{
                Name:        "ECS-ReadOnly-Custom",
                Description: ptrString("Custom read-only policy for ECS"),
                Type:        "AX",
                Content:     string(contentJSON),
            },
        },
    }
    
    response, err := client.CreatePolicy(request)
    if err != nil {
        return nil, err
    }
    return response, nil
}
```

---

## 10. Pagination

### 10.1 Offset-Based Pagination

```go
func listAllUsers(client *iam.IamClient, domainId string) ([]iam_model.UserResult, error) {
    var allUsers []iam_model.UserResult
    offset := 0
    limit := 100
    
    for {
        request := &iam_model.ListUsersRequest{
            DomainId: domainId,
            Limit:    &limit,
            Offset:   &offset,
        }
        
        response, err := client.ListUsers(request)
        if err != nil {
            return nil, err
        }
        
        allUsers = append(allUsers, response.Users...)
        
        if len(response.Users) < limit {
            break
        }
        
        offset += limit
    }
    
    return allUsers, nil
}
```

---

## 11. Error Handling

### 11.1 Common IAM Error Codes

| Error Code | HTTP Status | Description | Recovery Action |
|------------|-------------|-------------|-----------------|
| IAM.0001 | 400 | Invalid parameter | Fix parameters; retry |
| IAM.0002 | 409 | Resource already exists | Change name or use existing |
| IAM.0003 | 400 | Quota exceeded | HALT; request quota increase |
| IAM.0004 | 404 | Resource not found | Verify ID; check if deleted |
| IAM.0005 | 403 | Permission denied | Check IAM policy; delegate to IAM skill |
| IAM.0006 | 401 | Authentication failed | Verify credentials; check AK/SK |
| IAM.0007 | 400 | Invalid policy document | Fix policy JSON syntax |
| IAM.0008 | 409 | Resource in use | Wait for operation; retry |
| IAM.0009 | 400 | MFA required | Provide MFA code |
| IAM.0010 | 429 | Rate limit exceeded | Retry with exponential backoff |
| IAM.0011 | 500 | Internal server error | Retry with backoff |
| IAM.0012 | 403 | Domain mismatch | Verify domain ID |
| IAM.0013 | 400 | Invalid credential type | Use correct credential type |
| IAM.0014 | 400 | Password policy violation | Fix password to meet policy |
| IAM.0015 | 403 | Account locked | Wait lockout duration or admin unlock |

### 11.2 Retry Strategy

| Error Category | Retryable | Max Retries | Backoff Strategy |
|----------------|-----------|-------------|------------------|
| Throttling (429) | Yes | 3 | Exponential: 1s, 2s, 4s |
| Internal Error (5xx) | Yes | 3 | Exponential: 2s, 4s, 8s |
| Quota Exceeded | No | 0 | HALT |
| Invalid Parameter | No | 0 | Fix and retry |
| Permission Denied | No | 0 | Check IAM policy |

---

*This document defines API and SDK usage patterns for IAM operations. Refer to official Huawei Cloud OpenAPI documentation for complete API specifications.*
