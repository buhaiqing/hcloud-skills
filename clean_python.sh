#!/usr/bin/env bash
# clean_python.sh — 清理当前目录及子目录下所有 Python 临时生成内容
set -euo pipefail

DRY_RUN=false
FORCE_YES=false

usage() {
    cat <<'EOF'
用法: ./clean_python.sh [选项]

选项:
  --dry-run    预览模式，仅列出将删除的内容和释放空间，不实际执行
  --yes        跳过确认，直接清理
  -h, --help   显示此帮助信息
EOF
    exit 0
}

# 解析参数
for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=true ;;
        --yes)     FORCE_YES=true ;;
        -h|--help) usage ;;
        *)         echo "未知参数: $arg"; usage ;;
    esac
done

# 颜色输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 格式化字节为人类可读
format_size() {
    local bytes=$1
    if (( bytes >= 1073741824 )); then
        awk -v b="$bytes" 'BEGIN { printf "%.2f GB", b / 1073741824 }'
    elif (( bytes >= 1048576 )); then
        awk -v b="$bytes" 'BEGIN { printf "%.2f MB", b / 1048576 }'
    elif (( bytes >= 1024 )); then
        awk -v b="$bytes" 'BEGIN { printf "%.2f KB", b / 1024 }'
    else
        echo "${bytes} B"
    fi
}

# 定义清理模式（数组）
# 格式: "类型描述|查找模式|类型(d=dir f=file)"
PATTERNS=(
    "__pycache__ 缓存|__pycache__|d"
    ".pyc 文件|*.pyc|f"
    ".pyo 文件|*.pyo|f"
    ".pytest_cache|.pytest_cache|d"
    ".mypy_cache|.mypy_cache|d"
    ".ruff_cache|.ruff_cache|d"
    "*.egg-info|*.egg-info|d"
    "*.egg 包|*.egg|f"
    ".eggs 缓存|.eggs|d"
    ".coverage|.coverage|f"
    "htmlcov 报告|htmlcov|d"
    ".hypothesis|.hypothesis|d"
)

TOTAL_COUNT=0
TOTAL_SIZE_KB=0

# 收集所有待清理路径到一个临时文件
TEMP_FILE=$(mktemp /tmp/clean_python.XXXXXX)
trap 'rm -f "$TEMP_FILE"' EXIT

MODE_LABEL=""
$DRY_RUN && MODE_LABEL="${CYAN}[DRY-RUN]${NC} "

echo ""
echo -e "${BLUE}${MODE_LABEL}Python 临时文件清理 | $(pwd)${NC}"
echo ""

# 扫描阶段（聚合统计，不逐条输出）
SUMMARY_LINES=()

for entry in "${PATTERNS[@]}"; do
    desc="${entry%%|*}"
    pattern="${entry#*|}"
    pattern="${pattern%|*}"
    ptype="${entry##*|}"

    kind_flag="-type d"
    [ "$ptype" = "f" ] && kind_flag="-type f"

    cat_count=0
    cat_size_kb=0

    find . $kind_flag -name "$pattern" \
        -not -path '*/.venv/*' -not -path '*/.venv' \
        -not -path '*/venv/*'  -not -path '*/venv' \
        -not -path '*/.git/*'  -not -path '*/.git' \
        -print0 2>/dev/null | \
    while IFS= read -r -d '' path; do
        if [ "$ptype" = "d" ]; then
            s=$(du -sk "$path" 2>/dev/null | cut -f1 | head -1)
            [ -z "$s" ] && s=0
        else
            if [[ "$(uname)" == "Darwin" ]]; then
                s=$(stat -f%z "$path" 2>/dev/null || echo 0)
                s=$((s / 1024))
            else
                s=$(stat -c%s "$path" 2>/dev/null || echo 0)
                s=$((s / 1024))
            fi
        fi
        cat_count=$((cat_count + 1))
        cat_size_kb=$((cat_size_kb + s))
        echo "$path|$ptype" >> "$TEMP_FILE"
    done

    if [ "$cat_count" -gt 0 ]; then
        human=$(format_size $((cat_size_kb * 1024)))
        SUMMARY_LINES+=("$(printf "  %-20s %4d 项  %s" "${desc}" "${cat_count}" "${human}")")
        TOTAL_COUNT=$((TOTAL_COUNT + cat_count))
        TOTAL_SIZE_KB=$((TOTAL_SIZE_KB + cat_size_kb))
    fi
done

if [ "$TOTAL_COUNT" -eq 0 ]; then
    echo -e "${GREEN}未发现 Python 临时文件，无需清理。${NC}"
    exit 0
fi

# 输出聚合摘要
echo -e "${BLUE}类别                    数量    大小${NC}"
echo -e "${BLUE}----                    ----    ----${NC}"
for line in "${SUMMARY_LINES[@]}"; do
    echo -e "$line"
done
echo ""

EST_SIZE=$(format_size $((TOTAL_SIZE_KB * 1024)))
echo -e "${CYAN}合计: ${TOTAL_COUNT} 项，预计释放 ${EST_SIZE}${NC}"

if $DRY_RUN; then
    echo -e "\n${BLUE}提示: ./clean_python.sh 执行 | ./clean_python.sh --yes 静默执行${NC}"
    exit 0
fi

# 确认
if ! $FORCE_YES; then
    echo ""
    read -r -p $'\033[1;31m确认删除以上内容并释放 '"${EST_SIZE}"$'\033[1;31m 空间? [y/N]: \033[0m' confirm
    if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
        echo "已取消。"
        exit 0
    fi
fi

# 清理阶段
file_cnt=0
dir_cnt=0
while IFS='|' read -r path ptype; do
    if [ "$ptype" = "d" ]; then
        rm -rf "$path"
        dir_cnt=$((dir_cnt + 1))
    else
        rm -f "$path"
        file_cnt=$((file_cnt + 1))
    fi
done < "$TEMP_FILE"

echo -e "\n${GREEN}清理完成: ${TOTAL_COUNT} 项 | 文件 ${file_cnt} + 目录 ${dir_cnt} | 释放 ${EST_SIZE}${NC}"
