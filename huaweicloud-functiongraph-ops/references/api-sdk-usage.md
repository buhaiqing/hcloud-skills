# API & SDK Usage — Huawei Cloud FunctionGraph

## JIT Go SDK Setup

```bash
mkdir -p /tmp/fg-sdk-workspace && cd /tmp/fg-sdk-workspace
go mod init fg-script
go get github.com/huaweicloud/huaweicloud-sdk-go-v3/services/functiongraph/v2
```

## Operation Map

| Operation | Go SDK Method | API Path | Description |
|-----------|--------------|----------|-------------|
| CreateFunction | `CreateFunction` | POST /v2/{project_id}/fgs/functions | Create a new function |
| ShowFunctionConfig | `ShowFunctionConfig` | GET /v2/{project_id}/fgs/functions/{function_urn} | Get function configuration |
| ListFunctions | `ListFunctions` | GET /v2/{project_id}/fgs/functions | List all functions |
| UpdateFunctionCode | `UpdateFunctionCode` | PUT /v2/{project_id}/fgs/functions/{function_urn}/code | Update function code |
| UpdateFunctionConfig | `UpdateFunctionConfig` | PUT /v2/{project_id}/fgs/functions/{function_urn}/config | Update function config |
| DeleteFunction | `DeleteFunction` | DELETE /v2/{project_id}/fgs/functions/{function_urn} | Delete a function |
| InvokeFunction | `InvokeFunction` | POST /v2/{project_id}/fgs/functions/{function_urn}/invocations | Sync invoke |
| AsyncInvokeFunction | `AsyncInvokeFunction` | POST /v2/{project_id}/fgs/functions/{function_urn}/async-invocations | Async invoke |
| CreateFunctionVersion | `CreateFunctionVersion` | POST /v2/{project_id}/fgs/functions/{function_urn}/versions | Publish version |
| ListFunctionVersions | `ListFunctionVersions` | GET /v2/{project_id}/fgs/functions/{function_urn}/versions | List versions |
| CreateAlias | `CreateAlias` | POST /v2/{project_id}/fgs/functions/{function_urn}/aliases | Create alias |
| UpdateAlias | `UpdateAlias` | PUT /v2/{project_id}/fgs/functions/{function_urn}/aliases/{alias_name} | Update alias |
| ListFunctionTriggers | `ListFunctionTriggers` | GET /v2/{project_id}/fgs/functions/{function_urn}/triggers | List triggers |
| CreateFunctionTrigger | `CreateFunctionTrigger` | POST /v2/{project_id}/fgs/functions/{function_urn}/triggers | Create trigger |
| DeleteFunctionTrigger | `DeleteFunctionTrigger` | DELETE /v2/{project_id}/fgs/functions/{function_urn}/triggers/{trigger_type}/{trigger_id} | Delete trigger |
| ListAsyncInvocations | `ListAsyncInvocations` | GET /v2/{project_id}/fgs/functions/{function_urn}/async-invocations | List async invocations |
| ShowFunctionMetrics | `ShowFunctionMetrics` | GET /v2/{project_id}/fgs/functions/{function_urn}/statistics | Get function metrics |
| ListQuotas | `ListQuotas` | GET /v2/{project_id}/fgs/quotas | List resource quotas |

## Common Request/Response Patterns

### Create Function

**Request body**:
```json
{
    "function_name": "my-function",
    "runtime": "Python3.9",
    "handler": "index.handler",
    "code_type": "obs",
    "code_url": "https://bucket.obs.cn-north-4.myhuaweicloud.com/code-package.zip",
    "timeout": 30,
    "memory_size": 256,
    "description": "My serverless function",
    "vpc_config": {
        "vpc_name": "vpc-xxx",
        "vpc_id": "vpc-id",
        "subnet_id": "subnet-id"
    }
}
```

**Response**:
```json
{
    "func_urn": "urn:fss:cn-north-4:project-id:function:my-function:latest",
    "func_name": "my-function",
    "runtime": "Python3.9",
    "handler": "index.handler",
    "code_type": "obs",
    "timeout": 30,
    "memory_size": 256,
    "state": "Active",
    "version": "latest"
}
```

### Invoke Function (Sync)

**Request headers**:
```
X-Cff-Invocation-Type: Sync
```

**Response** (200):
```
Function execution result as string
```

### Invoke Function (Async)

**Request headers**:
```
X-Cff-Invocation-Type: Async
```

**Response** (202):
```json
{
    "request_id": "a1b2c3d4-1234-5678-9abc-def012345678"
}
```

## Pagination

ListFunctions supports pagination with `marker` (last function URN from previous page) and `max_items` (default 50, max 500).

```go
request := &model.ListFunctionsRequest{
    Marker:   func() *string { v := ""; return &v }(),
    MaxItems: func() *string { v := "50"; return &v }(),
}
```

## Idempotency

- Function name must be unique within a project — duplicate name returns `FSS.0102`
- Trigger creation: same trigger type + event source for same function returns `FSS.0501`
- Version publish: each function supports up to 50 versions
- Delete: idempotent — deleting non-existent function succeeds (404 swallowed)
