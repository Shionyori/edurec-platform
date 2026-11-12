#!/usr/bin/env bash
# =============================================================================
# start.sh — 本地开发/演示一键启动（只起服务，不准备数据）
#
# 依次完成：起 MySQL/Redis 容器 -> 等 MySQL 就绪 -> 后台启动后端 ->
# 后台启动前端 -> 打印访问信息。
#
# 数据准备（二选一，启动前/后执行均可）：
#   bash scripts/seed.sh        # 播种 sim 演示数据
#   bash scripts/bilibili.sh    # 采集 B 站真实数据
# 个性化推荐刷新：bash scripts/handoff.sh
#
# 用法：
#   bash scripts/start.sh           # 一键启动
#   bash scripts/start.sh --with-kg # 另起 engine 的 KG 在线服务（可选旁路）
#   bash scripts/stop.sh            # 停止服务
#   bash scripts/stop.sh --with-db  # 连容器一起停
#
# 可用环境变量覆盖：
#   CONFIG_PATH    后端配置（默认 configs/config.yaml，相对 backend/）
#   MYSQL_PASSWORD MySQL root 密码（默认 root，与 config.yaml 对齐）
#   MYSQL_DATABASE 数据库名（默认 edurec）
#   ENGINE_ROOT    engine 仓库（默认 ../edurec-engine）
#   KG_PORT        KG 服务端口（默认 8090）
#   KNOWLEDGE_GENERATOR_TOKEN  KG 服务 token（可选；与 engine 侧一致）
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORM_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$PLATFORM_ROOT/backend"
FRONTEND_DIR="$PLATFORM_ROOT/frontend"
RUN_DIR="$SCRIPT_DIR/.run"

CONFIG_PATH="${CONFIG_PATH:-configs/config.yaml}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-root}"
MYSQL_DATABASE="${MYSQL_DATABASE:-edurec}"
BACKEND_PORT="${BACKEND_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"
ENGINE_ROOT="${ENGINE_ROOT:-$PLATFORM_ROOT/../edurec-engine}"
ENGINE_PYTHON="${ENGINE_PYTHON:-$ENGINE_ROOT/.venv/bin/python}"
KG_PORT="${KG_PORT:-8090}"

WITH_KG=0
for arg in "$@"; do
  case "$arg" in
    --with-kg) WITH_KG=1 ;;
    -h|--help) sed -n '2,22p' "$0"; exit 0 ;;
    *) echo "未知参数: $arg" >&2; exit 2 ;;
  esac
done

log() { echo "[start] $*"; }
die() { echo "[start] 错误: $*" >&2; exit 1; }
port_listening() { ss -tln 2>/dev/null | grep -qE ":${1}([[:space:]]|\$)"; }

# ---- 前置检查 ---------------------------------------------------------------
command -v docker >/dev/null || die "未找到 docker"
command -v go >/dev/null || die "未找到 go"
# 前端需要 node/pnpm（nvm 通常挂在 .zshrc，bash 里手动加载一次）
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" || true
command -v pnpm >/dev/null || die "未找到 pnpm（前端需要）"

# ---- [1] 容器 -----------------------------------------------------------------
log "[1] 启动 MySQL / Redis 容器"
if docker ps -a --format '{{.Names}}' | grep -qx edurec-mysql; then
  docker start edurec-mysql >/dev/null 2>&1 || true
  echo "  MySQL: 已存在，复用"
else
  docker run -d --name edurec-mysql -p 3306:3306 \
    -e MYSQL_ROOT_PASSWORD="$MYSQL_PASSWORD" -e MYSQL_DATABASE="$MYSQL_DATABASE" mysql:8 >/dev/null
  echo "  MySQL: 已创建"
fi
if docker ps -a --format '{{.Names}}' | grep -qx edurec-redis; then
  docker start edurec-redis >/dev/null 2>&1 || true
  echo "  Redis: 已存在，复用"
else
  docker run -d --name edurec-redis -p 6379:6379 redis:7 >/dev/null
  echo "  Redis: 已创建"
fi

# ---- [2] 等 MySQL --------------------------------------------------------------
log "[2] 等待 MySQL 就绪"
for i in $(seq 1 30); do
  if docker exec edurec-mysql mysqladmin ping -uroot -p"$MYSQL_PASSWORD" --silent >/dev/null 2>&1; then
    echo "  MySQL 就绪（第 ${i} 次探测）"
    break
  fi
  [ "$i" = 30 ] && die "MySQL 迟迟未就绪，查看日志：docker logs edurec-mysql"
  sleep 2
done

# ---- [3] 后端 ------------------------------------------------------------------
log "[3] 启动后端 (port=${BACKEND_PORT})"
mkdir -p "$RUN_DIR"
if port_listening "$BACKEND_PORT"; then
  echo "  端口 ${BACKEND_PORT} 已被占用，跳过"
else
  ( cd "$BACKEND_DIR" && CONFIG_PATH="$CONFIG_PATH" \
    KNOWLEDGE_GENERATOR_TOKEN="${KNOWLEDGE_GENERATOR_TOKEN:-}" \
    nohup go run ./cmd/server > "$RUN_DIR/backend.log" 2>&1 & \
    echo $! > "$RUN_DIR/backend.pid" )
  echo "  后端已启动，日志: scripts/.run/backend.log"
fi

# ---- [4] 前端 ------------------------------------------------------------------
log "[4] 启动前端 (port=${FRONTEND_PORT})"
if port_listening "$FRONTEND_PORT"; then
  echo "  端口 ${FRONTEND_PORT} 已被占用，跳过"
else
  ( cd "$FRONTEND_DIR" && \
    nohup env VITE_MOCK=0 pnpm dev > "$RUN_DIR/frontend.log" 2>&1 & \
    echo $! > "$RUN_DIR/frontend.pid" )
  echo "  前端已启动，日志: scripts/.run/frontend.log"
fi

# ---- [4.5] 可选：engine KG 在线服务 -------------------------------------------
if [ "$WITH_KG" = 1 ]; then
  log "[4.5] 启动 engine KG 服务 (port=${KG_PORT})"
  [ -d "$ENGINE_ROOT" ] || die "engine 目录不存在: $ENGINE_ROOT（可用 ENGINE_ROOT 覆盖）"
  [ -x "$ENGINE_PYTHON" ] || die "engine 依赖未安装: $ENGINE_PYTHON"
  if port_listening "$KG_PORT"; then
    echo "  端口 ${KG_PORT} 已被占用，跳过"
  else
    KG_ARGS=(--host 127.0.0.1 --port "$KG_PORT")
    [ -n "${KG_READY_FILE:-}" ] && KG_ARGS+=(--ready-file "$KG_READY_FILE")
    ( cd "$ENGINE_ROOT" && \
      nohup env KG_SERVICE_TOKEN="${KNOWLEDGE_GENERATOR_TOKEN:-}" \
      "$ENGINE_PYTHON" scripts/kg_server.py "${KG_ARGS[@]}" \
      > "$RUN_DIR/kg.log" 2>&1 & \
      echo $! > "$RUN_DIR/kg.pid" )
    echo "  KG 服务已启动，日志: scripts/.run/kg.log"
  fi
fi

# ---- [5] 总结 ------------------------------------------------------------------
log "[5] 全部完成"
echo "  后端:   http://localhost:${BACKEND_PORT}"
echo "  前端:   http://localhost:${FRONTEND_PORT}"
[ "$WITH_KG" = 1 ] && echo "  KG:     http://127.0.0.1:${KG_PORT}（可选在线能力）"
echo "  数据库: MySQL root/${MYSQL_PASSWORD} @ localhost:3306 / ${MYSQL_DATABASE}"
