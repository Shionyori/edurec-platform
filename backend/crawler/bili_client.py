"""B 站公开 JSON 接口客户端。

设计原则（见 docs/bilibili-import.md）：

- **只用官方 web 接口的 JSON 返回**，不解析 HTML 页面（B 站改版即失效）
- **不伪造设备指纹、不绕过人机校验**：`buvid3` 由直接访问首页让 B 站正常下发，
  与浏览器行为一致；遇到风控（-352/-412）直接报错退出，不做退避重试轰炸
- 强制请求间隔，默认 1.5–3.0 秒，可配置

接口参考：https://github.com/SocialSisterYi/bilibili-API-collect
"""

from __future__ import annotations

import hashlib
import random
import time
import urllib.parse
from typing import Any

import requests

# WBI 签名用的 64 位重排表，B 站前端固定值
MIXIN_KEY_ENC_TAB = [
    46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35,
    27, 43, 5, 49, 33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13,
    37, 48, 7, 16, 24, 55, 40, 61, 26, 17, 0, 1, 60, 51, 30, 4,
    22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11, 36, 20, 34, 44, 52,
]

USER_AGENT = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

# 风控相关错误码：命中即中止，不重试
RISK_CONTROL_CODES = {-352, -412}

NAV_URL = "https://api.bilibili.com/x/web-interface/nav"
SEARCH_URL = "https://api.bilibili.com/x/web-interface/wbi/search/type"
VIEW_URL = "https://api.bilibili.com/x/web-interface/view"
SPACE_URL = "https://api.bilibili.com/x/space/wbi/arc/search"


class BiliError(RuntimeError):
    """B 站接口返回非 0 code。"""

    def __init__(self, code: int, message: str, endpoint: str = ""):
        self.code = code
        self.message = message
        self.endpoint = endpoint
        super().__init__(f"[{code}] {message}" + (f" ({endpoint})" if endpoint else ""))


class RiskControlError(BiliError):
    """命中 B 站风控（-352 / -412）。不重试，直接向上抛。"""


class BiliClient:
    """B 站接口客户端。一次运行内复用 session 与 WBI 密钥。"""

    def __init__(
        self,
        cookie: str = "",
        delay: tuple[float, float] = (1.5, 3.0),
        max_retries: int = 3,
        timeout: int = 15,
        verbose: bool = True,
    ):
        self.delay = delay
        self.max_retries = max_retries
        self.timeout = timeout
        self.verbose = verbose
        self.session = requests.Session()
        self.session.headers.update(
            {
                "User-Agent": USER_AGENT,
                "Referer": "https://www.bilibili.com/",
                "Accept-Language": "zh-CN,zh;q=0.9",
            }
        )
        if cookie:
            self.session.headers["Cookie"] = cookie
        self._mixin_key: str | None = None
        self._request_count = 0

    # ---------------------------------------------------------------- 日志

    def _log(self, message: str) -> None:
        if self.verbose:
            print(f"[crawler] {message}", flush=True)

    # ------------------------------------------------------------ 初始化

    def bootstrap(self) -> None:
        """取真实 buvid3 与 WBI 密钥。

        WBI 密钥约每 2–4 周轮换，因此每次运行都重新拉取，不做持久化缓存。
        """
        self._log("访问 B 站首页以获取真实 buvid3 …")
        try:
            self.session.get("https://www.bilibili.com/", timeout=self.timeout)
        except requests.RequestException as exc:
            raise BiliError(0, f"访问 B 站首页失败: {exc}") from exc

        if "buvid3" not in self.session.cookies.get_dict():
            self._log("警告：未从首页拿到 buvid3，搜索接口可能返回 -412")

        nav = self._request(NAV_URL, allow_codes={0, -101})
        wbi_img = (nav.get("data") or {}).get("wbi_img") or {}
        img_url = wbi_img.get("img_url", "")
        sub_url = wbi_img.get("sub_url", "")
        if not img_url or not sub_url:
            raise BiliError(nav.get("code", 0), "nav 接口未返回 WBI 密钥")

        img_key = img_url.rsplit("/", 1)[-1].split(".")[0]
        sub_key = sub_url.rsplit("/", 1)[-1].split(".")[0]
        orig = img_key + sub_key
        self._mixin_key = "".join(orig[i] for i in MIXIN_KEY_ENC_TAB)[:32]
        self._log("WBI 密钥就绪，可以开始采集")

    def close(self) -> None:
        self.session.close()

    # ------------------------------------------------------------ 请求层

    def _throttle(self) -> None:
        """强制请求间隔。第一个请求不等待。"""
        if self._request_count == 0:
            return
        low, high = self.delay
        if high > 0:
            time.sleep(random.uniform(low, high))

    def _request(
        self,
        url: str,
        params: dict[str, Any] | None = None,
        *,
        signed: bool = False,
        allow_codes: set[int] | None = None,
    ) -> dict:
        """发起一次 GET，返回响应 JSON。

        - 网络/5xx 失败按 max_retries 重试，指数退避
        - 风控错误码（-352/-412）立即抛出 RiskControlError，不重试
        """
        allow_codes = allow_codes or {0}
        if signed:
            if self._mixin_key is None:
                raise BiliError(0, "WBI 密钥未初始化，请先调用 bootstrap()")
            params = self._sign(params or {})

        last_exc: Exception | None = None
        for attempt in range(self.max_retries):
            self._throttle()
            try:
                resp = self.session.get(url, params=params, timeout=self.timeout)
                self._request_count += 1
            except requests.RequestException as exc:
                last_exc = exc
                wait = 2**attempt
                self._log(f"请求失败（{exc}），{wait}s 后重试 …")
                time.sleep(wait)
                continue

            if resp.status_code >= 500:
                wait = 2**attempt
                self._log(f"服务端 {resp.status_code}，{wait}s 后重试 …")
                time.sleep(wait)
                last_exc = BiliError(resp.status_code, f"HTTP {resp.status_code}", url)
                continue

            try:
                body = resp.json()
            except ValueError as exc:
                raise BiliError(resp.status_code, f"响应不是 JSON: {resp.text[:120]}", url) from exc

            code = body.get("code", 0)
            if code in RISK_CONTROL_CODES:
                raise RiskControlError(code, body.get("message", "风控校验失败"), url)
            if code not in allow_codes:
                raise BiliError(code, body.get("message", "接口返回错误"), url)
            return body

        raise BiliError(0, f"重试 {self.max_retries} 次后仍失败: {last_exc}", url)

    def _sign(self, params: dict[str, Any]) -> dict[str, Any]:
        """按 WBI 算法补 wts 与 w_rid。"""
        signed = dict(params)
        signed["wts"] = int(time.time())
        query = urllib.parse.urlencode(sorted(signed.items()))
        signed["w_rid"] = hashlib.md5((query + self._mixin_key).encode()).hexdigest()
        return signed

    # ------------------------------------------------------------ 业务接口

    def search_videos(self, keyword: str, page: int = 1, page_size: int = 20) -> list[dict]:
        """按关键词搜索视频。返回原始 result 列表（title 含 <em> 高亮标签，由上层清洗）。"""
        body = self._request(
            SEARCH_URL,
            {
                "search_type": "video",
                "keyword": keyword,
                "page": page,
                "page_size": page_size,
            },
            signed=True,
        )
        return (body.get("data") or {}).get("result") or []

    def video_detail(self, bvid: str) -> dict:
        """按 BV 号取视频详情。该接口无需登录、无需签名。"""
        body = self._request(VIEW_URL, {"bvid": bvid})
        return body.get("data") or {}

    def space_videos(self, mid: int, page: int = 1, page_size: int = 30) -> list[dict]:
        """取 UP 主投稿列表。

        ⚠️ 该接口风控较严，免登录调用通常返回 -352。命中时会抛
        RiskControlError，由调用方提示用户改用 --cookie 提供自己登录后的凭证。
        本客户端不实现任何人机校验绕过。
        """
        body = self._request(
            SPACE_URL,
            {
                "mid": mid,
                "ps": page_size,
                "pn": page,
                "order": "pubdate",
                "platform": "web",
                "web_location": 1550101,
                "dm_img_list": "[]",
                "dm_img_str": "",
                "dm_img_inter": '{"ds":[],"wh":[0,0,0],"of":[0,0,0]}',
            },
            signed=True,
        )
        return ((body.get("data") or {}).get("list") or {}).get("vlist") or []
