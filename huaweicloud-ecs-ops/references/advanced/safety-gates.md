# ECS Safety Gates — High-Risk Operation Controls

> Advanced safety controls for Elastic Cloud Server.
> Load when issuing destructive instance operations or fleet-wide changes.

## 1. Destructive Operation Catalogue

| Operation | Risk class | Default gate |
|-----------|-----------|--------------|
| `DeleteServers` | irreversible (data on data disk) | snapshot first + confirmation |
| `StopServers` (force) | may corrupt disk | grace period + notification |
| `ResizeServers` | requires reboot | maintenance window |
| `ResetPassword` | SSH key rotation | confirm new key delivery |
| `MigrateServer` | cross-host | drain + smoke test |

## 2. Safety Gate Workflow

1. **Inventory**: list all affected instance IDs
2. **Snapshot**: trigger `CreateServerGroupSnapshot` for system + data disks
3. **Confirm**: collect `{{user.confirm_destructive}}` per instance or batch
4. **Execute**: dry-run first, then apply
5. **Verify**: poll `status` until `SHUTOFF` / `ACTIVE`
6. **Audit**: emit CTS event + surface `{{output.ecs_change_record}}`

## 3. Cross-Skill Delegation

- `huaweicloud-ecs-ops → huaweicloud-vpc-ops` for security group changes
- `huaweicloud-ecs-ops → huaweicloud-evs-ops` for disk operations
- `huaweicloud-ecs-ops → huaweicloud-iam-ops` for agency / AK-SK rotation

> **Security-Sensitive**: destructive operations above MUST pass the Safety
> Gate. Always snapshot before delete; never delete more than 1 instance per
> minute without an explicit maintenance window.