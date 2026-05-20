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
