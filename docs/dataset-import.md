# 第三方数据集导入（慕课等）

平台的第三条内容来源：把**已下载到本地**的第三方数据集（慕课课程等）解析、字段映射后导入 `resources` 表，
以 `type=course` 的普通资源身份混入现有列表，与 B 站视频（见 [bilibili-import.md](./bilibili-import.md)）同构。

```
数据集文件 (json / jsonl / csv) ──► LoadDataset（配置驱动的字段映射）──► cmd/import_dataset ──► resources 表
       用户自行下载                    外部字段 → 平台契约                 复用判重/分类/截断        type=course
```

## 为什么是离线导入，而不是在线爬慕课网

最初的做法是给慕课网（imooc.com）写在线爬虫，与 B 站并列为第二个在线来源。**该方案已验证为不可行，故放弃**：

imooc 全站（`www` / `m.` / `/api/` / `/learn/{id}` 详情页）由**腾讯 EdgeOne 机器人管理**接管，
对非浏览器客户端统一下发 29180 字节的混淆 JS 挑战页。同机、同 IP、同 UA 实测：

| 客户端 | 结果 |
|---|---|
| `curl` | 200，89844B，真实 JSON |
| Python `requests` / `httpx`（HTTP/1.1）/ `urllib` | 200，29180B，挑战页 |

换浏览器头、`Referer`、`Accept-Language`、`X-Requested-With`、`Accept-Encoding: identity`、`Connection: close`
及 60 秒冷却均无效。关键现象：`curl` 换成 `Python-urllib` 的 UA 会被拦、换回浏览器 UA 又能过，
而 Python 换任何 UA 都过不去 —— 说明拦截发生在 **TLS 指纹（JA3）** 层，不在 HTTP 头层。
要打通只能解 JS 挑战或伪造浏览器指纹（`curl_cffi` 一类），二者都违反本项目的合规红线
（见「合规边界」），**故不做**。

于是改为**纯离线数据集导入**：数据来源是用户合法获取的本地文件，Go 侧直接解析入库。
附带好处是不引入任何新的 Python 代码，也就完全避开了改动无测试覆盖的采集脚本的风险。

## 快速上手

```bash
cd backend

# ① 先核对字段映射对不对（不连数据库，最安全）
go run ./cmd/import_dataset --dataset mooc --preview

# ② 再看到底会新增/刷新多少条（连库走完整判重与分类，但不写库）
go run ./cmd/import_dataset --dataset mooc --dry-run

# ③ 确认无误后真导入
go run ./cmd/import_dataset --dataset mooc
```

`data/mooc/` 下有 `sample.json` 与 `sample.csv` 两份手写样例（各 5 条，含故意缺字段的行），
可用来观察映射与跳过原因的输出，**不是真实数据**。

## 配置

写在 `configs/config.yaml` 的 `datasets.<名称>` 段。字段对应关系全部外置到配置——
各家数据集的字段名和格式都不一样且会更换，换数据集只改 YAML、不动代码。

```yaml
datasets:
  mooc:
    file: "data/mooc/courses.json"   # 数据集文件路径，相对 backend/ 运行目录
    format: "auto"                   # auto | json | jsonl | csv
    items_path: "data.list"          # 记录数组在 JSON 里的位置，留空表示根节点就是数组
    resource_type: "course"          # 落库的 resources.type
    default_category: "慕课课程"      # 取不到分类时的兜底分类名
    source_url_template: ""          # 取不到来源链接时用 {字段名} 模板拼
    tag_separator: ","               # tags 是字符串时的分隔符
    fields:                          # 内部字段 → 外部字段候选（按序回退）
      title: ["title", "name"]
      description: ["short_description", "description"]
      cover_url: ["pic"]
      author: ["author_nickname"]
      source_url: ["target_url"]
      category: ["category"]
      tags: ["tags"]
      view_count: ["numbers"]
    metadata: ["obj_id", "price"]    # 原样收进 metadata 的外部字段
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `file` | 是 | 数据集文件路径，相对 `backend/` |
| `format` | 否 | `auto`（默认）/ `json` / `jsonl` / `csv` |
| `items_path` | 否 | 点号路径，如 `data.list`；留空表示根节点即数组 |
| `resource_type` | 是 | `course` / `article` / `video`，写库前会再校验一次 |
| `default_category` | 否 | 记录取不到分类时使用 |
| `source_url_template` | 否 | `{外部字段名}` 占位，如 `https://x/learn/{obj_id}` |
| `tag_separator` | 否 | 默认 `,` |
| `fields` | 是 | 内部字段 → 外部字段候选列表 |
| `metadata` | 否 | 原样收进 `metadata` 的外部字段名 |

配置在 `config.Load()` 阶段就会校验（未知的 `fields` key、非法的 `resource_type` / `format` 都会直接报错），
**拼错的 key 会让服务启动失败**，而不是等导入跑完才发现整批数据都因缺字段被跳过。

> `config.yaml` 是提交进仓库的模板；本机实际运行用的常是 `config.local.yaml`（被 `.gitignore` 忽略）。
> 后者不补上 `datasets` 段的话，`--dataset mooc` 会报「配置里没有 datasets.mooc」。
> 默认读 `configs/config.yaml`，可用 `CONFIG_PATH` 覆盖。

## 字段映射

`fields` 的左边只能填下面 8 个内部字段名，写错直接报错：

| 内部字段 | 落到 `resources` 的列 | 归一化 |
|---|---|---|
| `title` | `title` | 按字符截断至 256 |
| `description` | `description` | 为空时**回退到 `title`**（卡片与详情页直接展示它，留空很难看） |
| `cover_url` | `cover_url` | `//` 前缀补 `https:`，`http://` 升 `https://`；截断至 512 |
| `author` | `author` | 截断至 128 |
| `source_url` | `source_url` | **判重键**；截断至 512 |
| `category` | `category_id` | 按名称 find-or-create |
| `tags` | `tags` | JSON 数组字符串，去重、上限 10 |
| `view_count` | `view_count` | 见下 |

右边是候选列表，**按序取第一个非空值**——空字符串、空数组、空对象都算「没值」，会继续往后回退。
候选可以是点号路径，如 `author.nickname`；数据集里若有 `"a.b"` 这种扁平键，会优先按字面量键匹配。

### 归一化细则

- **`view_count`**：兼容 `25612`（数字）、`"25612"`、`"25,612"`、`"2.5万"`、`"1.2w"`、`"3k"`。
  识别不了的一律记 `0`——播放量是展示字段，不值得为它让整条记录失败。
- **`tags`**：兼容 JSON 数组、分隔符字符串（`tag_separator`）、`null`。去重后截断至 10 条。
- **`metadata`**：`metadata` 列出的外部字段原样收进 `metadata` map，空值不收。
  留作将来换推荐模型时取用（对齐决策 #24「统一抽象 + JSON metadata」）。

### 容器格式

| `format` | 行为 |
|---|---|
| `auto` | 按扩展名判定（`.jsonl`/`.ndjson` → jsonl，`.csv` → csv，`.json` → json）；扩展名不认识时看首个非空字符，`[` 或 `{` 视为 json，否则按 jsonl 逐行试 |
| `json` | 按 `items_path` 取数组；**单个对象也接受**（省去为一条记录包一层数组）；取到的节点不是数组则报错并提示检查 `items_path` |
| `jsonl` | 逐行解析，空行跳过；某行出错会报出**行号** |
| `csv` | 首行为表头，每行转成字段映射；容忍列数不一致的行（数据集常有末尾缺列）；所有值按字符串处理，交给下游归一化 |

文件开头的 UTF-8 BOM 会被剥掉——Windows 上导出的 CSV 常带 BOM，不剥会让首个列名/字段名多出一个
`U+FEFF` 前缀而导致首列静默映射不上。

## 行丢弃条件

下列三种情况整行跳过并被归类计数，**原因文案会指明该改哪个配置 key**：

| 原因 | 触发条件 |
|---|---|
| `缺少标题（检查 fields.title）` | `title` 归一化后为空 |
| `缺少分类（检查 fields.category 或 default_category）` | 映射值为空且 `default_category` 也为空 |
| `无法确定来源链接（检查 fields.source_url 或 source_url_template）` | 映射值为空且模板渲染不出（模板为空，或任一 `{占位符}` 取不到值） |

`source_url` 是判重的**唯一依据**，所以它必须能确定下来，否则每次导入都会重复插入同一行——
这是必须拦住的失败模式，故宁可不导入。拼出半截 URL 同样会让判重失效，因此模板只要有任一占位符
取不到值，整体即返回空、该行跳过。

## 命令行

```bash
cd backend
go run ./cmd/import_dataset --dataset mooc [--file X] [--format auto] [--limit N] [--preview] [--dry-run]
```

| 参数 | 默认值 | 说明 |
|---|---|---|
| `--dataset` | 无（**必填**） | `datasets.<名称>` 的配置名；写错时会列出已配置的名字 |
| `--file` | 配置里的 `file` | 覆盖文件路径，便于拿同一套映射试不同文件 |
| `--format` | 配置里的 `format` | 覆盖容器格式 |
| `--limit` | 不限 | 最多处理多少条；`--preview` 未显式指定时取 5 条 |
| `--preview` | `false` | 只解析并打印映射结果，**不连数据库**；优先于 `--dry-run` |
| `--dry-run` | `false` | 连库走完整判重与分类逻辑，但不写库 |

`--file` / `--format` **只覆盖这两项**，字段映射、兜底分类与落库类型仍以 YAML 为准——
否则同一次导入究竟用了哪套映射就说不清了。

输出分「解析」与「落库」两阶段报数，阶段内各有一个「跳过」计数，别把它们看成一个：

```
[import_dataset] 数据集：data/mooc/sample.json（auto，course）
[import_dataset] 解析：共 5 条，映射成功 3 条，跳过 2 条
[import_dataset] 落库：新增资源 3 条，刷新资源 0 条，跳过 0 条，新建分类 1 个
[import_dataset] 解析阶段跳过原因：
[import_dataset]   1 条  无法确定来源链接（检查 fields.source_url 或 source_url_template）
[import_dataset]   1 条  缺少标题（检查 fields.title）
```

解析阶段跳过原因按条数倒序，调映射时优先解决排第一的那条。

## 判重与幂等

判重键是 `source_url`，语义与 B 站导入完全一致（同一段代码，见
[crawl_import.go](../backend/internal/service/crawl_import.go) 的 `ImportItems`）：

- **未命中** → 插入新行
- **命中** → **不新增行**，只刷新 `view_count` 与 `metadata`。标题、简介、分类等**保持平台侧的值**，
  导入不会覆盖人工编辑过的内容

因此命令可反复执行，第二次跑应当全部走更新分支：

```
第 1 次：新增资源 3 条，刷新资源 0 条，跳过 0 条，新建分类 1 个
第 2 次：新增资源 0 条，刷新资源 3 条，跳过 0 条，新建分类 0 个
```

数据集内部若出现重复的 `source_url`，后续条目按「已存在」处理，只刷新不新建。

## 换一份数据集怎么调

1. 把文件放到 `backend/data/` 下（该目录已被 `.gitignore` 忽略，不入库），
   或直接 `--file` 指过去。
2. `--preview` 看输出：`映射成功 0 条` 或大比例跳过，基本都是 `fields` 没对上。
   跳过原因会告诉你是哪个字段缺，对照数据集的真实字段名改 YAML。
3. 数据集的记录嵌在 JSON 里（如 `{"data": {"list": [...]}}`）就填 `items_path: data.list`；
   根节点直接是数组则留空。
4. 数据集没有 URL 字段但有课程 id，就留空 `fields.source_url` 并配
   `source_url_template: "https://站点/课程/{字段名}"`。
5. 反复 `--preview` 直到映射全部命中，再 `--dry-run` 看条数，最后真导入。

## 前端展示

导入的课程走与站内资源完全相同的列表与详情页，无需额外配置：

- **封面**：卡片与详情页的 `<img>` 均带 `referrerpolicy="no-referrer"`，
  可绕开慕课网 CDN 的防盗链（实测无 Referer 返回 200，带 `Referer: http://localhost:5173/` 返回 403，
  与 B 站同款行为）。
- **筛选**：资源库页选「类型 = 课程」即可筛出本链路导入的内容。

## 合规边界

- 只导入**用户合法获取的公开数据集**。数据集本身必须是公开可得、允许使用的；请遵守其许可条款，
  不要导入来源不明或含个人信息的转储数据。
- **不实现任何风控绕过**：不解决 JS 挑战、不伪造浏览器/设备指纹、不使用代理池（这也是本链路选择
  离线导入而非在线爬取的原因，见上文）。
- 导入产物落在已被 `.gitignore` 忽略的 `backend/data/`，数据集文件不入库。
- 页面展示时回链原始来源（`source_url`）；仅供学习研究。

## 已知限制

1. **导入不覆盖已有字段**。命中判重时只刷新 `view_count` / `metadata`，所以数据集的标题/简介更新了
   不会同步过来。需要同步时先删掉该行再导入。
2. **分类自动创建**。`default_category` 或数据里的分类名写错不会报错，会静默新建一个分类
   （与 B 站导入同款行为）。
3. **`resources` 表无 `source_url` 唯一索引**（仅有普通索引，见
   [resource.go](../backend/internal/model/resource.go)）。本命令单进程串行执行，不会产生重复行；
   若将来引入并发导入，需要用 migration 加唯一索引。
4. **`resource_type` 只影响落库的 `type` 列**，不影响推荐链路——课程与视频在推荐里同等对待。
5. **数据集若没有课程元数据**（如公开的 XuetangX/KDD 交互日志只有 user–course 行为，没有标题封面），
   `title` 会全空导致整批跳过。`--preview` 会立刻暴露这种情况，属数据集本身不适用，需换数据集。
