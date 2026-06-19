# IAM SecOps — Least-Privilege & Zero-Trust Deep Dive

> Advanced IAM security patterns layered below the runbook.
> Load when designing custom policies, federation, or AK/SK rotation flows.

## 1. Least-Privilege Template

```json
{
  "Version": "1.1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["<product>:<ReadOnlyOp>"],
      "Resource": ["acs:<product>:*:*:resource/*"]
    }
  ]
}
```

- One IAM User per Skill executor (no shared AK/SK)
- MFA enforced for all interactive operations
- AK/SK rotation: ≤ 90 days; rotate immediately on personnel change

## 2. Federation Patterns

- SAML 2.0 federation with corporate IdP
- Attribute-based access: `tags/cost-center` mapped to permission scope
- JIT elevation via STS token (≤ 1 hour lifetime)

## 3. Agency Chains

| Scenario | Pattern |
|----------|---------|
| Cross-account ECS → OBS | ECS agency with `obs:PutObject` on target bucket only |
| Cross-region DR | dedicated DR agency, KMS-encrypted evidence |
| Third-party integration | STS temporary credentials + IP whitelist |

## 4. Audit & Compliance

- Enable CTS recorder for IAM events; ≥ 365-day retention
- Quarterly access review: remove unused AK/SK within 7 days of detection
- Alert on policy changes via CES rule `IAM:policy:update`

> **Security-Sensitive**: `DeleteUser`, `DeleteAccessKey`, `DisableUser`, and
> `DetachPolicy` operations MUST require explicit operator confirmation and
> preserve CTS evidence. The agent MUST surface `{{output.iam_change_record}}`
> for every privileged mutation.