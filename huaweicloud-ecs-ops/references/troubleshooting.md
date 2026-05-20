# Troubleshooting — Huawei Cloud ECS

## Common API Error Codes

| Code | HTTP | Meaning | Agent Action |
|------|------|---------|--------------|
| `Ecs.0801` | 400 | Insufficient resource | Downgrade flavor or request quota |
| `Ecs.0804` | 400 | Image not found | Verify image ID, list available |
| `Ecs.0805` | 400 | Flavor not found | Verify flavor, list available |
| `Ecs.0820` | 400 | Security group not found | Create or verify SG ID |
| `Ecs.4600` | 400 | VPC/subnet not found | Create via VPC skill first |
| `Ecs.4601` | 409 | IP address conflict | Use different subnet or auto-allocate |
| `Ecs.4603` | 409 | Instance name duplicate | Choose unique name |
| `Ecs.4610` | 403 | Quota exceeded | Request quota increase |
| `Ecs.4615` | 400 | AZ insufficient resources | Try different AZ |
| `Ecs.4620` | 400 | Invalid parameter | Verify params against API docs |
| `Ecs.4625` | 404 | Server not found | Verify server ID |
| `Ecs.4630` | 409 | Invalid server state | Wait for state transition to complete |
| `Ecs.4640` | 403 | Insufficient balance | Recharge account |
| `Ecs.CloudCell.403` | 403 | Agent not installed | Install via CLI or marketplace |
| `Ecs.CloudCell.500` | 500 | Remote exec failed | Check network + agent status |

## Diagnostic Order

1. **Verify instance existence**: `ShowServerDetail(server_id)`
2. **Check instance state**: `status` field (`ACTIVE`, `SHUTOFF`, `BUILD`, `ERROR`)
3. **Check job status**: `ShowJobStatus(job_id)` for async operations
4. **Verify Cloud-Cell Agent**: `ShowServerCloudCellDetail` → `is_install` + `status`
5. **Check regional endpoint**: confirm `RegionId` consistency
6. **Verify CLI metadata coverage**: `hcloud ecs --help`
7. **Check CES metrics**: CPU, memory, disk usage for resource pressure
8. **Check security group rules**: inbound SSH (22), outbound HTTPS (443) for agent

## Cloud-Cell Agent Issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| `is_install: false` | Agent not installed | Run install via marketplace or manual script |
| `status: STOPPED` | Agent crashed or disabled | Restart agent on ECS |
| `status: ERROR` | Connectivity or config issue | Check outbound HTTPS, IAM permissions |
| Command timeout | Network or agent overloaded | Increase timeout, check ECS load |
| Upload fails | Disk full or permission denied | Free space, check file path permissions |

## Instance Unreachable

```
Instance → Security Group → VPC Route → ELB → Internet
   │            │              │          │
   ▼            ▼              ▼          ▼
 SSH open?    Rule 22 OK   Route correct  ELB healthy
 SG allows    Inbound 22  Default exists  Backend active
```

1. Verify security group allows SSH (22) from source IP
2. Verify VPC route table has correct routes
3. Check ECS system status via Cloud-Cell command: `systemctl status sshd`
4. Verify instance has public IP or EIP bound
5. Check NACL rules if configured
