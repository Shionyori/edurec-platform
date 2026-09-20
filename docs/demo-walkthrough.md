# 老师现场查看 · 演示动线脚本

> 适用仓库：`E:\work\learn\edurec-platform`（后端 Go+Gin+GORM / 前端 Vue3+Vite+Element Plus+Pinia）
> 配套仓库：edurec-engine（Python + PyTorch，独立仓库，**本演示可选**）
> 本脚本所有命令、URL、期望输出均在 2026-09-19 本机实测通过；决策编号均对应 [design.md](./design.md) 的「决策记录」表。

## 0. 一句话总纲（开口第一句）

「平台做**内容与行为的生产与消费**，推荐算法在**另一个仓库离线跑**，两边只通过**数据快照文件 + 推荐结果文件**交接，
所以 platform 侧没有任何模型推理代码——首页个性化与否，只看 `recommendations` 缓存表有没有被 engine 的结果填过。」
—— 对应决策 **#1 / #2 / #3 / #12**。

### 演示前必须具备的四个前提

| 组件 | 地址 | 账号/密码 |
|---|---|---|
| MySQL | 127.0.0.1:**3308**，库 `edurec` | root / 123456 |
| Redis | 127.0.0.1:**6380** | 密码 `284835` |
| 后端 | http://127.0.0.1:8080 | — |
| 前端 dev | http://127.0.0.1:**5173**（`/api` 代理到 8080） | — |

演示账号：

| 角色 | 账号 | 密码 | 说明 |
|---|---|---|---|
| 演示用户（无缓存行，走兜底） | `demo_fresh` | `demo123456` | users.id=2001，**没有**推荐缓存行 → 首页走热门兜底并回写。见 1.8 节 |
| 演示管理员 | `demo_admin` | `demo123456` | users.id=2000，`admins` 表有行；管理后台 `/admin`；**同时也有缓存行**，可演个性化读取 |

> ⚠️ **2026-09-19 变更：模拟数据已彻底清除**。按你的要求"页面里不要有模拟资源"，
> 已删除 `模拟资源*`（347 条）、模拟用户 `demo0..demo1999`、`模拟类别*` 分类，及其行为/评分/推荐缓存。
> **`demo1` 等 `demo<数字>` 账号已不存在**（它们是模拟数据的一部分），别再写进演示里。
> 库里现在只有 **4 个真实用户**：`BTXL`(id 2)、`biliverify`(id 3)、`demo_admin`(id 2000)、`demo_fresh`(id 2001)。
> `BTXL` / `biliverify` 是真实注册用户，**密码不是 `demo123456`**，不要拿来演示登录。
> 因此**新注册** `demo_fresh`（见 1.8 节命令）就是最干净的演示账号。

---

## 1. 演示前 5 分钟自检

> 逐条执行，**每条的「期望」都对得上再往下走**。全部命令在 PowerShell 中执行。

### 1.1 启动 Redis（6380 + requirepass 284835）

```powershell
Test-Path E:\work\cppsoft\Redis\redis-server.exe
& E:\work\cppsoft\Redis\redis-server.exe E:\work\cppsoft\Redis\redis.conf
```

- 期望：第二条命令**前台常驻**，输出里有 `Ready to accept connections tcp` 与 `The server is now ready to accept connections on port 6380`。
- 若 6380 已被占用（第二条报 `Could not create server TCP listening socket` / `bind: Unknown error` 一类错误），
  说明 Redis 已由服务或别的窗口拉起，**跳过本步**；先用下面的 `PING` 确认 6380 到底通不通。
- 该目录里有 `RedisService.exe` / `install_redis_service.bat`（服务方式安装）与 `redis_6379.pid`。
  ⚠️ **不要**直接用默认 `redis.conf` 之外的配置或 6379 端口的实例 —— 后端配置写死的是 **6380 + requirepass 284835**。

验证（新开一个窗口）：

```powershell
& E:\work\cppsoft\Redis\redis-cli.exe -p 6380 -a 284835 PING
```

期望输出：`PONG`

> Redis 存什么：JWT refresh token（决策 **#5**）+ 推荐结果缓存通道（决策 **#3/#12**）。
> 若 Redis 未起，表现是登录成功但刷新后立刻掉登录 —— 见第 4 节兜底 B。

### 1.2 启动后端（8080，用 local 配置）

```powershell
cd E:\work\learn\edurec-platform\backend
$env:CONFIG_PATH='configs/config.local.yaml'
go run ./cmd/server
```

期望：日志出现 `MySQL 连接成功 host=localhost port=3308`，Gin 打印路由表并监听 `:8080`。
（`go run` 若打印 `error acquiring upload token: ... Access is denied` 属 Go 遥测写盘告警，**与项目无关，可忽略**。）

验证：

```powershell
Invoke-RestMethod http://127.0.0.1:8080/api/v1/health | ConvertTo-Json -Compress
```

期望：`{"code":0,"message":"ok","data":{"status":"healthy"}}`

### 1.3 启动前端 dev（5173）

```powershell
cd E:\work\learn\edurec-platform\frontend
Get-Content .env.development.local
pnpm run dev
```

- 期望第一行输出是 `VITE_MOCK=0`（关掉 MSW，请求走 vite proxy → 真实后端 8080，决策 **#12**）。
  仓库里默认的 `.env.development` 是 `VITE_MOCK=1`，**本机的 `.local` 覆盖掉了它**，这是关键。
- 期望第二条打印 `Local: http://localhost:5173/`。

> ⚠️ **本机 `pnpm` 不在 PATH**（`Get-Command pnpm` 无结果，只有 `C:\Program Files\nodejs\corepack.cmd`）。
> 现场若 `pnpm run dev` 报「无法将"pnpm"项识别为 cmdlet」，**直接用 `npm run dev`**（`node_modules` 已装好，无需 install）。
>
> 想确认 mock 到底关没关（`"0"` 才对；`"1"` 会走 MSW 假数据，页面"全都正常"但数据是假的）：
>
> ```powershell
> node -e "console.log(require('vite').loadEnv('development',process.cwd(),'VITE_').VITE_MOCK)"
> ```
>
> 2026-09-19 本机实测输出 `"0"`（vite 8.2.2）。**这条务必在开场前确认一次。**

### 1.4 代理连通性验证（前端 → 代理 → 后端）

```powershell
Invoke-RestMethod http://127.0.0.1:5173/api/v1/health | ConvertTo-Json -Compress
```

期望：与 1.2 完全相同的 JSON（用的是 `vite.config.ts` 里 `'/api' → http://localhost:8080` 那条 proxy）。

### 1.5 数据现状核对（决定下面每一幕看到什么）

```powershell
cd E:\work\learn\edurec-platform\backend
$env:MYSQL_PWD='123456'
mysql -h 127.0.0.1 -P 3308 -u root -e @"
SELECT type, COUNT(*) AS n FROM edurec.resources GROUP BY type;
SELECT COUNT(*) AS bili_videos FROM edurec.resources WHERE source_url LIKE '%bilibili.com%';
SELECT COUNT(*) AS users FROM edurec.users;
SELECT COUNT(*) AS rec_rows FROM edurec.recommendations;
SELECT COUNT(*) AS behaviors FROM edurec.user_behaviors;
SELECT COUNT(*) AS ratings FROM edurec.ratings;
SELECT COUNT(*) AS bili_comments FROM edurec.resource_comments;
SELECT u.id, u.username, COUNT(r.id) AS rec FROM edurec.users u
  LEFT JOIN edurec.recommendations r ON r.user_id = u.id
  WHERE u.username IN ('demo_admin','demo_fresh','BTXL','biliverify') GROUP BY u.id, u.username;
"@
```

**本机当前实测值（2026-09-19 清除模拟数据后实查）**：

| 指标 | 当前值 | 备注 |
|---|---|---|
| resources 总数 | **299** | **全部是真实 B 站采集资源**（模拟资源已清除）；在线抓取会持续增加 |
| 其中 `source_url like %bilibili.com%` | **299** | 全部有来源链接 |
| 资源类型分布 | **course 69 · video 230** | 长合集/系统课程在**导入时按规则自动判为 course**（`content_rules` 可配）；详见 `docs/bilibili-import.md` |
| users | **4** | `BTXL`(2) · `biliverify`(3) · `demo_admin`(2000) · `demo_fresh`(2001) |
| recommendations（缓存行） | **3** | 属于 BTXL / biliverify / demo_admin；`demo_fresh` 无行 → 走兜底（1.8 节） |
| user_behaviors | **50** | BTXL / biliverify 的真实行为（view/click/favorite 三类齐全） |
| ratings | **5** | 真实评分 |
| resource_comments（B 站评论缓存） | **46** | 打开 B 站视频详情页会自动追加 |
| categories | 2 | 人工智能、B站视频（`模拟类别*` 已清除） |

> ⚠️ **这些数字一定会「长大」，别当固定值报**：1.7 节或第 3 幕里执行 `python online.py ...`、在搜索页滚到底、
> 打开 B 站视频详情页，都会**真实写库**（新增 B 站资源行 / 评论行）。这不是脏数据，是链路正常工作的证据。
> **上台前重新查一遍**，用当时的值说 —— 一行 SQL 拿全部：
>
> ```powershell
> $env:MYSQL_PWD='123456'
> mysql -h 127.0.0.1 -P 3308 -u root -e "SELECT (SELECT COUNT(*) FROM edurec.resources) resources, (SELECT COUNT(*) FROM edurec.resources WHERE source_url LIKE '%bilibili.com%') bili, (SELECT COUNT(*) FROM edurec.users) users, (SELECT COUNT(*) FROM edurec.recommendations) recs, (SELECT COUNT(*) FROM edurec.resource_comments) comments;"
> ```

> ⚠️ **补充：`avg_rating` 曾被回算修复**。清理时发现全库只有 1 条资源有 `avg_rating`，
> 其余为 0 导致「按评分降序」的兜底退化成 id 倒序（首页被同主题资源刷屏）。
> 已按 `ratings` 表回算：现 **153/170** 条资源有真实均分（如《泛函分析》3.2、《解析几何》3.1），
> 兜底列表因此是像样的热门内容。若老师问「热门怎么算的」，照实说：`AVG(score)` 降序、无评分者为 0 排后。

> ⚠️ **ID 重叠事实（仍然重要，答辩会被问）**：`backend/data/recommendations.json` 是 engine 的**模拟数据集**产物，
> key 是模拟用户 id `0..1999`、value 是模拟资源 id `0..499`；而平台真实 B 站视频占用 id `1..153`（数值重叠）。
> 导入服务只保留「库中真实存在的用户 / 资源 id」（`recommendation_import.go:99-111`），其余计入 `skipped_*`。
> **当前库已清除模拟数据**，所以现在导入会**绝大部分被跳过** —— 见下面对 1.6 节的警示。

### 1.6 ⚠️ 不要执行 demo_seed（会重新引入模拟资源）

```powershell
# ⛔ 不要运行：这一条会把模拟资源/用户/分类/行为重新写回库，
#    首页与搜索页会再次出现「模拟资源228」这类占位标题。
# cd E:\work\learn\edurec-platform\backend
# $env:CONFIG_PATH='configs/config.local.yaml'
# go run ./cmd/demo_seed -with-behaviors
```

**背景**：按你的要求「页面里不要有模拟资源」，2026-09-19 已把模拟数据**彻底清除**：

```powershell
# 已执行（留档备查）：删模拟资源 → 连带其行为/评分 → 删模拟用户及关联 → 删模拟分类
DELETE b FROM user_behaviors b      JOIN resources r ON r.id=b.resource_id WHERE r.title LIKE '模拟资源%';
DELETE t FROM ratings t             JOIN resources r ON r.id=t.resource_id WHERE r.title LIKE '模拟资源%';
DELETE FROM resources               WHERE title LIKE '模拟资源%';
DELETE FROM recommendations         WHERE resource_ids='[]' OR JSON_LENGTH(resource_ids)=0;
DELETE b FROM user_behaviors b      JOIN users u ON u.id=b.user_id WHERE u.username LIKE 'demo%' AND u.username NOT IN ('demo_admin','demo_fresh');
DELETE t FROM ratings t             JOIN users u ON u.id=t.user_id    WHERE u.username LIKE 'demo%' AND u.username NOT IN ('demo_admin','demo_fresh');
DELETE rc FROM recommendations rc   JOIN users u ON u.id=rc.user_id   WHERE u.username LIKE 'demo%' AND u.username NOT IN ('demo_admin','demo_fresh');
DELETE FROM users                   WHERE username LIKE 'demo%' AND username NOT IN ('demo_admin','demo_fresh');
DELETE FROM categories              WHERE name LIKE '模拟类别%';
```

> 于是 `demo<数字>` 账号（`demo1`…`demo1999`）**已不存在**；`demo_seed` 的用途也随之变成「只在需要恢复模拟数据时才跑」。
> **演示前若发现库被清空**，正确做法不是跑 `demo_seed`，而是重建两个演示账号（见 1.8 / 第 2 幕的 register 命令）。

**关于导入 engine 推荐结果**：`data/recommendations.json` 引用的是模拟资源 id `0..499`。
在**已清除模拟数据**的库上导入，结果是绝大部分被跳过（只有 id `1..153` 中真实存在的那几条会命中）：

```powershell
$login = Invoke-RestMethod http://127.0.0.1:8080/api/v1/auth/login -Method Post `
  -ContentType 'application/json' -Body '{"username":"demo_admin","password":"demo123456"}'
$tok = $login.data.access_token
Invoke-RestMethod http://127.0.0.1:8080/api/v1/admin/recommendations/import -Method Post `
  -Headers @{ Authorization = "Bearer $tok" } | ConvertTo-Json -Compress
```

> ⚠️ 现在跑它会得到（**已按当前数据精确推算**）：
> `imported_users=3, imported_resources=16, skipped_users=1998, skipped_resources=40004` —— 命中率极低。
> 因为这份 engine 产物来自**模拟数据集**，而库里已清除模拟数据，只有 id `1..153` 里真实存在的那几条会命中。
> 它还会**覆盖**当前那 3 行真实缓存（写入这 16 条）。**建议只在老师明确想看"导入接口"时才跑**，
> 并照着解释：「导入按平台真实存在的 ID 过滤，所以 `skipped_*` 很大 —— 这正说明**过滤逻辑生效**，
> 引擎的内部编号换到平台库不会直接错配。」想恢复当前基线，见第 6 节复位说明。

### 1.7 在线抓取通路预检（第 3 幕要用，30 秒）

```powershell
cd E:\work\learn\edurec-platform\backend\crawler
python --version
python online.py search --keyword "雅思" --limit 3 --page 1
```

期望：`python --version` → `Python 3.14.0`；第二条**退出码 0**，stdout 打印**一行** JSON，形如
`{"items":[{"bvid":"BV...","title":"...","source_url":"https://www.bilibili.com/video/BV..."}]}`。
若这里失败，第 3 幕直接走兜底 C。

> ⚠️ **注意**：`online.py` 本身只负责抓取、**不写库**（落库是 Go 侧 `ImportItems` 干的），
> 所以这条自检命令不会改数据；但第 3 幕在前端触发的在线抓取**会**写库（这正是要演示的效果）。

### 1.8 让第 2 幕的兜底演示成立（30 秒，**必做**）

第 2 幕要演示「缓存缺失 → 兜底并回写」，所以演示前必须确认 `demo_fresh` **没有**缓存行
（只要有人用它打开过首页，就会写上缓存行）：

```powershell
$env:MYSQL_PWD='123456'
mysql -h 127.0.0.1 -P 3308 -u root -e "DELETE FROM edurec.recommendations WHERE user_id = 2001;"
mysql -h 127.0.0.1 -P 3308 -u root -N -e "SELECT COUNT(*) FROM edurec.recommendations WHERE user_id = 2001;"
```

期望：第二条输出 `0`。**清完就不要再拿 demo_fresh 登录首页**（一登录就写回缓存行了），
留到第 2 幕现场第一次打开时才触发兜底。

> 库里当前应有 **3 行**缓存（user_id 2 / 3 / 2000，属于 BTXL / biliverify / demo_admin），`demo_fresh`(2001) 为 0 行。
> 若 `demo_fresh` 不存在（库被清过），按第 2 幕的 `register` 命令重建，id 会变，把上面的 `2001` 换成它的真实 id。

### 1.9 自检快照（30 秒内扫一眼）

- [ ] Redis `PING` → `PONG`
- [ ] `8080/api/v1/health` → `healthy`
- [ ] `5173/api/v1/health` → `healthy`（代理通）
- [ ] 浏览器无痕窗口打开 `http://localhost:5173/login`，用 `demo_admin / demo123456` 能进首页
- [ ] `demo_fresh` 的缓存行已清空（1.8 节，查询返回 `0`）
- [ ] 浏览器另开无痕窗口（或退出登录后）用 `demo_admin / demo123456` 能进 `http://localhost:5173/admin`
- [ ] 首页、列表页、搜索结果里**看不到任何「模拟资源」字样**（搜索「模拟」应返回 0 条）
- [ ] 终端里 4 个窗口：Redis、后端、前端、备用命令窗口

> 建议准备**两个浏览器 profile/无痕窗口**：一个用 `demo_fresh`（普通用户视角），一个用 `demo_admin`（管理员），
> 避免每幕都要退出登录（退出登录会清 token，刷新 token 在 Redis 里）。

---

## 2. 演示动线（8 幕）

| 幕 | 页面 | 讲什么（决策编号） |
|---|---|---|
| 1 | `/`（demo_admin） | 首页推荐 = 读缓存表（#1 #2 #3 #12 #24） |
| 2 | `/login`（demo_fresh）→ `/` | 个性化 vs 热门兜底的分界 + 兜底回写缓存（#12 #24） |
| 3 | `/search` | 三条内容渠道 + 在线抓 B 站 + 无限滚动（#28 #29 #30 #31 #32） |
| 4 | `/resources/:id` | 资源统一抽象 / B 站评论独立建模（#24 #29 #30） |
| 5 | `/resources/:id` 评分 + `/user/me` | 行为与评分埋点、个人中心行为历史（#24） |
| 6 | `/admin/*` | 管理后台四页（#4 #25 #18 #28） |
| 7 | 终端 + `/admin` | 数据闭环：export_snapshot → engine → 导入（#1 #2 #12 #15） |
| 8 | 终端 | 测试与工程化收尾（#21 #9 #19） |

---

### 第 1 幕 · 首页：读推荐缓存表（约 2 分钟）

**操作**
1. 打开 `http://localhost:5173/login`，输入 `demo_admin` / `demo123456`，回车登录。
   （用 `BTXL` / `biliverify` 效果相同，但它们没有已知密码；`demo_admin` 同时还能进后台，一举两得。）
2. 登录后自动跳到 `http://localhost:5173/`（首页）。

**说什么（一句话要点）**
> 「首页这个列表不是后端现算的，是**读 `recommendations` 缓存表**返回的；缓存里存的是资源 ID 的有序列表，
> platform 只做查询与顺序还原。这就是决策 #12『离线批量 + 结果落库』——**个性化与否，取决于这张表有没有这一行**。」
> 再补一句现状（诚实）：「当前库里是 3 行真实缓存（BTXL / biliverify / demo_admin），共 170 条真实 B 站资源，
> **页面上已经没有任何模拟数据**。」

**期望看到**
- 页面标题「发现优质教育资源」，副标题「为你推荐的精选课程、文章与视频」，**12 张真实卡片**（首页固定请求 `limit=12`）。
- 卡片全是真实 B 站教育视频（如《高等数学》宋浩、《泛函分析》、《解析几何》），**没有「模拟资源」字样**。
- 打开 DevTools Network，能看到 `GET /api/v1/recommendations?limit=12` 返回 `code:0`，且 `data.updated_at` 有值（Excel 序列号那种格式不用管，是 `time.Unix(...)` 转出来的）。

**证据链（可选，在备用终端敲）**

```powershell
$login = Invoke-RestMethod http://127.0.0.1:8080/api/v1/auth/login -Method Post `
  -ContentType 'application/json' -Body '{"username":"demo_admin","password":"demo123456"}'
$tok = $login.data.access_token
$rec = Invoke-RestMethod 'http://127.0.0.1:8080/api/v1/recommendations?limit=12' -Headers @{ Authorization = "Bearer $tok" }
$rec.data.list | Select-Object id, title     # 应返回 12 条真实资源
```

**对应决策**：#1（集成方式）、#2（文件/目录交接）、#3（engine 独立仓库）、#12（离线批量 + 结果落库）、#24（资源统一抽象 + JSON metadata）。
**代码位置**：`frontend/src/pages/home/index.vue` → `backend/internal/service/recommendation.go` 的 `Get()`（命中缓存分支）。
**兜底**：若缓存行为 0（库被清过），首页会自动走热门兜底 —— 直接接第 2 幕讲，不影响演示。

---

### 第 2 幕 · 个性化 vs 热门兜底（约 2 分钟）

这一幕**不点 UI**，讲的是第 1 幕背后的分支，是最容易被追问的点。

**操作（用没有缓存行的用户 `demo_fresh`）**

`demo_fresh` **在本机已经建好了**（用户名 `demo_fresh` / 密码 `demo123456`，users.id=2001），
演示前按 **1.8 节**确认它的推荐缓存行为 0 即可；下面这条只在「库被清过、账号没了」时才需要跑：

```powershell
Invoke-RestMethod http://127.0.0.1:8080/api/v1/auth/register -Method Post `
  -ContentType 'application/json' `
  -Body '{"username":"demo_fresh","email":"demo_fresh@sim.local","password":"demo123456","display_name":"新注册演示用户"}'
```

> ⚠️ **账号说明**：库里现在只有 4 个真实用户 —— `BTXL`(id 2) · `biliverify`(id 3) · `demo_admin`(id 2000) · `demo_fresh`(id 2001)。
> `BTXL` / `biliverify` 是真实注册用户、**密码不是 `demo123456`**，不要拿来演示登录。
> （原 `demo0..demo1999` 属模拟数据，已随模拟资源一并清除，**不再存在**。）

现场操作：
1. 退出当前登录，用 `demo_fresh` / `demo123456` 登录，落到首页。
2. 打开 `backend/internal/service/recommendation.go`，指给老师看 `Get()` 的三段：
   `1. 优先读缓存` → `2. 缓存命中：按缓存中的顺序返回资源` → `3. 缓存未命中：兜底生成（按评分降序取热门资源）并写入缓存`。

**说什么（一句话要点）**
> 「缓存表里**没有这一行**的用户（新注册、engine 还没覆盖到）不会没有推荐：后端按 `avg_rating DESC, id DESC` 取一批资源返回，并顺手写一行缓存。所以『引擎还没跑/没覆盖到我』的降级表现是**一个非空的通用列表**，不是空白页 —— 决策 #12 明确要求服务始终有返回。」

**期望看到**
- `demo_fresh` 首页有 **12 张卡片**（它没有缓存行，第一次请求必然走兜底）。
- 列表内容是**真实评分最高的那批 B 站资源**：实测为《泛函分析 8-1：完备化》(id 103, 3.2)、《解析几何的本质是什么？》(id 139, 3.1)、
  《复变函数与积分变换》正课 (id 119, 3.0) 等 —— **全是真实内容，无模拟资源**。
- 再说一句**容易被追问的细节**：兜底排序键是 `avg_rating DESC, id DESC`。清理阶段已按 `ratings` 表回算过 `avg_rating`
  （现 **153/170** 条有真实均分，其余为 0 排在后），所以这个列表是像样的热门内容，而不是退化成 id 倒序。
  早期版本里"评分降序会退化成新上架优先"的说法**在当前数据上已不成立**，别照旧讲。
- 再到 1.5 节的 SQL 里查一次 `recommendations`，`demo_fresh` 的缓存行**从 0 变成 1**（用 `WHERE user_id=2001`）—— 兜底确实写回了缓存。
  顺手说清**这正是「陈旧但可用」语义的由来**：下一次导入（第 6.5 / 第 7 幕）会用 engine 的新结果**覆盖**这一行，在那之前首页一直读的是它。

**对应决策**：#12（未命中按评分兜底并写缓存）、#3（platform 无在线推理）、#24（`avg_rating` 是资源表字段）。
**兜底**：如果不想动库，就只讲代码（第 2、3 段分支），效果一样能讲清。

---

### 第 3 幕 · 搜索页：在线抓 B 站 + 无限滚动（约 4 分钟，全场最亮的一幕）

**操作**
1. 顶部导航点「课程」（等价于直接打开 `http://localhost:5173/search`）。
2. 在关键词框输入 **`雅思`**，回车（或点「搜索」）。

   > ✅ **顶栏搜索框已经修好了**（2026-09-19）：页面顶部 header 里占位符为「搜索课程、文章、视频…」的那个框，
   > 原先**没有绑定 `v-model`**，输入的内容会被渲染清掉（表现为"打不进字"）；现在已绑定并支持回车跳转
   > —— 跳转时会带 `?keyword=`，搜索页挂载即按该关键词搜索，且输入框会回填。
   > 演示时**两个框都能用**；若老师问起，可以照实说「这里原先是个没接参数的占位入口，已修」。
   > （改动见 `FrontLayout.vue` 的 `submitSearch()` 与 `SearchPage.vue` 的 `keywordFromQuery()` + `watch`。）

3. 本地结果立刻耗尽 → 底部**自动**出现「正在加载…」，**不用手动滚到底**（本地 0 条时哨兵直接可见）。
4. 观察底部文案从「正在加载…」→ 列表**追加新卡片**（这些是刚从 B 站实时抓来并落库的）。
5. 打开 DevTools Network，找到带 `online_page=1` 的那条 `GET /api/v1/resources`。

**为什么选「雅思」**（2026-09-19 用 API 实测过，别凭 SQL 猜）：

| 关键词 | 本地 `total`（page_size=12） | 第一屏是否触发在线抓取 | B 站侧 |
|---|---|---|---|
| **`雅思`** ← 首选 | **0** | ✅ **立刻**（本地空，哨兵直接进视口） | 17 条 |
| `考研英语` / `量子力学` | **0** | ✅ 立刻（可互换） | 17 / 18 条 |
| `数据结构` | 1 | ✅ 会 | 20 条 |
| `概率论` / `离散数学` / `大学物理` / `操作系统` / `计算机网络` | 2 | ✅ 会（本地耗尽即触发） | 18–20 条 |
| `高等数学` | 6 | ⚠️ 需先滚到底才耗尽，第一屏不触发 | 19 条 |
| `线性代数` / `机器学习` | 24 | ❌ 本地就有两页多，要滚很久才轮在线 | 19 条 |

> ⚠️ **为什么首选本地 0 条的关键词（重要）**：在线抓取会**真实写库**，所以每演示一次，该关键词的本地命中数就涨一截。
> 用 `数据结构`（原有 2 条）演一次之后本地就变成 18 条，**第二次演示「第一屏即触发」就不成立了**。
> 选 `雅思` 这类本地 0 条的词，即使演过一轮，下次也只是本地变成十几条、仍能立刻触发（且它不在任何既有资源的字段里，最干净）。
> **如果连续演示多次，每次换一个没演过的词**（`考研英语`、`量子力学`）。

> ⚠️ **另一个坑（判据）**：`title like '%高等数学%'` 的 SQL 只数出 **2 条**，但**搜索是跨字段匹配**的
> （标题 / 描述 / **标签**），实际 API 返回 **6 条**。「高等数学能立刻触发」曾出现在本文件早期版本里，是错的。
> **判据要用 API 的 `total`，不要用 `title like` 的 SQL。**

**说什么（一句话要点）**
> 「搜索先是**只查本地库**；本地结果翻完之后，如果是**纯关键词**搜索（没加分类/类型/标签筛选），前端才改带 `online_page=M` 去请求，后端 `exec` 调 Python 的 `online.py search` 实时抓 B 站第 M 页，**复用与离线导入完全相同的落库内核** `ImportItems` 写库，再把本次新导入的资源返回给前端 —— 这就是决策 #30 的『离线批量 + 在线实时并存』。」

**期望看到（一一指给老师看）**
1. 搜索后**本地 0 条**，页面几乎立刻进入「正在加载…」，说明在线抓取自动触发了。
2. 几秒后**出现十几张卡片**，卡片的 `source_url` 都是 `https://www.bilibili.com/video/BV...`，分类是 `B站视频`。
   （实测：`雅思` 在线抓取 6.9s 返回 **17 条**，新行 id 从 555 起；`数据结构` 早先实测 4.1s / 18 条。）
3. Network 里 `online_page=1` 的响应带 `has_more` 字段；`total` 字段为 **0**（在线结果不计入本地 total，走的是另一条分支）。
4. 底部随即显示到底、不再继续追加 —— **见下面的「已知边界」，别在这里说会一直翻页**。

> ⚠️ **已知边界（老师若追问，照实说）**：`has_more` 的判定是
> `len(resp.Items) >= search_limit(20) && page < search_max_pages(10)`
> （`bilibili_online.go:144`），而 B 站对多数关键词单页只返回 **17–20 条**（字段缺失的会被 `normalize_search_item` 丢掉），
> 一旦少于 20 就判为没有下一页。**实测 `雅思`（17 条）与 `数据结构`（18 条）的 `online_page=1` 都返回 `has_more=false`**，
> 所以无限滚动在本机实际表现是「**自动追加一批（十几条）在线内容，然后停止**」，而不是连续翻十页。
> 说它是「本地耗尽后自动补一批 B 站实时内容」是准确的；说「会一直往后翻」是不准确的。

**顺手讲的三个设计点（全是决策记录原文）**- **判重 + 幂等**（决策 **#29**）：落库按 `source_url` 判重，命中**不新增行**，只刷新 `view_count` / `metadata`，所以标题、简介等平台侧人工编辑不会被覆盖。用命令行验证：

  ```powershell
  cd E:\work\learn\edurec-platform\backend
  $env:CONFIG_PATH='configs/config.local.yaml'
  go run ./cmd/import_bilibili            # 第 1 次：新增 X 条，刷新 0 条
  go run ./cmd/import_bilibili            # 第 2 次：新增 0 条，刷新 X 条  ← 幂等
  ```

- **契约泛化**（决策 **#32**）：B 站和第三方数据集共用同一个 `CrawlImportService.ImportItems(items, opts{ResourceType, SourceURLTemplate}, write)`，
  差异只剩「落库 type」和「source_url 怎么来」两个参数。
- **合规边界**（决策 **#28 / #30**）：走官方 JSON 接口、不解析 HTML、`buvid3` 靠访问首页获取而**不伪造设备指纹**、
  30s 子进程超时、风控 `-352/-412` **不重试**、`search_max_pages` 防无界爬取。
- **内容质量闸门（教育分区白名单）**：B 站搜索是**按关键词返回**的，不区分内容性质 ——
  搜非教育主题（如影视剧角色名）会把影视剪辑、娱乐杂谈、网络游戏一并返回。
  因此落库前按 B 站**分区名** `metadata.typename` 做白名单拦截
  （`config.DefaultEducationalTypenames`，在线抓取与离线导入共用）。
  **现场可以演示**：搜一个非教育关键词（如 `杨真真`），页面只会追加极少量命中、库内不新增影视内容；
  而搜 `高等代数` 会正常入库。计数里的 `skipped_typenames` 就是被拦下的条数。
  细节见 `docs/bilibili-import.md` 的「教育分区白名单」。

**对应决策**：#28（外部内容来源）、#29（采集内容建模）、#30（在线搜索 + 评论 + 无限滚动）、#31（第三方数据集的对照）、#32（落库契约泛化）。
**代码位置**：`frontend/src/pages/search/SearchPage.vue`（`canCrawlOnline` / `loadMore` / `fetchOnline`）→ `backend/internal/service/resource.go` 的 `List()`（`OnlinePage > 0` 分支）→ `backend/internal/service/bilibili_online.go`。
**兜底**：在线失败 → 见第 4 节兜底 C / D。

---

### 第 4 幕 · 资源详情页（真实 B 站视频 + B 站评论区）（约 3 分钟）

**操作**
1. 地址栏直接输入 `http://localhost:5173/resources/63`（宋浩《高等数学》全程教学视频 2.0 版）。
2. 等页面加载完，往下滚到「**B 站评论**」区块。

**说什么（一句话要点）**
> 「资源用一张表统一抽象，靠 `type` 字段区分 course / article / video，B 站视频就是 `type=video` 的**普通资源**，没有专区、不改推荐链路；而 B 站评论落的是**独立的 `resource_comments` 表**，绝不污染站内评分。」

**期望看到**
- 顶部封面图能正常显示（`<img>` 带 `referrerpolicy="no-referrer"`，绕过 B 站 CDN 防盗链；万一 403 会**降级成占位块**而不是破图）。
- 标题下方一行元信息：`B站视频` 标签、分类名、`#标签`、作者、`★ 评分`、`N 次浏览`。
- 「元信息」区块是 `metadata` 的键值表原样渲染，其中 `pubdate`（Unix 秒）被**特判格式化成 `YYYY-MM-DD`**，`duration`、`like` 等直接展示。
- 「用户评价（N）」是**站内评分**列表（`ratings` 表），与下面的 B 站评论**是两块完全不同的数据**。
- 「B 站评论」区块显示**作者昵称 + 正文 + 点赞数**（默认抓 `comment_limit: 20` 条热门评论）。
- 点页面上的「前往原站学习」会新窗口打开 B 站原视频（回链原始来源，见合规边界）。

**验证评论落库（说明第一次是实时抓的、第二次读缓存）**

```powershell
$env:MYSQL_PWD='123456'
mysql -h 127.0.0.1 -P 3308 -u root -N -e "SELECT COUNT(*) FROM edurec.resource_comments WHERE bvid='BV1DgxCzREbM';"
```

第一次打开详情页前计数可能为 0，打开后变成 20（或本次真实抓到的条数）；**再刷新页面计数不变**，因为第二次直接读缓存。

**对应决策**：#24（资源统一抽象 + JSON metadata）、#29（B 站视频以普通资源身份混入）、#30（评论独立建模、30s 超时、风控不重试）。
**兜底**：评论抓取失败 → 见第 4 节兜底 E。

---

### 第 5 幕 · 评分 / 行为埋点 + 个人中心行为历史（约 3 分钟）

**操作**
1. 仍在 `/resources/63`，右侧评分卡：「你的评分」点 **5 星**，文本框输入「讲得很清楚」，点「**提交评价**」。
2. 期望 Toast「评价成功」，右上角「N 条评价」数字 +1，左侧「用户评价」列表出现自己那条。
3. 再点页面里任意一张资源卡片（或直接跳 `/resources/9`），**这一步会产生一条 `view` 行为**。
4. 右上角头像下拉 →「**个人中心**」（`http://localhost:5173/user/me`）。
5. 在「行为历史」里用 `全部 / 浏览 / 点击 / 收藏` 单选按钮切着看。

**说什么（一句话要点）**
> 「详情页加载成功后才上报 `view`（404 不上报），评分走 `POST /resources/:id/ratings` 的 upsert（同一用户同一资源只有一条）；个人中心直接读 `GET /users/me/behaviors`，这些行为/评分就是 `export_snapshot` 导出给 engine 的训练输入。」

**期望看到**
- 提交后资源页的 `avg_rating` 与「N 条评价」**立即刷新**（提交成功后页面前端 `loadResource()` + `loadRatings()` 双刷）。
- 个人中心上方是 ProfileCard（昵称/邮箱等，可改），下方是「行为历史」卡片，条目按时间倒序，
  每条左侧是 `view / click / favorite` 彩色标签，右侧是「资源标题」（点标题能跳回详情页）+ 时间。
- 切到「浏览」只留 `view`，切到「收藏」可能为空 —— 这是筛选生效的正常表现。
- 库里能看到刚才那两条记录：

  ```powershell
  $env:MYSQL_PWD='123456'
  # 注意：users 表里已没有 id=1 这个用户（真实用户只有 id 2/3/2000/2001），
  # 所以下面用「取当前演示用户的 id」而不是写死 1 —— 拿 id 查空表就白演了。
  mysql -h 127.0.0.1 -P 3308 -u root -e @"
  SELECT action, resource_id, created_at FROM edurec.user_behaviors
    WHERE user_id=(SELECT id FROM edurec.users WHERE username='demo_fresh') ORDER BY id DESC LIMIT 5;
  SELECT resource_id, score, comment FROM edurec.ratings
    WHERE user_id=(SELECT id FROM edurec.users WHERE username='demo_fresh') ORDER BY id DESC LIMIT 3;
  "@
  ```

**衔接下一幕**：「刚才这些新行为和评分，**本轮推荐是看不到的** —— batch 形态的时效口径是『反映截至导出时刻的快照』，要等下一轮 导出 → 训练/推理 → 导入 才生效。」（决策 #1 / #2 / #12，见 `docs/engine-integration.md`「刷新节奏与时效口径」）

**对应决策**：#24（行为/评分作为引擎输入）、#12（batch 时效口径）。
**兜底**：如果 toast 报「登录已过期」，见第 4 节兜底 B。

---

### 第 6 幕 · 管理后台（约 4 分钟）

**入口（重要）**：前台导航里**没有**后台入口，必须**直接输 URL**。

```
http://localhost:5173/admin
```

用 `demo_admin` / `demo123456` 登录。后台是侧边栏 + 顶栏的独立布局（决策 **#18**），
侧边栏四项：仪表盘 / 资源管理 / 用户管理 / 分类管理。

**6.1 仪表盘 `/admin`（30 秒）**
- 期望：两张可点的大数字卡 ——「资源总数 **170**」「用户总数 **4**」（清除模拟数据后的实测值；**以 1.5 节上台前实查的为准**，在线抓取会让资源数只增不减），下面三个快捷入口。
- 说什么：「管理员身份不是用户表里的一个 flag，而是**独立的 `admins` 表**（`id, user_id`）——决策 #25。所以 `demo_admin` 是『users 里一行 + admins 里一行』。」

**6.2 用户管理 `/admin/users`（1 分钟）**
- 操作：搜索框输入 `demo_admin` 或 `BTXL`，回车。
- 期望：表格列 ID / 用户名 / 邮箱 / 昵称 / 角色 / 注册时间；`demo_admin` 那行「角色」是**黄色的「管理员」标签**，其余是灰色「普通用户」。
- 说什么：「角色标签就是查 `admins` 表的结果，`GET /admin/users` 需要 `AdminRequired` 中间件放行（决策 #25、#20）。」

**6.3 资源管理 `/admin/resources`（1.5 分钟）**
- 操作：类型下拉选「**视频**」，回车/触发搜索；再输入关键词（如 `高等数学`）。
- 期望：表格列 ID / 标题 / 类型标签（video 绿色）/ 分类 / 评分 / 浏览 / 创建时间 / 操作；点「新增资源」跳 `/resources/upload`（管理员上传表单）。
- 说什么：「B 站采集进来的资源**就是这张表里的普通行**，后台一视同仁能搜、能改、能删 —— 决策 #29：不新增专区、不改推荐链路；删除是 `DELETE /resources/:id`，同样要管理员。」
- ⚠️ **不要在现场演示删除**（会连带影响详情页和推荐列表的演示稳定性）。

**6.4 分类管理 `/admin/categories`（1 分钟）**
- 操作：输入分类名「**答辩演示分类**」，描述随便写，点创建。
- 期望：分类列表里立刻出现新行（当前库里 **2 个**分类：人工智能、B站视频）。
- 说什么：「分类是 find-or-create 语义 —— 采集/导入时遇到没见过的分类名会**静默新建**（决策 #28/#29 的已知限制），所以后台这里能直接管。」
- ⚠️ 如果不想污染数据，这一步可以**只讲不点**（把输入框填好，说「这里点下去就建了」）。

**6.5 导入 engine 推荐结果（1.5 分钟，终端演示，因为前端没有这个按钮）**

> 事实核对：`frontend/src/api/admin.ts` **只有**用户和资源两个接口，**没有**导入推荐的 UI；
> 导入是后端管理员接口 `POST /api/v1/admin/recommendations/import`（决策 #12），所以这一步在终端演示。

```powershell
$login = Invoke-RestMethod http://127.0.0.1:8080/api/v1/auth/login -Method Post `
  -ContentType 'application/json' -Body '{"username":"demo_admin","password":"demo123456"}'
$tok = $login.data.access_token
Invoke-RestMethod http://127.0.0.1:8080/api/v1/admin/recommendations/import -Method Post `
  -Headers @{ Authorization = "Bearer $tok" } | ConvertTo-Json -Compress
```

- 期望（**清除模拟数据后的当前值，按现有数据精确推算**）：
  `{"code":0,"message":"ok","data":{"imported_users":3,"skipped_users":1998,"imported_resources":16,"skipped_resources":40004}}`
  —— 只有 3 个真实用户命中，其余 1998 个 key 是已删除的模拟用户 id；`skipped_resources` 巨大是因为文件引用的是模拟资源 id。
  **别把它当失败**：接口返回 `code:0`，过滤行为完全正确。
- 说什么：「这个接口**只认一个文件**：`engine.recommendations_file`（`data/recommendations.json`，配置见 `configs/config.local.yaml` 的 `engine:` 段）。
  格式是 `{"<user_id>": [<resource_id>, ...]}`，**平台原始 ID**；导入时按库里真实存在的 user/resource 过滤，
  其余跳过 —— 返回里的 `skipped_*` 就是被过滤掉的数量（决策 #12、#15 的 Viper 配置外置）。」
- 顺手补两句现状（**重要，别说反**）：
  1. 「`recommendations.json` 是 engine **模拟数据集**的产物（引用资源 id `0..499`），而库里现在只有 170 条**真实** B 站资源，
     所以命中率天然很低 —— 这正是『两套 ID 空间靠数值相同才匹配』的体现。」
  2. ⛔ **不要为了"让导入好看"去跑 `demo_seed`** —— 那会把模拟资源重新写回库、首页又出现「模拟资源228」，
     违背今天"页面里不要有模拟资源"的要求（见 1.6 节）。导入这一步**只讲过滤语义**即可。
  3. 跑完它会把那 3 行真实缓存**覆盖**成这 16 条；想恢复基线看第 6 节。

**对应决策**：#4（Handler→Service→Repository）、#25（独立 admins 表）、#18（后台布局）、#28/#29（采集资源在后台可管）、#12/#15（导入接口与配置外置）、#20（统一错误处理与中间件）。

---

### 第 7 幕 · 数据闭环：export_snapshot → engine → 导入（约 4 分钟，可选）

这是**唯一需要开两个终端**的一幕。engine 仓库不在本项目目录下，本幕设计为「**平台侧两步现场跑 + engine 侧一步口头交代**」。

**操作 ①：平台导出快照（现场跑，约 10 秒）**

```powershell
cd E:\work\learn\edurec-platform\backend
$env:CONFIG_PATH='configs/config.local.yaml'
go run ./cmd/export_snapshot
```

期望：

```
[export_snapshot] 快照导出完成 -> data\snapshots\20260919_HHMMSS
```

看看导出了什么（**这是整个项目最硬的证据**）：

```powershell
$run = (Get-ChildItem data\snapshots | Sort-Object LastWriteTime -Descending | Select-Object -First 1).Name
Get-ChildItem "data\snapshots\$run" | Select-Object Name, Length
Get-Content "data\snapshots\$run\meta.json"
```

期望文件：`meta.json` + `users.csv` + `resources.csv` + `categories.csv` + `behaviors.csv` + `ratings.csv`。
`meta.json` 实机内容如下（**2026-09-19 12:16:13 清除模拟数据后的真实导出**，数值随导出时刻变化，字段结构固定）：

```json
{
  "behavior_window": { "end": 1789556016, "start": 0 },
  "behaviors_count": 37,
  "categories_count": 2,
  "contract_version": 1,
  "exported_at": 1789791373,
  "files_sha256": {
    "behaviors.csv": "8a0e0c4d...",
    "categories.csv": "f0efe8a6...",
    "ratings.csv": "25d90957...",
    "resources.csv": "26e26ea2...",
    "users.csv": "2c222f86..."
  },
  "ratings_count": 3,
  "resources_count": 170,
  "run_id": "20260919_121613",
  "users_count": 4
}
```

> ⚠️ **照终端实际输出念，别念这份示例**：你现场重新 `export_snapshot` 会得到**新的 `run_id`**，
> 而且 `resources_count` 会随着第 3 幕的在线抓取变大（演示过搜索就会多几十条）。
> 讲的重点是**结构**：`contract_version` + `files_sha256`（可校验）+ 各表 count，而不是具体数字。

指给老师看四个点：**`contract_version`**（契约版本，换字段要升版本）、**`run_id`**（与目录名一致，一轮一 id）、
**`behavior_window`**（行为时间窗，engine 据此切训练集）、**`files_sha256`**（每个文件的校验和，engine 可验完整性）。

说什么（一句话要点）：
> 「平台导出的是**契约**，不是模型：`users.csv` 只导 ID、**不含任何个人字段**；快照带 sha256 和 contract_version，
> engine 拿到的是一个可校验的、版本化的数据集。两仓之间不互相调用，交叉接口就是这份文件（决策 #2、#3）。」

**操作 ②：交给 engine 训练 + 推理（口头交代 / 若 engine 在场可现场跑）**

```bash
cp -r backend/data/snapshots/<run_id> <engine>/dataset/platform_snapshot/<run_id>/
cd <engine>
python -m scripts.train_all       --data-source platform --snapshot-dir dataset/platform_snapshot/<run_id>
python -m scripts.run_batch_infer --data-source platform --snapshot-dir dataset/platform_snapshot/<run_id>
cp model/recommendations.json <platform>/backend/data/recommendations.json
```

说什么：「engine 里是 DSSM 召回 → DeepFM 精排 → MMR 重排，读同一个快照目录保证训练/推理口径一致，
最后吐一个 `{用户ID: [资源ID...]}` 的排名列表 —— **模型权重只留在 engine，平台永远拿不到也跑不了模型**。」

**操作 ③：回到平台导入（复用第 6.5 节的导入命令）**

```powershell
Invoke-RestMethod http://127.0.0.1:8080/api/v1/admin/recommendations/import -Method Post `
  -Headers @{ Authorization = "Bearer $tok" } | ConvertTo-Json -Compress
```

然后**刷新首页**，Network 里 `GET /api/v1/recommendations` 的 `updated_at` 变了 → 说明缓存被新结果覆盖。

**如果对方有 `bash`（Git Bash / WSL）**，整轮一条命令搞定：

```bash
bash scripts/handoff.sh              # 导出 + 训练 + 推理 + 导入
bash scripts/handoff.sh --infer-only # 复用 model/models.pt，只导出 + 推理 + 导入
```

`ENGINE_ROOT` / `CONFIG_PATH` / `SNAPSHOT_DIR` / `BASE_URL` / `ADMIN_USER` / `ADMIN_PASS` 都可环境变量覆盖。

**期望看到的现象**
- 快照目录下 6 个文件齐全，`meta.json` 的 `behaviors_count` ≈ 37（含第 5 幕现场产生的那几条）。
- 导入返回的 `imported_users` 与库里用户数同量级。
- 首页 `updated_at` 刷新。

**对应决策**：#1、#2、#3、#12（闭环主链路）、#15（配置外置可覆盖）、#19（slog 结构化日志）。
**兜底**：engine 没装/没跑 → 见第 4 节兜底 F。

---

### 第 8 幕 · 工程化收尾（约 1 分钟，视时间取舍）

```powershell
cd E:\work\learn\edurec-platform\backend
go test ./... ; go vet ./...

cd E:\work\learn\edurec-platform\frontend
npm test ; npm run type-check
```

**2026-09-19 本机实测（照实说，别夸大）**：

| 命令 | 实测结果 | 说明 |
|---|---|---|
| `go test ./...` | **通过（exit 0）** | `middleware`/`model`/`service`/`util/jwt`/`util/refresh` 有测试；**`handler`/`repository`/`router` 三个目录当前没有测试文件** |
| `go vet ./...` | **通过（exit 0）** | 输出里的 `error acquiring upload token ... Access is denied` 是 Go 遥测告警，与本项目无关 |
| `npm run type-check` | **通过（exit 0）** | `vue-tsc --noEmit` |
| `npm test` | **111 个用例全部通过（27 个文件）** | 新增了「顶栏搜索框」等回归用例；本机一次全量约 45–50 秒（worker 冷启动较慢）。看输出里的 `Test Files 27 passed (27)` / `Tests 111 passed (111)` |

说什么：「后端 Go 单测覆盖认证/推荐/CRUD 关键路径，前端 Vitest + Vue Test Utils 覆盖核心交互（决策 #21）；
测试策略是**实用主义**——保关键路径，不追覆盖率数字（design.md 第 9 节）。」

> ⚠️ **现场提示**：全量 `npm test` 在本机要 45 秒以上（vitest worker 冷启动慢），现场时间紧就别跑全量，
> 用 `npx vitest run src/pages/search` 或直接讲代码。同时**主动承认** handler/repository/router 三层目前缺测试
> （`qa-prep.md` 与 `presentation-notes.md` 都已把这条列为待补项 —— 主动承认比被问出来得分高）。

---

## 3. 一页速查卡（打印出来放手边）

| 我要做什么 | 命令 / URL |
|---|---|
| 起 Redis | `& E:\work\cppsoft\Redis\redis-server.exe E:\work\cppsoft\Redis\redis.conf` |
| 验 Redis | `& E:\work\cppsoft\Redis\redis-cli.exe -p 6380 -a 284835 PING` |
| 起后端 | `cd backend; $env:CONFIG_PATH='configs/config.local.yaml'; go run ./cmd/server` |
| 起前端 | `cd frontend; pnpm run dev` |
| 健康检查 | `Invoke-RestMethod http://127.0.0.1:8080/api/v1/health` |
| 代理检查 | `Invoke-RestMethod http://127.0.0.1:5173/api/v1/health` |
| 登录页 | http://localhost:5173/login |
| 首页 | http://localhost:5173/ |
| 搜索（触发在线抓取） | http://localhost:5173/search → 关键词「雅思」→ 不用滚到底，自动加载 |
| 详情页（B 站评论） | http://localhost:5173/resources/63 |
| 个人中心 | http://localhost:5173/user/me |
| 管理后台 | http://localhost:5173/admin （`demo_admin`/`demo123456`） |
| 导出快照 | `cd backend; $env:CONFIG_PATH='configs/config.local.yaml'; go run ./cmd/export_snapshot` |
| 导入推荐 | `POST /api/v1/admin/recommendations/import`（管理员 Bearer Token） |
| 播种演示数据 | `go run ./cmd/demo_seed -with-behaviors` |
| B 站离线导入 | `go run ./cmd/import_bilibili [-dry-run]` |
| 数据集导入 | `go run ./cmd/import_dataset --dataset mooc --preview / --dry-run` |
| 一键闭环（bash） | `bash scripts/handoff.sh [--infer-only]` |

---

## 4. 兜底方案（某个环节挂了怎么接）

### 兜底 A · MySQL 挂了 / 连不上 3308

- **症状**：后端启动即 `MySQL 初始化失败`，前端所有页面红字「网络错误」。
- **替代**：改为**展示代码 + 文档**：打开 `docs/design.md` 第 5 节数据模型表、第 4.1 节架构图，
  讲清 7 张表的关系（`users ──1:N── user_behaviors / ratings / recommendations`，`resources ──N:1── categories`）。
  再用 `backend/migrations/` 里的 SQL 说明「迁移用 golang-migrate 独立 up/down 脚本管理」（决策 **#9**）。
- **话术**：「数据库没起，正好说明分层是干净的 —— 起不来的是 repository 层，handler/service 的契约没变。」

### 兜底 B · 登录成功但一刷新就掉登录 / 报 401

- **根因**：Redis 没起或密码不对 → refresh token 存不进去（决策 **#5**：refresh token 存 Redis）。
- **处理**：按 1.1 起 Redis；确认 `configs/config.local.yaml` 的 `redis.password: "284835"`、`port: 6380`。
- **替代演示**：**完全绕开登录**，用 `curl`/`Invoke-RestMethod` 演示认证链路：
  `POST /api/v1/auth/login` 拿 `access_token` / `refresh_token` / `expires_in`（15 分钟），
  `POST /api/v1/auth/refresh` 换新 token —— 接口契约在 `backend/internal/router/router.go` 第 69-72 行，
  讲决策 **#5**（Access 15 分钟 + Refresh 7 天，Refresh 落 Redis）与 **#16**（未登录路由守卫跳 `/login`）。

### 兜底 C · 在线抓 B 站失败（断网 / 风控 -352 / -412 / 30s 超时）

- **症状**：搜索页第一屏之后不再追加卡片，底部直接显示「没有更多内容」；
  后端日志有 `WARN B站搜索爬取失败 keyword=...`。
- **这是设计好的降级**（决策 **#30**：在线失败**不阻塞已加载内容**，前端 `fetchOnline()` 的 `catch` 直接把列表标记为到底；
  后端 `resource.go` 的 `List()` 出错时返回**空列表 + hasMore=false**，不返回 5xx）。
- **替代演示路径**（三层，任选）：
  1. **展示已落库的 B 站真实资源**：搜索页清空关键词、类型选「视频」，或直接打开 `/resources/63`、`/resources/9` —— 这些是之前采集并导入的（当前 170 条，`id 1..153` 是基线），**不依赖网络**。
  2. **演示离线导入链路**（离线批量，与在线共用同一段落库内核，决策 #28/#32）：
     ```powershell
     cd backend
     $env:CONFIG_PATH='configs/config.local.yaml'
     go run ./cmd/import_bilibili -dry-run   # 只统计，不写库
     go run ./cmd/import_bilibili            # 真导入（幂等：第二次全是「刷新」）
     ```
     期望输出形如 `[import_bilibili] 新增资源 0 条，刷新资源 27 条，跳过 0 条，新建分类 0 个`。
  3. **直接演示爬虫本体**：`cd backend\crawler && python run.py --config config.yaml --dry-run` —— 先只看抓到什么不写文件。
- **话术**：「在线抓取是**增量补充**，不是主链路。主链路是离线批量 + 命令导入，实时抓取失败只会让这一页少几条，不会让平台不可用。」

### 兜底 D · 无限滚动没被触发（关键词本地命中就 ≥ 12 条，或没滚到底）

- **根因**：`canCrawlOnline` 要求**纯关键词**搜索（无分类/类型/标签筛选），且本地结果翻完才走在线。
- **处理**：
  1. 确认关键词框里是 `雅思`（本地 `total` 为 0，最干净），且**分类/类型/标签三个筛选都是空的**；
  2. 把浏览器窗口缩小到一屏放不下，再滚到底；
  3. 兜底：直接在地址栏请求在线接口，让老师看后端实时抓取（用 demo_admin 的 token）：
     ```powershell
     (Invoke-RestMethod 'http://127.0.0.1:8080/api/v1/resources?keyword=雅思&online_page=1' `
       -Headers @{ Authorization = "Bearer $tok" }).data | ConvertTo-Json -Depth 4 -Compress
     ```
     响应里的 `list` 就是**本次实时抓取并落库**的新资源（实测一次追加 18 条、约 4 秒）；
     `has_more` 表示 B 站还有没有下一页 —— 注意多数关键词它会直接是 `false`（原因见第 3 幕的「已知边界」）。

### 兜底 E · 详情页评论区空 / 「评论暂不可用」

- **这是设计好的空态**（决策 **#30**）：评论抓取失败不阻塞页面主体，展示空态即可。
- **替代演示**：
  1. 指着页面说「站内评分（`ratings`）和 B 站评论（`resource_comments`）是**两张表、两条链路**，评论挂了评分照样用」；
  2. 用库里已缓存的 27 条评论证明链路通过：
     ```powershell
     $env:MYSQL_PWD='123456'
     mysql -h 127.0.0.1 -P 3308 -u root -e "SELECT resource_id, bvid, author_name, LEFT(content,40), like_count FROM edurec.resource_comments ORDER BY id LIMIT 5;"
     ```
  3. 需要现场再抓一次就手动跑：`cd backend\crawler && python online.py comments --bvid BV1DgxCzREbM --limit 5`（1.7 节已预检）。

### 兜底 F · engine 不在场 / 训练跑不动

- **症状**：第 7 幕操作 ② 无法执行。
- **替代演示路径**：
  1. **只跑平台侧第一步**（导出快照）—— 这一步已经完整证明了「platform → engine」契约，快照 + `meta.json` + sha256 都在磁盘上。
  2. 展示**已有的 engine 产物**：`backend/data/recommendations.json`（211 KB，key 为模拟用户 id 0..1999，value 为 20 个资源 id），
     说明「engine 的输出就是这个文件，平台只认这个格式」。
  3. 现场执行第 6.5 节的**导入**，让首页个性化刷新 —— 闭环的「结果上链」半程即可完整演示。
- **话术**：「训练/推理是 engine 的职责，平台这边**不需要 PyTorch、没有推理依赖**，所以我今天在这台机器上跑全额演示也不影响 —— 这正是决策 #1/#12 想要的解耦效果。」

### 兜底 G · 导入推荐报错

- **`Internal` / 文件读取失败**：`backend/data/recommendations.json` 不存在或不是合法 JSON。
  → 用 1.6 节的命令重建缓存；或先讲「整份文件为空/导入失败 → **缓存不变**，服务继续返回旧结果或兜底（陈旧但可用）」（`docs/engine-integration.md` 的「服务语义：空 / 缺失 / 陈旧」）。
- **`imported_resources` 很小、`skipped_resources` 很大**：
  → **这是清除模拟数据后的正常结果，不是故障**。`recommendations.json` 是 engine **模拟数据集**的产物（资源 id `0..499`），
  而 2026-09-19 已把模拟资源清出库，库里只剩 170 条真实 B 站视频（`id 1..153` 是基线），两套 id 只有数值相同才匹配得上。
  讲解要点正是 1.5 节的 ⚠️「两套 ID 空间数值重叠」：导入只写「库里真实存在的 id」，
  所以**导入后 `recommendations` 缓存表仍只有那 3 行**（user_id 2 / 3 / 2000，属于 BTXL / biliverify / demo_admin），
  但被**覆盖**成过滤后的少量结果，首页照常返回非空列表 —— 说明「过滤后仍然命中」。⛔ 别为了让它"好看"去跑 `demo_seed`（见 1.6 节）。
- **401 / 403**：token 过期（Access 15 分钟）。重新执行登录那两行拿新 token 即可；或讲决策 **#20**（HTTP 状态码 + 业务错误码 + middleware 统一封装）。

### 兜底 H · 前端编译/热更报错，页面白屏

- **处理**：`cd frontend && pnpm install && pnpm run dev` 重启；
  确认 `.env.development.local` 里是 `VITE_MOCK=0`（若被改回 1，前端会走 MSW 假数据 —— 那时**所有页面都"能用"但数据是假的**，答辩时会被看穿）。
- **快速辨别真假**（旧版「看有没有模拟资源」的判据已失效，模拟数据清除后不再适用）：
  1. 直接请求代理：`Invoke-RestMethod http://127.0.0.1:5173/api/v1/health` —— 真后端返回 `{"code":0,...,"status":"healthy"}`，MSW 拦不到这个路径；
  2. 用 `node -e "console.log(require('vite').loadEnv('development',process.cwd(),'VITE_').VITE_MOCK)"` 确认输出是 `"0"`（1.3 节）；
  3. 真后端首页是 12 条真实 B 站视频（带封面、作者、浏览量）；MSW 假数据是它自己造的固定条目。
- **终极兜底**：直接展示 `docs/design.md` 的架构图与决策记录表，按第 1、2 幕的话术讲架构与推荐口径。

---

## 5. 可能被追问的问题（备用弹药）

| 问题 | 一句话回答 | 决策/文档 |
|---|---|---|
| 为什么不直接在线调 engine 推理？ | batch 是**既定接入方式**：训练与全量推理代价高、按周期跑，serving 只消费预计算结果；实时服务化列为可选演进 | #1 #2 #12，`engine-integration.md`「演进方向」 |
| 平台里有模型代码吗？ | 没有。platform 不做推理、engine 不做在线服务，模型权重只留 engine | #3 #12 |
| 引擎没跑，首页会空吗？ | 不会。未命中缓存按 `avg_rating DESC, id DESC` 兜底**并写缓存**（实测因资源评分多为 0，效果是"新上架优先"）；文件空/导入失败则缓存不变，陈旧但可用 | #12，`engine-integration.md` |
| 推荐结果多久更新一次？ | 反映「截至导出时刻」的快照，用户新行为/新资源要等**下一轮完整刷新**才生效 | #12，`data-handoff.md` |
| 为什么资源只有一张表？ | 统一抽象 + JSON `metadata`，`type` 区分 course/article/video，新增内容源不用改表结构 | #24 #29 |
| 采集会不会覆盖人工编辑？ | 不会。判重键是 `source_url`，命中只刷新 `view_count`/`metadata`，标题简介分类保持平台侧的值（幂等可反复执行） | #29 #32 |
| 爬虫合规吗？ | 只采公开元数据与公开热门评论；不下载视频、不伪造设备指纹、不绕验证码；风控错误不重试；随机间隔 1.5–3s | #28 #30，`bilibili-import.md`「合规边界」 |
| 慕课网为什么不爬？ | 全站由腾讯 EdgeOne 接管，非浏览器客户端只拿到 JS 挑战页（curl 能过、Python 任何 UA 都过不去 ⇒ 拦在 TLS 指纹层），打通就要伪造指纹/解挑战，违反合规红线，故改为**离线数据集导入** | #31，`dataset-import.md` |
| 配置文件改了要重新编译吗？ | 不用。Viper + 环境变量覆盖（`CONFIG_PATH`），`datasets.<名称>` 的字段映射全在 YAML 里外置 | #15 #31 |
| 为什么管理后台要单独的表？ | 独立 `admins` 表（`id, user_id`），权限判断是「在不在表里」，不是用户表加 flag | #25 |
| 前端 mock 和后端怎么切换？ | 一个环境变量：`VITE_MOCK=1` 走 MSW，`0` 走 vite proxy → 真实后端 | #12，`vite.config.ts` |
| 错误处理统一吗？ | 统一响应体 `{code, message, data}` + HTTP 状态码 + 业务错误码 + Gin middleware 封装 | #6 #20 |
| 日志用什么？ | Go 标准库 `log/slog` 结构化日志，零依赖 | #19 |

---

## 6. 演示后复位（让下一次演示仍是同一套数据）

```powershell
$env:MYSQL_PWD='123456'

# 1) 删掉本场新建的演示分类（如果第 6.4 幕真点了创建）
mysql -h 127.0.0.1 -P 3308 -u root -e "DELETE FROM edurec.categories WHERE name='答辩演示分类';"

# 2) 让第 2 幕的「兜底」在下一场依然成立（最常需要做的一步）
#    只要有人用 demo_fresh 打开过首页，它就有缓存行了 → 下次不再走兜底。
mysql -h 127.0.0.1 -P 3308 -u root -e "DELETE FROM edurec.recommendations WHERE user_id = 2001;"
mysql -h 127.0.0.1 -P 3308 -u root -N -e "SELECT COUNT(*) FROM edurec.recommendations WHERE user_id = 2001;"
#    期望输出 0。等价做法：现场现场注册一个新账号（见第 2 幕 register 命令）。

# 3) 第 3 幕现场抓进来的 B 站资源：默认【留着】。
#    它们是真实内容，不影响下次演示，反而让库更丰富（唯一影响是某些关键词的本地命中数变大，
#    所以走查建议"连续演示就换一个没演过的关键词"，见第 3 幕）。
#    若确实想清掉本场新增的，按导入时间倒序删（没有 id 区间可依赖了）：
# mysql -h 127.0.0.1 -P 3308 -u root -e "DELETE FROM edurec.resources WHERE created_at > '2026-09-19 12:20:00';"
#    注意：会因外键失败（行为/评分引用），需要先删引用行；不确定就别删。

# 4) 若第 6.5 幕真跑了「导入 engine 推荐结果」，那 3 行缓存会被覆盖成 16 条。
#    恢复方式二选一：
#    a) 不必恢复 —— 首页照样有内容，只是个性化排序变弱；
#    b) 重新灌入"评分最高"的列表（等价于兜底结果），或直接清掉让兜底重建：
# mysql -h 127.0.0.1 -P 3308 -u root -e "DELETE FROM edurec.recommendations;"
#    （清空后每个用户首次访问都会走兜底并回写，首页始终非空）

# 5) ⛔ 不要用 demo_seed 来"恢复演示数据" —— 那会重新引入模拟资源（见 1.6 节）。
```

> 一句话记法：**要复位的只有一条 —— `demo_fresh` 的缓存行**（第 2 步）。
> 其余都无害；而**唯一会造成"页面出现模拟资源"的操作是 `demo_seed`，别跑它**。

---

## 附：本脚本引用的真实文件清单

| 文件 | 用途 |
|---|---|
| `backend/configs/config.local.yaml` | MySQL 3308 / Redis 6380+密码 / server 8080 / engine 三个路径 / bilibili 在线参数 / datasets 映射 |
| `frontend/.env.development.local` | `VITE_MOCK=0`（本机覆盖，走真实后端） |
| `frontend/vite.config.ts` | 5173 + `/api → http://localhost:8080` 代理 |
| `frontend/src/router/index.ts` | 路由表（`/`、`/search`、`/resources/:id`、`/user/me`、`/resources/upload`、`/admin/*`） |
| `frontend/src/router/guards.ts` | 未登录 → `/login`，非管理员访问 `/admin` → 首页 |
| `frontend/src/pages/home/index.vue` | 首页推荐（`limit=12`） |
| `frontend/src/pages/search/SearchPage.vue` | 无限滚动 + `online_page` 在线抓取触发条件 |
| `frontend/src/pages/resource/ResourceDetailPage.vue` | 元信息渲染、`view` 埋点、评分、B 站评论区 |
| `frontend/src/pages/user/ProfilePage.vue` + `components/profile/BehaviorHistory.vue` | 个人中心与行为历史 |
| `frontend/src/pages/admin/*.vue` | 仪表盘 / 用户 / 资源 / 分类 |
| `frontend/src/api/client.ts` | `CRAWL_TIMEOUT=35000`（比后端 30s 宽，避免"后端已落库、前端已超时"） |
| `backend/internal/router/router.go` | 全部接口路径清单（含 `/admin/recommendations/import`） |
| `backend/internal/service/recommendation.go` | 读缓存 + 热门兜底 + 回写缓存 |
| `backend/internal/service/recommendation_import.go` | 按库中真实存在的 user/resource 过滤后覆盖写缓存 |
| `backend/internal/service/resource.go` | `OnlinePage > 0` 走在线抓取；失败降级为空列表 |
| `backend/internal/service/bilibili_online.go` | `exec python online.py`、30s 超时、`has_more` 判定 |
| `backend/cmd/{server,demo_seed,export_snapshot,import_bilibili,import_dataset}` | 五个可执行入口 |
| `backend/data/{recommendations.json,sim/,bilibili/latest.json,mooc/,snapshots/}` | engine 产物、模拟数据集、采集产物、快照 |
| `scripts/handoff.sh` | 一键「导出 + 训练 + 推理 + 导入」 |
