# KMS Monitoring & Alerts — Huawei Cloud Key Management Service

## Metrics (via CES)

| Metric | Namespace | Dimension | Meaning |
|---|---|---|---|
| `kms_key_api_invoke_count` | SYS.KMS | key_id | API call count per key |
| `kms_key_api_fail_count` | SYS.KMS | key_id | Failed API calls per key |
| `kms_key_encrypt_count` | SYS.KMS | key_id | Encrypt operations |
| `kms_key_decrypt_count` | SYS.KMS | key_id | Decrypt operations |
| `kms_hsm_request_latency` | SYS.KMS | region | HSM operation latency |
| `kms_quota_usage_ratio` | SYS.KMS | region | CMK count / quota |

## Alarm Templates

```yaml
# KMS API error rate alarm
alarm_name: "kms-api-error-rate"
namespace: SYS.KMS
metric: kms_key_api_fail_count
threshold: 10  # per key per 5 min
comparison: ">="
period: 300  # 5 min
statistic: sum
actions: SMN critical topic

# KMS quota usage warning
alarm_name: "kms-quota-usage"
namespace: SYS.KMS
metric: kms_quota_usage_ratio
threshold: 0.8  # 80%
comparison: ">="
period: 3600  # 1 hour
statistic: average
actions: SMN warning topic
```

## Cost Metrics

| Item | Metric | Alert |
|---|---|---|
| CMK count | `kms_key_count` | > 80% of quota |
| API call volume | `kms_key_api_invoke_count` | Spike detection |
| Failed operations | `kms_key_api_fail_count` | Anomaly detection |

> Delegate alarm wiring to `huaweicloud-ces-ops`.
