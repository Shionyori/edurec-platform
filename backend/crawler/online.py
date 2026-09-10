"""在线 B 站采集入口（供 Go 后端 os/exec 调用）。

用法：

    python online.py search --keyword X [--limit N] [--category Y]
    python online.py comments --bvid BVxxx [--limit N]

约定（Go 侧 `exec.Command().Output()` 依赖）：

- **stdout 只输出一行 JSON**（成功时为 `{"items": [...]}` 或 `{"comments": [...]}`），
  日志与错误一律走 stderr
- 编码统一 UTF-8：stdout 经 `sys.stdout.buffer` 写原始字节，stderr 显式 reconfigure 为 UTF-8，
  规避 Windows 子进程默认 GBK 导致的乱码
- 失败时返回非 0 退出码、stdout 为空；Go 侧据此区分成功/失败
"""

from __future__ import annotations

import argparse
import json
import sys

from bili_client import BiliClient, BiliError, RiskControlError
from collect import normalize_search_item, normalize_comment

DEFAULT_CATEGORY = "B站视频"


def _force_utf8() -> None:
    """stdout/stderr 统一 UTF-8，避免 Windows 控制台默认 GBK 乱码。"""
    for stream in (sys.stdout, sys.stderr):
        if hasattr(stream, "reconfigure"):
            try:
                stream.reconfigure(encoding="utf-8", errors="replace")
            except (ValueError, OSError):
                pass


def _emit(payload: dict) -> None:
    """向 stdout 写一行 UTF-8 JSON（成功路径专用）。"""
    body = json.dumps(payload, ensure_ascii=False)
    sys.stdout.buffer.write(body.encode("utf-8") + b"\n")
    sys.stdout.buffer.flush()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="在线采集 B 站内容（供 Go 后端调用）")
    sub = parser.add_subparsers(dest="command", required=True)

    search = sub.add_parser("search", help="按关键词搜索视频")
    search.add_argument("--keyword", required=True, help="搜索关键词")
    search.add_argument("--limit", type=int, default=20, help="返回条数，默认 20")
    search.add_argument("--category", default=DEFAULT_CATEGORY, help="落库分类名")

    comments = sub.add_parser("comments", help="取视频评论")
    comments.add_argument("--bvid", required=True, help="视频 BV 号")
    comments.add_argument("--limit", type=int, default=20, help="返回条数，默认 20")

    parser.add_argument("--cookie", default="", help="可选：自定义 cookie")
    return parser.parse_args()


def _make_client(args: argparse.Namespace) -> BiliClient:
    # 在线单次调用，verbose=False：过程日志不落 stdout，错误经异常→stderr 暴露
    return BiliClient(
        cookie=args.cookie,
        delay=(1.5, 3.0),
        max_retries=3,
        verbose=False,
    )


def _do_search(client: BiliClient, args: argparse.Namespace) -> dict:
    results = client.search_videos(args.keyword, page=1, page_size=args.limit)
    items: list[dict] = []
    for raw in results:
        item = normalize_search_item(raw, args.category)
        if item is not None:
            items.append(item)
    return {"items": items}


def _do_comments(client: BiliClient, args: argparse.Namespace) -> dict:
    replies = client.get_comments(args.bvid, limit=args.limit)
    comments: list[dict] = []
    for raw in replies:
        comment = normalize_comment(raw)
        if comment is not None:
            comments.append(comment)
    return {"comments": comments}


def main() -> int:
    _force_utf8()
    args = parse_args()
    client = _make_client(args)

    try:
        client.bootstrap()
        if args.command == "search":
            payload = _do_search(client, args)
        else:
            payload = _do_comments(client, args)
    except RiskControlError as exc:
        print(f"[online] 被 B 站风控拦截：{exc.message}", file=sys.stderr)
        return 2
    except BiliError as exc:
        print(f"[online] 接口调用失败：{exc}", file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        print("[online] 已中断", file=sys.stderr)
        return 130
    finally:
        client.close()

    _emit(payload)
    return 0


if __name__ == "__main__":
    sys.exit(main())
