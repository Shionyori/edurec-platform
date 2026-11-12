#!/usr/bin/env bash
# =============================================================================
# set-kg-source.sh — 一键切换知识图谱数据源路径（MOOCCube）
#
# 修改 backend/configs/config.local.yaml 里的 knowledge.source_dir，
# 首次修改前保留 .bak 备份。数据目录是平台本地的外部数据，不指向 engine 仓库。
#
# 用法：
#   bash scripts/set-kg-source.sh --show
#   bash scripts/set-kg-source.sh data/MOOCCube
#   bash scripts/set-kg-source.sh /opt/data/MOOCCube backend/configs/config.yaml
#
# 数据目录应包含 relations/prerequisite-dependency.json 等 MOOCCube 文件。
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

CONFIG="backend/configs/config.local.yaml"
if [ "${1:-}" = "--show" ]; then
  if [ ! -f "$REPO_ROOT/$CONFIG" ]; then
    echo "配置文件不存在: $REPO_ROOT/$CONFIG" >&2
    exit 1
  fi
  grep -A2 '^knowledge:' "$REPO_ROOT/$CONFIG" || echo "未配置 knowledge.source_dir"
  exit 0
fi

if [ $# -lt 1 ] || [ -z "${1:-}" ]; then
  echo "用法: bash scripts/set-kg-source.sh <MOOCCube 目录> [配置文件]" >&2
  exit 2
fi

PATH_VALUE="$1"
if [ $# -ge 2 ]; then
  CONFIG="$2"
fi
CONFIG_PATH="$REPO_ROOT/$CONFIG"

if [ ! -f "$CONFIG_PATH" ]; then
  echo "配置文件不存在: $CONFIG_PATH" >&2
  exit 1
fi

# 相对 backend/ 解析校验
CHECK_DIR="$PATH_VALUE"
if [ ! -d "$CHECK_DIR" ]; then
  CHECK_DIR="$REPO_ROOT/backend/$PATH_VALUE"
fi
if [ ! -d "$CHECK_DIR" ]; then
  echo "数据目录不存在: $PATH_VALUE（也尝试过 backend/$PATH_VALUE）" >&2
  exit 1
fi
if [ ! -f "$CHECK_DIR/relations/prerequisite-dependency.json" ]; then
  echo "目录里缺少 relations/prerequisite-dependency.json，请确认指向 MOOCCube 根目录" >&2
  exit 1
fi

if [ ! -f "$CONFIG_PATH.bak" ]; then
  cp "$CONFIG_PATH" "$CONFIG_PATH.bak"
  echo "已备份原配置: $CONFIG_PATH.bak"
fi

TMP_FILE="$(mktemp)"
awk -v value="$PATH_VALUE" '
  BEGIN { in_block = 0; found = 0 }
  /^knowledge:[[:space:]]*$/ { in_block = 1; print; next }
  in_block && /^[^[:space:]]/ {
    if (!found) { print "  source_dir: \"" value "\""; found = 1 }
    in_block = 0
  }
  in_block && /^[[:space:]]+source_dir:/ { print "  source_dir: \"" value "\""; found = 1; next }
  { print }
  END {
    if (in_block && !found) { print "  source_dir: \"" value "\""; found = 1 }
    if (!found) {
      print ""
      print "# 知识图谱数据源（平台本地外部数据，或用 scripts/set-kg-source.sh 切换）"
      print "knowledge:"
      print "  source_dir: \"" value "\""
    }
  }
' "$CONFIG_PATH" > "$TMP_FILE"
mv "$TMP_FILE" "$CONFIG_PATH"

echo "已更新 knowledge.source_dir -> $PATH_VALUE"
echo "配置文件: $CONFIG_PATH"
echo "下一步: (cd backend && CONFIG_PATH=configs/config.local.yaml go run ./cmd/seed_knowledge -dry-run)"

