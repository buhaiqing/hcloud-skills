# GaussDB Error Handling Reference

## HTTP Status Codes

| Code | Meaning | Common Scenarios |
|------|---------|-----------------|
| 200 | Success | All read/list operations |
| 202 | Accepted | Async operations (create, delete, resize) |
| 400 | Bad Request | Missing/invalid parameters, bad JSON body |
| 401 | Unauthorized | Expired AK/SK, wrong region token |
| 403 | Forbidden | IAM policy denies the operation |
| 404 | Not Found | Instance/backup/config ID doesn't exist |
| 409 | Conflict | Concurrent operation in progress |
| 500 | Internal Error | Server-side failure, retry safe |
| 503 | Service Unavailable | Regional service degradation |

## GaussDB Error Code Matrix

### Instance Management Errors

| HTTP | Service Code | Message Pattern | Root Cause | Fix |
|------|-------------|----------------|------------|-----|
| 400 | DBS.200001 | Invalid parameter value | Request body contains invalid field | Validate against API spec; check required fields |
| 400 | DBS.200010 | Instance status does not support this operation | Instance in CREATING/BACKING UP/FAULT | Wait for ACTIVE status |
| 400 | DBS.200012 | Cannot perform operation on a replica instance | Target is a read replica | Switch to primary instance_id |
| 400 | DBS.200013 | Flavor not available in current AZ | Requested spec code not supported | Run `ListFlavors()` for available specs |
| 403 | DBS.200301 | Insufficient instance quota | Account quota exhausted | Request quota increase via ticket |
| 403 | DBS.200302 | Insufficient storage quota | Storage quota exceeded | Request storage quota increase |
| 404 | DBS.200404 | Instance does not exist | Wrong instance_id or region | Verify instance_id with `ListInstances()` |
| 409 | DBS.200409 | Instance is being operated | Another operation in progress | Wait for completion; check `ListTasks()` |
| 500 | DBS.200500 | Internal server error | Temporary backend failure | Retry with exponential backoff (1s, 2s, 4s) |
| 500 | DBS.200501 | Flavor change failed | Backend resource insufficient | Contact support |

### Backup Errors

| HTTP | Service Code | Message Pattern | Root Cause | Fix |
|------|-------------|----------------|------------|-----|
| 400 | DBS.200601 | Backup name already exists | Duplicate backup name | Use unique naming with timestamp |
| 400 | DBS.200602 | Backup quota reached | Max 50 manual backups | Delete old backups via `DeleteManualBackup()` |
| 400 | DBS.200603 | Backup in progress | Concurrent backup running | Wait for completion before retry |
| 400 | DBS.200604 | Insufficient disk space for backup | Storage near full | Scale storage or delete old backups |
| 404 | DBS.200605 | Backup not found | Wrong backup_id | Verify backup_id with `ListBackups()` |

### Parameter Template Errors

| HTTP | Service Code | Message Pattern | Root Cause | Fix |
|------|-------------|----------------|------------|-----|
| 400 | DBS.200701 | Template already exists | Duplicate template name | Use unique name |
| 400 | DBS.200702 | Template parameter value out of range | Parameter exceeds allowed range | Check allowed values in `ShowConfigurationSetting()` |
| 400 | DBS.200703 | Template version mismatch | Template engine version != instance version | Use matching template version |
| 404 | DBS.200704 | Template not found | Wrong config_id | Verify with `ListConfigurations()` |

### Database & Account Errors

| HTTP | Service Code | Message Pattern | Root Cause | Fix |
|------|-------------|----------------|------------|-----|
| 400 | DBS.200801 | Database name already exists | Duplicate database name | Choose unique name |
| 400 | DBS.200802 | Database user already exists | Duplicate user name | Use different name or drop existing |
| 400 | DBS.200803 | Password does not meet complexity requirements | Weak password | Enforce 8+ chars, mixed case, digits, special chars |
| 400 | DBS.200804 | Cannot drop the last database user | Would leave instance with no admin | Create another admin user first |

## Diagnostic Workflow

```
1. Capture the error
   ↓
2. Identify HTTP status code + Service Code
   ↓
3. (If 4xx) Check parameters, permissions, quotas
   (If 5xx) Retry with backoff
   ↓
4. Verify instance status:
   hcloud GaussDB ShowInstanceDetail --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"
   ↓
5. Check for concurrent operations:
   hcloud GaussDB ListTasks --cli-query="tasks[?status=='Running']"
   ↓
6. If still blocked → escalate with full error payload to support
```

## Retry Strategy

```bash
# Exponential backoff for transient errors (500, 503)
max_retries=3
retry_delay=1
for i in $(seq 1 $max_retries); do
  output=$(hcloud GaussDB $OPERATION --cli-region="{{env.REGION}}" 2>&1)
  if echo "$output" | grep -q "500\|503"; then
    echo "Retry $i/$max_retries after ${retry_delay}s..."
    sleep $retry_delay
    retry_delay=$((retry_delay * 2))
  else
    echo "$output"
    break
  fi
done
```
