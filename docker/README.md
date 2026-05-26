# Docker Sandbox for Huawei Cloud Ops Skills

This directory contains Docker configuration for running all hcloud-skills in an isolated sandbox environment.

## Quick Start

### 1. Configure Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit with your Huawei Cloud credentials
vim .env
```

### 2. Build and Run

```bash
# Build the Docker image
docker-compose build

# Start interactive sandbox
docker-compose up hcloud-skills

# Or run in detached mode
docker-compose up -d hcloud-skills

# Attach to running container
docker attach hcloud-skills-sandbox
```

### 3. Using the Sandbox

```bash
# Inside the container:

# Check environment
check-env

# List available skills
skill-list

# Read a specific skill
skill-read huaweicloud-ecs-ops

# Execute CLI commands
hcloud ecs list-servers --region cn-north-4

# Run skill operations
/workspace/scripts/skill-exec.sh huaweicloud-ecs-ops list-servers
```

## Services

| Service | Purpose | Profile |
|---------|---------|---------|
| `hcloud-skills` | Interactive CLI sandbox | default |
| `hcloud-worker` | Background task execution | default |
| `hcloud-test` | Isolated test environment | test |
| `hcloud-sdk-builder` | Go SDK compilation | build |

## Profiles

```bash
# Run test environment
docker-compose --profile test up hcloud-test

# Run SDK builder
docker-compose --profile build up hcloud-sdk-builder

# Run all services
docker-compose --profile test --profile build up
```

## Directory Structure

```
docker/
├── entrypoint.sh      # Container startup script
├── scripts/
│   ├── run-tests.sh   # Skill test runner
│   └── skill-exec.sh  # Skill execution wrapper
├── .credentials/      # Credential storage (mounted)
├── .outputs/          # Output capture directory
├── logs/              # Execution logs
└── go-cache/          # Go SDK cache
```

## Security Notes

1. **Credentials**: Store in `.env` file (never commit to git)
2. **Read-only mounts**: Skills directory mounted read-only for safety
3. **Network isolation**: Bridge network mode by default
4. **Resource limits**: CPU/memory limits prevent resource exhaustion

## Common Operations

```bash
# Execute command in running container
docker exec hcloud-skills-sandbox hcloud ecs list-servers

# Run background task via worker
docker exec hcloud-skills-worker /workspace/scripts/skill-exec.sh huaweicloud-ces-ops list-alarm-rules

# View logs
docker logs hcloud-skills-sandbox

# Copy output from container
docker cp hcloud-skills-sandbox:/workspace/.outputs ./local-output
```

## Troubleshooting

### CLI not found
```bash
# Verify installation
docker exec hcloud-skills-sandbox hcloud version
```

### Credentials not working
```bash
# Check environment inside container
docker exec hcloud-skills-sandbox check-env
```

### Build failures
```bash
# Rebuild with no cache
docker-compose build --no-cache
```