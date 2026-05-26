#!/bin/bash
# Skill executor - run a specific skill operation in the sandbox

set -e

SKILL="$1"
OPERATION="$2"
ARGS="${@:3}"

if [ -z "$SKILL" ]; then
    echo "Usage: skill-exec <skill-name> <operation> [args...]"
    echo ""
    echo "Available skills:"
    skill-list
    exit 1
fi

SKILL_PATH="/workspace/hcloud-skills/$SKILL/SKILL.md"
if [ ! -f "$SKILL_PATH" ]; then
    echo "Error: Skill '$SKILL' not found at $SKILL_PATH"
    exit 1
fi

echo "=== Executing: $SKILL / $OPERATION ==="
echo "Args: $ARGS"
echo "Timestamp: $(date)"
echo ""

# Log execution
LOG_FILE="/workspace/logs/skill-exec-$(date +%Y%m%d-%H%M%S).log"
echo "Log: $LOG_FILE" | tee "$LOG_FILE"

# Execute via hcloud CLI
# Skills define CLI commands; this script provides context
case "$SKILL" in
    huaweicloud-ces-ops)
        hcloud ces $OPERATION $ARGS 2>&1 | tee -a "$LOG_FILE"
        ;;
    huaweicloud-ecs-ops)
        hcloud ecs $OPERATION $ARGS 2>&1 | tee -a "$LOG_FILE"
        ;;
    huaweicloud-vpc-ops)
        hcloud vpc $OPERATION $ARGS 2>&1 | tee -a "$LOG_FILE"
        ;;
    huaweicloud-rds-ops)
        hcloud rds $OPERATION $ARGS 2>&1 | tee -a "$LOG_FILE"
        ;;
    huaweicloud-elb-ops)
        hcloud elb $OPERATION $ARGS 2>&1 | tee -a "$LOG_FILE"
        ;;
    huaweicloud-cce-ops)
        hcloud cce $OPERATION $ARGS 2>&1 | tee -a "$LOG_FILE"
        ;;
    huaweicloud-obs-ops)
        hcloud obs $OPERATION $ARGS 2>&1 | tee -a "$LOG_FILE"
        ;;
    huaweicloud-iam-ops)
        hcloud iam $OPERATION $ARGS 2>&1 | tee -a "$LOG_FILE"
        ;;
    *)
        echo "Direct CLI execution for: $SKILL"
        hcloud $OPERATION $ARGS 2>&1 | tee -a "$LOG_FILE"
        ;;
esac

echo ""
echo "=== Execution Complete ==="
echo "Output saved to: $LOG_FILE"