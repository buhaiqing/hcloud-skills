#!/bin/bash
# Test runner script for hcloud-skills sandbox
# Executes skill validation tests

set -e

SKILLS_DIR="/workspace/hcloud-skills"
TEST_OUTPUT="/workspace/.outputs"

echo "=== Huawei Cloud Skills Test Runner ==="
echo "Timestamp: $(date)"
echo ""

# List available skills
echo "Skills available:"
skill-list
echo ""

# Run basic connectivity test
echo "=== Connectivity Test ==="
if hcloud version; then
    echo "✓ CLI available"
else
    echo "✗ CLI not available"
    exit 1
fi

# Check environment
echo ""
echo "=== Environment Check ==="
check-env

# Test skill file integrity
echo ""
echo "=== Skill File Integrity ==="
for skill in $(skill-list | head -10); do
    if [ -f "$SKILLS_DIR/$skill/SKILL.md" ]; then
        echo "✓ $skill/SKILL.md exists"
    else
        echo "✗ $skill/SKILL.md missing"
    fi
done

echo ""
echo "=== Tests Complete ==="
echo "Output saved to: $TEST_OUTPUT"