# B 站教育资源采集与导入

平台的第二条内容来源：用 Python 采集 B 站公开视频元数据 → 输出 JSON → Go 命令导入 `resources` 表 →
前端资源列表自然展示。视频以 `type=video` 的普通资源身份**混入现有列表**，不新增专区、不改动推荐链路。

> 慕课等第三方数据集走**完全不同的链路**（纯离线导入，配置驱动字段映射），见
> [dataset-import.md](./dataset-import.md)。两条链路复用同一段落库内核
> （[crawl_import.go](../backend/internal/service/crawl_import.go) 的 `ImportItems`），
> 故下文的判重与幂等语义对二者同样成立。

链路分两种模式，**离线批量 + 在线实时并存**：

```
离线批量：backend/crawler/ ──JSON──► backend/data/bilibili/latest.json ──► cmd/import_bilibili ──► resources 表
            (Python 采集)                    (gitignore)                     (Go 导入)              type=video

在线实时：GET /resources?keyword=X（本地耗尽后 online_page=M）─► exec python online.py search --page M ─► ImportItems 落库 ─► 返回新导入资源
          GET /resources/:id/comments（无缓存）──► exec python online.py comments ─► 落库 resource_comments ─► 返回评论
```

在线实时部分见下文「在线搜索 + 评论（实时）」，离线批量链路见「全流程」。

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
[import_bilibili] 新增资源 27 条，刷新资源 0 条，跳过 0 条，非教育分区拦截 0 条，新建分类 1 个
```

| 计数 | 含义 |
|---|---|
| `created_resources` | 新插入的 `type=video` 资源 |
| `updated_resources` | `source_url` 命中已有记录，只刷新了动态字段 |
| `skipped_resources` | 缺必填字段（`bvid` / `title` / `category`），未落库 |
| `skipped_typenames` | **非教育分区被白名单拦下**，未落库（见下一节） |
| `created_categories` | 按名称找不到、本次新建的分类 |

## 教育分区白名单（内容质量闸门）

B 站搜索是**按关键词返回**的，不区分内容性质。搜「杨真真」会把《夏家三千金》影视剪辑、
娱乐杂谈、网络游戏等一并返回；这些条目若直接落库，就会和正常课程一起出现在首页与搜索页。
因此落库前按 **B 站分区名（`typename`）白名单**拦截。

- **判定依据**：爬虫在 `metadata.typename` 里写了 B 站分区名（`collect.py` 取搜索结果的 `typename`）。
- **生效范围**：在线搜索抓取与离线导入**共用同一套**（都走 `CrawlImportService.ImportItems`）；
  第三方数据集条目没有 `typename` 字段，**不受影响**。
- **配置**：`backend/configs/config.yaml` 的 `bilibili.allowed_typenames`。
  不写这一项 → 用代码内默认白名单（`config.DefaultEducationalTypenames`）；写 `[]` 表示不过滤（不推荐）。

白名单之所以不能"只留校园学习"，是因为 B 站对正经课程的误分类非常普遍。实测（2026-09-19）：

| 分区 | 为什么保留 |
|---|---|
| `校园学习` / `计算机技术` / `科学科普` / `野生技能协会` | 明确的教学与知识分区 |
| `人文历史` / `社科·法律·心理` | 通识学科内容 |
| `日常` / `数码` / `运动文化` / `竞技体育` | **误分类重灾区**：「数学分析」「泛函分析」「高等代数」「数学建模国赛」都被归进这些分区 |
| `软件应用` / `职业职场` / `科工机械` / `财经商业` | 技能与职业教育向：「机器学习」「数据分析」教程落在这里 |

刻意**不**收录：`其他`（内容不可控）、`预告·资讯`、`原创音乐`、
`影视剪辑`/`影视杂谈`/`娱乐粉丝创作`/`娱乐杂谈`/`明星综合`/`网络游戏`/`音乐综合` 等。
未收录的分区一律不导入 —— 新分区默认拦住，宁可漏也不要放回非教育内容。

> ⚠️ **白名单是关键词搜索的兜底，不是万能的**：分区本身也可能误标（实测有数学视频被归进「搞笑」），
> 因此仍可能有少量非教育条目落进白名单分区。发现后按标题清理即可：
>
> ```sql
> -- 先查（把关键词换成要清理的主题）
> SELECT id, title, JSON_UNQUOTE(JSON_EXTRACT(metadata,'$.typename')) tn
> FROM resources WHERE title REGEXP '关键词1|关键词2';
> -- 确认后删（有行为/评分引用时需先删引用行，外键会拦住）
> DELETE FROM resources WHERE title REGEXP '关键词1|关键词2';
> ```

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

## 资源类型：长合集/系统课程自动判为 course

B 站采集**统一按 `type=video` 落库**（`BilibiliImportOptions.ResourceType`），因为 B 站搜索结果
本身不区分「单集视频」与「系列课程」——一条搜索结果就是一个视频，即便它是 151 集的合集。
于是 `resources` 里会出现「十几分钟的单集」和「150 小时的全套课程」类型相同、无法按类型筛选的情况。

**导入时自动判定**（`config.ContentRulesConfig.IsLongCourse`，在线抓取与离线导入共用）：每条 B 站内容
在落库前判一次，符合规则就落 `course`，否则落 `video`。规则**配置化**，在 `configs/config.yaml` 的
`content_rules` 段调整，不用改代码：

| 档位 | 触发条件 | 默认门槛 |
|---|---|---|
| 强标记 | 标题含 `course_keywords`（课程/精讲）或 `collection_markers`（合集/全集/全套/系列）或形如「全N集/共N讲」 | 时长 ≥ `course_min_minutes`（120 分钟） |
| 弱关键词 | 标题含 `longform_keywords`（教程/讲解） | 时长 ≥ `longform_min_minutes`（**600 分钟 / 10 小时**） |

弱词档门槛更高的原因：`教程`/`讲解` 在 B 站**短片**标题里极常见
（「17分钟让你看懂所有机器学习算法」写作【入门教程】），只靠关键词会把短片误判成课程。
两档都要求时长达标，这是规则的核心。

> ⚠️ **`duration` 是「分钟:秒」，不是「时:分」**。采集器 `collect.py` 的 `_format_duration`
> 用 `f"{seconds//60}:{seconds%60:02d}"` 格式化，所以 `9003:20` 表示 **9003 分钟**。
> 曾按「时:分」解析，把 `119:59` 算成 7199 分钟，导致时长门槛形同失效。

实测校准（2026-09-19）：单看时长 → 174 条都成课程，连没写「课程」字样的长课也算；
单看关键词 → 17 分钟短片也被判课程（14 条）；本口径 → 命中 63 条，逐条核对均为长课程/系列
（121 分钟 ~ 150 小时），结果 `course 63 / video 206`。

**历史数据对齐**用一次性脚本（口径调整后也可重跑）：

```bash
mysql -h 127.0.0.1 -P 3308 -u root -p edurec < scripts/reclassify-long-courses.sql
```

判重命中时导入只刷新 `view_count` 与 `metadata`、**不覆盖 `type`**
（`internal/service/crawl_import.go` 的 `ImportItems`），所以改判结果不会被后续导入改回，
也可以随时在管理后台 `/admin/resources` 逐条调整。

> ⚠️ **连带影响已处理**：详情页原先用 `type === 'video'` 判断是否显示 B 站评论区，
> 改判后这些课程会丢失评论区。判据已改为「来源是 B 站」（`source_url` 含 `bilibili.com`），
> 因此 **course 类型的 B 站内容同样有评论区**。

## 前端展示

- **封面**：卡片与详情页的 `<img>` 均带 `referrerpolicy="no-referrer"`。
  B 站 CDN 有防盗链——实测无 Referer 返回 200，带 `Referer: http://localhost:5173/` 返回 403，
  去掉 Referer 即可正常加载。外链仍可能失效，故 `@error` 时降级为占位块。
- **元信息**：详情页「元信息」直接渲染 `metadata` 键值表；`pubdate`（Unix 秒）按 key 特判格式化为 `YYYY-MM-DD`。

## 在线搜索 + 评论（实时）

除离线批量链路外，后端还支持**在线实时**调用爬虫，把「纯离线」扩展为「离线批量 + 在线实时」并存：

1. **搜索时逐页爬取（无限滚动）**：`GET /resources?keyword=X&page=N` 先翻本地 `resources` 表；本地匹配结果翻完后（`resources.length >= total`），且是**纯关键词搜索**（只带 keyword、未加分类/类型/标签筛选），前端改带 `online_page=M` 请求，后端 `exec` 调用 `crawler/online.py search --page M` 实时抓取 B 站第 M 页，复用 `ImportItems` 落库后**返回本次新导入的资源**，并用 `has_more` 标记 B 站是否还有下一页。B 站翻页受 `search_max_pages` 上限约束，防无界爬取。
2. **打开详情页自动爬评论**：`GET /resources/:id/comments` 先查 `resource_comments` 缓存；无缓存且 `source_url` 含 `bilibili.com` 时，`exec` 调用 `crawler/online.py comments` 抓取该视频评论并落库，后续请求直接读缓存。

```text
搜索：GET /resources?keyword=X&page=N →（本地耗尽）→ GET /resources?keyword=X&online_page=M → exec python online.py search --page M → ImportItems 落库 → 返回新导入资源 + has_more
评论：GET /resources/:id/comments（无缓存）→ exec python online.py comments → 落库 resource_comments → 返回
```

约定：

- Python 子进程 **stdout 只输出单行 UTF-8 JSON**，日志与错误走 stderr；Go 侧 `exec.CommandContext` 带 30s 超时，按 UTF-8 解析，规避 Windows GBK 编码问题。
- 评论落独立的 `resource_comments` 表，**不污染站内 `Rating` 评分**；每条存作者昵称、内容、点赞、楼层、发布时间。
- 评论默认只抓公开热门 top N（`comment_limit`，默认 20）；命中风控（`-352` / `-412`）不重试，前端展示「评论暂不可用」空态，不阻塞页面主体。
- 相关配置见 `configs/config.yaml` 的 `bilibili:` 段（`python_path` / `crawler_dir` / `category` / `search_limit` / `search_max_pages` / `comment_limit`）。

## 合规边界

- 只采集**公开的视频元数据**（标题/简介/封面/UP 主名/公开播放量）与**公开热门评论 top N**（作者昵称、正文、楼层、点赞、发布时间）：不下载视频内容、不采集评论中的个人信息、不绕过登录墙
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
