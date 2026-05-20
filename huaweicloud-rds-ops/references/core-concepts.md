# Core Concepts — Huawei Cloud RDS

> **Purpose:** Defines RDS architecture, limits, regions, quotas, and resource relationships.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [Service Overview](#1-service-overview)
2. [Architecture](#2-architecture)
3. [Supported Database Engines](#3-supported-database-engines)
4. [Instance Types](#4-instance-types)
5. [Regions and Availability Zones](#5-regions-and-availability-zones)
6. [Quotas and Limits](#6-quotas-and-limits)
7. [Resource Relationships](#7-resource-relationships)
8. [SPOF Analysis](#8-spof-analysis)

---

## 1. Service Overview

Huawei Cloud Relational Database Service (RDS) is a reliable, scalable, and manageable online relational database service running on Huawei Cloud. It provides comprehensive performance monitoring, multi-level security protection, and professional database management capabilities.

### Key Features

| Feature | Description |
|---------|-------------|
| Multi-Engine Support | MySQL, PostgreSQL, SQL Server |
| High Availability | Primary/Standby deployment across AZs |
| Automatic Backup | Automated daily backups with configurable retention |
| Monitoring & Alerts | Integration with Cloud Eye Service (CES) |
| Security | VPC isolation, security groups, TDE encryption |
| Scalability | Online storage and compute scaling |

---

## 2. Architecture

### 2.1 Single Instance Architecture

```
┌─────────────────────────────────────┐
│           Application Layer           │
└───────────────┬───────────────────────┘
                │
┌───────────────▼───────────────────────┐
│        Security Group (Port 3306)       │
└───────────────┬───────────────────────┘
                │
┌───────────────▼───────────────────────┐
│        Subnet (Private Network)       │
└───────────────┬───────────────────────┘
                │
┌───────────────▼───────────────────────┐
│              RDS Instance             │
│  ┌─────────────────────────────┐    │
│  │    Database Engine (MySQL)  │    │
│  │  ┌─────────┐  ┌─────────┐   │    │
│  │  │   DB1   │  │   DB2   │   │    │
│  │  └─────────┘  └─────────┘   │    │
│  └─────────────────────────────┘    │
└─────────────────────────────────────┘
```

### 2.2 Primary/Standby Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     VPC Network                             │
│  ┌──────────────────┐                ┌──────────────────┐  │
│  │   Primary Node   │◄──────────────►│   Standby Node   │  │
│  │  (Active)        │  Replication   │  (Standby)       │  │
│  │  cn-north-4a     │                │  cn-north-4b     │  │
│  └──────────────────┘                └──────────────────┘  │
│           │                                    │            │
│           └──────────────┬─────────────────────┘            │
│                          ▼                                  │
│                  ┌──────────────┐                          │
│                  │  Shared Disk  │                          │
│                  └──────────────┘                          │
└─────────────────────────────────────────────────────────────┘
```

### 2.3 Read Replica Architecture

```
┌──────────────────┐
│  Primary Node    │
│   (Read/Write)   │
└────────┬─────────┘
         │ Replication
    ┌────┴────┬────────┐
    ▼         ▼        ▼
┌───────┐ ┌───────┐ ┌───────┐
│Replica│ │Replica│ │Replica│
│ (RO)  │ │ (RO)  │ │ (RO)  │
└───────┘ └───────┘ └───────┘
```

---

## 3. Supported Database Engines

### 3.1 MySQL

| Version | Supported | Notes |
|---------|-----------|-------|
| 5.6 | Deprecated | Migration to 5.7+ recommended |
| 5.7 | ✅ | Primary support version |
| 8.0 | ✅ | Recommended for new deployments |

### 3.2 PostgreSQL

| Version | Supported | Notes |
|---------|-----------|-------|
| 9.5 | Deprecated | Migration recommended |
| 10 | ✅ | Primary support version |
| 11 | ✅ | Recommended for new deployments |
| 12 | ✅ | Recommended for new deployments |
| 13 | ✅ | Latest stable version |

### 3.3 SQL Server

| Version | Supported | Notes |
|---------|-----------|-------|
| 2014 | Deprecated | Limited support |
| 2016 | ✅ | Primary support version |
| 2017 | ✅ | Recommended |
| 2019 | ✅ | Latest stable version |

---

## 4. Instance Types

### 4.1 MySQL Instance Flavors

| Flavor Code | vCPU | Memory (GB) | Max Connections | Max IOPS |
|-------------|------|-------------|-----------------|----------|
| rds.mysql.s1.medium | 1 | 2 | 300 | 2,000 |
| rds.mysql.s1.large | 2 | 4 | 600 | 4,000 |
| rds.mysql.s1.xlarge | 4 | 8 | 1,200 | 6,000 |
| rds.mysql.m1.2xlarge | 8 | 16 | 2,400 | 12,000 |
| rds.mysql.m1.4xlarge | 16 | 32 | 4,800 | 24,000 |
| rds.mysql.c1.8xlarge | 32 | 64 | 9,600 | 48,000 |

### 4.2 PostgreSQL Instance Flavors

| Flavor Code | vCPU | Memory (GB) | Max Connections | Max IOPS |
|-------------|------|-------------|-----------------|----------|
| rds.pg.s1.medium | 1 | 2 | 200 | 2,000 |
| rds.pg.s1.large | 2 | 4 | 400 | 4,000 |
| rds.pg.s1.xlarge | 4 | 8 | 800 | 6,000 |
| rds.pg.m1.2xlarge | 8 | 16 | 1,600 | 12,000 |
| rds.pg.m1.4xlarge | 16 | 32 | 3,200 | 24,000 |

### 4.3 Storage Types

| Type | Description | Use Case |
|------|-------------|----------|
| ULTRAHIGH | SSD cloud disk | General purpose, balanced performance |
| ULTRAHIGHPRO | Provisioned IOPS SSD | High I/O, low latency requirements |
| ESSD | Enhanced SSD | Ultra-low latency, highest IOPS |

---

## 5. Regions and Availability Zones

### 5.1 Supported Regions

| Region | Region ID | Description |
|--------|-----------|-------------|
| 华北-北京四 | cn-north-4 | Beijing Region |
| 华北-北京一 | cn-north-1 | Beijing Region (Original) |
| 华东-上海二 | cn-east-2 | Shanghai Region |
| 华东-上海一 | cn-east-3 | Shanghai Region |
| 华南-广州 | cn-south-1 | Guangzhou Region |
| 西南-贵阳一 | cn-southwest-2 | Guiyang Region |
| 中国-香港 | ap-southeast-1 | Hong Kong Region |
| 亚太-曼谷 | ap-southeast-2 | Bangkok Region |
| 亚太-新加坡 | ap-southeast-3 | Singapore Region |

### 5.2 Availability Zones

- Each region has 2-3 availability zones
- AZs are physically isolated within the same region
- Primary/Standby deployments should span multiple AZs
- Cross-region replication requires separate instances

---

## 6. Quotas and Limits

### 6.1 Instance Quotas

| Resource | Default Quota | Maximum |
|----------|---------------|---------|
| Instances per region | 50 | 500 (request increase) |
| Read replicas per instance | 10 | 20 (request increase) |
| Manual backups per instance | 7 | 50 |
| Automated backups retention | 7 days | 35 days |

### 6.2 Storage Limits

| Resource | Minimum | Maximum |
|----------|---------|---------|
| Storage size | 40 GB | 4,000 GB (MySQL/PostgreSQL) |
| Storage size (SQL Server) | 100 GB | 2,000 GB |
| Storage scaling increment | 10 GB | N/A |

### 6.3 Connection Limits

| Engine | Minimum | Maximum |
|--------|---------|---------|
| MySQL | 100 | 10,000 (depends on flavor) |
| PostgreSQL | 100 | 5,000 (depends on flavor) |
| SQL Server | 50 | 32,767 |

### 6.4 API Rate Limits

| Operation | Rate Limit |
|-----------|------------|
| CreateInstance | 10/minute |
| DeleteInstance | 10/minute |
| DescribeInstance | 100/minute |
| ListInstances | 50/minute |
| CreateBackup | 5/minute |

---

## 7. Resource Relationships

### 7.1 Dependency Graph

```
RDS Instance
    │
    ├─► VPC (Required)
    │   └─► Subnet (Required)
    │
    ├─► Security Group (Required)
    │
    ├─► Parameter Group (Optional)
    │
    ├─► Backup
    │   ├─► Manual Backup
    │   └─► Automated Backup
    │
    ├─► Database (Internal)
    │   └─► User
    │
    └─► Read Replica (Optional)
```

### 7.2 Creation Order

1. **VPC** - Must exist first
2. **Subnet** - Must exist within VPC
3. **Security Group** - Should allow database port access
4. **RDS Instance** - Created last, references above

### 7.3 Deletion Order

1. **Read Replicas** - Delete first
2. **Manual Backups** - Delete or retain
3. **Automated Backups** - Auto-deleted with instance
4. **RDS Instance** - Can be deleted
5. **Security Group** - Can be deleted if unused

---

## 8. SPOF Analysis

### 8.1 Single Points of Failure

| Component | Risk Level | Mitigation |
|-----------|------------|------------|
| Single-AZ deployment | **HIGH** | Use Primary/Standby across AZs |
| Single manual backup | **MEDIUM** | Multiple manual backups + automated backups |
| Single connection endpoint | **LOW** | Use read replicas for read scaling |
| Single security group | **LOW** | Multiple security groups for different access patterns |

### 8.2 HA Recommendations

| Scenario | Recommendation |
|----------|----------------|
| Production | Primary/Standby across AZs |
| Critical Production | Primary/Standby + Cross-region read replica |
| Development | Single instance acceptable |
| Testing | Single instance acceptable |

### 8.3 Disaster Recovery

| DR Level | RTO | RPO | Implementation |
|----------|-----|-----|----------------|
| AZ-level | 1-5 min | 0 | Primary/Standby across AZs |
| Region-level | 30-60 min | 5-10 min | Cross-region read replica |
| Point-in-time | 15-60 min | 5 min | Automated backups + restore |

---

*This document defines the core architectural concepts for RDS operations. Refer to official Huawei Cloud documentation for the latest limits and specifications.*
