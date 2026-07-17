# Prompts — Huawei Cloud VPC

> **Purpose:** Structured prompts for VPC (Virtual Private Cloud) AIOps operations.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 VPC Health Check
```
Analyze VPC {{resource_id}} health status:
- VPC CIDR: {{vpc_cidr}}
- Subnets: {{subnet_count}} (AZs: {{az_count}})
- Route tables: {{route_table_count}}
- Security groups: {{sg_count}}
- ENIs: {{eni_count}} (active: {{eni_active}})
- Peering connections: {{peering_count}}
Determine VPC health and recommend actions.

Applicable CES metrics: SYS.VPC.bandwidth, SYS.VPC.connection_count, AGT.VPC.security_group_rules
```

### 1.2 Bandwidth Utilization Analysis
```
Analyze VPC bandwidth on {{resource_id}}:
- VPN bandwidth: {{vpn_bandwidth}}Mbps / {{vpn_max}}Mbps
- Direct Connect: {{dc_bandwidth}}Mbps / {{dc_max}}Mbps
- Peering bandwidth: {{peering_bandwidth}}Mbps
- NAT gateway: {{nat_bandwidth}}Mbps
- Utilization trend: {{utilization_trend}}%
Identify bandwidth bottlenecks.

VPC bandwidth: VPN gateways, Direct Connect, VPC peering, NAT gateway, EIP
```

### 1.3 Security Group Analysis
```
Analyze VPC security group {{resource_id}}:
- Rules: {{rule_count}} (ingress: {{ingress_count}}, egress: {{egress_count}})
- Resources attached: {{attached_resources}}
- Rule changes (24h): {{rule_changes}}
- Overpermissive rules: {{overpermissive_count}}
- Unused rules: {{unused_rules}}
Assess security posture.

VPC security groups: port-based rules, IP ranges, security group references
```

### 1.4 Route Table Analysis
```
Analyze VPC route table {{resource_id}}:
- Routes: {{route_count}}
- Active routes: {{active_routes}}
- Blackhole routes: {{blackhole_routes}}
- Route conflicts: {{route_conflicts}}
- Peering routes: {{peering_routes}}
- VPN routes: {{vpn_routes}}
Diagnose routing issues.

VPC routing: local, peering, VPN, NAT, Internet, custom vs system routes
```

### 1.5 Connection Analysis
```
Analyze VPC connections on {{resource_id}}:
- VPN connections: {{vpn_count}} (active: {{vpn_active}})
- Direct Connect: {{dc_count}} (dedicated: {{dc_dedicated}}, shared: {{dc_shared}})
- VPC peering: {{peering_count}} (active: {{peering_active}})
- Transit gateway: {{tgw_usage}}%
- Connection latency: {{conn_latency}}ms
Assess connectivity health.

VPC connections: IPsec VPN, Direct Connect, VPC peering, Transit Gateway, SD-WAN
```

---

## 2. Root Cause Analysis Prompts

### 2.1 Connectivity Failure Analysis
```
Analyze VPC connectivity failure on {{resource_id}}:
- Source: {{source_resource}} ({{source_type}})
- Destination: {{dest_resource}} ({{dest_type}})
- Protocol: {{protocol}} (TCP/UDP/ICMP)
- Port: {{port}}
- Failure type: {{failure_type}} (timeout/refused/no_route)
- Firewall: {{firewall_check}}
Diagnose connection issue.

VPC connectivity: ENI issues, security group rules, NACL rules, route tables, peering
```

### 2.2 VPN Tunnel Failure Analysis
```
Analyze VPN tunnel failure on {{resource_id}}:
- VPN gateway: {{vpn_gateway_id}}
- Connection: {{connection_id}}
- Tunnel status: {{tunnel_status}} (up/down)
- IKE errors: {{ike_errors}}
- IPSec errors: {{ipsec_errors}}
- Peer gateway: {{peer_gateway}}
Diagnose VPN issue.

VPN failure causes: peer gateway unreachable, Phase 1/2 mismatch, encryption domain, bandwidth limit
```

### 2.3 Direct Connect Failure Analysis
```
Analyze Direct Connect failure on {{resource_id}}:
- Connection ID: {{connection_id}}
- Virtual interface: {{vif_id}}
- Connection status: {{conn_status}} (ordered/active/down)
- BGP status: {{bgp_status}}
- MAC address: {{mac_address}}
- VLAN ID: {{vlan_id}}
Identify DX issue.

Direct Connect failures: physical link, BGP session, VLAN configuration, MAC security
```

### 2.4 Security Group Misconfiguration
```
Analyze security group misconfiguration on {{resource_id}}:
- Affected ENI: {{eni_id}}
- Expected traffic: {{expected_traffic}}
- Actual allowed: {{actual_allowed}}
- Rule causing block: {{blocking_rule}}
- Recent changes: {{recent_changes}}
- Related alerts: {{related_alerts}}
Identify blocking rule.

VPC security group issues: overly restrictive rules, rule order, protocol mismatch, source/dest confusion
```

### 2.5 NACL Issue Analysis
```
Analyze NACL issue on {{resource_id}}:
- Subnet: {{subnet_id}}
- NACL: {{nacl_id}}
- Rule causing block: {{blocking_rule}}
- Traffic type: {{traffic_type}}
- Rule order: {{rule_order}}
- Stateless inspection impact: {{stateless_impact}}
Diagnose NACL issue.

VPC NACL: stateless rules, rule order processing, ephemeral ports, allow vs deny
```

---

## 3. Capacity Prompts

### 3.1 CIDR Capacity Planning
```
Plan VPC CIDR capacity for {{resource_id}}:
- Current CIDR: {{vpc_cidr}}
- Used address space: {{used_addresses}} / {{total_addresses}} ({{utilization}}%)
- Subnets allocated: {{subnet_count}}
- Available AZs: {{available_azs}}
- Future requirements: {{future_requirements}}
- Recommended CIDR: {{recommended_cidr}}
Prevent IP exhaustion.

VPC CIDR: primary/secondary CIDR, subnet sizing, RFC 1918, CIDR overlap
```

### 3.2 Security Group Rule Optimization
```
Optimize security group rules for {{resource_id}}:
- Current rules: {{rule_count}} (limit: {{sg_rule_limit}})
- Duplicate rules: {{duplicate_count}}
- Overly broad rules: {{broad_rules}}
- Unused rules: {{unused_count}}
- Recommended consolidation: {{consolidation_plan}}
- Rule efficiency score: {{efficiency_score}}
Reduce rule count.

VPC SG optimization: CIDR notation, security group references, rule consolidation
```

### 3.3 Route Capacity Planning
```
Plan VPC route capacity for {{resource_id}}:
- Current routes: {{route_count}} (limit: {{route_limit}})
- Routes per table: {{routes_per_table}}
- Peering routes: {{peering_routes}}
- VPN routes: {{vpn_routes}}
- Future scale needs: {{future_routes}}
- Recommended architecture: {{recommended_arch}}
Prevent route exhaustion.

VPC routing: route limits per table, transitive peering not allowed, TGW routing
```

### 3.4 Peering Capacity Planning
```
Plan VPC peering capacity for {{resource_id}}:
- Current peerings: {{peering_count}} (limit: {{peering_limit}})
- Route limits: {{route_limit}} per peering
- Regional constraints: {{regional_constraint}}
- Transit traffic needs: {{transit_needs}}
- Recommended: {{recommended_setup}}
Optimize peering design.

VPC peering: non-transitive, route limits, same-account vs cross-account, regional
```

---

## 4. Availability Prompts

### 4.1 HA Configuration Check
```
Check VPC HA configuration for {{resource_id}}:
- VPN HA: {{vpn_ha}} (active/standby)
- Direct Connect: {{dc_ha}} (primary/backup)
- Multi-AZ subnets: {{multiaz_subnets}}
- Route redundancy: {{route_redundancy}}
- TGW redundancy: {{tgw_redundancy}}
Assess HA readiness.

VPC HA: VPN dual-tunnel, DX dual-connection, multi-AZ deployment, route redundancy
```

### 4.2 DR Network Assessment
```
Assess VPC DR capability for {{resource_id}}:
- Primary VPC: {{primary_vpc}} (region: {{primary_region}})
- DR VPC: {{dr_vpc}} (region: {{dr_region}})
- Peering: {{peering_setup}}
- VPN: {{vpn_setup}}
- RTO: {{rto_achieved}}min (target: {{rto_target}}min)
- RPO: {{rpo_achieved}}min (target: {{rpo_target}}min)
Evaluate DR network.

VPC DR: cross-region peering, VPN backup, DX backup, DNS failover
```

### 4.3 Network Monitoring Coverage
```
Check VPC monitoring coverage for {{resource_id}}:
- Flow logs: {{flow_log_enabled}}
- Traffic monitored: {{monitored_traffic}}%
- VPN metrics: {{vpn_metrics}} (latency, tunnel status)
- DX metrics: {{dx_metrics}} (BGP, connection)
- Peering metrics: {{peering_metrics}}
- Cloud Eye alarms: {{alarm_count}}
Verify observability.

VPC monitoring: Flow Log, VPN Monitor, DX Connector, CES metrics, CTS traces
```

### 4.4 SLA Compliance Report
```
Report VPC SLA compliance for {{resource_id}}:
- VPN availability: {{vpn_availability}}% (target: {{target}}%)
- Direct Connect: {{dx_availability}}% (target: {{target}}%)
- VPC Peering: {{peering_availability}}%
- Route availability: {{route_availability}}%
- Configuration changes: {{config_changes}}/day
Report SLA violations.

VPC SLA: gateway availability, connection uptime, routing availability
```

---

## 5. Inspection Prompts

### 5.1 Routine Health Inspection
```
Perform routine VPC inspection:
- List all VPCs in {{scope}}
- Check CIDR utilization > {{cidr_threshold}}%
- Identify unused security groups
- Flag rules approaching limits
- Check peering health
- Verify route table consistency
- Check for orphaned resources
Report findings.

Scope: region, project, tag-based resource group
```

### 5.2 Security Compliance Check
```
Audit VPC security compliance:
- Public subnets: {{public_subnet_count}} (avoid if possible)
- Exposed ENIs: {{exposed_enis}}
- Security group rules: overly permissive ({{overpermissive_count}})
- NACL rules: default allow ({{default_allow_count}})
- VPN encryption: {{vpn_encryption}} (AES256 recommended)
- Direct Connect: MACsec {{macsec_enabled}}
Report compliance status.

Severity: Critical = public subnet with DB, High = 0.0.0.0/0 SG rule, Medium = no MACsec
```

### 5.3 Cost Optimization Scan
```
Scan VPC for cost optimization:
- Idle VPN connections: {{idle_vpn_count}}
- Oversized EIPs: {{oversized_eips}}
- Unused Direct Connect: {{unused_dx_count}}
- Peering efficiency: {{peering_efficiency}}
- NAT gateway usage: {{natgw_usage}}
- VPN bandwidth vs utilization: {{vpn_utilization}}%
Provide action list with estimated savings.

VPC cost: EIP idle fees, VPN bandwidth, Direct Connect dedicated, NAT gateway
```

### 5.4 Network Architecture Review
```
Review VPC network architecture:
- CIDR design: {{cidr_design_score}} (1-10)
- Subnet allocation: {{subnet_design_score}}
- AZ distribution: {{az_coverage}}
- Route efficiency: {{route_efficiency_score}}
- Security group design: {{sg_design_score}}
- Peering complexity: {{peering_complexity}}
Identify architectural improvements.

VPC architecture: hub-spoke vs full mesh, subnet sizing, AZ placement, security zoning
```

### 5.5 Configuration Drift Detection
```
Detect VPC configuration drift:
- VPC: {{vpc_id}}
- Expected CIDR: {{expected_cidr}}
- Actual CIDR: {{actual_cidr}}
- Expected routes: {{expected_routes}}
- Actual routes: {{actual_routes}}
- Expected SG rules: {{expected_sg_rules}}
- Actual SG rules: {{actual_sg_rules}}
Recommend reconciliation.

VPC drift: CIDR changes, route modifications, SG rule changes, peering changes
```

---

## Appendix: VPC-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{resource_id}}` | VPC ID | `vpc-abcd1234` |
| `{{vpc_cidr}}` | VPC CIDR block | `192.168.0.0/16` |
| `{{subnet_id}}` | Subnet ID | `subnet-1234` |
| `{{security_group_id}}` | Security group ID | `sg-5678` |
| `{{route_table_id}}` | Route table ID | `rtb-9abc` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance*
