# CLI Usage — Huawei Cloud OBS

## CLI / obsutil Command Map

| Operation | hcloud obs Command | obsutil Command | Key Flags |
|-----------|-------------------|-----------------|-----------|
| Create Bucket | `hcloud obs create-bucket` | `obsutil mb obs://bucket` | `--bucket`, `--acl`, `--storage-class` |
| List Buckets | `hcloud obs list-buckets` | `obsutil ls -s` | — |
| Delete Bucket | `hcloud obs rb obs://bucket` | `obsutil rb obs://bucket` | `--bucket` |
| List Objects | `hcloud obs ls obs://bucket` | `obsutil ls obs://bucket` | `--prefix`, `--delimiter` |
| Upload Object | `hcloud obs cp local obs://bucket/key` | `obsutil cp local obs://bucket/key` | `--acl`, `--storage-class` |
| Download Object | `hcloud obs cp obs://bucket/key local` | `obsutil cp obs://bucket/key local` | `--threadNum` |
| Delete Object | `hcloud obs rm obs://bucket/key` | `obsutil rm obs://bucket/key` | `--bucket`, `--key` |
| Copy Object | `hcloud obs cp obs://src/key obs://dest/key` | `obsutil cp obs://src/key obs://dest/key` | `--source-endpoint` |
| Set ACL | `hcloud obs set-bucket-acl` | `obsutil set-acl obs://bucket acl` | `--acl` |
| Set Lifecycle | `hcloud obs set-bucket-lifecycle` | `obsutil set-lifecycle obs://bucket` | `--lifecycle-file` |
| Set Versioning | `hcloud obs set-bucket-versioning` | `obsutil set-versioning obs://bucket` | `--status` |
| Set Website | `hcloud obs set-bucket-website` | `obsutil set-website obs://bucket` | `--index`, `--error` |
| Set CORS | `hcloud obs set-bucket-cors` | `obsutil set-cors obs://bucket` | `--cors-file` |
| Generate Presigned URL | `hcloud obs sign obs://bucket/key` | `obsutil sign obs://bucket/key` | `-ef` (expires in seconds) |
| Sync Directory | — | `obsutil sync local/ obs://bucket/` | `-r`, `-f`, `-threadNum` |
| Stat Object | — | `obsutil stat obs://bucket/key` | `--bucket`, `--key` |

## Coverage Gap Table

| Operation | hcloud obs | obsutil | Fallback |
|-----------|-----------|---------|----------|
| Bucket CRUD | ✅ | ✅ | — |
| Object upload/download | ✅ | ✅ | — |
| Multipart upload | ⚠️ Basic | ✅ Full (parallel, resume) | JIT Go SDK |
| Lifecycle rules | ✅ (JSON file) | ✅ (JSON file) | — |
| Versioning | ✅ | ✅ | — |
| CORS | ✅ | ✅ | — |
| Bucket Policy | ✅ | ✅ | — |
| Static Website | ✅ | ✅ | — |
| Cross-region replication | ⚠️ Partial | ⚠️ Partial | JIT Go SDK |
| Bucket logging | ⚠️ | ✅ | JIT Go SDK |
| Presigned URL | ✅ | ✅ | — |
| Large file sync | ⚠️ | ✅ (sync, resume) | JIT Go SDK |
| Object tagging | ⚠️ | ✅ | JIT Go SDK |
| Restore archive object | ⚠️ | ✅ | JIT Go SDK |

## Common Invocation Patterns

```bash
# Configure obsutil credentials once
obsutil config -i=$HW_ACCESS_KEY_ID -k=$HW_SECRET_ACCESS_KEY -e=$HW_ENDPOINT

# List all buckets
obsutil ls -s

# List objects with prefix
obsutil ls obs://my-bucket/logs/ -r

# Upload large file with multipart (10MB parts, 10 threads)
obsutil cp ./data.tar.gz obs://my-bucket/backups/data.tar.gz \
  -f -partSize 10m -threadNum 10

# Download entire directory
obsutil sync obs://my-bucket/data/ ./local-data/ -r -threadNum 5

# Set bucket to private
obsutil set-acl obs://my-bucket private

# Generate presigned URL (valid for 1 hour)
obsutil sign obs://my-bucket/secret.pdf -ef=3600

# Set lifecycle rule
obsutil set-lifecycle obs://my-bucket lifecycle.json

# Delete all versions of an object (permanent)
obsutil rm obs://my-bucket/key --versionId=xxx
```

## Bucket Policy Example

```json
{
  "Statement": [
    {
      "Sid": "AllowReadFromApp",
      "Effect": "Allow",
      "Principal": {"ID": ["user-xxx"]},
      "Action": ["GetObject", "ListBucket"],
      "Resource": ["my-bucket", "my-bucket/*"]
    },
    {
      "Sid": "DenyInsecureAccess",
      "Effect": "Deny",
      "Principal": {"ID": ["*"]},
      "Action": ["*"],
      "Resource": ["my-bucket/*"],
      "Condition": {"Bool": {"obs:SecureTransport": "false"}}
    }
  ]
}
```

## CLI Error Handling

- obsutil returns **exit code 0** on success, **non-zero** on failure
- Error messages include OBS error code (e.g., `NoSuchKey`, `AccessDenied`)
- Map error codes to troubleshooting entries in `references/troubleshooting.md`
