# EIP Troubleshooting — Huawei Cloud Elastic IP

## Top 8 Failure Patterns

### P1 — EIP Allocated, But Traffic Doesn't Reach the Resource

| Step | Check | Fix |
|---|---|---|
| 1 | `hcloud eip describe` — is `status=ACTIVE` and `port_id` set? | If `DOWN`, retry bind; if `ERROR`, unbind + rebind |
| 2 | Security group on target resource: does it allow ingress on the expected port? | Add SG rule; delegate to `huaweicloud-vpc-ops` |
| 3 | OS firewall (iptables / firewalld / Windows Firewall) | `iptables -L -n -v` and add rule |
| 4 | Route table — does subnet have `0.0.0.0/0 → igw`? | Delegate to `huaweicloud-vpc-ops` |

### P2 — Bandwidth Saturation / "Slow" Reports

| Step | Check | Fix |
|---|---|---|
| 1 | `hcloud eip describe` — current `bandwidth.size` (Mbps) | Resize via `hcloud eip update-bandwidth` |
| 2 | If 按流量 — is egress approaching 100 Mbps? | Note: 100 Mbps is the **hard ceiling** of `traffic` mode |
| 3 | CES metric `outgoing_bytes / bandwidth_size > 0.9` for >5 min | Trigger AIOps pattern → `references/advanced/aiops-patterns.md` |
| 4 | DDoS? | Delegate to `huaweicloud-ddos-ops` / `huaweicloud-hss-ops` |

### P3 — `EipAllocateFailed` on Allocate

| Step | Check | Fix |
|---|---|---|
| 1 | Region in stockout? Try `hcloud eip list --region <adjacent>` first | Move to adjacent region |
| 2 | Quota? `hcloud eip describe-quota` | HALT; quota raise |
| 3 | Account balance? | HALT; recharge |

### P4 — `release-eip` Returns `EipInUse`

EIP is still bound. Sequence:

```bash
hcloud eip unbind --eip-id "{{user.eip_id}}" --region "{{user.region}}"
# poll until port_id == null
hcloud eip delete --eip-id "{{user.eip_id}}" --region "{{user.region}}"
```

### P5 — Idle EIP (Cost Leak)

```bash
# All EIPs unbound for ≥7 days
hcloud eip list --region {{env.HW_REGION_ID}} --output json \
  | jq '.publicips[] | select(.port_id == null) | {id, public_ip_address, created_at}'
```

Cross-skill: feed the list to `huaweicloud-billing-ops` for cost attribution.

### P6 — DNS Still Resolves Old EIP After Release

- Cloud DNS (DNS) TTL still caches the old A record.
- Action: lower TTL **before** the planned release; after release, manually flush
  recursive resolvers if the domain is critical.
- Delegate to `huaweicloud-dns-ops` (when present) or manual DNS provider.

### P7 — Cross-Region EIP Bind Failure

EIP is region-scoped; binding to a resource in another region is **impossible**.
Fix: allocate a new EIP in the target region, re-bind.

### P8 — Bandwidth Resize Cooldown (95计费)

If the EIP is in `WHOLE` shared bandwidth and the subscription is 95计费, each
bandwidth change triggers a **cooldown window** (typically 24h). Plan accordingly;
batch resize requests.

## Diagnostic Command Bundle

```bash
# Snapshot the current state
hcloud eip list --region {{env.HW_REGION_ID}} --output json \
  | jq '.publicips[] | {id, public_ip_address, status, port_id,
      bw_id: .bandwidth.id, bw_size: .bandwidth.size,
      charge_mode: .bandwidth.charge_mode, share_type: .bandwidth.share_type}'

# Check quota
hcloud eip describe-quota --region {{env.HW_REGION_ID}} --output json
```
