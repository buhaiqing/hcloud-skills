# SWR SecOps — Container Registry Security Deep Dive

> Advanced security patterns for Software Repository for Containers.
> Load when designing image signing, vulnerability scanning, or cross-account
> registry sharing.

## 1. Image Signing & Verification

- Sign images with `cosign` (keyless, OIDC-based) at CI time
- Verify signature in admission webhook before cluster pull
- Reject unsigned or stale (> 90 days) images

## 2. Vulnerability Scanning

| Severity | SLA | Action |
|----------|-----|--------|
| Critical | ≤ 24 h | block promotion |
| High | ≤ 7 d | warn + ticket |
| Medium / Low | best effort | backlog |

## 3. Network Isolation

- Registry endpoint in private VPC subnet; no public EIP
- Cross-account pulls use dedicated IAM agency with `swr:Pull*`
- Restrict per-namespace by `repository_policy`

## 4. Retention & Cleanup

- Keep `latest` + last 5 stable tags per image
- Auto-delete untagged images after 7 days
- Quarantine malware-flagged images in isolated namespace

> **Security-Sensitive**: image deletion, organization transfer, and policy
> changes MUST require explicit operator confirmation. Production image
> mutations must run inside a documented maintenance window.