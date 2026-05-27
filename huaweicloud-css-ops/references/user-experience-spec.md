# CSS User Experience Specification

## Quick Start Flow

### First-Time User (5-Minute Onboarding)

**Step 1**: Verify CLI Setup (60s)
```bash
hcloud version  # Should show version
```

**Step 2**: Configure Credentials (60s)
```bash
export HW_ACCESS_KEY_ID="your-ak"
export HW_SECRET_ACCESS_KEY="your-sk"
export HW_REGION_ID="cn-north-4"
```

**Step 3**: First Command (60s)
```bash
hcloud CSS ListClusters --region cn-north-4
```
Expected output: List of clusters (empty if first time)

**Step 4**: Create First Cluster (180s)
```bash
hcloud CSS CreateCluster --name "my-first-cluster" ...
```

## Interaction Design

### Prompt Strategy

| Scenario | Prompt Count | Information Strategy |
|----------|--------------|---------------------|
| List clusters | 0 | No prompts needed |
| Show cluster | 1 | Ask for cluster_id if not provided |
| Create cluster | 3-5 | Name, version, node count, VPC (with defaults) |
| Delete cluster | 2 | cluster_id + explicit confirmation |
| Create snapshot | 2 | cluster_id + snapshot_name |

### Smart Defaults

```yaml
defaults:
  engine_version: "7.10.2"
  node_count: 3
  node_flavor: "ess.spec-4u8g"
  storage_size: 100
  storage_type: "ULTRAHIGH"
  https_enabled: true
  encryption_enabled: true
```

### Error Message Format

```
[ERROR] {error_code}: {message}

Context:
- Cluster: {cluster_name} ({cluster_id})
- Operation: {operation}
- Time: {timestamp}

Suggested Fix:
{recovery_steps}

Documentation:
{relevant_doc_link}
```

### Progress Indicators

**Long-Running Operations**:
```
⏳ Creating cluster "prod-es"... 
   Step 1/5: Provisioning VPC resources... ✓
   Step 2/5: Creating ECS instances... ✓
   Step 3/5: Installing Elasticsearch... ▶ (2m elapsed)
   Step 4/5: Configuring security... 
   Step 5/5: Health checks... 
```

## Success Criteria

### Time-to-Value

| Task | Target Time | Maximum Time |
|------|-------------|--------------|
| First command execution | 2 min | 5 min |
| Cluster creation | 10 min | 20 min |
| Snapshot creation | 2 min | 5 min |
| Query execution | < 1s | < 5s |

### Error Recovery

| Error Type | Recovery Time Target |
|------------|---------------------|
| Configuration error | < 2 min |
| Quota exceeded | < 5 min |
| Network error | < 1 min (auto-retry) |
| Service error | < 10 min (escalation) |

## Accessibility

### CLI Help

```bash
# Contextual help
hcloud CSS --help
hcloud CSS CreateCluster --help
hcloud CSS CreateCluster --examples
```

### Output Formats

```bash
# Human-readable (default)
hcloud CSS ListClusters

# JSON for scripting
hcloud CSS ListClusters -o json

# Table for quick view
hcloud CSS ListClusters -o table

# YAML for configuration
hcloud CSS ShowClusterDetail -o yaml
```

## Feedback Mechanisms

### Operation Confirmation

**Destructive Operations**:
```
⚠️  You are about to DELETE cluster "prod-es" (cluster-xxx).
   This action is IRREVERSIBLE and will delete all data.

   To confirm, type: DELETE prod-es
   To cancel, type: cancel

> 
```

**Expensive Operations**:
```
💰 Estimated cost for this operation: ¥500/month
   Continue? [y/N]: 
```

## Documentation Integration

### Contextual Help Links

Every error message includes relevant documentation link:
```
[ERROR] CSS.0010: Invalid VPC configuration

Learn more: https://support.huaweicloud.com/api-css/css_03_0010.html
```

### Example Command Suggestions

After listing clusters, suggest next actions:
```
Your clusters:
  1. prod-es (AVAILABLE)
  2. dev-es (AVAILABLE)

Next, you can:
  • View details: hcloud CSS ShowClusterDetail --cluster_id prod-es-id
  • Create snapshot: hcloud CSS CreateSnapshot --cluster_id prod-es-id
  • Monitor: hcloud CES ShowMetricData --cluster_id prod-es-id
```
