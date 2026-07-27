// Package learning provides Go ports of:
//   - scripts/gen_skill_knowledge.py  → WriteSkillAssets / Products
//   - scripts/trace_learning.py        → trace.go (signature key, extract, aggregate)
//
// The byte-level output of WriteSkillAssets must match the Python baseline so
// downstream tooling (gcl_runner pre_execution_risk) sees no diff.
package learning

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Pattern mirrors the failure-patterns/v1 schema. The fix block intentionally
// omits playbook_ref — only the remediation-playbooks schema carries it.
type Pattern struct {
	ID         string         `json:"id"`
	Category   string         `json:"category"`
	Signature  PatternSig     `json:"signature"`
	RootCause  string         `json:"root_cause"`
	Fix        PatternFix     `json:"fix"`
	Prevention string         `json:"prevention"`
	Stats      map[string]any `json:"stats,omitempty"`
}

// PatternSig is the signature block.
type PatternSig struct {
	ErrorCode         string `json:"error_code"`
	ErrorMessageRegex string `json:"error_message_regex"`
	CommandPattern    string `json:"command_pattern"`
}

// PatternFix is the fix block. (playbook_ref intentionally absent — see above.)
type PatternFix struct {
	Strategy string `json:"strategy"`
	Action   string `json:"action"`
}

// Playbook mirrors the remediation-playbooks/v1 schema.
type Playbook struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Trigger     map[string]any `json:"trigger"`
	Diagnosis   map[string]any `json:"diagnosis"`
	Remediation map[string]any `json:"remediation"`
}

// SkillPayload is the per-skill PRODUCTS entry shape.
type SkillPayload struct {
	Patterns  []Pattern  `json:"patterns"`
	Playbooks []Playbook `json:"playbooks"`
}

// Products is the regenerated knowledge base for the high-frequency skills.
// Mirrors gen_skill_knowledge.py PRODUCTS exactly so re-running produces
// byte-identical output.
var Products = map[string]SkillPayload{
	"rds": {
		Patterns: []Pattern{
			{ID: "RDS-FP001", Category: "resource_state", Signature: PatternSig{ErrorCode: "DBS.200001", ErrorMessageRegex: "instance.*not found|InstanceNotFound", CommandPattern: "describe-instance"}, RootCause: "DB instance ID invalid or instance deleted", Fix: PatternFix{Strategy: "retry", Action: "List instances to verify ID: hcloud RDS listInstances --region {{env.HW_REGION_ID}} --limit 50"}, Prevention: "Cache instance ID from listInstances before state operations"},
			{ID: "RDS-FP002", Category: "resource_state", Signature: PatternSig{ErrorCode: "DBS.200013", ErrorMessageRegex: "instance.*not.*running|InvalidDBInstanceState", CommandPattern: "*"}, RootCause: "Operation attempted while DB in transitional state (Creating/Restarting/FlavorChanging)", Fix: PatternFix{Strategy: "retry", Action: "Poll showInstance until status==ACTIVE, then retry"}, Prevention: "Pre-check status before state-mutating operations"},
			{ID: "RDS-FP003", Category: "permission", Signature: PatternSig{ErrorCode: "DBS.200028", ErrorMessageRegex: "Quota exceeded|InstanceQuota", CommandPattern: "create-instance"}, RootCause: "RDS instance quota exhausted for project/region", Fix: PatternFix{Strategy: "halt", Action: "Delegate to huaweicloud-iam-ops for quota raise; do not retry"}, Prevention: "Check RDS quota via listQuotas before batch create"},
			{ID: "RDS-FP004", Category: "cross_skill", Signature: PatternSig{ErrorCode: "DBS.201011", ErrorMessageRegex: "VPC.*not found|subnet.*not found", CommandPattern: "create-instance"}, RootCause: "VPC/subnet does not exist or wrong region", Fix: PatternFix{Strategy: "delegate", Action: "Delegate to huaweicloud-vpc-ops to verify/create VPC+subnet, then retry RDS create"}, Prevention: "Verify vpc_id/subnet_id + cross-region match before create"},
			{ID: "RDS-FP005", Category: "network", Signature: PatternSig{ErrorCode: "DBS.201015", ErrorMessageRegex: "connection.*timeout|connect.*refused", CommandPattern: "*"}, RootCause: "Network ACL/security group blocks 3306/5432/1433 traffic to RDS", Fix: PatternFix{Strategy: "fallback", Action: "Inspect security group: hcloud VPC showSecurityGroup --id {{output.sg_id}}; ensure inbound allows client CIDR on DB port"}, Prevention: "Validate SG rules + NACL at provisioning time"},
			{ID: "RDS-FP006", Category: "runtime", Signature: PatternSig{ErrorCode: "", ErrorMessageRegex: "RequestLimitExceeded|Throttling", CommandPattern: "*"}, RootCause: "API rate limit exceeded during burst operations", Fix: PatternFix{Strategy: "retry", Action: "Exponential backoff: 2^n seconds (cap 60s), respect Retry-After header"}, Prevention: "Throttle batch operations; respect per-API rate limits"},
			{ID: "RDS-FP007", Category: "resource_state", Signature: PatternSig{ErrorCode: "DBS.200301", ErrorMessageRegex: "storage.*full|diskfull", CommandPattern: "*"}, RootCause: "Disk space exhausted; writes failing", Fix: PatternFix{Strategy: "fallback", Action: "Extend volume: hcloud RDS enlargeInstanceVolume --instance_id {{output.id}} --size {{output.new_size}}; or prune old backups"}, Prevention: "Monitor disk_util via CES; alarm at 80%"},
			{ID: "RDS-FP008", Category: "cli_parameter", Signature: PatternSig{ErrorCode: "DBS.200021", ErrorMessageRegex: "InvalidParameter|flavor.*invalid", CommandPattern: "create-instance"}, RootCause: "flavor_ref / engine_version / HA mode invalid", Fix: PatternFix{Strategy: "retry", Action: "List valid flavors: hcloud RDS listFlavors --region {{env.HW_REGION_ID}} --engine_type {{output.engine}}"}, Prevention: "Validate flavor via listFlavors before create"},
			{ID: "RDS-FP009", Category: "cli_parameter", Signature: PatternSig{ErrorCode: "DBS.200019", ErrorMessageRegex: "password.*policy|weak.*password", CommandPattern: "create-instance"}, RootCause: "DB root password violates complexity policy", Fix: PatternFix{Strategy: "retry", Action: "Use a password meeting: 8-32 chars, 3 of {upper,lower,digit,special}; do NOT log the password"}, Prevention: "Generate password client-side with policy validator"},
			{ID: "RDS-FP010", Category: "runtime", Signature: PatternSig{ErrorCode: "DBS.202005", ErrorMessageRegex: "backup.*failed|BackupError", CommandPattern: "create-backup"}, RootCause: "Backup window collision or storage backend unavailable", Fix: PatternFix{Strategy: "fallback", Action: "Retry with --wait_until off-peak or delegate to huaweicloud-cbr-ops for cross-region backup"}, Prevention: "Schedule backups in low-traffic windows; verify CBR policy"},
		},
		Playbooks: []Playbook{
			{ID: "RDS-R001", Name: "DB CPU High Restart Process", Trigger: map[string]any{"metric": "rds_cpu_util", "condition": "> 90 for 5min", "namespace": "SYS.RDS"}, Diagnosis: map[string]any{"steps": []string{"hcloud CES getMetricData --namespace SYS.RDS --metric_name cpu_util --period 60 --filter avg --dim instance_id={{output.instance_id}}", "hcloud RDS listSlowLogs --instance_id {{output.instance_id}} --top 20"}, "confidence_factors": []string{"cpu_sustained_high", "slow_queries_present", "no_long_running_txn"}}, Remediation: map[string]any{"risk_level": "medium", "auto_execute_threshold": 0.85, "preconditions": []string{"hcloud RDS showInstance --instance_id {{output.instance_id}} | grep -q ACTIVE"}, "dry_run": "hcloud RDS restartInstance --instance_id {{output.instance_id}} --dry-run", "execute": "hcloud RDS restartInstance --instance_id {{output.instance_id}}", "verification": "hcloud RDS showInstance --instance_id {{output.instance_id}} | grep -q ACTIVE"}},
			{ID: "RDS-R002", Name: "Disk Space Low Volume Expand", Trigger: map[string]any{"metric": "rds_disk_util", "condition": "> 85 for 5min", "namespace": "SYS.RDS"}, Diagnosis: map[string]any{"steps": []string{"hcloud RDS showInstance --instance_id {{output.instance_id}}", "hcloud CES getMetricData --namespace SYS.RDS --metric_name disk_util --period 300 --filter avg --dim instance_id={{output.instance_id}}"}, "confidence_factors": []string{"disk_growing", "no_recent_expand", "quota_available"}}, Remediation: map[string]any{"risk_level": "medium", "auto_execute_threshold": 0.9, "preconditions": []string{"hcloud RDS showInstanceVolume --instance_id {{output.instance_id}} | grep -q available"}, "dry_run": "hcloud RDS enlargeInstanceVolume --instance_id {{output.instance_id}} --size {{output.new_size}} --dry-run", "execute": "hcloud RDS enlargeInstanceVolume --instance_id {{output.instance_id}} --size {{output.new_size}}", "verification": "hcloud RDS showInstanceVolume --instance_id {{output.instance_id}}"}},
			{ID: "RDS-R003", Name: "Connection Count High Failover", Trigger: map[string]any{"metric": "rds_connections", "condition": "> 90% of max_connections", "namespace": "SYS.RDS"}, Diagnosis: map[string]any{"steps": []string{"hcloud CES getMetricData --namespace SYS.RDS --metric_name connections --period 60 --filter avg --dim instance_id={{output.instance_id}}", "hcloud RDS listSessions --instance_id {{output.instance_id}} --limit 50"}, "confidence_factors": []string{"connections_saturated", "long_idle_sessions", "pool_misconfigured"}}, Remediation: map[string]any{"risk_level": "low", "auto_execute_threshold": 0.8, "preconditions": []string{"hcloud RDS showInstance --instance_id {{output.instance_id}} | grep -q ACTIVE"}, "dry_run": "hcloud RDS killSessions --instance_id {{output.instance_id}} --session_ids {{output.session_ids}} --dry-run", "execute": "hcloud RDS killSessions --instance_id {{output.instance_id}} --session_ids {{output.session_ids}}", "verification": "hcloud CES getMetricData --namespace SYS.RDS --metric_name connections --period 60 --filter avg --dim instance_id={{output.instance_id}}"}},
			{ID: "RDS-R004", Name: "Primary Failover Promote Standby", Trigger: map[string]any{"metric": "rds_ha_role", "condition": "primary unreachable for 60s, standby healthy", "namespace": "SYS.RDS"}, Diagnosis: map[string]any{"steps": []string{"hcloud RDS listInstances --status ACTIVE --ha_mode HA --limit 50", "hcloud CES getMetricData --namespace SYS.RDS --metric_name ha_role --period 30 --filter avg"}, "confidence_factors": []string{"primary_down_confirmed", "standby_healthy", "network_partition_or_hw_failure"}}, Remediation: map[string]any{"risk_level": "high", "auto_execute_threshold": 0.95, "preconditions": []string{"hcloud RDS showInstance --instance_id {{output.standby_id}} | grep -q ACTIVE", "hcloud RDS showInstance --instance_id {{output.primary_id}} | grep -qE 'FAILED|ERROR'"}, "dry_run": "hcloud RDS failoverInstance --instance_id {{output.standby_id}} --dry-run", "execute": "hcloud RDS failoverInstance --instance_id {{output.standby_id}}", "verification": "hcloud RDS showInstance --instance_id {{output.standby_id}} | grep -q ACTIVE"}},
		},
	},
	"vpc": {
		Patterns: []Pattern{
			{ID: "VPC-FP001", Category: "resource_state", Signature: PatternSig{ErrorCode: "VPC.0001", ErrorMessageRegex: "VPC.*not found|VpcNotFound", CommandPattern: "*"}, RootCause: "VPC ID invalid or VPC deleted", Fix: PatternFix{Strategy: "retry", Action: "List VPCs: hcloud VPC listVpcs --region {{env.HW_REGION_ID}}"}, Prevention: "Cache VPC ID from listVpcs before operations"},
			{ID: "VPC-FP002", Category: "cli_parameter", Signature: PatternSig{ErrorCode: "VPC.0007", ErrorMessageRegex: "CIDR.*conflict|CidrConflict", CommandPattern: "create-subnet"}, RootCause: "Subnet CIDR overlaps existing subnet in same VPC", Fix: PatternFix{Strategy: "retry", Action: "List existing subnets: hcloud VPC listSubnets --vpc_id {{output.vpc_id}}; pick a non-overlapping /24"}, Prevention: "Pre-allocate non-overlapping CIDR plan; validate before create"},
			{ID: "VPC-FP003", Category: "resource_state", Signature: PatternSig{ErrorCode: "VPC.0501", ErrorMessageRegex: "security.*group.*not.*found|SgNotFound", CommandPattern: "*"}, RootCause: "Security group ID invalid or already deleted", Fix: PatternFix{Strategy: "retry", Action: "hcloud VPC listSecurityGroups --vpc_id {{output.vpc_id}}"}, Prevention: "Cache SG ID at provisioning time"},
			{ID: "VPC-FP004", Category: "permission", Signature: PatternSig{ErrorCode: "VPC.0601", ErrorMessageRegex: "Quota exceeded|ResourceQuota", CommandPattern: "*"}, RootCause: "VPC/subnet/EIP quota exhausted", Fix: PatternFix{Strategy: "halt", Action: "Delegate to huaweicloud-iam-ops for quota raise"}, Prevention: "Check quota via showQuota before bulk operations"},
			{ID: "VPC-FP005", Category: "cli_parameter", Signature: PatternSig{ErrorCode: "VPC.0010", ErrorMessageRegex: "rule.*conflict|RuleConflict", CommandPattern: "create-security-group-rule"}, RootCause: "Duplicate/conflicting SG rule (same direction+port+remote) exists", Fix: PatternFix{Strategy: "retry", Action: "List existing rules: hcloud VPC showSecurityGroup --id {{output.sg_id}}; dedupe"}, Prevention: "Diff existing rules before add; idempotent apply"},
			{ID: "VPC-FP006", Category: "runtime", Signature: PatternSig{ErrorCode: "", ErrorMessageRegex: "RequestLimitExceeded|Throttling", CommandPattern: "*"}, RootCause: "API rate limit on VPC control plane", Fix: PatternFix{Strategy: "retry", Action: "Exponential backoff with jitter; respect Retry-After"}, Prevention: "Throttle batch operations to <5 req/s"},
			{ID: "VPC-FP007", Category: "network", Signature: PatternSig{ErrorCode: "VPC.0700", ErrorMessageRegex: "EIP.*bind.*failed|elastic.*ip.*in.*use", CommandPattern: "associate-eip"}, RootCause: "EIP already bound to another resource", Fix: PatternFix{Strategy: "fallback", Action: "Unbind first: hcloud VPC disassociateEip --eip_id {{output.eip_id}}; or pick another EIP"}, Prevention: "Track EIP-resource binding via CTS before reuse"},
			{ID: "VPC-FP008", Category: "cli_parameter", Signature: PatternSig{ErrorCode: "VPC.0011", ErrorMessageRegex: "peering.*not.*found|PeeringNotFound", CommandPattern: "*"}, RootCause: "VPC peering ID invalid or rejected by remote VPC", Fix: PatternFix{Strategy: "retry", Action: "Verify acceptance: hcloud VPC listVpcPeerings --status PENDING_ACCEPTANCE; ensure both VPCs in same project"}, Prevention: "Validate peering config + acceptance state before traffic"},
			{ID: "VPC-FP009", Category: "cross_skill", Signature: PatternSig{ErrorCode: "VPC.0012", ErrorMessageRegex: "route.*table.*conflict|RouteTableConflict", CommandPattern: "*"}, RootCause: "Default route table + custom route table both applied to same subnet", Fix: PatternFix{Strategy: "fallback", Action: "Disassociate custom route table: hcloud VPC disassociateRouteTable --subnet_id {{output.subnet_id}}"}, Prevention: "Use custom route tables only; document subnet associations"},
			{ID: "VPC-FP010", Category: "network", Signature: PatternSig{ErrorCode: "VPC.0701", ErrorMessageRegex: "bandwidth.*insufficient|BandwidthExceeded", CommandPattern: "*"}, RootCause: "EIP bandwidth cap reached; throttle or upgrade", Fix: PatternFix{Strategy: "fallback", Action: "Modify EIP bandwidth: hcloud VPC updateEipBandwidth --eip_id {{output.eip_id}} --bandwidth {{output.bw_mbps}}"}, Prevention: "Monitor CES metric `eip_bandwidth_util`; alarm at 80%"},
		},
		Playbooks: []Playbook{
			{ID: "VPC-R001", Name: "Security Group Rule Stale Cleanup", Trigger: map[string]any{"metric": "sg_unused_rules", "condition": "> 5 rules unused for 30d", "namespace": "SYS.VPC"}, Diagnosis: map[string]any{"steps": []string{"hcloud VPC showSecurityGroup --id {{output.sg_id}}", "hcloud CTS listTraces --resource_type security_group --limit 100"}, "confidence_factors": []string{"rules_unused", "no_recent_match", "no_compliance_lock"}}, Remediation: map[string]any{"risk_level": "low", "auto_execute_threshold": 0.85, "preconditions": []string{"hcloud VPC showSecurityGroup --id {{output.sg_id}} | grep -q active"}, "dry_run": "hcloud VPC deleteSecurityGroupRule --sg_id {{output.sg_id}} --rule_id {{output.rule_id}} --dry-run", "execute": "hcloud VPC deleteSecurityGroupRule --sg_id {{output.sg_id}} --rule_id {{output.rule_id}}", "verification": "hcloud VPC showSecurityGroup --id {{output.sg_id}}"}},
			{ID: "VPC-R002", Name: "Subnet CIDR Auto-Allocate", Trigger: map[string]any{"metric": "subnet_available_ips", "condition": "< 10 remaining", "namespace": "SYS.VPC"}, Diagnosis: map[string]any{"steps": []string{"hcloud VPC showSubnet --subnet_id {{output.subnet_id}}", "hcloud VPC listSubnets --vpc_id {{output.vpc_id}}"}, "confidence_factors": []string{"cidr_exhausted", "vpc_cidr_room_available", "no_overlap"}}, Remediation: map[string]any{"risk_level": "medium", "auto_execute_threshold": 0.9, "preconditions": []string{"hcloud VPC listVpcs --vpc_id {{output.vpc_id}} | grep -q ACTIVE"}, "dry_run": "hcloud VPC createSubnet --vpc_id {{output.vpc_id}} --cidr {{output.next_cidr}} --dry-run", "execute": "hcloud VPC createSubnet --vpc_id {{output.vpc_id}} --cidr {{output.next_cidr}}", "verification": "hcloud VPC showSubnet --subnet_id {{output.new_subnet_id}}"}},
			{ID: "VPC-R003", Name: "EIP Bandwidth Upgrade", Trigger: map[string]any{"metric": "eip_bandwidth_util", "condition": "> 85 for 5min", "namespace": "SYS.VPC"}, Diagnosis: map[string]any{"steps": []string{"hcloud CES getMetricData --namespace SYS.VPC --metric_name eip_bandwidth_util --period 60 --filter avg", "hcloud VPC showEip --eip_id {{output.eip_id}}"}, "confidence_factors": []string{"bw_sustained_high", "quota_available", "no_maintenance_window"}}, Remediation: map[string]any{"risk_level": "low", "auto_execute_threshold": 0.9, "preconditions": []string{"hcloud VPC showEip --eip_id {{output.eip_id}} | grep -q ACTIVE"}, "dry_run": "hcloud VPC updateEipBandwidth --eip_id {{output.eip_id}} --bandwidth {{output.bw_mbps}} --dry-run", "execute": "hcloud VPC updateEipBandwidth --eip_id {{output.eip_id}} --bandwidth {{output.bw_mbps}}", "verification": "hcloud VPC showEip --eip_id {{output.eip_id}}"}},
			{ID: "VPC-R004", Name: "ACL Route Drift Repair", Trigger: map[string]any{"metric": "route_drift_count", "condition": "> 0 missing routes for 5min", "namespace": "SYS.VPC"}, Diagnosis: map[string]any{"steps": []string{"hcloud VPC showRouteTable --route_table_id {{output.rt_id}}", "hcloud VPC listVpcPeerings --status ACTIVE"}, "confidence_factors": []string{"routes_missing", "peering_active", "config_known"}}, Remediation: map[string]any{"risk_level": "medium", "auto_execute_threshold": 0.9, "preconditions": []string{"hcloud VPC showRouteTable --route_table_id {{output.rt_id}} | grep -q ACTIVE"}, "dry_run": "hcloud VPC createRoutes --route_table_id {{output.rt_id}} --routes {{output.routes_json}} --dry-run", "execute": "hcloud VPC createRoutes --route_table_id {{output.rt_id}} --routes {{output.routes_json}}", "verification": "hcloud VPC showRouteTable --route_table_id {{output.rt_id}}"}},
		},
	},
	"elb": {
		Patterns: []Pattern{
			{ID: "ELB-FP001", Category: "resource_state", Signature: PatternSig{ErrorCode: "ELB.0001", ErrorMessageRegex: "loadbalancer.*not found|LbNotFound", CommandPattern: "*"}, RootCause: "LB ID invalid or LB deleted", Fix: PatternFix{Strategy: "retry", Action: "hcloud ELB listLoadBalancers --region {{env.HW_REGION_ID}}"}, Prevention: "Cache LB ID at provisioning time"},
			{ID: "ELB-FP002", Category: "resource_state", Signature: PatternSig{ErrorCode: "ELB.0104", ErrorMessageRegex: "backend.*unhealthy|HealthCheckFailed", CommandPattern: "*"}, RootCause: "Backend instance health check failing: port blocked, app down, or SG mismatch", Fix: PatternFix{Strategy: "fallback", Action: "Inspect backend: hcloud ELB listMembers --lb_id {{output.lb_id}}; check SG allows LB->backend traffic on health-check port"}, Prevention: "Validate SG rules + health-check port at LB add time"},
			{ID: "ELB-FP003", Category: "cli_parameter", Signature: PatternSig{ErrorCode: "ELB.0201", ErrorMessageRegex: "listener.*port.*conflict|PortConflict", CommandPattern: "create-listener"}, RootCause: "Port already used by another listener on same LB", Fix: PatternFix{Strategy: "retry", Action: "hcloud ELB listListeners --lb_id {{output.lb_id}}; pick unused port"}, Prevention: "Track listener ports in inventory"},
			{ID: "ELB-FP004", Category: "permission", Signature: PatternSig{ErrorCode: "ELB.0301", ErrorMessageRegex: "Quota exceeded|LbQuota", CommandPattern: "*"}, RootCause: "LB / listener / backend quota exhausted", Fix: PatternFix{Strategy: "halt", Action: "Delegate to huaweicloud-iam-ops for quota raise"}, Prevention: "Check quota via showQuota before bulk LB ops"},
			{ID: "ELB-FP005", Category: "cross_skill", Signature: PatternSig{ErrorCode: "ELB.0401", ErrorMessageRegex: "subnet.*not.*found|invalid subnet", CommandPattern: "create-loadbalancer"}, RootCause: "LB subnet does not exist or in different VPC than backend", Fix: PatternFix{Strategy: "delegate", Action: "Delegate to huaweicloud-vpc-ops to verify subnet; ensure LB subnet + backend subnet share VPC"}, Prevention: "Validate LB subnet + backend subnet VPC alignment"},
			{ID: "ELB-FP006", Category: "runtime", Signature: PatternSig{ErrorCode: "", ErrorMessageRegex: "RequestLimitExceeded|Throttling", CommandPattern: "*"}, RootCause: "API rate limit on ELB control plane", Fix: PatternFix{Strategy: "retry", Action: "Exponential backoff with jitter; respect Retry-After"}, Prevention: "Throttle to <10 req/s for ELB API"},
			{ID: "ELB-FP007", Category: "resource_state", Signature: PatternSig{ErrorCode: "ELB.0501", ErrorMessageRegex: "certificate.*invalid|cert.*expired", CommandPattern: "*"}, RootCause: "ELB HTTPS listener cert expired or uploaded wrong format", Fix: PatternFix{Strategy: "fallback", Action: "Delegate to huaweicloud-kms-ops to upload new cert; update listener: hcloud ELB updateListener --cert_id {{output.new_cert_id}}"}, Prevention: "Alarm cert expiry at 30d via CES"},
			{ID: "ELB-FP008", Category: "resource_state", Signature: PatternSig{ErrorCode: "ELB.0601", ErrorMessageRegex: "backend.*weight.*invalid|WeightInvalid", CommandPattern: "update-member"}, RootCause: "Member weight outside [0, 100]", Fix: PatternFix{Strategy: "retry", Action: "Clamp weight to 0..100; 0 means drain (no new connections)"}, Prevention: "Validate weight range client-side before submit"},
			{ID: "ELB-FP009", Category: "network", Signature: PatternSig{ErrorCode: "ELB.0700", ErrorMessageRegex: "bandwidth.*exceeded|LbBandwidth", CommandPattern: "*"}, RootCause: "LB bandwidth cap reached", Fix: PatternFix{Strategy: "fallback", Action: "Upgrade LB spec: hcloud ELB changeLoadbalancerChargeMode --lb_id {{output.lb_id}} --type payg"}, Prevention: "Monitor CES `lb_bandwidth_util`; alarm at 80%"},
			{ID: "ELB-FP010", Category: "resource_state", Signature: PatternSig{ErrorCode: "ELB.0801", ErrorMessageRegex: "draining.*timeout|ConnectionDrained", CommandPattern: "*"}, RootCause: "Backend removal blocked by lingering connections past drain timeout", Fix: PatternFix{Strategy: "fallback", Action: "Force remove after drain timeout: hcloud ELB batchRemoveMembers --lb_id {{output.lb_id}} --members {{output.member_ids}} --force"}, Prevention: "Tune connection-drain timeout to <300s; reduce stuck connections"},
		},
		Playbooks: []Playbook{
			{ID: "ELB-R001", Name: "Unhealthy Backend Replace", Trigger: map[string]any{"metric": "lb_member_unhealthy", "condition": "1+ backend unhealthy for 3min", "namespace": "SYS.ELB"}, Diagnosis: map[string]any{"steps": []string{"hcloud ELB listMembers --lb_id {{output.lb_id}}", "hcloud CES getMetricData --namespace SYS.ELB --metric_name health_check_status --period 60"}, "confidence_factors": []string{"unhealthy_confirmed", "as_group_available", "backup_in_cbr"}}, Remediation: map[string]any{"risk_level": "medium", "auto_execute_threshold": 0.9, "preconditions": []string{"hcloud ELB showLoadBalancer --lb_id {{output.lb_id}} | grep -q ACTIVE"}, "dry_run": "hcloud ELB batchRemoveMembers --lb_id {{output.lb_id}} --members {{output.unhealthy_ids}} --dry-run", "execute": "hcloud ELB batchRemoveMembers --lb_id {{output.lb_id}} --members {{output.unhealthy_ids}}", "verification": "hcloud CES getMetricData --namespace SYS.ELB --metric_name health_check_status --period 60"}},
			{ID: "ELB-R002", Name: "Backend Weight Drain", Trigger: map[string]any{"metric": "lb_member_error_rate", "condition": "> 50% errors for 5min", "namespace": "SYS.ELB"}, Diagnosis: map[string]any{"steps": []string{"hcloud CES getMetricData --namespace SYS.ELB --metric_name 4xx_rate --period 60", "hcloud ELB listMembers --lb_id {{output.lb_id}}"}, "confidence_factors": []string{"errors_sustained", "instance_isolated", "no_other_traffic_drop"}}, Remediation: map[string]any{"risk_level": "low", "auto_execute_threshold": 0.85, "preconditions": []string{"hcloud ELB listMembers --lb_id {{output.lb_id}} | grep -q {{output.member_id}}"}, "dry_run": "hcloud ELB updateMember --lb_id {{output.lb_id}} --member_id {{output.member_id}} --weight 0 --dry-run", "execute": "hcloud ELB updateMember --lb_id {{output.lb_id}} --member_id {{output.member_id}} --weight 0", "verification": "hcloud CES getMetricData --namespace SYS.ELB --metric_name 4xx_rate --period 60"}},
			{ID: "ELB-R003", Name: "HTTPS Cert Rotation", Trigger: map[string]any{"metric": "cert_expires_in_days", "condition": "< 30 days", "namespace": "SYS.ELB"}, Diagnosis: map[string]any{"steps": []string{"hcloud KMS listCertificates --status ACTIVE", "hcloud ELB listListeners --lb_id {{output.lb_id}} --protocol HTTPS"}, "confidence_factors": []string{"cert_expiring", "new_cert_uploaded", "old_cert_archived"}}, Remediation: map[string]any{"risk_level": "medium", "auto_execute_threshold": 0.95, "preconditions": []string{"hcloud KMS showCertificate --cert_id {{output.new_cert_id}} | grep -q ACTIVE"}, "dry_run": "hcloud ELB updateListener --lb_id {{output.lb_id}} --listener_id {{output.listener_id}} --cert_id {{output.new_cert_id}} --dry-run", "execute": "hcloud ELB updateListener --lb_id {{output.lb_id}} --listener_id {{output.listener_id}} --cert_id {{output.new_cert_id}}", "verification": "hcloud ELB showListener --lb_id {{output.lb_id}} --listener_id {{output.listener_id}}"}},
			{ID: "ELB-R004", Name: "Listener Rule Auto-Scale Out", Trigger: map[string]any{"metric": "lb_active_connections", "condition": "> 80% of capacity", "namespace": "SYS.ELB"}, Diagnosis: map[string]any{"steps": []string{"hcloud CES getMetricData --namespace SYS.ELB --metric_name active_connections --period 60", "hcloud AS listScalingGroups --lb_id {{output.lb_id}}"}, "confidence_factors": []string{"connections_saturated", "as_group_healthy", "quota_available"}}, Remediation: map[string]any{"risk_level": "medium", "auto_execute_threshold": 0.85, "preconditions": []string{"hcloud AS showScalingGroup --scaling_group_id {{output.sg_id}} | grep -q ACTIVE"}, "dry_run": "hcloud AS scalingInstances --scaling_group_id {{output.sg_id}} --dry-run", "execute": "hcloud AS executeScalingAction --scaling_group_id {{output.sg_id}} --action ADD --instance_number 1", "verification": "hcloud AS showScalingGroup --scaling_group_id {{output.sg_id}}"}},
		},
	},
	"cce": {
		Patterns: []Pattern{
			{ID: "CCE-FP001", Category: "resource_state", Signature: PatternSig{ErrorCode: "CCE.0140", ErrorMessageRegex: "cluster.*not found|ClusterNotFound", CommandPattern: "*"}, RootCause: "Cluster ID invalid or cluster deleted", Fix: PatternFix{Strategy: "retry", Action: "hcloud CCE listClusters --region {{env.HW_REGION_ID}}"}, Prevention: "Cache cluster ID at provisioning time"},
			{ID: "CCE-FP002", Category: "resource_state", Signature: PatternSig{ErrorCode: "CCE.0150", ErrorMessageRegex: "node.*not.*ready|NodeNotReady", CommandPattern: "*"}, RootCause: "Worker node unreachable: kubelet down or network partition", Fix: PatternFix{Strategy: "fallback", Action: "Inspect node: hcloud CCE showNode --cluster_id {{output.cluster_id}} --node_id {{output.node_id}}; check kubelet + network"}, Prevention: "Monitor node_ready via CES; alarm at <0.8"},
			{ID: "CCE-FP003", Category: "resource_state", Signature: PatternSig{ErrorCode: "CCE.0200", ErrorMessageRegex: "pod.*Pending|PodSchedulingFailed", CommandPattern: "*"}, RootCause: "Insufficient cluster resources or node selector mismatch", Fix: PatternFix{Strategy: "fallback", Action: "Describe pod events: hcloud CCE showPod --cluster_id {{output.cluster_id}} --namespace {{output.ns}} --name {{output.pod}}; check node capacity"}, Prevention: "Set HPA + resource requests/limits on all deployments"},
			{ID: "CCE-FP004", Category: "permission", Signature: PatternSig{ErrorCode: "CCE.0300", ErrorMessageRegex: "Quota exceeded|ClusterQuota", CommandPattern: "*"}, RootCause: "Cluster / node / pod quota exhausted", Fix: PatternFix{Strategy: "halt", Action: "Delegate to huaweicloud-iam-ops for quota raise"}, Prevention: "Check quota via showQuota before bulk node add"},
			{ID: "CCE-FP005", Category: "cross_skill", Signature: PatternSig{ErrorCode: "CCE.0400", ErrorMessageRegex: "image.*not found|ImageNotFound", CommandPattern: "*"}, RootCause: "SWR image tag missing or auth failed", Fix: PatternFix{Strategy: "delegate", Action: "Delegate to huaweicloud-swr-ops to verify/push image; verify pull secret"}, Prevention: "Pre-pull images to local registry; cache tags"},
			{ID: "CCE-FP006", Category: "runtime", Signature: PatternSig{ErrorCode: "", ErrorMessageRegex: "RequestLimitExceeded|Throttling", CommandPattern: "*"}, RootCause: "API rate limit on CCE control plane", Fix: PatternFix{Strategy: "retry", Action: "Exponential backoff with jitter"}, Prevention: "Throttle to <5 req/s"},
			{ID: "CCE-FP007", Category: "resource_state", Signature: PatternSig{ErrorCode: "CCE.0501", ErrorMessageRegex: "PersistentVolumeClaim.*Pending|PvcPending", CommandPattern: "*"}, RootCause: "No available PV or storage class misconfigured", Fix: PatternFix{Strategy: "fallback", Action: "hcloud CCE listPvcs --cluster_id {{output.cluster_id}}; check storage class + CSI driver"}, Prevention: "Validate storage class in deployment manifest"},
			{ID: "CCE-FP008", Category: "resource_state", Signature: PatternSig{ErrorCode: "CCE.0601", ErrorMessageRegex: "addons.*install.*failed|AddonInstallFailed", CommandPattern: "install-addon"}, RootCause: "CCE addon installation failed: chart repo unreachable or version mismatch", Fix: PatternFix{Strategy: "retry", Action: "Retry with explicit version: hcloud CCE installAddon --cluster_id {{output.id}} --addon {{output.name}} --version {{output.v}}"}, Prevention: "Pin addon versions; mirror chart repos"},
			{ID: "CCE-FP009", Category: "network", Signature: PatternSig{ErrorCode: "CCE.0700", ErrorMessageRegex: "NetworkPolicy.*conflict|NetworkPolicyInvalid", CommandPattern: "*"}, RootCause: "Conflicting NetworkPolicy rules blocking pod-to-pod traffic", Fix: PatternFix{Strategy: "fallback", Action: "List NetworkPolicies: kubectl get networkpolicy -A; dedupe overlapping selectors"}, Prevention: "Adopt deny-by-default + explicit allow model"},
			{ID: "CCE-FP010", Category: "runtime", Signature: PatternSig{ErrorCode: "CCE.0800", ErrorMessageRegex: "etcd.*unhealthy|EtcdNotHealthy", CommandPattern: "*"}, RootCause: "Control plane etcd unhealthy: disk pressure or quorum loss", Fix: PatternFix{Strategy: "halt", Action: "Escalate to CCE support; do not attempt repair (risk of data loss)"}, Prevention: "Monitor etcd metrics; alarm on leader changes > 5/min"},
		},
		Playbooks: []Playbook{
			{ID: "CCE-R001", Name: "NotReady Node Cordon + Drain", Trigger: map[string]any{"metric": "node_ready", "condition": "== 0 (NotReady) for 3min", "namespace": "SYS.CCE"}, Diagnosis: map[string]any{"steps": []string{"hcloud CCE showNode --cluster_id {{output.cluster_id}} --node_id {{output.node_id}}", "hcloud CES getMetricData --namespace SYS.CCE --metric_name node_ready --period 60"}, "confidence_factors": []string{"node_notready_confirmed", "as_group_available", "no_etcd_pressure"}}, Remediation: map[string]any{"risk_level": "medium", "auto_execute_threshold": 0.9, "preconditions": []string{"hcloud CCE showNode --cluster_id {{output.cluster_id}} --node_id {{output.node_id}} | grep -q NotReady"}, "dry_run": "hcloud CCE cordonNode --cluster_id {{output.cluster_id}} --node_id {{output.node_id}} --dry-run", "execute": "hcloud CCE cordonNode --cluster_id {{output.cluster_id}} --node_id {{output.node_id}}", "verification": "hcloud CCE showNode --cluster_id {{output.cluster_id}} --node_id {{output.node_id}} | grep -q SchedulingDisabled"}},
			{ID: "CCE-R002", Name: "Pending Pod Auto-Scale", Trigger: map[string]any{"metric": "pending_pods", "condition": "> 5 pending for 5min", "namespace": "SYS.CCE"}, Diagnosis: map[string]any{"steps": []string{"hcloud CCE listPods --cluster_id {{output.cluster_id}} --phase Pending", "hcloud CCE showNode --cluster_id {{output.cluster_id}} --limit 50"}, "confidence_factors": []string{"resources_exhausted", "as_group_healthy", "quota_available"}}, Remediation: map[string]any{"risk_level": "medium", "auto_execute_threshold": 0.85, "preconditions": []string{"hcloud AS showScalingGroup --scaling_group_id {{output.sg_id}} | grep -q ACTIVE"}, "dry_run": "hcloud AS scalingInstances --scaling_group_id {{output.sg_id}} --dry-run", "execute": "hcloud AS executeScalingAction --scaling_group_id {{output.sg_id}} --action ADD --instance_number 1", "verification": "hcloud CCE listPods --cluster_id {{output.cluster_id}} --phase Pending"}},
			{ID: "CCE-R003", Name: "Addon Restart", Trigger: map[string]any{"metric": "addon_health", "condition": "== Abnormal for 5min", "namespace": "SYS.CCE"}, Diagnosis: map[string]any{"steps": []string{"hcloud CCE listAddons --cluster_id {{output.cluster_id}}", "hcloud CES getMetricData --namespace SYS.CCE --metric_name addon_status --period 60"}, "confidence_factors": []string{"addon_abnormal", "version_pinned", "no_active_upgrade"}}, Remediation: map[string]any{"risk_level": "low", "auto_execute_threshold": 0.85, "preconditions": []string{"hcloud CCE showAddon --cluster_id {{output.cluster_id}} --addon_name {{output.name}} | grep -q Abnormal"}, "dry_run": "hcloud CCE restartAddon --cluster_id {{output.cluster_id}} --addon_name {{output.name}} --dry-run", "execute": "hcloud CCE restartAddon --cluster_id {{output.cluster_id}} --addon_name {{output.name}}", "verification": "hcloud CCE showAddon --cluster_id {{output.cluster_id}} --addon_name {{output.name}} | grep -q Running"}},
			{ID: "CCE-R004", Name: "Evicted Pod Workload Restart", Trigger: map[string]any{"metric": "evicted_pods", "condition": "> 3 in 5min", "namespace": "SYS.CCE"}, Diagnosis: map[string]any{"steps": []string{"hcloud CCE listPods --cluster_id {{output.cluster_id}} --phase Failed", "hcloud CCE showEvents --cluster_id {{output.cluster_id}} --reason Evicted --limit 20"}, "confidence_factors": []string{"disk_pressure_confirmed", "node_drainable", "quota_available"}}, Remediation: map[string]any{"risk_level": "medium", "auto_execute_threshold": 0.9, "preconditions": []string{"hcloud CCE showNode --cluster_id {{output.cluster_id}} --node_id {{output.evictor_node_id}} | grep -q MemoryPressure\\|DiskPressure"}, "dry_run": "hcloud CCE cordonNode --cluster_id {{output.cluster_id}} --node_id {{output.evictor_node_id}} --dry-run", "execute": "hcloud CCE cordonNode --cluster_id {{output.cluster_id}} --node_id {{output.evictor_node_id}}", "verification": "hcloud CCE listPods --cluster_id {{output.cluster_id}} --node_id {{output.evictor_node_id}} --phase Failed"}},
		},
	},
}

// NowISO is the current UTC timestamp in ISO-8601 (Z) form.
// Mirrors Python's _now_iso() in gen_skill_knowledge.py.
func NowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// WriteSkillAssets writes failure_patterns.json + remediation-playbooks.json
// for one product under <root>/huaweicloud-<short>-ops/assets/.
//
// Field order matches scripts/gen_skill_knowledge.py output exactly:
//
//	failure_patterns.json: $schema, skill_id, patterns, meta  (meta last)
//	remediation-playbooks.json: $schema, skill_id, playbooks
func WriteSkillAssets(root, short string) error {
	skillID := "huaweicloud-" + short + "-ops"
	targetDir := filepath.Join(root, skillID, "assets")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", targetDir, err)
	}
	payload, ok := Products[short]
	if !ok {
		return fmt.Errorf("unknown skill short=%q", short)
	}
	now := NowISO()
	// failure-patterns/v1 — meta goes LAST, no playbook_ref in fix block
	fp := struct {
		Schema   string    `json:"$schema"`
		SkillID  string    `json:"skill_id"`
		Patterns []Pattern `json:"patterns"`
		Meta     any       `json:"meta"`
	}{
		Schema:   "failure-patterns/v1",
		SkillID:  skillID,
		Patterns: payload.Patterns,
		Meta: map[string]any{
			"total_patterns":         len(payload.Patterns),
			"last_aggregation":       now,
			"source_traces_analyzed": 0,
		},
	}
	// remediation-playbooks/v1
	rp := struct {
		Schema    string     `json:"$schema"`
		SkillID   string     `json:"skill_id"`
		Playbooks []Playbook `json:"playbooks"`
	}{
		Schema:    "remediation-playbooks/v1",
		SkillID:   skillID,
		Playbooks: payload.Playbooks,
	}
	if err := writeJSON(filepath.Join(targetDir, "failure_patterns.json"), fp); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(targetDir, "remediation-playbooks.json"), rp); err != nil {
		return err
	}
	return nil
}

// GenerateAll writes assets for every product in Products.
func GenerateAll(root string) (int, error) {
	count := 0
	for short := range Products {
		if err := WriteSkillAssets(root, short); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// PatternIDPrefix is exposed for tests; not part of the public CLI.
func PatternIDPrefix(short string) string {
	return strings.ToUpper(short)
}

// PitfallReportEntry is one line in the common-pitfalls report.
type PitfallReportEntry struct {
	Skill      string `json:"skill"`
	ID         string `json:"id"`
	Category   string `json:"category"`
	RootCause  string `json:"root_cause"`
	Prevention string `json:"prevention"`
}

// GeneratePitfallReport scans all failure_patterns.json under root,
// aggregates prevention lessons cross-skill, and writes a markdown
// reference to huaweicloud-skill-generator/references/common-pitfalls.md.
// The report is ordered by category then frequency.
func GeneratePitfallReport(root string) (int, error) {
	skillDirs, err := os.ReadDir(root)
	if err != nil {
		return 0, err
	}
	var entries []PitfallReportEntry
	for _, d := range skillDirs {
		if !d.IsDir() || !strings.HasPrefix(d.Name(), "huaweicloud-") || !strings.HasSuffix(d.Name(), "-ops") {
			continue
		}
		fpPath := filepath.Join(root, d.Name(), "assets", "failure_patterns.json")
		raw, err := os.ReadFile(fpPath)
		if err != nil {
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			continue
		}
		patterns, _ := data["patterns"].([]any)
		short := strings.TrimPrefix(strings.TrimPrefix(d.Name(), "huaweicloud-"), "-ops")
		for _, p := range patterns {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			cat, _ := pm["category"].(string)
			rootCause, _ := pm["root_cause"].(string)
			prevention, _ := pm["prevention"].(string)
			if prevention == "" {
				continue
			}
			entries = append(entries, PitfallReportEntry{
				Skill:      short,
				ID:         pm["id"].(string),
				Category:   cat,
				RootCause:  rootCause,
				Prevention: prevention,
			})
		}
	}

	// Group by category for ordered output.
	byCat := map[string][]PitfallReportEntry{}
	for _, e := range entries {
		byCat[e.Category] = append(byCat[e.Category], e)
	}
	var cats []string
	for c := range byCat {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	// Write markdown report.
	outPath := filepath.Join(root, "huaweicloud-skill-generator", "references", "common-pitfalls.md")
	var buf bytes.Buffer
	buf.WriteString("# Common Pitfalls — Cross-Skill Failure Patterns\n\n")
	buf.WriteString("> Auto-generated by `hwcloud-skillcheck learning pitfall-report`. ")
	buf.WriteString("Do not edit manually; changes will be overwritten.\n\n")
	buf.WriteString("This file aggregates prevention lessons from all `failure_patterns.json` entries\n")
	buf.WriteString("across skills. Generator must reference this before writing execution flows\n")
	buf.WriteString("to avoid repeating known failure patterns.\n\n")
	buf.WriteString("## Categories\n\n")
	for _, cat := range cats {
		es := byCat[cat]
		buf.WriteString(fmt.Sprintf("### %s (%d patterns)\n\n", cat, len(es)))
		buf.WriteString("| Skill | ID | Root Cause | Prevention |\n")
		buf.WriteString("|-------|----|------------|------------|\n")
		for _, e := range es {
			buf.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				e.Skill, e.ID, e.RootCause, e.Prevention))
		}
		buf.WriteString("\n")
	}
	buf.WriteString(fmt.Sprintf("---\n*Generated at %s · %d patterns from %d skills*\n",
		NowISO(), len(entries), len(cats)))

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		return 0, err
	}
	return len(entries), nil
}
