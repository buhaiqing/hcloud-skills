# API & SDK Usage — Huawei Cloud RDS

> **Purpose:** Maps RDS operations to API calls and Go SDK patterns.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [API Overview](#1-api-overview)
2. [Instance Operations](#2-instance-operations)
3. [Backup Operations](#3-backup-operations)
4. [Database & User Operations](#4-database--user-operations)
5. [Parameter Operations](#5-parameter-operations)
6. [Go SDK Usage Patterns](#6-go-sdk-usage-patterns)
7. [Pagination](#7-pagination)
8. [Error Handling](#8-error-handling)

---

## 1. API Overview

### Base URL

```
https://rds.{region}.myhuaweicloud.com
```

### API Version

- **Current Version:** v3 (Recommended)
- **Legacy Version:** v1 (Deprecated, avoid for new development)

### Authentication

All API requests require:
- `X-Auth-Token` header with valid IAM token, OR
- AK/SK signature (SDK handles automatically)

### Common Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Content-Type` | Yes | `application/json` |
| `X-Auth-Token` | Yes* | IAM authentication token |
| `X-Project-Id` | Yes | Project ID |

---

## 2. Instance Operations

### 2.1 Create Instance

**API Endpoint:**
```
POST /v3/{project_id}/instances
```

**Request Body:**
```json
{
  "name": "rds-mysql-prod-01",
  "datastore": {
    "type": "MySQL",
    "version": "8.0"
  },
  "flavor_ref": "rds.mysql.s1.large",
  "vpc_id": "vpc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "subnet_id": "subnet-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "security_group_id": "sg-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "availability_zone": "cn-north-4a",
  "volume": {
    "type": "ULTRAHIGH",
    "size": 100
  },
  "backup_strategy": {
    "start_time": "02:00-03:00",
    "keep_days": 7
  },
  "ha": {
    "mode": "Ha",
    "replication_mode": "async"
  },
  "port": "3306",
  "password": "YourSecurePassword123!",
  "region": "cn-north-4",
  "enterprise_project_id": "0"
}
```

**Response Body:**
```json
{
  "instance": {
    "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "name": "rds-mysql-prod-01",
    "status": "BUILD",
    "datastore": {
      "type": "MySQL",
      "version": "8.0"
    },
    "flavor_ref": "rds.mysql.s1.large",
    "vpc_id": "vpc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "subnet_id": "subnet-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "security_group_id": "sg-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "availability_zone": "cn-north-4a,cn-north-4b",
    "port": "3306",
    "private_ips": ["192.168.0.100"],
    "region": "cn-north-4",
    "created": "2026-05-20T10:30:00+0800"
  },
  "job_id": "job-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

### 2.2 List Instances

**API Endpoint:**
```
GET /v3/{project_id}/instances
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `limit` | Integer | No | Records per page (default: 10, max: 100) |
| `offset` | Integer | No | Offset for pagination (default: 0) |
| `name` | String | No | Filter by instance name (fuzzy match) |
| `id` | String | No | Filter by instance ID |
| `status` | String | No | Filter by status (BUILD, ACTIVE, FAILED, etc.) |
| `datastore_type` | String | No | Filter by engine type (MySQL, PostgreSQL, SQLServer) |
| `vpc_id` | String | No | Filter by VPC ID |

**Response Body:**
```json
{
  "instances": [
    {
      "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "name": "rds-mysql-prod-01",
      "status": "ACTIVE",
      "datastore": {
        "type": "MySQL",
        "version": "8.0"
      },
      "flavor_ref": "rds.mysql.s1.large",
      "cpu": "2",
      "mem": "4",
      "volume": {
        "type": "ULTRAHIGH",
        "size": 100,
        "used": 25.5
      },
      "vpc_id": "vpc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "subnet_id": "subnet-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "private_ips": ["192.168.0.100"],
      "public_ips": [],
      "port": "3306",
      "region": "cn-north-4",
      "created": "2026-05-20T10:30:00+0800",
      "updated": "2026-05-20T10:35:00+0800"
    }
  ],
  "total_count": 1
}
```

### 2.3 Describe Instance

**API Endpoint:**
```
GET /v3/{project_id}/instances/{instance_id}
```

**Response Body:**
```json
{
  "instance": {
    "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "name": "rds-mysql-prod-01",
    "status": "ACTIVE",
    "datastore": {
      "type": "MySQL",
      "version": "8.0"
    },
    "flavor_ref": "rds.mysql.s1.large",
    "cpu": "2",
    "mem": "4",
    "volume": {
      "type": "ULTRAHIGH",
      "size": 100,
      "used": 25.5
    },
    "vpc_id": "vpc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "subnet_id": "subnet-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "security_group_id": "sg-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "availability_zone": "cn-north-4a,cn-north-4b",
    "private_ips": ["192.168.0.100"],
    "public_ips": [],
    "port": "3306",
    "region": "cn-north-4",
    "created": "2026-05-20T10:30:00+0800",
    "updated": "2026-05-20T10:35:00+0800",
    "backup_strategy": {
      "start_time": "02:00-03:00",
      "keep_days": 7
    },
    "ha": {
      "mode": "Ha",
      "replication_mode": "async",
      "replication_status": "normal"
    }
  }
}
```

### 2.4 Delete Instance

**API Endpoint:**
```
DELETE /v3/{project_id}/instances/{instance_id}
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `delete_backup` | Boolean | No | Whether to delete automated backups (default: false) |

**Response:** HTTP 202 Accepted

### 2.5 Modify Instance

**API Endpoint:**
```
PUT /v3/{project_id}/instances/{instance_id}/name
```

**Request Body:**
```json
{
  "name": "rds-mysql-prod-01-renamed"
}
```

### 2.6 Resize Instance

**API Endpoint:**
```
POST /v3/{project_id}/instances/{instance_id}/flavor
```

**Request Body:**
```json
{
  "flavor_ref": "rds.mysql.s1.xlarge",
  "is_auto_pay": true
}
```

### 2.7 Expand Volume

**API Endpoint:**
```
POST /v3/{project_id}/instances/{instance_id}/volume/extend
```

**Request Body:**
```json
{
  "size": 200,
  "is_auto_pay": true
}
```

### 2.8 Restart Instance

**API Endpoint:**
```
POST /v3/{project_id}/instances/{instance_id}/action
```

**Request Body:**
```json
{
  "restart": {}
}
```

---

## 3. Backup Operations

### 3.1 Create Manual Backup

**API Endpoint:**
```
POST /v3/{project_id}/backups
```

**Request Body:**
```json
{
  "instance_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "name": "manual-backup-20260520",
  "description": "Manual backup before upgrade",
  "type": "manual"
}
```

**Response Body:**
```json
{
  "backup": {
    "id": "backup-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "name": "manual-backup-20260520",
    "instance_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "type": "manual",
    "status": "BUILDING",
    "size": 0,
    "created": "2026-05-20T14:00:00+0800"
  }
}
```

### 3.2 List Backups

**API Endpoint:**
```
GET /v3/{project_id}/backups
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `instance_id` | String | No | Filter by instance ID |
| `backup_type` | String | No | Filter by type (auto, manual) |
| `limit` | Integer | No | Records per page |
| `offset` | Integer | No | Offset for pagination |

### 3.3 Delete Backup

**API Endpoint:**
```
DELETE /v3/{project_id}/backups/{backup_id}
```

### 3.4 Restore from Backup

**API Endpoint:**
```
POST /v3/{project_id}/instances/{instance_id}/restore
```

**Request Body:**
```json
{
  "backup_id": "backup-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "is_auto_pay": true
}
```

### 3.5 Point-in-Time Recovery (PITR)

**API Endpoint:**
```
POST /v3/{project_id}/instances/{instance_id}/restore/point-in-time
```

**Request Body:**
```json
{
  "restore_time": 1716187200000,
  "is_auto_pay": true
}
```

---

## 4. Database & User Operations

### 4.1 Create Database

**API Endpoint:**
```
POST /v3/{project_id}/instances/{instance_id}/database
```

**Request Body:**
```json
{
  "name": "myapp_database",
  "character_set": "utf8mb4",
  "collate": "utf8mb4_general_ci"
}
```

### 4.2 List Databases

**API Endpoint:**
```
GET /v3/{project_id}/instances/{instance_id}/database
```

**Response Body:**
```json
{
  "databases": [
    {
      "name": "myapp_database",
      "character_set": "utf8mb4",
      "collate": "utf8mb4_general_ci"
    }
  ]
}
```

### 4.3 Delete Database

**API Endpoint:**
```
DELETE /v3/{project_id}/instances/{instance_id}/database/{db_name}
```

### 4.4 Create User

**API Endpoint:**
```
POST /v3/{project_id}/instances/{instance_id}/db_user
```

**Request Body:**
```json
{
  "name": "app_user",
  "password": "SecurePassword123!",
  "databases": [
    {
      "name": "myapp_database",
      "readonly": false
    }
  ]
}
```

### 4.5 List Users

**API Endpoint:**
```
GET /v3/{project_id}/instances/{instance_id}/db_user
```

### 4.6 Delete User

**API Endpoint:**
```
DELETE /v3/{project_id}/instances/{instance_id}/db_user/{user_name}
```

---

## 5. Parameter Operations

### 5.1 Modify Parameters

**API Endpoint:**
```
PUT /v3/{project_id}/instances/{instance_id}/parameters
```

**Request Body:**
```json
{
  "values": {
    "max_connections": "500",
    "innodb_buffer_pool_size": "2147483648",
    "wait_timeout": "600"
  }
}
```

### 5.2 Reset Parameters

**API Endpoint:**
```
POST /v3/{project_id}/instances/{instance_id}/parameters/default
```

**Request Body:**
```json
{
  "parameter_names": ["max_connections", "wait_timeout"]
}
```

### 5.3 Describe Parameters

**API Endpoint:**
```
GET /v3/{project_id}/instances/{instance_id}/parameters
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | String | No | Filter by parameter name |

---

## 6. Go SDK Usage Patterns

### 6.1 Client Initialization

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "rds" "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3"
    "rds_model" "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3/model"
)

func initRdsClient(region string) *rds.RdsClient {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    
    client := rds.RdsClientBuilder().
        WithEndpoint(fmt.Sprintf("rds.%s.myhuaweicloud.com", region)).
        WithCredential(basic.NewCredentialsBuilder().
            WithAk(ak).WithSk(sk).Build()).
        WithHttpConfig(config.DefaultHttpConfig()).
        Build()
    
    return client
}
```

### 6.2 Create Instance Pattern

```go
func createRdsInstance(client *rds.RdsClient, projectId string) (*rds_model.CreateInstanceResponse, error) {
    request := &rds_model.CreateInstanceRequest{
        Body: &rds_model.CreateInstanceRequestBody{
            Name: "rds-mysql-test",
            Datastore: &rds_model.Datastore{
                Type:    "MySQL",
                Version: "8.0",
            },
            FlavorRef: "rds.mysql.s1.large",
            VpcId:     "vpc-xxx",
            SubnetId:  "subnet-xxx",
            SecurityGroupId: "sg-xxx",
            AvailabilityZone: "cn-north-4a",
            Volume: &rds_model.Volume{
                Type: "ULTRAHIGH",
                Size: 100,
            },
            Password: "SecurePassword123!",
            Port:     "3306",
            Region:   "cn-north-4",
        },
    }
    
    response, err := client.CreateInstance(request)
    if err != nil {
        return nil, err
    }
    return response, nil
}
```

### 6.3 Polling Pattern

```go
func pollInstanceStatus(client *rds.RdsClient, instanceId string, targetStatus string, timeoutSec int) error {
    projectId := os.Getenv("HW_PROJECT_ID")
    startTime := time.Now()
    
    for {
        if time.Since(startTime) > time.Duration(timeoutSec)*time.Second {
            return fmt.Errorf("timeout waiting for status %s", targetStatus)
        }
        
        request := &rds_model.ShowInstanceRequest{
            InstanceId: instanceId,
        }
        
        response, err := client.ShowInstance(request)
        if err != nil {
            return err
        }
        
        status := *response.Instance.Status
        fmt.Printf("Current status: %s\n", status)
        
        if status == targetStatus {
            return nil
        }
        
        if status == "FAILED" {
            return fmt.Errorf("instance creation failed")
        }
        
        time.Sleep(30 * time.Second)
    }
}
```

### 6.4 Error Handling Pattern

```go
func handleRdsError(err error) string {
    if err == nil {
        return ""
    }
    
    // Check for specific error types
    if sdkErr, ok := err.(model.SdkError); ok {
        errorCode := sdkErr.ErrorCode()
        errorMsg := sdkErr.ErrorMessage()
        
        switch errorCode {
        case "DBS.0001":
            return fmt.Sprintf("Quota exceeded: %s", errorMsg)
        case "DBS.0002":
            return fmt.Sprintf("Invalid parameter: %s", errorMsg)
        case "DBS.0005":
            return fmt.Sprintf("Resource not found: %s", errorMsg)
        default:
            return fmt.Sprintf("RDS Error [%s]: %s", errorCode, errorMsg)
        }
    }
    
    return err.Error()
}
```

---

## 7. Pagination

### 7.1 Offset-Based Pagination

```go
func listAllInstances(client *rds.RdsClient, projectId string) ([]rds_model.InstanceResponse, error) {
    var allInstances []rds_model.InstanceResponse
    offset := 0
    limit := 100
    
    for {
        request := &rds_model.ListInstancesRequest{
            Limit:  &limit,
            Offset: &offset,
        }
        
        response, err := client.ListInstances(request)
        if err != nil {
            return nil, err
        }
        
        allInstances = append(allInstances, *response.Instances...)
        
        if len(*response.Instances) < limit {
            break // No more pages
        }
        
        offset += limit
    }
    
    return allInstances, nil
}
```

---

## 8. Error Handling

### 8.1 Common Error Codes

| Error Code | HTTP Status | Description | Recovery Action |
|------------|-------------|-------------|-----------------|
| DBS.0001 | 400 | Quota exceeded | HALT; request quota increase |
| DBS.0002 | 400 | Invalid parameter | Fix parameters; retry |
| DBS.0003 | 400 | Insufficient balance | HALT; recharge account |
| DBS.0004 | 400 | Resource already exists | Change name or use existing |
| DBS.0005 | 404 | Resource not found | Verify ID; retry |
| DBS.0006 | 403 | Permission denied | Check IAM permissions |
| DBS.0007 | 409 | Resource in use | Wait for completion; retry |
| DBS.0008 | 500 | Internal server error | Retry with backoff |
| DBS.0009 | 503 | Service unavailable | Retry with backoff |
| DBS.0010 | 429 | Rate limit exceeded | Retry with exponential backoff |

### 8.2 Retry Strategy

| Error Category | Retryable | Max Retries | Backoff Strategy |
|----------------|-----------|-------------|------------------|
| Throttling (429) | Yes | 3 | Exponential: 1s, 2s, 4s |
| Internal Error (5xx) | Yes | 3 | Exponential: 2s, 4s, 8s |
| Quota Exceeded | No | 0 | HALT |
| Invalid Parameter | No | 0 | Fix and retry |
| Not Found | No | 0 | Verify and retry |

---

*This document defines API and SDK usage patterns for RDS operations. Refer to official Huawei Cloud OpenAPI documentation for complete API specifications.*
