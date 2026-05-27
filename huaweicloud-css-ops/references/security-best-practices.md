# CSS Security Best Practices (SecOps)

## Identity and Access Management

### Minimum IAM Permissions

```json
{
  "Version": "1.1",
  "Statement": [
    {
      "Sid": "CSSReadOnly",
      "Effect": "Allow",
      "Action": [
        "css:cluster:get",
        "css:cluster:list",
        "css:snapshot:get",
        "css:snapshot:list",
        "css:dict:get",
        "css:dict:list",
        "css:config:get"
      ],
      "Resource": "*"
    },
    {
      "Sid": "CSSWrite",
      "Effect": "Allow",
      "Action": [
        "css:cluster:create",
        "css:cluster:delete",
        "css:cluster:resize",
        "css:cluster:restart",
        "css:snapshot:create",
        "css:snapshot:delete",
        "css:snapshot:restore",
        "css:dict:create",
        "css:dict:delete",
        "css:config:modify"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "vpc:id": "{{user.authorized_vpc_id}}"
        }
      }
    }
  ]
}
```

### Role-Based Access Control

| Role | Permissions | Use Case |
|------|-------------|----------|
| CSS Admin | Full access | Platform administrators |
| CSS Operator | Read + Create/Modify | Application teams |
| CSS ReadOnly | Read only | Monitoring/auditing |
| CSS Auditor | Read + Snapshot access | Compliance teams |

## Network Security

### VPC Isolation

```
┌─────────────────────────────────────────────────────────────┐
│                        VPC                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Private Subnet (CSS)                               │   │
│  │  ┌─────────────────────────────────────────────┐   │   │
│  │  │ CSS Cluster                                  │   │   │
│  │  │ - No public IP                               │   │   │
│  │  │ - Security group restricted                  │   │   │
│  │  └─────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────┘   │
│                            │                                │
│  ┌─────────────────────────┴─────────────────────────────┐ │
│  │  Public Subnet (Bastion/ALB)                          │ │
│  │  ┌─────────────────────────────────────────────┐     │ │
│  │  │ Bastion Host / Application Load Balancer     │     │ │
│  │  └─────────────────────────────────────────────┘     │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Security Group Rules

```yaml
security_group_rules:
  ingress:
    - protocol: tcp
      port: 9200
      source: vpc_cidr  # 172.16.0.0/16
      description: "Elasticsearch REST API"
    
    - protocol: tcp
      port: 9300
      source: vpc_cidr
      description: "Elasticsearch transport"
    
    - protocol: tcp
      port: 5601
      source: vpc_cidr
      description: "Kibana (if enabled)"
  
  egress:
    - protocol: all
      destination: 0.0.0.0/0
      description: "Allow all outbound"
```

### Network ACL Rules

```yaml
network_acl_rules:
  ingress:
    - rule: 100
      protocol: tcp
      port: 9200
      source: vpc_cidr
      action: allow
    
    - rule: 200
      protocol: tcp
      port: 9300
      source: vpc_cidr
      action: allow
    
    - rule: 999
      protocol: all
      source: 0.0.0.0/0
      action: deny
  
  egress:
    - rule: 100
      protocol: all
      destination: 0.0.0.0/0
      action: allow
```

## Data Encryption

### Encryption at Rest

```bash
# Create cluster with encryption
hcloud CSS CreateCluster \
  --name "secure-es-cluster" \
  --disk-encryption-enabled true \
  --disk-encryption-key "{{user.kms_key_id}}" \
  ...
```

**KMS Key Policy**:
```json
{
  "Version": "1.1",
  "Statement": [
    {
      "Sid": "Allow CSS Service",
      "Effect": "Allow",
      "Principal": {
        "Service": "css.myhuaweicloud.com"
      },
      "Action": [
        "kms:cmk:encrypt",
        "kms:cmk:decrypt",
        "kms:cmk:generateDataKey"
      ],
      "Resource": "{{user.kms_key_id}}"
    }
  ]
}
```

### Encryption in Transit

```yaml
cluster_config:
  https_enabled: true
  tls_version: "TLSv1.2+"
  certificate:
    type: "managed"  # or "custom"
    custom_cert: "{{user.cert_pem}}"
    custom_key: "{{user.key_pem}}"
```

**Client Connection**:
```bash
# Verify TLS
curl -v --cacert ca.pem https://cluster-endpoint:9200

# With client cert (if mTLS enabled)
curl --cert client.crt --key client.key https://cluster-endpoint:9200
```

## Snapshot Security

### OBS Bucket Security

```yaml
obs_bucket:
  name: "{{user.bucket_name}}"
  encryption:
    enabled: true
    algorithm: "AES256"
    kms_key_id: "{{user.kms_key_id}}"
  
  policy:
    Version: "2012-10-17"
    Statement:
      - Sid: "Allow CSS Service"
        Effect: "Allow"
        Principal:
          Service: "css.myhuaweicloud.com"
        Action:
          - "s3:PutObject"
          - "s3:GetObject"
          - "s3:DeleteObject"
        Resource: "arn:aws:s3:::{{user.bucket_name}}/css-snapshots/*"
  
  lifecycle:
    - prefix: "css-snapshots/"
      transitions:
        - days: 30
          storage_class: "WARM"
        - days: 90
          storage_class: "COLD"
```

## Access Control

### Elasticsearch Security Features

```yaml
elasticsearch_security:
  authentication:
    enabled: true
    method: "native"  # or "ldap", "saml"
  
  authorization:
    enabled: true
    roles:
      - name: "read_only"
        indices:
          - names: ["logs-*"]
            privileges: ["read", "view_index_metadata"]
      
      - name: "write_access"
        indices:
          - names: ["app-*"]
            privileges: ["read", "write", "create_index"]
      
      - name: "admin"
        cluster: ["all"]
        indices:
          - names: ["*"]
            privileges: ["all"]
```

### Password Policy

```yaml
password_policy:
  min_length: 16
  require_uppercase: true
  require_lowercase: true
  require_numbers: true
  require_special: true
  max_age_days: 90
  history_count: 5
```

## Audit Logging

### CTS Integration

```yaml
cts_config:
  enabled: true
  events:
    - css:cluster:create
    - css:cluster:delete
    - css:cluster:modify
    - css:snapshot:create
    - css:snapshot:restore
    - css:config:modify
  
  log_destination:
    type: "obs"
    bucket: "{{user.audit_bucket}}"
    prefix: "css-audit/"
    retention_days: 365
```

### Elasticsearch Audit Log

```yaml
elasticsearch_audit:
  enabled: true
  events:
    - authentication_success
    - authentication_failed
    - access_denied
    - connection_granted
    - connection_denied
  
  indices:
    - security_log
    - slow_log
    - deprecation_log
```

## Threat Detection

### Anomaly Detection Rules

```yaml
security_anomalies:
  - name: "Multiple Failed Logins"
    condition: |
      count(failed_login_events) > 5 
      AND time_window < 5m
    severity: high
    action: alert_and_block_ip
  
  - name: "Unusual Query Pattern"
    condition: |
      query_volume > baseline * 5
      AND query_complexity > threshold
    severity: medium
    action: alert
  
  - name: "Off-Hours Access"
    condition: |
      access_time NOT IN business_hours
      AND source_ip NOT IN whitelist
    severity: low
    action: log_and_alert
```

### Security Monitoring Checklist

- [ ] Enable CTS logging
- [ ] Enable Elasticsearch audit logs
- [ ] Configure failed login alerts
- [ ] Monitor for privilege escalation attempts
- [ ] Review access patterns weekly
- [ ] Rotate credentials quarterly
- [ ] Audit security group rules monthly
- [ ] Verify encryption settings

## Compliance

### Data Protection

| Requirement | Implementation | Verification |
|-------------|----------------|--------------|
| Encryption at rest | KMS encryption | `disk_encryption_enabled: true` |
| Encryption in transit | TLS 1.2+ | `https_enabled: true` |
| Access logging | CTS + ES audit | Log storage verified |
| Data retention | ILM policies | Snapshot retention configured |
| Data deletion | Secure wipe | Cluster deletion with data purge |

### Security Hardening Checklist

- [ ] HTTPS only (no HTTP)
- [ ] Disk encryption enabled
- [ ] Snapshot encryption enabled
- [ ] VPC isolated deployment
- [ ] Security group restricted
- [ ] IAM least privilege
- [ ] Strong passwords enforced
- [ ] Audit logging enabled
- [ ] Regular security patches
- [ ] Snapshot access controlled
