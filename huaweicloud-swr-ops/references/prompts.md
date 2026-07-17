# Prompts — Huawei Cloud SWR

> **Purpose:** Structured prompts for SWR (Software Repository for Container) AIOps operations.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Organization Health Check
```
Analyze SWR organization {{resource_id}} health status:
- Repository count: {{repo_count}}
- Total image count: {{image_count}}
- Storage used: {{storage_used}}GB / {{storage_quota}}GB
- Build tasks: {{build_count}} total, {{build_success}} success
- Organization members: {{member_count}}
- Access permissions: {{permission_count}}
Determine organization health and recommend actions.

Applicable CES metrics: SYS.SWR.repo_capacity, SYS.SWR.image_count, AGT.SWR.build_stats
```

### 1.2 Repository Usage Analysis
```
Analyze SWR repository usage on {{resource_id}}:
- Repository: {{repo_name}}
- Images: {{image_count}} (tags: {{tag_count}})
- Storage: {{storage_used}}GB
- Pull count: {{pull_count}}/day
- Last push: {{last_push_time}}
- Image size: {{avg_image_size}}GB
Identify optimization opportunities.

SWR storage: image layers, manifest, blob storage, tag overhead
```

### 1.3 Build Task Analysis
```
Analyze SWR build task on {{resource_id}}:
- Task ID: {{task_id}}
- Status: {{task_status (building/success/failed)}}
- Dockerfile: {{dockerfile_path}}
- Build duration: {{duration}} minutes
- Build steps: {{steps_completed}}/{{total_steps}}
- Image size: {{image_size}}MB
Diagnose build issues.

SWR build: Docker build, multi-stage build, build cache, layer efficiency
```

### 1.4 Image Vulnerability Analysis
```
Analyze SWR image vulnerability on {{resource_id}}:
- Image: {{image_name}}:{{tag}}
- Vulnerability scan: {{scan_status}} (passed/failed/not_scanned)
- Critical: {{critical_count}}, High: {{high_count}}, Medium: {{medium_count}}
- Base image: {{base_image}}
- Last scan: {{last_scan_time}}
Assess security risk.

SWR vulnerability: CVE database, base image age, layer scanning, image signing
```

### 1.5 Permission Analysis
```
Analyze SWR permissions on {{resource_id}}:
- Organization: {{org_name}}
- Repository: {{repo_name}}
- Global permissions: {{global_perms}}
- Repository permissions: {{repo_perms}}
- Service accounts: {{service_accounts}}
- OAuth apps: {{oauth_apps}}
Identify overpermissioned users.

SWR permissions: org admin, repo read/write, image delete, build permissions
```

---

## 2. Root Cause Analysis Prompts

### 2.1 Build Failure Analysis
```
Analyze SWR build failure on {{resource_id}}:
- Task ID: {{task_id}}
- Error code: {{error_code}}
- Error stage: {{error_stage}} (pull/resolve/build/push)
- Dockerfile: {{dockerfile}}
- Build context: {{build_context}}
- Layer cache: {{cache_status}} (hit/miss/disabled)
Diagnose build failure.

SWR build failures: syntax error, layer pull failure, OOM, network timeout, quota exceeded
```

### 2.2 Image Push Failure Analysis
```
Analyze SWR image push failure on {{resource_id}}:
- Image: {{image_name}}:{{tag}}
- Error: {{error_message}}
- Layer failed: {{failed_layer}}
- Layer size: {{layer_size}}MB
- Network: {{network_status}}
- Quota remaining: {{quota_remaining}}GB
Identify push issue.

SWR push failures: quota exceeded, network timeout, layer invalid, manifest rejected
```

### 2.3 Image Pull Failure Analysis
```
Analyze SWR image pull failure on {{resource_id}}:
- Image: {{image_name}}:{{tag}}
- Error: {{error_message}}
- Registry: {{registry_endpoint}}
- Auth: {{auth_status}}
- Network path: {{network_path}}
- Region: {{region}}
Diagnose pull issue.

SWR pull failures: auth expired, network policy, region restriction, image not found
```

### 2.4 Quota Exhaustion Analysis
```
Analyze SWR quota exhaustion on {{resource_id}}:
- Quota type: {{quota_type}} (storage/traffic/requests)
- Limit: {{quota_limit}}GB
- Used: {{quota_used}}GB ({{utilization}}%)
- Usage by repo: {{usage_by_repo}}
- Projected exhaustion: {{exhaustion_date}}
Assess quota risk.

SWR quotas: per-org storage, per-repo traffic, API rate limits
```

### 2.5 Vulnerability Scan Failure Analysis
```
Analyze SWR vulnerability scan failure on {{resource_id}}:
- Image: {{image_name}}:{{tag}}
- Scan status: {{scan_status}}
- Error: {{error_message}}
- Retry count: {{retry_count}}
- Last retry: {{last_retry_time}}
- Base image: {{base_image}}
Diagnose scan issue.

SWR scan failures: large image, unsupported base, scan service unavailable, timeout
```

---

## 3. Capacity Prompts

### 3.1 Storage Capacity Planning
```
Plan SWR storage capacity for {{resource_id}}:
- Current usage: {{storage_used}}GB / {{storage_quota}}GB
- Daily growth: {{daily_growth}}GB
- Image count growth: {{image_growth}}/month
- Average image size: {{avg_image_size}}GB
- Retention: {{retention_days}} days (auto-cleanup: {{auto_cleanup}})
- Projected exhaustion: {{exhaustion_date}}
Recommend storage optimization.

SWR storage: image layers, build cache, cleanup policies, storage tiering
```

### 3.2 Repository Organization
```
Optimize SWR repository organization for {{resource_id}}:
- Current repos: {{repo_count}}
- Images per repo: {{images_per_repo}} avg
- Unused repos: {{unused_repos}} (> 90 days no push)
- Recommended structure: {{recommended_structure}}
- Namespace usage: {{namespace_usage}}
Reduce management overhead.

SWR organization: per-team/per-service repos, namespace conventions, access patterns
```

### 3.3 Build Capacity Planning
```
Plan SWR build capacity for {{resource_id}}:
- Concurrent builds: {{concurrent_builds}} / {{max_builds}}
- Build queue: {{queue_depth}}
- Avg build time: {{avg_build_time}} minutes
- Build success rate: {{success_rate}}%
- Build cost: {{build_cost}} CNY/month
Recommend scaling.

SWR build: concurrent builds, build minutes, priority queues, cache optimization
```

### 3.4 Traffic Capacity Forecast
```
Forecast SWR traffic for {{resource_id}}:
- Current traffic: {{traffic_used}}GB / {{traffic_quota}}GB
- Daily traffic: {{daily_traffic}}GB
- Pull traffic: {{pull_traffic}}GB/day
- Push traffic: {{push_traffic}}GB/day
- CDN traffic: {{cdn_traffic}}GB
- Projected exhaustion: {{exhaustion_date}}
Recommend traffic optimization.

SWR traffic: image pulls, image pushes, CDN vs origin, regional distribution
```

---

## 4. Availability Prompts

### 4.1 Image Availability Check
```
Check SWR image availability on {{resource_id}}:
- Image: {{image_name}}:{{tag}}
- Region: {{region}}
- Replicas: {{replica_count}}
- Last synced: {{last_sync_time}}
- CDN coverage: {{cdn_coverage}}%
- Status: {{image_status}} (available/syncing/unavailable)
Assess availability.

SWR availability: multi-region replication, CDN caching, manifest distribution
```

### 4.2 Backup Status Verification
```
Verify SWR backup status for {{resource_id}}:
- Automatic backup: {{auto_backup_enabled}}
- Backup retention: {{retention_days}} days
- Last backup: {{last_backup_time}}
- Backup coverage: {{backup_coverage}}%
- Disaster recovery: {{dr_enabled}}
Validate data protection.

SWR backup: image layer redundancy, cross-region replication, disaster recovery
```

### 4.3 Access Control Audit
```
Audit SWR access control on {{resource_id}}:
- Org members: {{member_count}}
- External users: {{external_users}}
- Service accounts: {{service_accounts}}
- API tokens: {{api_tokens}}
- Expired permissions: {{expired_perms}}
- Overpermissioned: {{overpermed_users}}
Review access security.

SWR access: IAM integration, org roles, repo permissions, token management
```

### 4.4 SLA Compliance Report
```
Report SWR SLA compliance for {{resource_id}}:
- Image availability: {{availability}}% (target: {{target}}%)
- Build success rate: {{build_success_rate}}%
- Push success rate: {{push_success_rate}}%
- Pull latency P99: {{pull_latency_p99}}ms
- Vulnerability scan coverage: {{scan_coverage}}%
Report SLA violations.

SWR SLA: image availability, API success rate, build completion, scan coverage
```

---

## 5. Inspection Prompts

### 5.1 Routine Health Inspection
```
Perform routine SWR inspection:
- List all SWR organizations in {{scope}}
- Check storage utilization > {{threshold_util}}%
- Identify unused repos > {{unused_days}} days
- Flag vulnerable images (> 0 critical)
- Check build queue depth > {{queue_threshold}}
- Verify all critical images replicated
- Check for expired tokens
Report findings.

Scope: region, project, tag-based resource group
```

### 5.2 Security Compliance Check
```
Audit SWR security compliance:
- Image signing: {{image_signing}} (enabled/not enabled)
- Vulnerability scanning: {{vuln_scanning}}
- Access control: {{access_control}} (IAM/private/public)
- Service account keys: {{service_key_rotation}}
- External access: {{external_access}}
- Audit logging: {{audit_logging}}
Report compliance status.

Severity: Critical = public exposure, High = no vulnerability scan, Medium = no signing
```

### 5.3 Cost Optimization Scan
```
Scan SWR for cost optimization:
- Identify unused repos (no pulls > 60 days)
- Find large unused images
- Check image duplication across repos
- Verify build cache cleanup
- Review retention policies
- Check reserved vs pay-per-use
Provide action list with estimated savings.

SWR cost: storage, traffic, build minutes, API calls
```

### 5.4 Build Efficiency Audit
```
Audit SWR build efficiency:
- Success rate: {{success_rate}}%
- Failed builds: {{failed_builds}}/{{total_builds}}
- Cache hit rate: {{cache_hit_rate}}%
- Layer efficiency: {{layer_efficiency}}
- Multi-stage usage: {{multistage_usage}}%
- Build time distribution: {{build_time_dist}}
Recommend efficiency improvements.

SWR build: layer caching, multi-stage builds, build args optimization
```

### 5.5 Configuration Drift Detection
```
Detect configuration drift for SWR {{resource_id}}:
- Expected retention: {{expected_retention}} days
- Actual retention: {{actual_retention}} days
- Expected auto-cleanup: {{expected_cleanup}}
- Actual auto-cleanup: {{actual_cleanup}}
- Expected permission policy: {{expected_policy}}
- Actual permission policy: {{actual_policy}}
Recommend reconciliation.

SWR drift: retention policy, cleanup rules, permission changes
```

---

## Appendix: SWR-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{resource_id}}` | SWR organization ID | `org-abc123` |
| `{{repo_name}}` | Repository name | `my-app` |
| `{{image_name}}` | Image name | `myregistry.com/my-app` |
| `{{tag}}` | Image tag | `v1.2.3` |
| `{{task_id}}` | Build task ID | `build-12345` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance*
