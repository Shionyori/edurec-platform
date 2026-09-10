"""采集编排与字段归一化。

把 B 站两种来源（搜索接口 / 视频详情接口）的原始返回，统一成同一种 item 结构，
供 run.py 写出 JSON，再交给 Go 侧 cmd/import_bilibili 落库。

item 结构见 docs/bilibili-import.md 的「交接文件格式」一节。
"""

from __future__ import annotations

import html
import re
from typing import Any

from bili_client import BiliClient, RiskControlError

# B 站搜索结果的标题带 <em class="keyword"> 高亮标签，需剥离
_TAG_RE = re.compile(r"<[^>]+>")

SOURCE_URL_TEMPLATE = "https://www.bilibili.com/video/{bvid}"

# 标签数量上限，避免个别视频塞几十个标签撑爆字段
MAX_TAGS = 10


def _clean_text(value: Any) -> str:
    """剥离 HTML 标签并反转义实体。"""
    if value is None:
        return ""
    text = _TAG_RE.sub("", str(value))
    return html.unescape(text).strip()


def _to_int(value: Any) -> int:
    """B 站有些计数字段是字符串（如 "1.2万" 不会出现，但 "170306" 会出现）。"""
    if value is None or value == "":
        return 0
    try:
        return int(value)
    except (TypeError, ValueError):
        return 0


def _normalize_cover(url: Any) -> str:
    """封面可能是 //i1.hdslb.com/... 或 http://... ，统一成 https。"""
    if not url:
        return ""
    text = str(url).strip()
    if text.startswith("//"):
        return "https:" + text
    if text.startswith("http://"):
        return "https://" + text[len("http://") :]
    return text


def _format_duration(seconds: Any) -> str:
    """秒数 → MM:SS / HH:MM:SS。

    搜索接口返回的是 "16:08" 这种字符串，详情接口返回的是秒数；
    统一成字符串，前端 metadata 键值表直接可读。
    """
    if seconds is None or seconds == "":
        return ""
    if isinstance(seconds, str) and ":" in seconds:
        return seconds.strip()
    try:
        total = int(seconds)
    except (TypeError, ValueError):
        return ""
    hours, rem = divmod(total, 3600)
    minutes, secs = divmod(rem, 60)
    if hours:
        return f"{hours}:{minutes:02d}:{secs:02d}"
    return f"{minutes}:{secs:02d}"


def _split_tags(tag_field: Any) -> list[str]:
    """搜索接口的 tag 是逗号分隔字符串。"""
    if not tag_field:
        return []
    parts = [p.strip() for p in str(tag_field).split(",")]
    seen: list[str] = []
    for part in parts:
        if part and part not in seen:
            seen.append(part)
    return seen[:MAX_TAGS]


def _base_item(bvid: str, category: str) -> dict:
    return {
        "bvid": bvid,
        "title": "",
        "description": "",
        "cover_url": "",
        "author": "",
        "source_url": SOURCE_URL_TEMPLATE.format(bvid=bvid),
        "category": category,
        "tags": [],
        "view_count": 0,
        "metadata": {},
    }


def normalize_search_item(raw: dict, category: str) -> dict | None:
    """搜索接口 /x/web-interface/wbi/search/type 的单条结果 → item。"""
    bvid = _clean_text(raw.get("bvid"))
    if not bvid:
        return None

    item = _base_item(bvid, category)
    item["title"] = _clean_text(raw.get("title"))
    item["description"] = _clean_text(raw.get("description"))
    item["cover_url"] = _normalize_cover(raw.get("pic"))
    item["author"] = _clean_text(raw.get("author"))
    item["tags"] = _split_tags(raw.get("tag"))
    item["view_count"] = _to_int(raw.get("play"))
    item["metadata"] = {
        "bvid": bvid,
        "duration": _format_duration(raw.get("duration")),
        "pubdate": _to_int(raw.get("pubdate")),
        "typename": _clean_text(raw.get("typename")),
        "like": _to_int(raw.get("like")),
        "favorite": _to_int(raw.get("favorites")),
        "review": _to_int(raw.get("review")),
    }
    return item


def normalize_view_item(raw: dict, category: str, tags: list[str] | None = None) -> dict | None:
    """视频详情接口 /x/web-interface/view 的返回 → item。

    该接口不返回 tag 字段，标签只能由调用方（如 --tags 参数）补充。
    """
    bvid = _clean_text(raw.get("bvid"))
    if not bvid:
        return None

    owner = raw.get("owner") or {}
    stat = raw.get("stat") or {}

    item = _base_item(bvid, category)
    item["title"] = _clean_text(raw.get("title"))
    item["description"] = _clean_text(raw.get("desc"))
    item["cover_url"] = _normalize_cover(raw.get("pic"))
    item["author"] = _clean_text(owner.get("name"))
    item["tags"] = list(tags or [])
    item["view_count"] = _to_int(stat.get("view"))
    item["metadata"] = {
        "bvid": bvid,
        "aid": _to_int(raw.get("aid")),
        "duration": _format_duration(raw.get("duration")),
        "pubdate": _to_int(raw.get("pubdate")),
        "typename": _clean_text(raw.get("tname")),
        "like": _to_int(stat.get("like")),
        "favorite": _to_int(stat.get("favorite")),
        "review": _to_int(stat.get("reply")),
    }
    return item


class Collector:
    """按配置执行采集任务，返回去重后的 item 列表。"""

    def __init__(self, client: BiliClient, config: dict, verbose: bool = True):
        self.client = client
        self.config = config
        self.verbose = verbose
        self._seen: set[str] = set()
        self._items: list[dict] = []

    def _log(self, message: str) -> None:
        if self.verbose:
            print(f"[crawler] {message}", flush=True)

    def _add(self, item: dict | None) -> None:
        """按 bvid 跨任务去重。"""
        if item is None:
            return
        bvid = item["bvid"]
        if bvid in self._seen:
            return
        self._seen.add(bvid)
        self._items.append(item)

    def run(self) -> list[dict]:
        tasks = self.config.get("tasks") or []
        if not tasks:
            raise ValueError("配置里没有任何 tasks")

        for index, task in enumerate(tasks, start=1):
            category = (task.get("category") or "").strip()
            if not category:
                raise ValueError(f"第 {index} 个任务缺少 category")
            label = task.get("keyword") or task.get("up_mid") or task.get("bvids")
            self._log(f"任务 {index}/{len(tasks)}：{label} → 分类「{category}」")
            self._run_task(task, category)

        return self._items

    def _run_task(self, task: dict, category: str) -> None:
        if task.get("keyword"):
            self._collect_keyword(task, category)
        elif task.get("bvids"):
            self._collect_bvids(task, category)
        elif task.get("up_mid"):
            self._collect_space(task, category)
        else:
            raise ValueError("任务必须包含 keyword / bvids / up_mid 之一")

    def _collect_keyword(self, task: dict, category: str) -> None:
        keyword = str(task["keyword"]).strip()
        pages = int(task.get("pages", 1))
        page_size = int(task.get("page_size", 20))

        for page in range(1, pages + 1):
            results = self.client.search_videos(keyword, page=page, page_size=page_size)
            if not results:
                # B 站限流时会返回 code=0 但 result=null，和「真的没结果」无法区分
                if page == 1:
                    self._log(
                        f"  「{keyword}」首页无结果 —— 可能是关键词没匹配到，"
                        "也可能是被限流，建议稍后重试或放慢 delay_seconds"
                    )
                else:
                    self._log(f"  「{keyword}」第 {page} 页无结果，停止翻页")
                break
            for raw in results:
                self._add(normalize_search_item(raw, category))
            self._log(f"  「{keyword}」第 {page} 页 +{len(results)} 条")

    def _collect_bvids(self, task: dict, category: str) -> None:
        bvids = task.get("bvids") or []
        tags = task.get("tags") or []
        for bvid in bvids:
            bvid = str(bvid).strip()
            if not bvid:
                continue
            try:
                raw = self.client.video_detail(bvid)
            except Exception as exc:  # 单条失败不中断整批
                self._log(f"  {bvid} 获取失败：{exc}")
                continue
            item = normalize_view_item(raw, category, tags=tags)
            if item is None:
                self._log(f"  {bvid} 返回内容为空，跳过")
                continue
            self._add(item)
            self._log(f"  {bvid} -> {item['title'][:30]}")

    def _collect_space(self, task: dict, category: str) -> None:
        mid = task["up_mid"]
        pages = int(task.get("pages", 1))
        page_size = int(task.get("page_size", 30))
        try:
            for page in range(1, pages + 1):
                results = self.client.space_videos(mid, page=page, page_size=page_size)
                if not results:
                    break
                for raw in results:
                    item = normalize_view_item(
                        {
                            "bvid": raw.get("bvid"),
                            "title": raw.get("title"),
                            "desc": raw.get("description"),
                            "pic": raw.get("pic"),
                            "pubdate": raw.get("created"),
                            "duration": raw.get("length"),
                            "aid": raw.get("aid"),
                            "tname": raw.get("typename"),
                            "owner": {"name": raw.get("author")},
                            "stat": {
                                "view": raw.get("play"),
                                "reply": raw.get("comment"),
                            },
                        },
                        category,
                    )
                    self._add(item)
                self._log(f"  UP {mid} 第 {page} 页 +{len(results)} 条")
        except RiskControlError as exc:
            raise RiskControlError(
                exc.code,
                "UP 主接口被 B 站风控拦截，请用 --cookie 提供你自己登录后的 cookie 后重试"
                f"（原始信息：{exc.message}）",
                exc.endpoint,
            ) from exc
