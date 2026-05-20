# Prompt Library — Huawei Cloud Skill Generator

> **Purpose:** Structured prompts for the skill generation lifecycle. Reusable templates with variable placeholders.
> **Status:** Reference document

---

## 1. Generation Prompts

### 1.1 New Skill Scaffolding
```
Generate huaweicloud-[product]-ops skill for [Product Name].
API Docs: [docs URL]
Primary resources: [Resource types]
Operations: create, describe, modify, delete, list
CLI support: [confirmed/partial/none]
Go SDK: github.com/huaweicloud/huaweicloud-sdk-go-v3/services/[service]
Key requirements: [FinOps cost tracking, SecOps threat detection, AIOps monitoring]
```

### 1.2 Skill Realignment
```
Regenerate huaweicloud-[product]-ops after API documentation changes.
Changed operations: [list]
Deprecations: [list]
New parameters: [list]
Current version: [X.Y.Z] → New version: [X.Y+1.0]
```

## 2. Evaluation Prompts

### 2.1 Trigger Accuracy Test
```
Test the skill description trigger accuracy:
Description: [current description text]
Test queries:
- [should-trigger query 1]
- [should-not-trigger query 1]
...
Evaluate: Does the description correctly fire on appropriate queries?
```

### 2.2 P0 Compliance Check
```
Review the generated skill against P0 checklist:
Skill: [skill name]
Checklist items: [list items to verify]
Gaps found: [list gaps]
Required fixes: [list fixes]
```

## 3. Three-Pillar Assessment Prompts

### 3.1 FinOps Review
```
Evaluate FinOps coverage in huaweicloud-[product]-ops:
- Billing model table present?
- Idle detection pattern documented?
- Right-sizing guidance complete?
- Cost attribution/tagging guidance?
Gap analysis and recommended additions.
```

### 3.2 SecOps Review
```
Evaluate SecOps coverage in huaweicloud-[product]-ops:
- Minimum IAM policy table complete?
- VPC/network isolation guidance?
- Encryption at rest/in transit documented?
- HSS/WAF integration triggers defined?
Gap analysis and recommended additions.
```

### 3.3 AIOps Review
```
Evaluate AIOps coverage in huaweicloud-[product]-ops:
- Multi-metric correlation (≥ 4 patterns)?
- Cross-skill delegation matrix?
- Knowledge base populated?
- Alarm storm handling defined?
Gap analysis and recommended additions.
```

## 4. Well-Architected Prompts

### 4.1 Five-Pillar Assessment
```
Evaluate Well-Architected coverage:
- Security: IAM minimum permissions, credential masking, VPC endpoint?
- Stability: Backup/recovery with RTO/RPO, multi-AZ, DR runbook?
- Cost: Billing comparison, idle detection, right-sizing?
- Efficiency: Batch operations, CI/CD integration, escalation paths?
- Performance: Scaling triggers, baselines, auto-detection?
For any missing pillar, generate assessment table and integration guidance.
```

### 4.2 Cross-Pillar Conflict Check
```
Analyze for pillar conflicts:
- Do SecOps requirements (e.g., full audit logging) create Cost overhead?
- Do FinOps cost-cutting recommendations (e.g., spot instances) hurt Stability?
- Do Performance optimizations (e.g., large instance types) increase Cost?
For each conflict, document the trade-off and recommended balance.
```

## 5. Self-Reflection Prompts

### 5.1 Round 1: Foundation
```
Self-Reflection Round 1 for huaweicloud-[product]-ops:
1. FinOps: Are cost optimization patterns actionable? Billing table complete?
2. SecOps: IAM permissions minimum documented? Credential masking enforced?
3. AIOps: Multi-metric correlation defined? Delegation matrix present?
Report gaps per pillar with specific remediation steps.
```

### 5.2 Round 2: Critical Analysis
```
Self-Reflection Round 2 for huaweicloud-[product]-ops:
1. What would break if a user follows this skill in production?
2. Is there a better way to document this that reduces agent confusion?
3. Are HALT conditions clearly separated from retry scenarios?
4. Do any FinOps recommendations conflict with reliability requirements?
5. Does SecOps create performance bottlenecks that contradict Performance pillar?
Document findings and generate targeted fixes.
```

---

*This prompt library is maintained alongside the generator. Track prompt effectiveness and update quarterly.*
