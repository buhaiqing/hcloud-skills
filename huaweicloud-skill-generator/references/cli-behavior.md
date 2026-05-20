# CLI Behavioral Reference — Huawei Cloud

> **Purpose:** Verified HuaweiCloud CLI behavioral notes. Progressive disclosure — loaded on demand by agent.
> **Status:** Reference document

---

## 1. Conventions (Agent Execution)

- Huawei Cloud CLI (`hcloud`) supports both interactive and non-interactive modes
- Output is typically JSON when `--output json` is used, otherwise formatted text
- Credentials can be passed via environment variables `HW_ACCESS_KEY_ID` / `HW_SECRET_ACCESS_KEY`
- Region is specified as `--region cn-north-4` (Beijing4), `ap-southeast-1` (Singapore), etc.
- For RESTful APIs: `hcloud [service] [action] --region <region> --param value`

## 2. Environment Variables

| Variable | Purpose | Required |
|----------|---------|----------|
| `HW_ACCESS_KEY_ID` | Huawei Cloud AK | Yes |
| `HW_SECRET_ACCESS_KEY` | Huawei Cloud SK | Yes |
| `HW_REGION_ID` | Default region | Recommended |
| `HW_PROJECT_ID` | Project ID (for some APIs) | Required for specific operations |
| `HW_IAM_ENDPOINT` | Custom IAM endpoint | Optional |
| `HW_SECURITY_TOKEN` | STS temporary token | STS auth only |

## 3. Invocation Patterns

```bash
# Service-level command
hcloud ecs describe-instances --region cn-north-4

# With filtering
hcloud ecs describe-instances --region cn-north-4 --name "my-instance"

# JSON output for agent parsing
hcloud ecs describe-instances --region cn-north-4 --output json

# JMESPath extraction
hcloud ecs describe-instances --region cn-north-4 --output json | jq '.instances[0].id'
```

## 4. Coverage Gap Table

Some Huawei Cloud services may not be fully covered by CLI. In such cases, JIT Go SDK (`github.com/huaweicloud/huaweicloud-sdk-go-v3`) is the fallback. Services with confirmed CLI coverage should be verified against official documentation.

---

*This document uses progressive disclosure. Agent loads only the section needed.*
