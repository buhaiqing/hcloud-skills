# CSS Integration

## Cross-Skill Delegation Matrix

| CSS Operation | Depends On | Delegates To | Notes |
|---------------|------------|--------------|-------|
| Create Cluster | VPC, Subnet, SG | `huaweicloud-vpc-ops` | Network prerequisites |
| Create Cluster | KMS Key | `huaweicloud-kms-ops` | If encryption enabled |
| Create Snapshot | OBS Bucket | `huaweicloud-obs-ops` | Snapshot storage |
| Monitor Cluster | CES Metrics | `huaweicloud-ces-ops` | Alarms and dashboards |
| IAM Permissions | IAM Policies | `huaweicloud-iam-ops` | Access control |
| Log Analysis | Log Tank | `huaweicloud-lts-ops` | Audit logs |

## Go Bootstrap

### SDK Installation

```bash
# Initialize Go module
go mod init css-ops

# Add dependency
go get github.com/huaweicloud/huaweicloud-sdk-go-v3/services/css/v2
```

### go.mod

```go
module css-ops

go 1.21

require (
    github.com/huaweicloud/huaweicloud-sdk-go-v3 v0.1.x
)
```

### JIT Execution Script

```go
package main

import (
    "fmt"
    "os"
    "time"
    
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/css/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/css/v2/model"
)

func main() {
    // Get credentials from environment
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    region := os.Getenv("HW_REGION_ID")
    
    if ak == "" || sk == "" {
        fmt.Fprintln(os.Stderr, "ERROR: Missing credentials")
        os.Exit(1)
    }
    
    // Create client
    cfg := config.DefaultHttpConfig()
    client := css.CssClientBuilder().
        WithEndpoint(fmt.Sprintf("css.%s.myhuaweicloud.com", region)).
        WithCredential(basic.NewCredentialsBuilder().
            WithAk(ak).WithSk(sk).Build()).
        WithHttpConfig(cfg).Build()
    
    // Execute operation based on args
    operation := os.Args[1]
    switch operation {
    case "list-clusters":
        listClusters(client)
    case "show-cluster":
        showCluster(client, os.Args[2])
    default:
        fmt.Fprintf(os.Stderr, "Unknown operation: %s\n", operation)
        os.Exit(1)
    }
}

func listClusters(client *css.CssClient) {
    request := &model.ListClustersRequest{}
    response, err := client.ListClusters(request)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    
    for _, cluster := range *response.Clusters {
        fmt.Printf("%s\t%s\t%s\n", cluster.Id, cluster.Name, cluster.Status)
    }
}

func showCluster(client *css.CssClient, clusterId string) {
    request := &model.ShowClusterDetailRequest{
        ClusterId: clusterId,
    }
    response, err := client.ShowClusterDetail(request)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Printf("Cluster: %s\n", response.Name)
    fmt.Printf("Status: %s\n", response.Status)
    fmt.Printf("Endpoint: %s\n", response.Endpoint)
    fmt.Printf("Nodes: %d\n", response.InstanceNum)
}
```

## Dependency Configuration

### Network Dependencies

```yaml
# vpc-config.yaml
vpc:
  name: css-vpc
  cidr: 172.16.0.0/16
  subnets:
    - name: css-subnet
      cidr: 172.16.1.0/24
      az: cn-north-4a
  security_groups:
    - name: css-sg
      rules:
        - protocol: tcp
          port: 9200
          source: 172.16.0.0/16
        - protocol: tcp
          port: 9300
          source: 172.16.0.0/16
```

### OBS Dependencies

```yaml
# obs-config.yaml
obs:
  bucket_name: css-snapshots-bucket
  storage_class: STANDARD
  encryption:
    enabled: true
    algorithm: kms
    kms_key_id: "{{user.kms_key_id}}"
  lifecycle:
    - prefix: "snapshots/"
      expiration_days: 90
```

### IAM Dependencies

```json
{
  "Version": "1.1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "css:*:get",
        "css:*:list",
        "css:*:create",
        "css:*:delete",
        "css:*:modify"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "vpc:*:get",
        "vpc:*:list"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "obs:bucket:GetBucketLocation",
        "obs:bucket:ListBucket",
        "obs:object:PutObject",
        "obs:object:GetObject",
        "obs:object:DeleteObject"
      ],
      "Resource": [
        "obs:*:*:bucket:css-snapshots*",
        "obs:*:*:object:css-snapshots*/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "kms:cmk:encrypt",
        "kms:cmk:decrypt"
      ],
      "Resource": "*"
    }
  ]
}
```

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/css-deploy.yml
name: Deploy CSS Cluster

on:
  workflow_dispatch:
    inputs:
      cluster_name:
        required: true
      action:
        type: choice
        options: [create, update, delete]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Huawei Cloud CLI
        run: |
          curl -sSL https://cn-north-4.myhuaweicloud.com/cli/latest/hcloud_install.sh | bash
          hcloud configure set --cli-access-key="${{ secrets.HW_ACCESS_KEY_ID }}"
          hcloud configure set --cli-secret-key="${{ secrets.HW_SECRET_ACCESS_KEY }}"
          hcloud configure set --cli-region="${{ vars.HW_REGION_ID }}"
      
      - name: Deploy Cluster
        if: github.event.inputs.action == 'create'
        run: |
          hcloud CSS CreateCluster \
            --name "${{ github.event.inputs.cluster_name }}" \
            --datastore '{"type":"elasticsearch","version":"7.10.2"}' \
            --instance '{...}'
```

### Terraform Integration

```hcl
# main.tf
resource "huaweicloud_css_cluster" "main" {
  name        = var.cluster_name
  engine_type = "elasticsearch"
  engine_version = "7.10.2"
  
  node_config {
    flavor = "ess.spec-4u8g"
    network_info {
      vpc_id            = huaweicloud_vpc.main.id
      subnet_id         = huaweicloud_subnet.main.id
      security_group_id = huaweicloud_networking_secgroup.main.id
    }
    volume {
      volume_type = "ULTRAHIGH"
      size        = 100
    }
    availability_zone = var.availability_zone
  }
  
  nodes = var.node_count
  
  https_enabled             = true
  disk_encryption_enabled   = true
  disk_encryption_key_id    = huaweicloud_kms_key.main.id
  
  backup_strategy {
    period    = "00:00 GMT+08:00"
    prefix    = "snapshot"
    keep_days = 7
    bucket    = huaweicloud_obs_bucket.snapshots.bucket
    base_path = "css-backup"
    agency    = "css_obs_agency"
  }
}
```

## Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `HW_ACCESS_KEY_ID` | Yes | Access Key ID | `AKIA...` |
| `HW_SECRET_ACCESS_KEY` | Yes | Secret Access Key | `***` |
| `HW_REGION_ID` | Yes | Region code | `cn-north-4` |
| `HW_PROJECT_ID` | Yes | Project ID | `a1b2c3...` |
| `CSS_CLUSTER_ID` | Optional | Default cluster | `cluster-xxx` |
| `CSS_ENDPOINT` | Optional | Cluster endpoint | `https://...` |
| `CSS_USERNAME` | Optional | Admin username | `admin` |
| `CSS_PASSWORD` | Optional | Admin password | `***` |

## Cross-Product Workflow

### Create Full Stack

```bash
#!/bin/bash
# create-full-stack.sh

# 1. Create VPC (delegates to huaweicloud-vpc-ops)
echo "Creating VPC..."
hcloud VPC CreateVpc --name "css-vpc" --cidr "172.16.0.0/16"
VPC_ID=$(hcloud VPC ListVpcs -o json | jq -r '.vpcs[0].id')

# 2. Create Subnet
echo "Creating Subnet..."
hcloud VPC CreateSubnet --vpc_id "$VPC_ID" --name "css-subnet" --cidr "172.16.1.0/24"
SUBNET_ID=$(hcloud VPC ListSubnets --vpc_id "$VPC_ID" -o json | jq -r '.subnets[0].id')

# 3. Create Security Group
echo "Creating Security Group..."
hcloud VPC CreateSecurityGroup --name "css-sg"
SG_ID=$(hcloud VPC ListSecurityGroups -o json | jq -r '.security_groups[0].id')

# 4. Create OBS Bucket (delegates to huaweicloud-obs-ops)
echo "Creating OBS Bucket..."
hcloud OBS CreateBucket --bucket "css-snapshots-$(date +%s)"

# 5. Create CSS Cluster
echo "Creating CSS Cluster..."
hcloud CSS CreateCluster \
  --name "prod-es-cluster" \
  --datastore '{"type":"elasticsearch","version":"7.10.2"}' \
  --instance "{\"nics\":{\"vpcId\":\"$VPC_ID\",\"netId\":\"$SUBNET_ID\",\"securityGroupId\":\"$SG_ID\"}}"
```
