#!/usr/bin/env bash
# =============================================================================
# bilibili.sh — B 站教育资源采集 + 导入 一键脚本
#
# 依次完成：准备 crawler 环境（幂等建 venv）-> 采集 B 站公开视频元数据 ->
# 导入 resources 表。与 docs/guides/bilibili-import.md 的手动步骤一一对应。
#
# 用法：
#   bash scripts/bilibili.sh               # 采集 + 导入（一条龙）
#   bash scripts/bilibili.sh --dry-run     # 全程预览：采集不写文件、导入不写库
#   bash scripts/bilibili.sh --crawl-only  # 只采集不导入
#   bash scripts/bilibili.sh --import-only # 只导入已有 data/bilibili/latest.json
#
# 可用环境变量覆盖：
#   CRAWLER_DIR   crawler 目录（默认 backend/crawler）
#   CRAWL_CONFIG  采集任务配置（默认 config.yaml，相对 CRAWLER_DIR）
#   CONFIG_PATH   后端配置（默认 configs/config.yaml，相对 backend/）
#   IMPORT_FILE   导入文件（默认 data/bilibili/latest.json，相对 backend/）
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORM_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$PLATFORM_ROOT/backend"

CRAWLER_DIR="${CRAWLER_DIR:-$BACKEND_DIR/crawler}"
CRAWL_CONFIG="${CRAWL_CONFIG:-config.yaml}"
CONFIG_PATH="${CONFIG_PATH:-configs/config.yaml}"
IMPORT_FILE="${IMPORT_FILE:-data/bilibili/latest.json}"

log() { echo "[bilibili] $*"; }
die() { echo "[bilibili] 错误: $*" >&2; exit 1; }

DRY_RUN=0; CRAWL_ONLY=0; IMPORT_ONLY=0
for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=1 ;;
    --crawl-only) CRAWL_ONLY=1 ;;
    --import-only) IMPORT_ONLY=1 ;;
    -h|--help) sed -n '2,16p' "$0"; exit 0 ;;
    *) echo "未知参数: $arg" >&2; exit 2 ;;
  esac
done
[ "$CRAWL_ONLY" = 1 ] && [ "$IMPORT_ONLY" = 1 ] && die "--crawl-only 与 --import-only 互斥"

# ---- 前置检查 ---------------------------------------------------------------
command -v go >/dev/null || die "未找到 go"
[ -d "$CRAWLER_DIR" ] || die "crawler 目录不存在: $CRAWLER_DIR"

# ---- [1] 准备 crawler venv（幂等） ---------------------------------------------
CRAWLER_PYTHON="$CRAWLER_DIR/.venv/bin/python"
if [ ! -x "$CRAWLER_PYTHON" ]; then
  log "[1] 初始化 crawler 虚拟环境"
  ( cd "$CRAWLER_DIR" && python3 -m venv .venv && .venv/bin/pip install -q -r requirements.txt )
fi

# ---- [2] 采集 ------------------------------------------------------------------
if [ "$IMPORT_ONLY" = 1 ]; then
  log "[2] 跳过采集（--import-only）"
else
  log "[2] 采集 B 站数据 (config=$CRAWL_CONFIG)"
  [ -f "$CRAWLER_DIR/$CRAWL_CONFIG" ] || die "采集配置不存在: $CRAWLER_DIR/$CRAWL_CONFIG（先 cp config.example.yaml config.yaml）"
  CRAWL_ARGS=(--config "$CRAWL_CONFIG")
  [ "$DRY_RUN" = 1 ] && CRAWL_ARGS+=(--dry-run)
  ( cd "$CRAWLER_DIR" && "$CRAWLER_PYTHON" run.py "${CRAWL_ARGS[@]}" )
fi

# ---- [3] 导入 ------------------------------------------------------------------
if [ "$CRAWL_ONLY" = 1 ]; then
  log "[3] 跳过导入（--crawl-only）"
else
  log "[3] 导入资源 (file=$IMPORT_FILE)"
  IMPORT_ARGS=(-file "$IMPORT_FILE")
  [ "$DRY_RUN" = 1 ] && IMPORT_ARGS+=(-dry-run)
  ( cd "$BACKEND_DIR" && CONFIG_PATH="$CONFIG_PATH" go run ./cmd/import_bilibili "${IMPORT_ARGS[@]}" )
fi

log "完成"
