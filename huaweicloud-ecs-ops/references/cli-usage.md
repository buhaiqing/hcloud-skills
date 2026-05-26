# CLI Usage — Huawei Cloud ECS (`hcloud`)

## Install and Config

```bash
# Install KooCLI (official binary)
curl -sSL https://cn-north-4.myhuaweicloud.com/cli/latest/hcloud_install.sh -o ./hcloud_install.sh && bash ./hcloud_install.sh -y

# Configure credentials
hcloud init

# Or via environment variables
export HW_ACCESS_KEY_ID="{{env.HW_ACCESS_KEY_ID}}"
export HW_SECRET_ACCESS_KEY="{{env.HW_SECRET_ACCESS_KEY}}"
export HW_REGION_ID="{{env.HW_REGION_ID}}"
export HW_PROJECT_ID="{{env.HW_PROJECT_ID}}"
```

## Conventions (Agent Execution)

- `hcloud ecs` is the ECS service subcommand
- Flags: `--region`, `--server-id`, `--name`, `--flavor-id`, `--image-id`
- JSON output for agent parsing: add `--output json`
- JMESPath extraction: `--output json | jq '.path.to.field'`

## Command Map

| Goal | CLI Invocation | Notes |
|------|---------------|-------|
| List instances | `hcloud ecs list-instances --region CN` | All instances in region |
| Describe instance | `hcloud ecs describe-server --server-id ID` | Single instance details |
| Create instance | `hcloud ecs create-server --region CN --name NAME --flavor-id F --image-id I` | Returns `job_id` |
| Start instance | `hcloud ecs start-server --server-id ID` | — |
| Stop instance | `hcloud ecs stop-server --server-id ID --type SOFT` | SOFT or OS-STOP |
| Delete instance | `hcloud ecs delete-server --server-id ID` | Irreversible |
| Resize instance | `hcloud ecs resize-server --server-id ID --new-flavor-id F` | May require reboot |
| List flavors | `hcloud ecs list-flavors --region CN` | Available compute specs |
| List quotas | `hcloud ecs list-quotas --region CN` | Current quota usage |
| Execute CloudCell | `hcloud ecs execute-cloud-cell-command --server-id ID --command CMD` | Remote exec |
| Upload file | `hcloud ecs cloud-cell-upload --server-id ID --local-path P --remote-path P` | File to ECS |
| Download file | `hcloud ecs cloud-cell-download --server-id ID --remote-path P --local-path P` | File from ECS |
| Describe CloudCell | `hcloud ecs describe-server-cloud-cell --server-id ID` | Agent status |
| List costs | `hcloud bss list-bills --resource-type ecs --region CN` | Monthly ECS billing |
| Query daily cost | `hcloud bss query-daily-cost --resource-id ID` | Daily cost breakdown |
| List orders | `hcloud bss list-orders --resource-type ecs` | Order history |

## CLI vs API Coverage Gap

| Operation (API) | CLI Available | Notes |
|-----------------|--------------|-------|
| CreateServer | ✅ `create-server` | Full support |
| CreateServers (batch) | ✅ | Via count parameter |
| ShowServerDetail | ✅ `describe-server` | — |
| ListServersDetail | ✅ `list-instances` | Paginated |
| BatchStartServers | ✅ `start-server` | — |
| BatchStopServers | ✅ `stop-server` | — |
| BatchDeleteServers | ✅ `delete-server` | — |
| ResizeServer | ✅ `resize-server` | — |
| ShowJobStatus | ✅ `describe-job` | Poll async operations |
| AttachVolume | ⚠️ Partial via EVS CLI | Use EVS specific command |
| CloudCell execute | ⚠️ Verify version | May need latest CLI |
| CloudCell upload/download | ⚠️ Verify version | File transfer support |

## JSON Output Paths

| Operation | Key Path | Description |
|-----------|---------|-------------|
| CreateServer | `$.job_id` | Async job ID for polling |
| DescribeServer | `$.server.id` | Instance UUID |
| DescribeServer | `$.server.status` | Active/Stopped/Building |
| DescribeServer | `$.server.addresses` | Network interfaces with IPs |
| DescribeServer | `$.server.flavor.id` | Current flavor UUID |
| DescribeServer | `$.server.image.id` | Image UUID |
| ListInstances | `$.servers[].id` | Array of instance IDs |
| DescribeCloudCell | `$.server_cloud_cell_detail.is_install` | Agent installed |
| DescribeCloudCell | `$.server_cloud_cell_detail.status` | RUNNING/STOPPED/ERROR |

## Cost & Billing Commands

The BSS (Business Support System) CLI provides cost analysis and billing management for ECS resources.

### Common Cost Operations

```bash
# List monthly ECS bills
hcloud bss list-bills --resource-type ecs --region CN --cycle 2024-01

# Query daily cost breakdown for specific instance
hcloud bss query-daily-cost --resource-id "ecs-instance-id" --start-time 2024-01-01 --end-time 2024-01-31

# List all ECS-related orders
hcloud bss list-orders --resource-type ecs --status completed

# Check subscription renewal status
hcloud bss query-subscription --resource-id "ecs-instance-id"

# View quota balance for subscription resources
hcloud bss query-quota --product-line ecs
```

### Cost Analysis Output Paths

| Operation | Key Path | Description |
|-----------|---------|-------------|
| ListBills | `$.bills[].bill_id` | Monthly bill identifier |
| ListBills | `$.bills[].total_amount` | Total monthly cost (CNY) |
| ListBills | `$.bills[].resource_ids[]` | ECS instance IDs in bill |
| QueryDailyCost | `$.daily_costs[].amount` | Daily cost amount |
| QueryDailyCost | `$.daily_costs[].resource_id` | Instance ID |
| ListOrders | `$.orders[].order_id` | Order identifier |
| ListOrders | `$.orders[].status` | Order status (completed/pending) |

### FinOps Integration

Use BSS commands with CES metrics for idle resource detection:

```bash
# Identify low-utilization instances (CPU < 10% for 7 days)
hcloud ecs list-instances --output json | jq '.servers[].id' | while read id; do
  utilization=$(hcloud ces show-metric-data --namespace SYS.ECS --metric_name cpu_util --dim.0=instance_id,$id --output json | jq '.datapoints[-1].average')
  if [ "$utilization" -lt 10 ]; then
    # Get billing info for recommendation
    hcloud bss query-daily-cost --resource-id "$id" --output json | jq '{resource_id: .daily_costs[0].resource_id, daily_cost: .daily_costs[0].amount}'
  fi
done
```

### Billing Model Detection

```bash
# Determine billing model from instance metadata
hcloud ecs describe-server --server-id ID --output json | jq '{billing_type: .server.metadata.__billing_type, charging_mode: .server.charging_mode}'
# Values: "0" = pay-per-use (按需), "1" = subscription (包年包月), "2" = spot (竞价)
```
