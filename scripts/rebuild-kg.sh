#!/usr/bin/env bash
# =============================================================================
# rebuild-kg.sh — 一键重建知识图谱（学科层 + MOOCCube 概念层）
#
# 默认先清空上一版图谱派生数据再重建（--keep-old 保留旧数据做增量）。
# 只清理知识图谱相关表，资源/用户/行为/评分不受影响。
#
# 用法：
#   bash scripts/rebuild-kg.sh                      # 全量重建（推荐）
#   bash scripts/rebuild-kg.sh --dry-run            # 只统计
#   bash scripts/rebuild-kg.sh --source /path/MOOCCube
#   bash scripts/rebuild-kg.sh --keep-old           # 增量
#   bash scripts/rebuild-kg.sh --skip-concepts      # 只建学科层
#
# 可选环境变量：CONFIG_PATH（默认 configs/config.local.yaml）、GOCACHE
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$REPO_ROOT/backend"

CONFIG_PATH="${CONFIG_PATH:-configs/config.local.yaml}"
export GOCACHE="${GOCACHE:-$REPO_ROOT/.gocache}"

RESET=1
DRY=0
SKIP_CONCEPTS=0
SOURCE=""

while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run) DRY=1 ;;
    --keep-old) RESET=0 ;;
    --skip-concepts) SKIP_CONCEPTS=1 ;;
    --source)
      shift
      SOURCE="${1:-}"
      [ -n "$SOURCE" ] || { echo "--source 需要目录参数" >&2; exit 2; }
      ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    *) echo "未知参数: $1" >&2; exit 2 ;;
  esac
  shift
done

args=(run ./cmd/seed_knowledge)
[ "$RESET" -eq 1 ] && args+=(-reset)
[ "$DRY" -eq 1 ] && args+=(-dry-run)
[ "$SKIP_CONCEPTS" -eq 1 ] && args+=(-skip-concepts)
[ -n "$SOURCE" ] && args+=(-source "$SOURCE")

echo "[rebuild-kg] CONFIG_PATH=$CONFIG_PATH"
echo "[rebuild-kg] go ${args[*]}"

cd "$BACKEND_DIR"
CONFIG_PATH="$CONFIG_PATH" go "${args[@]}"
echo "[rebuild-kg] 完成。查看接口: GET /api/v1/knowledge-points"

