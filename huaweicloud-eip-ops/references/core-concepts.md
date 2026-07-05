# EIP Core Concepts — Huawei Cloud Elastic IP

## 1. What Is an EIP?

An **Elastic IP (EIP / 弹性公网IP)** is a public IPv4 address that:

- Belongs to your **account**, not to a specific resource (resource-independent)
- Is **region-scoped** — you can bind it to any resource in the region that supports EIP
- Continues to **bill** even when unbound (this is the #1 cost leak; see FinOps §3)
- Is **re-bindable** without re-allocation (move between ECS / ENI / NAT / ELB / VIP)

| Resource | Can Bind EIP? | Notes |
|---|---|---|
| ECS (弹性云服务器) | Yes | bind to ECS port (single NIC) |
| ENI (弹性网卡) | Yes | bind to NIC; one EIP per NIC |
| NAT Gateway | Yes | typically `5_bgp`, used for SNAT egress |
| ELB (Enhanced / Classic) | Yes (Enhanced only) | `5_dualStack` for IPv6 EIP |
| Virtual IP (虚拟IP) | Yes | for HA / keepalived setups |
| BMS (裸金属) | Yes | limited by physical NIC |

## 2. EIP Types

| Type | Code | Network | Use Case |
|---|---|---|---|
| 全动态 BGP | `5_bgp` | Standard, multi-carrier | Default; covers most workloads |
| 静态 BGP | `5_sbgp` | Single-carrier, lower cost | Legacy / non-critical |
| 优选 BGP | `5_premium` | Premium routing | Latency-sensitive |
| 双栈 | `5_dualStack` | IPv4 + IPv6 | IPv6 exposure required |

> Default to `5_bgp` unless user explicitly asks for `5_sbgp` (cost) or `5_dualStack` (IPv6).

## 3. Bandwidth Types

| Type | Code | Meaning |
|---|---|---|
| 独占带宽 (PER) | `PER` | Bandwidth attached to a single EIP |
| 共享带宽 (WHOLE) | `WHOLE` | Bandwidth pool; multiple EIPs share one Mbps ceiling |

Shared bandwidth is a FinOps lever: if 3 EIPs are never all busy at once, the WHOLE
bandwidth can be sized to the **sum-of-peaks** instead of the **sum-of-Mbps**, often
saving 30–60%.

## 4. Billing Modes (see also well-architected §3)

| Mode | API value | When it bills | Default use |
|---|---|---|---|
| 按带宽 | `bandwidth` | Mbps × hours | Stable 24×7 |
| 按流量 | `traffic` | Bytes egress | Spiky / dev / bursty |
| 共享带宽 | (wrapped under `WHOLE`) | Mbps × hours (per pool) | ≥2 EIPs |
| 95计费 | `95` (contract) | Monthly 5-min samples | Wholesale |

## 5. Lifecycle States

```
[NOT_EXIST] --allocate--> [DOWN] --bind--> [BINDING] --port attached--> [ACTIVE]
                                                              --unbind--> [DOWN]
[ACTIVE|DOWN] --release--> [DELETING] --> [NOT_EXIST]
```

| Status | Meaning | Agent Action |
|---|---|---|
| `DOWN` | EIP exists, not bound | safe to release or bind |
| `BINDING` | Transition | poll describe; do not retry |
| `ACTIVE` | Bound and traffic-able | release needs unbind first |
| `ERROR` | Last operation failed | diagnose; do not retry blindly |

## 6. Key Resource Identifiers

| Field | Source | Used as |
|---|---|---|
| `{{output.eip_id}}` | `publicip.id` | All EIP operations |
| `{{output.public_ip}}` | `publicip.public_ip_address` | DNS / allowlist |
| `{{output.bandwidth_id}}` | `bandwidth.id` | Bandwidth operations |
| `{{output.port_id}}` | `publicip.port_id` | Bind / unbind target |
| `{{user.region}}` | user input | All `hcloud eip` calls |

## 7. Region Quotas (Dynamic — Always Query)

```bash
hcloud eip describe-quota --region {{env.HW_REGION_ID}}
```

> Per `AGENTS.md` TE-1, **never hardcode** quota numbers. Always read from
> `hcloud eip describe-quota` at run time.
