#!/usr/bin/env bash
# =============================================================================
# check-contract-sync.sh — 校验 platform 侧的契约 fixture 与 engine 单一来源一致
#
# 契约的唯一来源是 engine 仓库的 contracts/v1/fixtures/。platform 把
# recommendations.json / recommendations.meta.json / invalid/* 复制到
# backend/internal/service/testdata/ 供 Go 测试使用；本脚本负责发现漂移。
#
# 用法（platform 仓库根目录）：
#   bash scripts/check-contract-sync.sh            # 只检查（默认）
#   bash scripts/check-contract-sync.sh --update   # 用 engine 的 fixture 覆盖本地副本
#   bash scripts/check-contract-sync.sh --allow-missing-engine
#                                                 # 找不到 engine 时跳过（CI 用）
#
# 环境变量：
#   ENGINE_ROOT   engine 仓库根目录（默认 ../edurec-engine）
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORM_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ENGINE_ROOT="${ENGINE_ROOT:-$PLATFORM_ROOT/../edurec-engine}"

CANON="$ENGINE_ROOT/contracts/v1/fixtures"
DEST="$PLATFORM_ROOT/backend/internal/service/testdata"
DEST_INVALID="$DEST/contract_invalid"

MODE="check"
ALLOW_MISSING=0
for arg in "$@"; do
  case "$arg" in
    --update) MODE="update" ;;
    --allow-missing-engine) ALLOW_MISSING=1 ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    *) echo "未知参数: $arg" >&2; exit 2 ;;
  esac
done

if [ ! -d "$CANON" ]; then
  if [ "$ALLOW_MISSING" -eq 1 ]; then
    echo "[contract-sync] engine 契约目录不存在，按 --allow-missing-engine 跳过"
    exit 0
  fi
  echo "[contract-sync] 错误: 找不到 engine 契约目录: $CANON" >&2
  echo "  设置 ENGINE_ROOT 指向 engine 仓库，或用 --allow-missing-engine 跳过" >&2
  exit 2
fi

map_src() {
  case "$(basename "$1")" in
    recommendations.json) echo "$DEST/recommendations.contract.json" ;;
    recommendations.meta.json) echo "$DEST/recommendations.contract.meta.json" ;;
    kg-generate.response.json) echo "$DEST/kg-generate.contract.json" ;;
    *) echo "$DEST_INVALID/$(basename "$1")" ;;
  esac
}

if [ "$MODE" = "update" ]; then
  mkdir -p "$DEST_INVALID"
  cp "$CANON/recommendations.json" "$DEST/recommendations.contract.json"
  cp "$CANON/recommendations.meta.json" "$DEST/recommendations.contract.meta.json"
  cp "$CANON/kg-generate.response.json" "$DEST/kg-generate.contract.json"
  cp "$CANON"/invalid/*.json "$DEST_INVALID/"
  echo "[contract-sync] 已从 $CANON 更新本地 fixture"
  exit 0
fi

failed=0
for src in "$CANON/recommendations.json" "$CANON/recommendations.meta.json" \
           "$CANON/kg-generate.response.json" "$CANON"/invalid/*.json; do
  dst="$(map_src "$src")"
  if [ ! -f "$dst" ]; then
    echo "[contract-sync] 缺失: $dst（先跑 --update）" >&2
    failed=1
  elif ! cmp -s "$src" "$dst"; then
    echo "[contract-sync] 漂移: $dst 与 $src 不一致" >&2
    failed=1
  fi
done

if [ "$failed" -ne 0 ]; then
  echo "[contract-sync] 失败：先改 engine 的 contracts/v1，再执行 --update" >&2
  exit 1
fi

echo "[contract-sync] OK：本地 fixture 与 engine 契约一致"
