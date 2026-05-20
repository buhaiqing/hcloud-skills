# Governance & Adversarial Review

> **Purpose:** Minimal adversarial review framework for generated skills. Catches destructive-action shortcuts, credential leaks, API hallucination, and gaps across FinOps/SecOps/AIOps before merge.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20
> **Status:** MANDATORY — no skill may be merged without passing this review

---

## 1. Review Process

### 1.1 Review Stages

| Stage | Focus | Artifact |
|-------|-------|----------|
| **Stage 1: Technical Review** | API fidelity, CLI accuracy, security across five pillars | Technical sign-off |
| **Stage 2: UX Review** | Onboarding, interaction, feedback, error handling | UX checklist |
| **Stage 3: Adversarial Review** | Destructive gates, credential safety, FinOps/SecOps/AIOps coverage | Adversarial report |

---

## 2. Adversarial Scenarios

### 2.1 Security Scenarios

#### Scenario 1: Destructive without Confirmation
**Test:** Search all delete/destroy/remove operations.
**Pass:** Every destructive operation has explicit user confirmation with resource identifier.

#### Scenario 2: Credential Echo / Masking Failure
**Test:** Search all execution flows for `HW_SECRET_ACCESS_KEY`, `SecretAccessKey`, or any credential value output.
**Pass:**
1. No secret value printed, logged, or echoed in any path.
2. ALL credential-related output uses `***` / `<masked>`.
3. Verification scripts check existence only (`test -n "$var"`).
4. JIT Go SDK scripts never print config struct or secret fields.

#### Scenario 3: API Hallucination
**Test:** Cross-reference all operationIds, field names, JSON paths against OpenAPI.
**Pass:** 100% traceability to OpenAPI or verified CLI output.

### 2.2 Resilience Scenarios

| Scenario | Test | Pass Criteria |
|----------|------|--------------|
| 4. Idempotency Gap | Execute same create twice | Behavior documented (error/reuse/duplicate) |
| 5. Throttling Blindness | Retry logic for 429 | Exponential backoff + max retries |
| 6. Region Drift | Search hardcoded regions | All use `{{env.*}}` or `{{user.*}}` |
| 7. Error Recovery Gap | Missing error patterns | Each error has documented recovery |

### 2.3 FinOps Scenarios

| Scenario | Test | Pass Criteria |
|----------|------|--------------|
| 8. Missing Cost Optimization | Search for billing/cost sections | Billing model comparison table present |
| 9. No Idle Detection | Check waste detection patterns | Idle resource detection pattern documented |
| 10. No Right-Sizing Guidance | Check utilization→recommendation mapping | Right-sizing matrix present |
| 11. No Unit Economics | Check for unit cost metrics | At least 1 unit cost metric (cost/request or cost/vCPU) defined |
| 12. No Cost Anomaly Detection | Check for cost anomaly rules | Cost anomaly detection logic documented |
| 13. No Reserved Coverage Analysis | Check for RI/包年包月 coverage | Coverage analysis template present |
| 14. No TCO Model | Check for total cost of ownership | TCO model with cost breakdown documented |

### 2.4 SecOps Scenarios

| Scenario | Test | Pass Criteria |
|----------|------|--------------|
| 15. Missing IAM Minimum | Search for IAM/PAM/RAM permissions table | Minimum permissions table documented |
| 16. No Network Isolation | Check for VPC/security group patterns | VPC isolation guidance present |
| 17. No Encryption Guidance | Check data security section | Encryption at rest/in transit documented |
| 18. Missing Threat Detection | Check HSS/WAF integration triggers | Threat detection triggers defined (applicable skills) |
| 19. No Zero Trust Alignment | Check for zero trust architecture guidance | Zero trust principles documented |
| 20. No Incident Response Runbook | Check for security incident response flow | Incident response runbook with severity levels |
| 21. No Supply Chain Security | Check for SDK dependency security | SDK vulnerability scanning + SBOM guidance |
| 22. No Key Lifecycle Management | Check for KMS key management | Key lifecycle strategy (create/rotate/audit/disable) documented |
| 23. No Compliance Automation | Check for automated compliance checks | 等保2.0/GDPR automated checklist present |

### 2.5 AIOps Scenarios

| Scenario | Test | Pass Criteria |
|----------|------|--------------|
| 24. Missing Multi-Metric Correlation | Search monitoring skills for ≥ 4 patterns | ≥ 4 anomaly patterns with detection logic |
| 25. No Delegation Matrix | Verify `integration.md` has alarm-to-Skill mapping | Delegation matrix complete |
| 26. No Knowledge Base | Check `references/knowledge-base.md` | ≥ 3 fault patterns + ≥ 1 cascade pattern |
| 27. No Alarm Storm Handling | Search for storm detection | Storm criteria + aggregation workflow |
| 28. Missing Self-Healing | Verify installation flows reference self-healing framework | Self-healing framework referenced |
| 29. No SLO/SLI Definition | Check for SLO with Error Budget | SLO + SLI + Error Budget + burn rate alerting |
| 30. No Change Correlation | Check for CTS change-anomaly correlation | Change correlation decision tree documented |
| 31. No Capacity Forecast | Check for capacity prediction methodology | 30-day capacity forecast methodology |
| 32. No Diagnosis Confidence | Check for confidence scoring model | Confidence score model + uncertainty declaration |

### 2.6 UX Scenarios

| Scenario | Test | Pass Criteria |
|----------|------|--------------|
| 20. Onboarding Friction | First-time user attempts first command | Succeeds within 60s |
| 21. Excessive Prompting | Count interactive prompts for CRUD | ≤ 3 prompts per operation |
| 22. Cryptic Errors | Simulate each error category | Error follows [ERROR] format |

---

## 3. Governance Checklist

### 3.1 Pre-Merge Checklist

- [ ] All `{{env.*}}` placeholders use correct environment variable names
- [ ] No secret literals in any generated file
- [ ] Credential masking enforced — every console/log output uses `***` / `<masked>`
- [ ] JIT Go SDK scripts never print credentials
- [ ] Verification commands check existence only, never echo value
- [ ] Both CLI and SDK paths documented for each operation (dual-path skills)
- [ ] Safety gates present before destructive operations
- [ ] Retry and timeout policies consistent across operations
- [ ] Quick Start section present (UX ≤ 30 seconds to read)
- [ ] Common operations require ≤ 3 prompts
- [ ] Success/failure messages follow standardized format
- [ ] Error messages follow `[ERROR] code → explanation → fix → next step`
- [ ] Error taxonomy covers ≥ 10 product-specific codes
- [ ] Recovery table distinguishes auto-remediation vs HALT
- [ ] Dependency mapping documented in `core-concepts.md`

#### FinOps Checklist
- [ ] Billing model comparison table present
- [ ] Idle resource detection pattern documented
- [ ] Right-sizing guidance with utilization thresholds
- [ ] Cost attribution / tagging guidance
- [ ] Unit economics metrics defined (cost/request or cost/vCPU)
- [ ] Cost anomaly detection rules documented
- [ ] Reserved coverage analysis template present
- [ ] TCO model with cost breakdown documented

#### SecOps Checklist
- [ ] Minimum IAM policy table documented
- [ ] VPC/network isolation guidance present
- [ ] Encryption recommendations (at rest + in transit)
- [ ] Threat detection integration triggers defined (when applicable)
- [ ] Zero Trust Architecture alignment guidance present
- [ ] Security incident response runbook with severity levels
- [ ] Supply chain security (SDK vulnerability + SBOM) documented
- [ ] KMS key lifecycle management strategy documented
- [ ] Compliance automation checklist (等保2.0 / GDPR) present

#### AIOps Checklist
- [ ] ≥ 4 anomaly patterns with detection logic (monitoring skills)
- [ ] Alarm-to-Diagnosis delegation matrix in `integration.md`
- [ ] Knowledge base with ≥ 3 fault patterns
- [ ] Alarm storm handling defined
- [ ] SLO/SLI with Error Budget and burn rate alerting
- [ ] Change correlation analysis with CTS integration
- [ ] Capacity forecasting methodology documented
- [ ] Diagnosis confidence scoring model with uncertainty declaration

### 3.2 Post-Merge Monitoring

- User escalation rate (target: < 10%)
- Task completion rate (target: > 90%)
- Error recovery rate (target: > 80%)
- Average prompts per operation (target: ≤ 3)

---

## 4. Review Templates

### 4.1 Adversarial Review Template

```markdown
## Adversarial Review: huaweicloud-[product]-ops

### Security
- [ ] Scenario 1: Destructive ops have confirmation
- [ ] Scenario 2: No credential echo — ALL use `***` / `<masked>`
- [ ] Scenario 3: All APIs traceable to OpenAPI

### Resilience
- [ ] Scenario 4: Idempotency documented
- [ ] Scenario 5: Throttling handled
- [ ] Scenario 6: No hardcoded regions
- [ ] Scenario 7: All errors have recovery

### FinOps
- [ ] Scenario 8: Billing model comparison present
- [ ] Scenario 9: Idle detection documented
- [ ] Scenario 10: Right-sizing guidance present
- [ ] Scenario 11: Unit economics metrics defined
- [ ] Scenario 12: Cost anomaly detection documented
- [ ] Scenario 13: Reserved coverage analysis present
- [ ] Scenario 14: TCO model documented

### SecOps
- [ ] Scenario 15: IAM minimum permissions documented
- [ ] Scenario 16: Network isolation guidance present
- [ ] Scenario 17: Encryption at rest/in transit documented
- [ ] Scenario 18: Threat detection triggers defined
- [ ] Scenario 19: Zero trust alignment documented
- [ ] Scenario 20: Security incident response runbook present
- [ ] Scenario 21: Supply chain security (SDK + SBOM) documented
- [ ] Scenario 22: KMS key lifecycle management documented
- [ ] Scenario 23: Compliance automation checklist present

### AIOps
- [ ] Scenario 24: Multi-metric correlation patterns (≥ 4)
- [ ] Scenario 25: Cross-skill delegation matrix
- [ ] Scenario 26: Knowledge base with fault patterns
- [ ] Scenario 27: Alarm storm handling defined
- [ ] Scenario 28: Self-healing framework referenced
- [ ] Scenario 29: SLO/SLI with Error Budget defined
- [ ] Scenario 30: Change correlation analysis documented
- [ ] Scenario 31: Capacity forecast methodology documented
- [ ] Scenario 32: Diagnosis confidence scoring model present

### UX
- [ ] Scenario 20: Onboarding ≤ 60s
- [ ] Scenario 21: ≤ 3 prompts per operation
- [ ] Scenario 22: Error messages user-friendly

### Reviewer Sign-off
Reviewer: _______________ Date: _______________ Result: PASS / FAIL
```

---

*This governance document is mandatory. No skill may be merged without passing all review stages.*
