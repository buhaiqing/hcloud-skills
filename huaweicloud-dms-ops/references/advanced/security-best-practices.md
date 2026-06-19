# DMS SecOps — Kafka / RabbitMQ Security Deep Dive

> Advanced security patterns for Distributed Message Service.
> Load when configuring SASL/SSL, IAM agency chains, or queue isolation.

## 1. Authentication Matrix

| Protocol | SASL | TLS | mTLS | Notes |
|---------|------|-----|------|-------|
| Kafka  | SASL_PLAINTEXT / SASL_SSL | optional | recommended for cross-account | use SCRAM-SHA-512 |
| RabbitMQ | PLAIN / EXTERNAL | optional | recommended | bind to dedicated IAM agency |
| RocketMQ | ACL + AK/SK | required | n/a | use ACL + tenant token |

## 2. Network Isolation

- Bind DMS endpoint to private VPC subnet; no public EIP
- Restrict security group ingress to application CIDR only
- For cross-account producers: dedicated IAM agency with `dms:Produce*`

## 3. Topic / Queue Hardening

- Per-environment topic naming: `prod.<service>.<event>`
- Quota per topic: `retention.ms = 7d`, `retention.bytes = 10 GB`
- Enable message trace for forensic queries; export to LTS

## 4. Disaster Recovery

- Mirror Maker 2 / RabbitMQ Shovel for cross-cluster replication
- Document `RTO ≤ 15 min` for critical topics
- Quarterly DR drill: failover consumer to DR cluster

> **Security-Sensitive**: topic deletion, consumer group reset, or quota
> increase MUST require explicit operator confirmation. Production topic
> mutations must run inside a documented maintenance window.