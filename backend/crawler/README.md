# B 站教育资源采集

按关键词 / BV 号 / UP 主采集 B 站公开视频元数据，输出 JSON 交给平台的导入命令落库。

采集结果在前台作为 `type=video` 的普通资源展示，不新增专区。

## 依赖

```bash
pip install -r requirements.txt
```

## 用法

```bash
cd backend/crawler

# 编辑任务配置（关键词 → 目标分类）
cp config.example.yaml config.yaml

# 只采集不写文件，先看看抓到什么
python run.py --config config.yaml --dry-run

# 正式采集，写出到 backend/data/bilibili/
python run.py --config config.yaml

# 采集 UP 主投稿（需要你自己登录后的 cookie）
python run.py --config config.yaml --cookie "SESSDATA=xxx; buvid3=yyy"
```

参数：

| 参数 | 说明 |
|---|---|
| `--config` | 任务配置文件，默认 `config.example.yaml` |
| `--out` | 输出目录，默认 `backend/data/bilibili` |
| `--cookie` | 自定义 cookie，仅 UP 主入口需要 |
| `--dry-run` | 只采集并打印，不写文件 |
| `--quiet` | 不打印过程日志 |

采完执行导入：

```bash
cd backend
CONFIG_PATH=configs/config.local.yaml go run ./cmd/import_bilibili
```

## 任务配置

每个任务必须指定 `category`（平台分类名，不存在会自动创建）。三种类型互斥：

```yaml
delay_seconds: [1.5, 3.0]   # 请求间隔（秒，区间内随机）
max_retries: 3

tasks:
  - keyword: "机器学习入门"    # ① 关键词搜索
    category: "人工智能"
    pages: 2
    page_size: 20

  - bvids: ["BV1DgxCzREbM"]  # ② 指定 BV 号
    category: "数据科学"
    tags: ["示例"]            # 详情接口不返回标签，可手动补

  - up_mid: 946974           # ③ UP 主投稿（通常需要 --cookie）
    category: "软件工程"
    pages: 1
```

## 输出格式

写到 `backend/data/bilibili/<run_id>.json`，并同步一份 `latest.json`（该目录已被 `.gitignore` 忽略）：

```json
{
  "version": 1,
  "generated_at": "2026-09-10T18:00:00+08:00",
  "source": "bilibili",
  "items": [
    {
      "bvid": "BV1DgxCzREbM",
      "title": "机器学习入门",
      "description": "……",
      "cover_url": "https://i1.hdslb.com/bfs/archive/xxx.jpg",
      "author": "某 UP 主",
      "source_url": "https://www.bilibili.com/video/BV1DgxCzREbM",
      "category": "人工智能",
      "tags": ["机器学习", "AI"],
      "view_count": 170306,
      "metadata": {
        "bvid": "BV1DgxCzREbM",
        "duration": "16:08",
        "pubdate": 1759988744,
        "typename": "科学科普",
        "like": 4757,
        "favorite": 7749,
        "review": 936
      }
    }
  ]
}
```

## 实现说明

### 走的接口

全部使用 **B 站官方 web 接口的 JSON 返回**，不解析 HTML 页面（页面结构一变正则就失效）：

| 入口 | 接口 | 是否需要登录/签名 |
|---|---|---|
| 关键词搜索 | `/x/web-interface/wbi/search/type` | 需要 WBI 签名 + `buvid3`，**不需要登录** |
| BV 详情 | `/x/web-interface/view` | **完全免登录** |
| UP 主投稿 | `/x/space/wbi/arc/search` | 需要 WBI 签名，且**风控较严** |

### 关于 buvid3 与 WBI 签名

- `buvid3` 通过**直接 GET `https://www.bilibili.com/`** 由 B 站正常下发，与浏览器首次访问的行为一致。
  **本工具不伪造设备指纹。**
- WBI 签名密钥从 `/x/web-interface/nav` 获取，按 B 站前端固定的 64 位重排表算出 `mixin_key`，
  再对排序后的参数 + 时间戳做 MD5 得到 `w_rid`。密钥约每 2–4 周轮换，因此**每次运行重新拉取，不做缓存**。

### UP 主入口为什么需要 cookie

`/x/space/wbi/arc/search` 在免登录时通常返回 **`-352 风控校验失败`** 并附带人机校验凭证 `v_voucher`。
本工具**不实现任何验证码绕过**：命中风控时直接报错退出，提示改用 `--cookie` 提供你自己的登录凭证。

即使带上自己的 cookie 也可能仍被拦截，这取决于 B 站当次的风控策略。

### 限流

- 请求之间强制随机间隔（默认 1.5–3.0 秒）
- 网络错误 / 5xx 按指数退避重试，默认最多 3 次
- **风控错误（-352 / -412）不重试**，直接中止并返回退出码 2

## 合规声明

- 只采集**公开的视频元数据**（标题、简介、封面、UP 主名、公开播放量），不下载视频内容、不采集评论、不绕过登录墙
- 不伪造设备指纹、不使用代理池、不实现验证码绕过
- 采集结果仅用于学习研究，页面展示时回链原始视频；请遵守 B 站用户协议，不要高频、大规模采集
