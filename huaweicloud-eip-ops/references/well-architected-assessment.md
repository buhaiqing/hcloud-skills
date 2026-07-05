# EIP Well-Architected + Three-Pillar Assessment

## 1. Security Pillar (安全支柱)

### IAM Minimum Permissions

| Role | Required Permissions | Use Case |
|---|---|---|
| EIP Viewer | `vpc:eip:list`, `vpc:eip:get`, `vpc:bandwidth:list`, `vpc:bandwidth:get` | Read-only inspection |
| EIP Operator | Viewer + `vpc:eip:create`, `vpc:eip:update`, `vpc:eip:delete` | Allocate / bind / unbind / release |
| EIP Admin | Operator + `vpc:bandwidth:create`, `vpc:bandwidth:update`, `vpc:bandwidth:delete` | Shared-bandwidth management |

### EIP Exposure Best Practices

| Principle | Rule |
|---|---|
| Default deny on SG | Only open ports with explicit ingress allowlist; do not use `0.0.0.0/0` for SSH/RDP/DB |
| Egress constraint | Restrict outbound to known CIDRs (e.g., specific API endpoints) |
| Audit | Tag every EIP with `owner`, `purpose`, `cost-center` so leaks are attributable |
| DDoS-aware | EIPs that face the public internet should be protected; delegate to `huaweicloud-ddos-ops` |
| Key hardening | Rotate `HW_ACCESS_KEY_ID` every 90 days; never share across skills |

### Encryption

- EIP itself carries no payload data; encryption applies to the **traffic** it carries.
- Use TLS for any application protocol (HTTPS / SMTPS / IMAPS); for SSH, prefer key-only auth.

### Network Isolation

- EIP lives in the **public** address space — do not bind to a private-only workload.
- For workloads that need both, prefer SNAT via NAT Gateway (delegate to `huaweicloud-nat-ops`).

## 2. Stability Pillar (稳定支柱)

### Multi-AZ / Multi-Region

| Practice | Recommendation |
|---|---|
| Single EIP per workload | Acceptable; EIP can move within a region |
| Cross-region failover | Allocate a parallel EIP in the standby region; switch DNS (delegate to `huaweicloud-dns-ops`) |
| EIP pool warm-up | Keep ≥1 EIP `DOWN` in production for fast re-bind during ECS rebuild |

### Capacity Sizing

| Metric | Recommendation |
|---|---|
| `bandwidth.size` | Sized to p95 of historical egress; re-evaluate monthly |
| `outgoing_bytes` (按流量) | Watch for >80% of `bandwidth.size` cap (按流量 has a hard ceiling) |
| Shared bandwidth pool | Pool size ≈ sum-of-peaks, not sum-of-Mbps |

### Backup / DR

EIP has no data to back up. The **disaster** to plan for is **unintended release**:
- EIP release is irreversible.
- Mitigation: enforce two-step confirmation for prod EIPs (S-rule S10).
- For DNS failover, ensure TTL is low (≤300s) before any planned EIP swap.

## 3. FinOps (财务运营)

### 3.1 Billing Model Comparison

| Mode | Best For | Cost Shape | Predictability | Risk |
|---|---|---|---|---|
| `bandwidth` (按带宽) | Stable 24×7 traffic | `Mbps × hours` | High | Over-pays during idle hours |
| `traffic` (按流量) | Spiky / dev / mostly idle | `bytes egress × unit_price` | Medium | Surprise bill on burst |
| `shared` (WHOLE) | ≥2 EIPs in complementary time zones | Pool `Mbps × hours` | High (after sizing) | Move-in/out complexity |
| `95` (95计费) | Wholesale / ISP-like shape | Monthly 5-min samples, top 5% discarded | Variable | Cooldown after each change |

**Default recommendation matrix:**

| Workload | Recommended Mode |
|---|---|
| Single EIP, prod 24×7 | `bandwidth` |
| Single EIP, dev/test, mostly idle | `traffic` |
| ≥3 EIPs, disjoint traffic hours | `shared` |
| Wholesale / ≥10 Gbps aggregate | `95` (contract only) |

### 3.2 Idle EIP Detection

```bash
# 7-day idle EIPs
hcloud eip list --region {{env.HW_REGION_ID}} --output json \
  | jq -r '.publicips[] | select(.port_id == null) | {id, alias, created_at: .create_time}'
```

> Idle EIP cost: 按带宽 EIP still bills `Mbps × hours` even when unbound. Even an
> idle 按流量 EIP carries a small per-EIP fee.

### 3.3 Right-Sizing

| Observed | Recommendation |
|---|---|
| p95 egress < 30% of `bandwidth.size` for 14 d | Downsize to nearest 5 Mbps |
| 按流量 + 24h egress < 1 GB | Keep current |
| Burst pattern with p99 / p50 > 5× | Switch to `traffic` mode |
| ≥3 EIPs with low individual p95 | Migrate to `shared` bandwidth pool |

### 3.4 Cost Tagging Strategy

Mandatory tags on every EIP:

| Tag | Source | Example |
|---|---|---|
| `owner` | user input | `team:platform` |
| `purpose` | user input | `prod-api`, `dev-jumpbox`, `nat-egress` |
| `cost-center` | user input | `CC-1234` |
| `lifecycle` | derived | `permanent`, `ephemeral-<ttl>` |

`huaweicloud-billing-ops` reads these tags for cost attribution.

## 4. SecOps (安全运营)

### 4.1 EIP Risk Surface

| Risk | Mitigation |
|---|---|
| Idle EIP exposes account | Detect via Pattern 3; release or tag with `warm-pool` |
| Prod EIP released by accident | S-rule S10: two-step confirmation on prod EIP release |
| EIP bind to wrong resource | Cross-skill: confirm `port_id` belongs to expected ECS / ENI |
| Bandwidth leak via shared pool | Audit `bandwidth.eip_count` weekly; release empty pools |

### 4.2 High-Risk Operations

| Operation | Risk | Required Gate |
|---|---|---|
| `release-eip` (any) | Irreversible | User confirmation + `port_id == null` check |
| `release-eip` (prod-named) | Production blast radius | Two-step confirmation |
| `release-eip` (in WHOLE pool) | Pool becomes empty | Admin role + confirmation |
| `unbind-eip` (prod) | Brief traffic interruption | Confirmation + low-traffic window |
| `resize-bandwidth` (95计费) | Cooldown triggers | Schedule + acknowledge |

### 4.3 Threat Detection Triggers

| Trigger | Action |
|---|---|
| `incoming_bandwidth` p99 > 10× p50 | Delegate to `huaweicloud-ddos-ops` / `huaweicloud-hss-ops` |
| Repeated `unbind`/`bind` from same AK | Audit; possible credential leak |
| New EIP with `0.0.0.0/0` SG exposure | Block via SG change; delegate to `huaweicloud-vpc-ops` |
| 95计费 bill shock | Cross-check with `huaweicloud-billing-ops` |

## 5. Operational Efficiency (性能效率)

- Use `client_token` on bind / unbind / add-to-shared for safe retry.
- Use list-then-act pattern on allocate (dedupe by `public_ip_address`).
- Pre-allocate warm EIPs for prod failover; tag with `warm-pool`.
- Audit shared bandwidth pools monthly for empties.

## 6. Cost Optimization Examples

| Before | After | Saving |
|---|---|---|
| 3 EIPs × 100 Mbps `bandwidth` = 300 Mbps ceiling | 1 WHOLE shared bandwidth 150 Mbps | ~50% on bandwidth cost |
| 1 EIP `bandwidth` 100 Mbps, average 5 Mbps | Switch to `traffic` mode, no cap | ~70% on idle hours |
| Idle EIPs (5×) × 5 Mbps `bandwidth` for 30 d | Release 4, tag 1 `warm-pool` | 80% of idle cost |
