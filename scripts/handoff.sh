#!/usr/bin/env bash
# =============================================================================
# handoff.sh — 一轮推荐刷新（platform → engine → platform）
#
# 把「导出 → 训练 → 推理 → 导入」整条离线交接链路固化为一条命令，
# 与 docs/data-handoff.md 的手动步骤一一对应。平台只认「CSV 快照进、
# 排名 ID 列表出」这一对契约，engine 内部算法（DSSM/DeepFM/MMR）不可见，
# 换算法或模型时本脚本无需改动。
#
# 前置条件：
#   1. 后端服务器已启动（第 ⑥ 步经 HTTP 导入，需管理员账号）
#   2. engine 已安装依赖（python 可执行，且 `python -m scripts.train_all` 可导入）
#
# 用法：
#   bash scripts/handoff.sh                 # 完整一轮：导出 + 训练 + 推理 + 导入
#   bash scripts/handoff.sh --infer-only    # 复用 model/models.pt，只导出 + 推理 + 导入
#
# 可用环境变量覆盖：
#   ENGINE_ROOT   engine 仓库根目录（默认 ../edurec-engine，与 platform 并列）
#   CONFIG_PATH   后端配置（默认 configs/config.yaml，相对 backend/）
#   SNAPSHOT_DIR  快照输出目录（默认 data/snapshots，相对 backend/）
#   BASE_URL      后端地址（默认 http://localhost:8080）
#   ADMIN_USER    管理员账号（默认 demo_admin）
#   ADMIN_PASS    管理员密码（默认 demo123456）
# =============================================================================
set -euo pipefail

# ---- 路径与参数 -------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORM_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$PLATFORM_ROOT/backend"

ENGINE_ROOT="${ENGINE_ROOT:-$PLATFORM_ROOT/../edurec-engine}"
CONFIG_PATH="${CONFIG_PATH:-configs/config.yaml}"
SNAPSHOT_DIR="${SNAPSHOT_DIR:-data/snapshots}"
BASE_URL="${BASE_URL:-http://localhost:8080}"
ADMIN_USER="${ADMIN_USER:-demo_admin}"
ADMIN_PASS="${ADMIN_PASS:-demo123456}"

INFER_ONLY=0
for arg in "$@"; do
  case "$arg" in
    --infer-only) INFER_ONLY=1 ;;
    -h|--help) sed -n '2,23p' "$0"; exit 0 ;;
    *) echo "未知参数: $arg" >&2; exit 2 ;;
  esac
done

# ---- 工具函数 ---------------------------------------------------------------
log() { printf '\n\033[1;36m[handoff]\033[0m %s\n' "$*"; }
die() { printf '\033[1;31m[handoff][错误]\033[0m %s\n' "$*" >&2; exit 1; }

# ---- 前置检查 ---------------------------------------------------------------
[ -d "$ENGINE_ROOT" ] || die "engine 目录不存在: $ENGINE_ROOT（可用 ENGINE_ROOT 覆盖）"
[ -d "$BACKEND_DIR" ] || die "backend 目录不存在: $BACKEND_DIR"

# ① 导出快照
log "① 导出平台数据快照 (CONFIG_PATH=$CONFIG_PATH)"
( cd "$BACKEND_DIR" && CONFIG_PATH="$CONFIG_PATH" go run ./cmd/export_snapshot )

run_id="$(ls -1t "$BACKEND_DIR/$SNAPSHOT_DIR" | head -1)"
[ -n "$run_id" ] || die "未在 $SNAPSHOT_DIR 下找到快照目录"
log "快照 run_id = $run_id"

# ② 快照交给 engine
log "② 拷贝快照到 engine: dataset/platform_snapshot/$run_id/"
mkdir -p "$ENGINE_ROOT/dataset/platform_snapshot"
cp -r "$BACKEND_DIR/$SNAPSHOT_DIR/$run_id" "$ENGINE_ROOT/dataset/platform_snapshot/"

# ③ 训练（--infer-only 跳过）
if [ "$INFER_ONLY" -eq 0 ]; then
  log "③ engine 训练 (train_all)"
  ( cd "$ENGINE_ROOT" && python -m scripts.train_all \
      --data-source platform --snapshot-dir "dataset/platform_snapshot/$run_id" )
else
  log "③ 跳过训练（--infer-only，复用 model/models.pt）"
fi

# ④ 全量推理
log "④ engine 全量推理 (run_batch_infer)"
( cd "$ENGINE_ROOT" && python -m scripts.run_batch_infer \
    --data-source platform --snapshot-dir "dataset/platform_snapshot/$run_id" )

# ⑤ 推理结果放回 platform
log "⑤ 拷贝推荐结果回 platform: data/recommendations.json"
mkdir -p "$BACKEND_DIR/data"
cp "$ENGINE_ROOT/model/recommendations.json" "$BACKEND_DIR/data/recommendations.json"

# ⑥ 管理员登录并导入缓存表
log "⑥ 管理员登录并导入推荐缓存 (POST /api/v1/admin/recommendations/import)"
login_resp="$(curl -sf -X POST "$BASE_URL/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}")" \
  || die "登录失败——请确认后端已启动 ($BASE_URL) 且管理员账号正确"

token="$(printf '%s' "$login_resp" | python -c \
  'import sys,json; print(json.load(sys.stdin)["data"]["access_token"])')"
curl -sf -X POST "$BASE_URL/api/v1/admin/recommendations/import" \
  -H "Authorization: Bearer $token" \
  -H 'Content-Type: application/json'
printf '\n'

log "完成。GET /api/v1/recommendations 将返回 engine 最新结果"
