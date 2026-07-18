# DNS Troubleshooting Guide — Huawei Cloud DNS

## Top DNS Failure Patterns

### T1: DNS Record Not Resolving

| Step | Check | Fix |
|---|---|---|
| 1 | Verify zone is `ACTIVE` | `hcloud dns show-zone --zone-id {{user.zone_id}}` → status must be `ACTIVE` |
| 2 | Verify NS records at registrar | `dig NS example.com.` — NS must point to Huawei Cloud DNS nameservers |
| 3 | Verify record set exists | `hcloud dns list-recordsets --zone-id {{user.zone_id}}` |
| 4 | Check TTL propagation | TTL expired? DNS changes take up to TTL duration to propagate |
| 5 | Check for duplicate record sets | Same name + type = conflict |

### T2: DNSSEC Validation Failure

| Step | Check | Fix |
|---|---|---|
| 1 | Check zone DNSSEC status | `hcloud dns show-zone --zone-id {{user.zone_id}}` → `dnssec_status` |
| 2 | Verify DS record at registrar | Registrar must host DS record for DNSSEC to work |
| 3 | Disable and re-enable DNSSEC | Force re-sign of all records |

### T3: CNAME + MX Conflict

| Problem | Cause | Fix |
|---|---|---|
| Cannot add MX to a CNAME record | RFC 1034 forbids MX pointing to CNAME | Add MX directly to the canonical name |
| Multiple CNAME targets | One name → one target | Remove duplicate CNAME; use A record |

## Error Code Quick Reference

| Code | Meaning | Immediate Action |
|---|---|---|
| `ZoneNotFound` | Zone ID invalid | Verify with `list-zones` |
| `RecordNotFound` | Recordset ID invalid | Verify with `list-recordsets` |
| `ZoneLocked` | DNSSEC transition in progress | Wait; poll status |
| `ZoneNotEmpty` | Zone has remaining records | Delete all records before deleting zone |
| `RecordExists` | Duplicate (same name+type) | `update-recordset` instead of `create` |
| `InvalidTTL` | TTL out of range | Use value 1–2147483647 |
| `Unauthorized` | IAM missing | Add `DNS FullAccess` policy |

## 官方错误码参考（API Error Codes）

> **权威说明**：下表为华为云 DNS API **真实返回**的错误码（`DNS.*` 命名空间），故障定位时优先匹配。上文「Error Code Quick Reference」中的 `ZoneNotFound` / `RecordNotFound` / `Unauthorized` 等为本 runbook 的**文档约定码**（语义化助记，非 API 原始返回码），仅用于正文 T1–T3 流程引用；当 `hcloud` 返回 `DNS.*` 码时以下表为准。

来源：[华为云 DNS API 参考 — 附录：错误码](https://support.huaweicloud.com/api-dns/ErrorCode.html)。以下为真实错误码，可直接用于故障定位。

| Error Code | Meaning | Recovery |
|---|---|---|
| `DNS.0202` | 域名格式非法（TLD/二级域名违规） | 检查域名格式，不可为 TLD 或公共二级域名 |
| `DNS.0208` | 域名已存在 | 确认域名未被任何租户创建 |
| `DNS.0211` | 域名被其他租户占用 | 通过主域名授权认领该名称 |
| `DNS.0304` | 记录集名称非法（须以区域域名为后缀） | 确保记录集名称为以区域结尾的合法 FQDN |
| `DNS.0308` | 记录集值非法 | 检查对应类型的记录值格式 |
| `DNS.0312` | 记录集名称已存在 | 使用已有记录集，勿重复创建 |
| `DNS.0319` | 记录集 TTL 超出范围 | TTL 设为允许区间内的值 |
| `DNS.0335` | 重复的记录集已存在 | 删除区域内相同记录集 |
| `DNS.0403` | 记录集配额不足 | 记录集数量达租户上限，联系工单提额 |
| `DNS.0404` | 域名配额不足 | 域名数量达租户上限，联系工单提额 |
| `DNS.0507` | PTR 记录不存在 | 核对 PTR 记录 ID/名称 |
| `DNS.1206` | 本账号未找到该域名 | 确认域名归属本账号 |
| `DNS.2302` | DNSSEC 已启用，重复开启 | 无需重复，DNSSEC 已开启 |
| `DNS.2305` | 区域存在 DS 记录，无法删除 | 先在注册商处删除 DS 记录 |
| `DNS.0005` | 需要认证（token/权限） | 检查 token 有效性及资源操作权限 |
| `DNS.0040` | 需要实名认证 | 完成实名认证后重试 |
