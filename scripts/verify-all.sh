#!/usr/bin/env bash
# =============================================================================
# verify-all.sh — 两仓契约 + 代码的一键验证
#
# 在 platform 仓库根目录执行；engine 默认在 ../edurec-engine。
# 依次检查：
#   1. engine 测试（含契约测试）
#   2. engine 的 golden fixture 自校验
#   3. platform 的契约 fixture 与 engine 单一来源是否一致
#   4. platform 后端 go test + go vet
#   5. （可选 --with-frontend）前端 test + type-check + eslint
#
# 用法：
#   bash scripts/verify-all.sh
#   bash scripts/verify-all.sh --with-frontend
#
# 环境变量：
#   ENGINE_ROOT   engine 仓库根目录（默认 ../edurec-engine）
#   ENGINE_PYTHON engine 的 python（默认 $ENGINE_ROOT/.venv/bin/python）
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORM_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ENGINE_ROOT="${ENGINE_ROOT:-$PLATFORM_ROOT/../edurec-engine}"
ENGINE_PYTHON="${ENGINE_PYTHON:-$ENGINE_ROOT/.venv/bin/python}"

WITH_FRONTEND=0
for arg in "$@"; do
  case "$arg" in
    --with-frontend) WITH_FRONTEND=1 ;;
    -h|--help) sed -n '2,18p' "$0"; exit 0 ;;
    *) echo "未知参数: $arg" >&2; exit 2 ;;
  esac
done

log() { echo "[verify] $*"; }
die() { echo "[verify] 错误: $*" >&2; exit 1; }

[ -d "$ENGINE_ROOT" ] || die "engine 目录不存在: $ENGINE_ROOT"
[ -x "$ENGINE_PYTHON" ] || die "engine python 不存在: $ENGINE_PYTHON"

log "[1/5] engine 测试"
( cd "$ENGINE_ROOT" && "$ENGINE_PYTHON" -m pytest -q )

log "[2/5] engine 契约 fixture 自校验"
( cd "$ENGINE_ROOT" &&
  "$ENGINE_PYTHON" contracts/v1/validate.py snapshot contracts/v1/fixtures/snapshot &&
  "$ENGINE_PYTHON" contracts/v1/validate.py recommendations \
    contracts/v1/fixtures/recommendations.json \
    --envelope contracts/v1/fixtures/recommendations.meta.json )

log "[3/5] platform fixture 与 engine 契约同步"
( cd "$PLATFORM_ROOT" && bash scripts/check-contract-sync.sh )

log "[4/5] platform 后端测试"
( cd "$PLATFORM_ROOT/backend" && go test ./... && go vet ./... )

if [ "$WITH_FRONTEND" -eq 1 ]; then
  log "[5/5] platform 前端测试"
  export NVM_DIR="$HOME/.nvm"
  [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" || true
  ( cd "$PLATFORM_ROOT/frontend" &&
    pnpm test && pnpm type-check && pnpm exec eslint . )
else
  log "[5/5] 跳过前端（用 --with-frontend 开启）"
fi

log "全部通过"
