# B 站教育资源采集与导入

平台的第二条内容来源：用 Python 采集 B 站公开视频元数据 → 输出 JSON → Go 命令导入 `resources` 表 →
前端资源列表自然展示。视频以 `type=video` 的普通资源身份**混入现有列表**，不新增专区、不改动推荐链路。

```
backend/crawler/  ──JSON──►  backend/data/bilibili/latest.json  ──►  cmd/import_bilibili  ──►  resources 表
   (Python 采集)                          (gitignore)                    (Go 导入)              type=video
```

## 全流程

```bash
# ① 采集
cd backend/crawler
cp config.example.yaml config.yaml      # 按需改任务与目标分类
python run.py --config config.yaml --dry-run   # 先只看抓到什么，不写文件
python run.py --config config.yaml             # 正式采集 → backend/data/bilibili/

# ② 导入
cd backend
CONFIG_PATH=configs/config.yaml go run ./cmd/import_bilibili -dry-run   # 只统计，不写库
CONFIG_PATH=configs/config.yaml go run ./cmd/import_bilibili            # 落库

# ③ 展示：前端资源列表 / 详情页直接可见，无需额外配置
```

## 采集器

详见 [backend/crawler/README.md](../backend/crawler/README.md)。要点：

- **走的接口**：关键词搜索 `/x/web-interface/wbi/search/type`（WBI 签名 + `buvid3`，**不需要登录**）、
  BV 详情 `/x/web-interface/view`（完全免登录）、UP 主投稿 `/x/space/wbi/arc/search`（风控较严）。
- **全部解析官方 JSON**，不解析 HTML 页面——页面结构一变正则就失效。
- **`buvid3` 由直接 GET `bilibili.com` 首页获得**（与浏览器首次访问行为一致），**不伪造设备指纹**。
- **UP 主入口**当前被 B 站风控（`-352`）拦截，工具**不实现任何验证码绕过**，命中即报错退出，
  提示改用 `--cookie` 提供自己的登录凭证。
- 每个任务必须指定 `category`（平台分类名，不存在时导入阶段自动创建）。

## 交接文件格式

写到 `backend/data/bilibili/<run_id>.json`，并同步一份 `latest.json`（该目录已被 `.gitignore` 忽略）：

```json
{
  "version": 1,
  "generated_at": "2026-09-10T18:23:52+08:00",
  "source": "bilibili",
  "items": [
    {
      "bvid": "BV1DgxCzREbM",
      "title": "机器学习入门",
      "description": "……",
      "cover_url": "https://i1.hdslb.com/bfs/archive/xxx.jpg",
      "author": "AI精品课程",
      "source_url": "https://www.bilibili.com/video/BV1DgxCzREbM",
      "category": "人工智能",
      "tags": ["机器学习", "AI"],
      "view_count": 170306,
      "metadata": {
        "bvid": "BV1DgxCzREbM", "duration": "16:08", "pubdate": 1759988744,
        "typename": "科学科普", "like": 4757, "favorite": 7749, "review": 936
      }
    }
  ]
}
```

## 导入命令

```bash
cd backend
go run ./cmd/import_bilibili [-file data/bilibili/latest.json] [-dry-run]
```

| 参数 | 默认值 | 说明 |
|---|---|---|
| `-file` | `data/bilibili/latest.json` | 爬虫输出文件路径（相对 `backend/`） |
| `-dry-run` | `false` | 走完全相同的解析/判重/分类逻辑，但不写库 |

输出统计：

```
[import_bilibili] 新增资源 27 条，刷新资源 0 条，跳过 0 条，新建分类 1 个
```

| 计数 | 含义 |
|---|---|
| `created_resources` | 新插入的 `type=video` 资源 |
| `updated_resources` | `source_url` 命中已有记录，只刷新了动态字段 |
| `skipped_resources` | 缺必填字段（`bvid` / `title` / `category`），未落库 |
| `created_categories` | 按名称找不到、本次新建的分类 |

## 字段映射

| `resources` 字段 | 来源 | 备注 |
|---|---|---|
| `title` | `title` | 超 256 字符按**字符**截断 |
| `description` | `description` | `TEXT`，不截断 |
| `cover_url` | `cover_url` | 超 512 字符截断 |
| `type` | 固定 `"video"` | |
| `category_id` | `category` | 按名称 find-or-create |
| `tags` | `tags` | JSON 数组字符串；`null` 兜底成 `[]`（否则 `JSON_CONTAINS` 查不到） |
| `metadata` | `metadata` | 原样序列化 |
| `author` | `author` | 超 128 字符截断 |
| `source_url` | `source_url` | **判重键**；为空时由 `bvid` 推导 |
| `avg_rating` | 固定 `0` | B 站无评分 |
| `view_count` | `view_count` | 用于 `sort=popular` |

## 判重与幂等

判重键是 `source_url`（`https://www.bilibili.com/video/<bvid>`）：

- **未命中** → 插入新行
- **命中** → **不新增行**，只刷新 `view_count` 与 `metadata`（播放量会变，`popular` 排序才不失真）。
  标题、简介、分类等**保持平台侧的值**，导入不会覆盖人工编辑过的内容

因此命令可反复执行，第二次跑应当全部走更新分支：

```
第 1 次：新增资源 27 条，刷新 0 条，跳过 0 条，新建分类 1 个
第 2 次：新增资源  0 条，刷新 27 条，跳过 0 条，新建分类 0 个
```

> `resources` 表当前**没有 `source_url` 唯一索引**（见「已知限制」）。导入命令是单进程串行执行的，
> 不存在并发写入，实际不会产生重复行。

## 前端展示

- **封面**：卡片与详情页的 `<img>` 均带 `referrerpolicy="no-referrer"`。
  B 站 CDN 有防盗链——实测无 Referer 返回 200，带 `Referer: http://localhost:5173/` 返回 403，
  去掉 Referer 即可正常加载。外链仍可能失效，故 `@error` 时降级为占位块。
- **元信息**：详情页「元信息」直接渲染 `metadata` 键值表；`pubdate`（Unix 秒）按 key 特判格式化为 `YYYY-MM-DD`。

## 合规边界

- 只采集**公开的视频元数据**（标题/简介/封面/UP 主名/公开播放量）：不下载视频内容、不采集评论、不绕过登录墙
- 不伪造设备指纹、不使用代理池、**不实现验证码 / 风控绕过**
- 请求间强制随机间隔（默认 1.5–3.0 秒），风控错误（`-352` / `-412`）不重试，直接中止
- 采集产物落在已被 `.gitignore` 忽略的 `backend/data/`，不入库
- 页面展示时回链原始视频（`source_url`）；仅供学习研究，请遵守 B 站用户协议，不要高频、大规模采集

## 已知限制

1. **`resources` 表无 `source_url` 唯一索引**。命令自身单进程串行，不会重复；但若将来引入并发导入，
   需要用 migration 给 `source_url` 加唯一索引。
2. **导入不覆盖已有字段**。命中判重时只刷新 `view_count` / `metadata`，所以 B 站侧改了标题不会同步过来。
   需要同步时先删掉该行再导入。
3. **UP 主入口不可用**（B 站风控 `-352`）。代码路径保留，需自带登录 cookie 且仍可能被拦截。
4. **分类自动创建**。配置里写错的分类名不会报错，会静默新建一个分类。
