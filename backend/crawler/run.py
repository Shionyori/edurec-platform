"""B 站教育资源采集入口。

用法：

    cd backend/crawler
    python run.py --config config.example.yaml              # 采集并写出 JSON
    python run.py --config config.example.yaml --dry-run    # 只采集，不写文件
    python run.py --config config.example.yaml --cookie "SESSDATA=xxx; buvid3=yyy"

产物写到 backend/data/bilibili/：
    <run_id>.json   本次采集的完整快照
    latest.json     同上，供 Go 侧导入命令默认读取

采完再执行导入：

    cd backend
    CONFIG_PATH=configs/config.local.yaml go run ./cmd/import_bilibili
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime
from pathlib import Path

import yaml

from bili_client import BiliClient, BiliError, RiskControlError
from collect import Collector

# backend/crawler/run.py → backend/data/bilibili
DEFAULT_OUT_DIR = Path(__file__).resolve().parent.parent / "data" / "bilibili"

ITEM_REQUIRED_FIELDS = ("bvid", "title", "source_url", "category")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="采集 B 站视频元数据，输出给平台导入命令使用",
    )
    parser.add_argument(
        "--config",
        default="config.example.yaml",
        help="任务配置文件（YAML），默认 config.example.yaml",
    )
    parser.add_argument(
        "--out",
        default=str(DEFAULT_OUT_DIR),
        help=f"输出目录，默认 {DEFAULT_OUT_DIR}",
    )
    parser.add_argument(
        "--cookie",
        default="",
        help="自定义 cookie（你自己登录后的凭证）。仅在采集 UP 主投稿时才需要",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="只采集并打印统计，不写文件",
    )
    parser.add_argument(
        "--quiet",
        action="store_true",
        help="不打印过程日志",
    )
    return parser.parse_args()


def load_config(path: str) -> dict:
    config_path = Path(path)
    if not config_path.is_file():
        raise FileNotFoundError(f"配置文件不存在: {config_path}")
    with config_path.open("r", encoding="utf-8") as handle:
        return yaml.safe_load(handle) or {}


def validate(items: list[dict]) -> tuple[list[dict], int]:
    """丢掉缺必填字段的条目，返回 (有效条目, 丢弃数)。"""
    valid: list[dict] = []
    dropped = 0
    for item in items:
        if all(str(item.get(field, "")).strip() for field in ITEM_REQUIRED_FIELDS):
            valid.append(item)
        else:
            dropped += 1
    return valid, dropped


def write_output(out_dir: Path, items: list[dict]) -> tuple[Path, Path]:
    out_dir.mkdir(parents=True, exist_ok=True)
    payload = {
        "version": 1,
        "generated_at": datetime.now().astimezone().isoformat(timespec="seconds"),
        "source": "bilibili",
        "items": items,
    }
    body = json.dumps(payload, ensure_ascii=False, indent=2)

    run_id = datetime.now().strftime("%Y%m%d_%H%M%S")
    run_file = out_dir / f"{run_id}.json"
    latest_file = out_dir / "latest.json"
    run_file.write_text(body, encoding="utf-8")
    latest_file.write_text(body, encoding="utf-8")
    return run_file, latest_file


def make_console_safe() -> None:
    """Windows 控制台默认是 GBK，遇到无法编码的字符会抛 UnicodeEncodeError 中断采集。

    这里只把错误处理改成 replace，不改编码 —— 中文在 GBK 下本来就能正常显示。
    """
    for stream in (sys.stdout, sys.stderr):
        if hasattr(stream, "reconfigure"):
            stream.reconfigure(errors="replace")


def main() -> int:
    make_console_safe()
    args = parse_args()
    verbose = not args.quiet

    try:
        config = load_config(args.config)
    except (FileNotFoundError, yaml.YAMLError) as exc:
        print(f"[crawler] 配置加载失败：{exc}", file=sys.stderr)
        return 1

    delay_cfg = config.get("delay_seconds") or [1.5, 3.0]
    client = BiliClient(
        cookie=args.cookie,
        delay=(float(delay_cfg[0]), float(delay_cfg[1])),
        max_retries=int(config.get("max_retries", 3)),
        verbose=verbose,
    )

    try:
        client.bootstrap()
        items = Collector(client, config, verbose=verbose).run()
    except RiskControlError as exc:
        print(f"\n[crawler] 被 B 站风控拦截：{exc.message}", file=sys.stderr)
        print("[crawler] 建议：降低频率、稍后重试，或对 UP 主任务提供 --cookie", file=sys.stderr)
        return 2
    except BiliError as exc:
        print(f"\n[crawler] 接口调用失败：{exc}", file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        print("\n[crawler] 已中断", file=sys.stderr)
        return 130
    finally:
        client.close()

    items, dropped = validate(items)
    print(f"\n[crawler] 采集完成：有效 {len(items)} 条，丢弃 {dropped} 条")

    if args.dry_run:
        print("[crawler] --dry-run：不写文件")
        for item in items[:5]:
            print(f"  {item['bvid']}  {item['title'][:40]}  [{item['category']}]")
        return 0

    run_file, latest_file = write_output(Path(args.out), items)
    print(f"[crawler] 已写出 {run_file}")
    print(f"[crawler] 已更新 {latest_file}")
    print("[crawler] 下一步：cd backend && CONFIG_PATH=configs/config.local.yaml go run ./cmd/import_bilibili")
    return 0


if __name__ == "__main__":
    sys.exit(main())
