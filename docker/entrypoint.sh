#!/bin/bash
# Entrypoint for Huawei Cloud Ops Skills Sandbox
# Validates environment and provides helpful startup messages

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Huawei Cloud Ops Skills Sandbox${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check required environment variables
MISSING_VARS=0

if [ -z "$HW_ACCESS_KEY_ID" ]; then
    echo -e "${YELLOW}⚠ HW_ACCESS_KEY_ID not set${NC}"
    MISSING_VARS=1
else
    echo -e "${GREEN}✓ HW_ACCESS_KEY_ID: configured${NC}"
fi

if [ -z "$HW_SECRET_ACCESS_KEY" ]; then
    echo -e "${YELLOW}⚠ HW_SECRET_ACCESS_KEY not set${NC}"
    MISSING_VARS=1
else
    echo -e "${GREEN}✓ HW_SECRET_ACCESS_KEY: configured (hidden)${NC}"
fi

if [ -z "$HW_REGION_ID" ]; then
    echo -e "${YELLOW}⚠ HW_REGION_ID not set (default: cn-north-4)${NC}"
    export HW_REGION_ID="cn-north-4"
else
    echo -e "${GREEN}✓ HW_REGION_ID: $HW_REGION_ID${NC}"
fi

if [ -z "$HW_PROJECT_ID" ]; then
    echo -e "${YELLOW}⚠ HW_PROJECT_ID not set (optional for most operations)${NC}"
else
    echo -e "${GREEN}✓ HW_PROJECT_ID: configured${NC}"
fi

echo ""

# Check CLI availability
if command -v hcloud &> /dev/null; then
    CLI_VERSION=$(hcloud version 2>/dev/null | head -1)
    echo -e "${GREEN}✓ HCloud CLI: $CLI_VERSION${NC}"
else
    echo -e "${RED}✗ HCloud CLI not found${NC}"
fi

# Check Go availability
if command -v go &> /dev/null; then
    GO_VERSION=$(go version 2>/dev/null)
    echo -e "${GREEN}✓ Go SDK: $GO_VERSION${NC}"
else
    echo -e "${RED}✗ Go not found${NC}"
fi

# Check workspace
if [ -d "/workspace/hcloud-skills" ]; then
    SKILL_COUNT=$(find /workspace/hcloud-skills -name "SKILL.md" -type f 2>/dev/null | wc -l)
    echo -e "${GREEN}✓ Skills loaded: $SKILL_COUNT skills${NC}"
    echo ""
    echo -e "${BLUE}Available skills:${NC}"
    find /workspace/hcloud-skills -name "SKILL.md" -type f 2>/dev/null | \
        sed "s|/workspace/hcloud-skills/||g" | \
        sed "s|/SKILL.md||g" | \
        head -20 | \
        while read skill; do
            echo -e "  ${GREEN}•${NC} $skill"
        done
else
    echo -e "${YELLOW}⚠ Skills directory not found${NC}"
fi

echo ""

if [ $MISSING_VARS -eq 1 ]; then
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}  Environment Configuration Required${NC}"
    echo -e "${YELLOW}========================================${NC}"
    echo ""
    echo "Set the following environment variables:"
    echo "  export HW_ACCESS_KEY_ID=\"your-access-key-id\""
    echo "  export HW_SECRET_ACCESS_KEY=\"your-secret-access-key\""
    echo "  export HW_REGION_ID=\"cn-north-4\""
    echo "  export HW_PROJECT_ID=\"your-project-id\""
    echo ""
    echo "Or pass via docker-compose environment section."
    echo ""
fi

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Quick Commands${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "  check-env      - Show current environment config"
echo "  skill-list     - List all available skills"
echo "  skill-read <n> - Read a specific skill's SKILL.md"
echo "  hcloud --help  - Show CLI help"
echo ""
echo -e "${GREEN}Ready. Type 'bash' for interactive shell.${NC}"
echo ""

# Execute the command passed to docker run
exec "$@"