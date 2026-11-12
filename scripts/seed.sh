#!/usr/bin/env bash
# =============================================================================
# seed.sh — 播种 sim 演示数据
#
# 把 engine 的模拟数据集（dataset/sim）拷到 backend/data/sim，再用 demo_seed
# 播种进 MySQL（用户/资源/类目/行为/评分 + 管理员 demo_admin）。
# 保证平台库 ID 与 engine 推荐产物 ID 一致，使个性化推荐可完整演示。
#
# 幂等：已播种（users 表非空）则跳过；如需重置请手动清库。
#
# 用法：
#   bash scripts/seed.sh
#
# 可用环境变量覆盖：
#   ENGINE_ROOT    engine 仓库根目录（默认 ../edurec-engine）
#   SIM_DIR        模拟数据目录（默认 $ENGINE_ROOT/dataset/sim）
#   CONFIG_PATH    后端配置（默认 configs/config.yaml，相对 backend/）
#   MYSQL_PASSWORD MySQL root 密码（默认 root）
#   MYSQL_DATABASE 数据库名（默认 edurec）
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORM_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$PLATFORM_ROOT/backend"

ENGINE_ROOT="${ENGINE_ROOT:-$PLATFORM_ROOT/../edurec-engine}"
SIM_DIR="${SIM_DIR:-$ENGINE_ROOT/dataset/sim}"
CONFIG_PATH="${CONFIG_PATH:-configs/config.yaml}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-root}"
MYSQL_DATABASE="${MYSQL_DATABASE:-edurec}"

log() { echo "[seed] $*"; }
die() { echo "[seed] 错误: $*" >&2; exit 1; }

# ---- 前置检查 ---------------------------------------------------------------
command -v docker >/dev/null || die "未找到 docker（需 MySQL 容器在运行）"
command -v go >/dev/null || die "未找到 go"
{ [ -d "$ENGINE_ROOT" ] || [ -d "$SIM_DIR" ]; } || \
  die "既没有 engine 目录 ($ENGINE_ROOT)，也没有模拟数据目录 ($SIM_DIR)"

# ---- [1] 拷贝 sim 数据 ---------------------------------------------------------
log "[1] 准备模拟数据"
if [ ! -d "$BACKEND_DIR/data/sim" ]; then
  if [ ! -d "$SIM_DIR" ]; then
    if [ -x "$ENGINE_ROOT/.venv/bin/python" ]; then
      echo "  未找到 $SIM_DIR，尝试用 engine 生成"
      ( cd "$ENGINE_ROOT" && "$ENGINE_ROOT/.venv/bin/python" -m scripts.gen_sim_data )
    fi
  fi
  [ -d "$SIM_DIR" ] || die "模拟数据不存在: $SIM_DIR（设置 SIM_DIR，或先在 engine 跑 python -m scripts.gen_sim_data）"
  cp -r "$SIM_DIR" "$BACKEND_DIR/data/sim"
  echo "  已从 $SIM_DIR 拷贝到 backend/data/sim"
else
  echo "  backend/data/sim 已存在，跳过"
fi

# ---- [2] 幂等播种 --------------------------------------------------------------
log "[2] 播种演示数据"
USER_COUNT="$(docker exec edurec-mysql mysql -uroot -p"$MYSQL_PASSWORD" -N -e \
  "SELECT COUNT(*) FROM $MYSQL_DATABASE.users" 2>/dev/null || echo 0)"
if [ "$USER_COUNT" = "0" ] || [ -z "$USER_COUNT" ]; then
  ( cd "$BACKEND_DIR" && CONFIG_PATH="$CONFIG_PATH" go run ./cmd/demo_seed -with-behaviors )
else
  echo "  已有 ${USER_COUNT} 个用户，跳过（重置请用 clean.sh）"
fi

log "完成，账号: demo1/demo123456（普通），demo_admin/demo123456（管理员）"
