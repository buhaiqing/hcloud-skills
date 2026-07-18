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

## 官方错误码参考（API Error Codes）

> **权威说明**：下表为华为云 EIP/VPC API **真实返回**的错误码（以 `VPC.*` 命名空间为主，辅以 `EIP.*`），故障定位时优先匹配。正文 P3/P4 中的 `EipAllocateFailed` / `EipInUse` 为本 runbook 的**文档约定码**（语义化助记，非 API 原始返回码）；当 `hcloud` 返回 `VPC.*` / `EIP.*` 码时以下表为准。

来源：[华为云 EIP API 参考 — 附录：错误码](https://support.huaweicloud.com/api-eip/ErrorCode.html)。EIP 复用 VPC/Neutron 错误空间，错误码以 `VPC.*` 命名空间为主，辅以 `EIP.*`。以下为真实错误码。

| Error Code | Meaning | Recovery |
|---|---|---|
| `VPC.0504` | EIP 不存在（Floating IP 未找到） | 检查指定的 EIP ID 是否有效 |
| `VPC.0510` | EIP 已绑定到 ECS | 先解绑 EIP 再操作 |
| `VPC.0511` | 端口已绑定 EIP | 先从现有 EIP 解绑端口 |
| `VPC.0517` | EIP 已绑定端口，无法释放 | 释放前先从 ECS 解绑 |
| `VPC.0521` | EIP 配额超限 | 释放未绑定 EIP 或提额 |
| `VPC.0532` | IP 池地址耗尽 | 释放未绑定 EIP 或稍后重试 |
| `VPC.0503` | 创建 publicIp 失败 | 联系技术支持 |
| `VPC.0501` | EIP 参数非法 | 对照 API 文档检查参数值 |
| `VPC.0301` | 带宽名称 / share_type / bandwidth 参数非法 | 检查带宽参数是否合法 |
| `VPC.0310` | 共享带宽配额不足 | 删除多余共享带宽或联系支持 |
| `VPC.0516` | EIP 已被 ELB 占用 | 先从 ELB 解绑 EIP |
| `VPC.0525` | 包年/包月 EIP 计费中，不可删除 | 走退订流程，勿直接释放 |
| `EIP.7901` | 输入参数非法（请求体错误） | 按提示检查 JSON 格式与取值范围 |
| `VPC.0502` | 租户状态受限（账号冻结/余额不足） | 检查账户余额或冻结状态 |
| `VPC.0008` | 请求头 token 非法（鉴权失败） | 检查请求头 token 是否有效 |
