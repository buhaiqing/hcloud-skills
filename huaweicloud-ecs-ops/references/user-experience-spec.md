# ECS User Experience Specification

> Mandatory UX requirements for the ECS ops skill. Mirrors the Huawei Cloud
> Skill Generator UX spec, applied to ECS lifecycle and remote-execution flows.

## Quick Start Flow

### First-Time User (5-Minute Onboarding)

**Step 1**: Verify CLI Setup (60s)
```bash
hcloud version
```

**Step 2**: Configure Credentials (60s)
```bash
export HW_ACCESS_KEY_ID="your-ak"
export HW_SECRET_ACCESS_KEY="your-sk"
export HW_REGION_ID="cn-north-4"
export HW_PROJECT_ID="your-project-id"
```

**Step 3**: First Command (60s)
```bash
hcloud ecs list-servers --region "$HW_REGION_ID"
```

**Step 4**: Verify a Specific Server (120s)
```bash
hcloud ecs show-server --server-id "$SERVER_ID"
```

## Interaction Design

### Prompt Strategy

| Scenario | Prompt Count | Information Strategy |
|----------|--------------|---------------------|
| List servers | 0 | No prompts needed |
| Show server | 1 | Ask for `server_id` if not provided |
| Create server | 5–7 | name, image, flavor, VPC/subnet, SG, keypair, password |
| Start / stop / reboot | 2 | `server_id` + explicit confirmation for stop/reboot |
| Resize | 2 | `server_id` + target flavor (with `list-flavors` suggestions) |
| Delete | 2 | `server_id` + explicit confirmation (irreversible) |
| CloudShell exec | 2 | `server_id` + command |
| File transfer | 3 | `server_id`, source, destination |

### Smart Defaults

```yaml
defaults:
  flavor: "s3.medium.4"        # 1 vCPU / 4 GB — safe baseline
  image_os: "CentOS 7.9 64bit"
  disk_type: "SSD"
  disk_size_gb: 40
  network:
    vpc_strategy: "default-vpc"  # auto-pick the first available VPC
    subnet_strategy: "default-subnet"
    security_group_strategy: "default-sg"
  charging_mode: "postPaid"      # always confirm before switching to prePaid
```

### Error Message Format

```
[ERROR] {error_code}: {message}

Context:
- Server: {server_name} ({server_id})
- Operation: {operation}
- Region: {region}
- Time: {timestamp}

Suggested Fix:
{recovery_steps}

Documentation:
{relevant_doc_link}
```

### Progress Indicators

**Long-Running Operations** (create / resize / restart):
```
⏳ Creating ECS "web-01"...
   Step 1/4: Allocating resources...     ✓ (3s)
   Step 2/4: Booting from image...       ▶ (30s elapsed)
   Step 3/4: Configuring network...      
   Step 4/4: Health checks...            
```

**CloudShell Remote Execution**:
```
🔌 Connecting to {server_id} via CloudShell...    ✓
⌨️  Executing: {command}                           ▶
📤 Output: {output_excerpt}
```

## Success Criteria

### Time-to-Value

| Task | Target Time | Maximum Time |
|------|-------------|--------------|
| First command execution | 2 min | 5 min |
| Server creation | 3 min | 8 min |
| Server start / stop | 30 s | 90 s |
| Resize | 5 min | 15 min |
| Delete | 30 s | 90 s |
| CloudShell exec | 5 s | 30 s |

### Error Recovery

| Error Type | Recovery Time Target |
|------------|---------------------|
| Parameter validation | < 30 s (re-prompt) |
| Quota exceeded | < 5 min (HALT) |
| Auth / IAM error | < 1 min (HALT, show missing permission) |
| Network / 5xx | < 1 min (auto-retry 3× with backoff) |
| Service unavailable | < 10 min (escalate) |

## Accessibility

### CLI Help

```bash
hcloud ecs --help
hcloud ecs create-server --help
hcloud ecs create-server --examples
```

### Output Formats

```bash
hcloud ecs list-servers                          # human-readable table
hcloud ecs list-servers -o json | jq '.[].id'    # machine-readable
hcloud ecs show-server --server-id "$ID" -o yaml # full detail
```

## Feedback Mechanisms

### Operation Confirmation

**Destructive Operations** (stop / reboot / resize / delete):
```
⚠️  You are about to STOP server "web-01" (i-xxx).
   This will INTERRUPT running services and disconnect SSH sessions.

   To confirm, type: STOP web-01
   To cancel, type: cancel

>
```

**Irreversible Operations** (delete):
```
🔥 You are about to DELETE server "web-01" (i-xxx).
   This action is IRREVERSIBLE. All attached disks (except those with
   delete_on_termination=false) and the server itself will be removed.

   To confirm, type: DELETE web-01
   To cancel, type: cancel

>
```

**Cost-Sensitive Operations** (prePaid resize / yearly conversion):
```
💰 Resizing "web-01" to s3.large.4 will change charging from postPaid
   to prePaid with an estimated cost of ¥250/month.

   Continue? [y/N]:
```

## Documentation Integration

### Contextual Help Links

Every error includes a doc link:
```
[ERROR] ECS.0101: Quota exceeded for ECS instances

Learn more: https://support.huaweicloud.com/api-ecs/ecs_01_0147.html
```

### Example Command Suggestions

After listing servers, suggest next actions:
```
Your ECS instances:
  1. web-01   (ACTIVE,   cn-north-4a)
  2. db-01    (SHUTOFF,  cn-north-4b)
  3. test-01  (ACTIVE,   cn-north-4a)

Next, you can:
  • View details:  hcloud ecs show-server --server-id web-01
  • Start:         hcloud ecs start-server --server-id db-01
  • Diagnose:      delegate to huaweicloud-ces-ops for metrics
  • Resize:        hcloud ecs resize-server --server-id web-01 --flavor s3.large.4
```

## CloudShell-Specific UX

### Command Echo Policy

- **NEVER** echo `HW_SECRET_ACCESS_KEY` or any value derived from it
- **DO** echo the `server_id` and the literal command for traceability
- Mask any string matching `AK[0-9A-Z]{16,}` or `***` pattern in logs

### File Transfer Confirmation

```
📁 Uploading /tmp/app.log (2.3 MB) to web-01:/var/log/app.log...   ✓
📁 Downloading /var/log/nginx/access.log (5.1 MB) to ./access.log... ✓
```

## References

- [Core Concepts](core-concepts.md) — ECS architecture and resource model
- [Troubleshooting](troubleshooting.md) — Common ECS failure modes
- [AIOps Best Practices](advanced/aiops-best-practices.md) — Anomaly patterns
