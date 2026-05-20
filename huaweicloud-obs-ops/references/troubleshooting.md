# Troubleshooting Guide — Huawei Cloud OBS

## Error Code Taxonomy

| Error Code | HTTP Status | Description | Retryable | Max Retries | Agent Action | User Guidance |
|-----------|------------|-------------|-----------|------------|--------------|---------------|
| NoSuchBucket | 404 | Bucket does not exist | No | 0 | Verify bucket name/region | `[ERROR] Bucket not found. Check bucket name: {{user.bucket_name}}` |
| NoSuchKey | 404 | Object does not exist | No | 0 | Verify object key exists | `[ERROR] Object not found. Check key: {{user.object_key}}` |
| AccessDenied | 403 | Permission denied | No | 0 | Check ACL, bucket policy, IAM | `[ERROR] Access denied. Check bucket ACL, policy, and IAM permissions.` |
| InvalidAccessKeyId | 403 | AK not found or disabled | No | 0 | HALT; verify AK | `[ERROR] Invalid AK. Check HW_ACCESS_KEY_ID.` |
| SignatureDoesNotMatch | 403 | Signature verification failed | No | 0 | HALT; verify SK | `[ERROR] Signature mismatch. Check HW_SECRET_ACCESS_KEY.` |
| InvalidSecretAccessKey | 403 | SK is invalid | No | 0 | HALT; verify SK | `[ERROR] Invalid SK. Check HW_SECRET_ACCESS_KEY.` |
| BucketAlreadyExists | 409 | Bucket name taken globally | No | 0 | Use different name | `[ERROR] Bucket name globally taken. Choose unique name.` |
| EntityTooLarge | 400 | Object exceeds size limit | No | 0 | Use multipart upload | `[ERROR] Object too large for single PUT. Use multipart for files > 5GB.` |
| InvalidBucketName | 400 | Bucket name format invalid | No | 0 | Fix name | `[ERROR] Invalid bucket name. 3-63 chars, lowercase letters/digits/hyphens.` |
| InvalidPart | 400 | Multipart part error | No | 0 | Verify part number/order | `[ERROR] Invalid multipart part. Check part number and uploadId.` |
| PartNumberInOrder | 400 | Parts must be uploaded sequentially | No | 0 | Fix part sequence | `[ERROR] Part number out of order. Upload parts 1, 2, 3... sequentially.` |
| RequestTimeout | 408 | Request timed out | Yes | 3 | Retry with larger timeout | `[ERROR] Request timeout. Check network and retry.` |
| SlowDown | 503 | Rate limit exceeded | Yes | 3 | Exponential backoff | `[ERROR] OBS rate limited. Retrying in {backoff}s...` |
| InternalError | 500 | OBS server error | Yes | 3 | Retry; then HALT | `[ERROR] OBS internal error. Retry or escalate with RequestId.` |

## Ordered Diagnostic Steps

### 1. 403 Access Denied After Policy Change

```
Step 1: Check bucket ACL
  obsutil get-acl obs://bucket
  → Should match expected access level

Step 2: Check bucket policy
  obsutil get-policy obs://bucket
  → Look for Deny statements or missing Allow

Step 3: Check IAM permissions
  List user policies for the AK being used
  → Verify obs:*Get* or obs:*Put* permissions

Step 4: Check if presigned URL expired
  If using signed URLs, verify expiry time
  → Generate new presigned URL

Step 5: Check VPC Endpoint whitelist
  If accessing via VPC Endpoint, verify IP is allowed
```

### 2. Upload Fails for Large Files

```
Step 1: Check file size
  ls -lh local-file
  → If > 5GB, single PUT will fail

Step 2: Use multipart upload
  obsutil cp local-file obs://bucket/key -f -partSize 10m -threadNum 10

Step 3: If multipart fails
  → Check if incomplete uploads exist: obsutil listuploads obs://bucket
  → Abort stale uploads: obsutil abortupload obs://bucket uploadId
  → Retry multipart

Step 4: Check bandwidth and network stability
  → Reduce threadNum if network is unstable
```

### 3. Lifecycle Rule Not Triggering

```
Step 1: Verify lifecycle rule is enabled
  obsutil get-lifecycle obs://bucket
  → Check Status = "Enabled"

Step 2: Check object creation date
  → Lifecycle applies based on object age (from creation date)
  → Objects must be older than specified Days

Step 3: Verify prefix match
  → Rule prefix must match object key
  → If prefix is "", rule applies to all objects

Step 4: Check storage class transition compatibility
  → Standard → WARM (IA) → COLD → Deep Cold
  → Cannot skip tiers
```

### 4. Cross-Region Replication Lag

```
Step 1: Check CRR rule status
  obsutil get-replication obs://bucket
  → Check Status = "Enabled"

Step 2: Check replication backlog
  → Monitor source bucket object count vs destination
  → New objects replicate within minutes (depends on size)

Step 3: Verify source and destination buckets
  → Source must have versioning enabled
  → Destination must exist in target region
```

### 5. CDN Cache Stale

```
Step 1: Check CDN cache status
  → Access object URL through CDN
  → Compare with direct OBS URL (bypass CDN)

Step 2: Check Cache-Control headers
  obsutil stat obs://bucket/key
  → Check Cache-Control header value
  → If max-age is large, CDN will serve stale

Step 3: Purge CDN cache
  → Use CDN skill to purge specific URL
  → Set appropriate Cache-Control for content type
```

### 6. Presigned URL Expired

```
Step 1: Check URL generation timestamp
  → Presigned URLs have max 7-day validity

Step 2: Regenerate with appropriate expiry
  obsutil sign obs://bucket/key -ef=3600 (1 hour)
  or obsutil sign obs://bucket/key -ef=604800 (7 days max)

Step 3: If long-term access needed
  → Use bucket policy with IP conditions
  → Or use CDN with token authentication
```

## Multi-Round Diagnosis Flow: Object Not Found

```
Round 1: Basic Verification
  1. obsutil ls obs://bucket/ → Is bucket accessible?
  2. obsutil ls obs://bucket/key → Does object exist?
  3. Check object key spelling (case-sensitive!)
  → If not found, proceed to Round 2.

Round 2: Versioning Check
  4. Is versioning enabled on bucket?
  5. obsutil listversions obs://bucket/key → Check all versions
  6. Is there a delete marker? If yes, the object was "deleted"
  → If a version exists, restore it. If not, proceed to Round 3.

Round 3: Lifecycle/Expiration
  7. Check lifecycle rules → Was object expired?
  8. Check if object was in Warm/Cold/DeepCold class → Was it deleted after retention?
  → If lifecycle caused deletion, object is permanently lost.

Round 4: Escalation
  9. Collect: bucket name, key, timestamp, AK ID (not value), operation attempted
  10. Submit ticket if object should exist but doesn't
```

## Common OBS Operational Issues

### Public Bucket Exposure
- **Symptom**: Sensitive data accessible without authentication
- **Detection**: Bucket ACL = public-read or public-read-write
- **Prevention**: Default to private ACL, audit bucket policies regularly

### Multipart Upload Stuck
- **Symptom**: Partial uploads consuming storage, incomplete objects
- **Detection**: Incomplete multipart uploads visible via ListMultipartUploads
- **Resolution**: Abort incomplete uploads, configure lifecycle rule to auto-abort after N days

### Egress Cost Surge
- **Symptom**: Unexpected high billing from outbound data transfer
- **Detection**: CES metrics show outbound traffic spike
- **Mitigation**: Use CDN (cheaper egress), optimize data transfer patterns

### Object Key Naming Performance Issue
- **Symptom**: Slow LIST operations on bucket with millions of objects
- **Cause**: Sequential naming creates hot partitions (e.g., `img-000001`, `img-000002`)
- **Fix**: Use hash prefix: `{hash(object_id)}/img-{id}` for even distribution
