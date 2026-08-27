# edurec-engine 接入说明

## 背景

edurec-engine 是独立仓库（`github.com/Shionyori/edurec-engine`），离线完成「双塔 DSSM 召回 → 多任务 DeepFM 精排 → MMR 重排」，批量推理后输出 `model/recommendations.json`。platform 与 engine 按「独立微服务 + REST」为远期设计，当前先以**路线 B（batch + 落库）**接入，engine 仓库零改动。

## 路线 B：batch + 落库（已实现）

### 数据流

```
edurec-engine 离线推理 → model/recommendations.json
   ↓  POST /api/v1/admin/recommendations/import（管理员触发）
Recommendation 缓存表（每个用户一条）
   ↓  GET /api/v1/recommendations
前端首页展示个性化推荐（缓存未命中时兜底按评分降序取热门）
```

### 配置文件

`backend/configs/config.yaml`：

```yaml
engine:
  recommendations_file: "/mnt/d/Project/edurec-engine/model/recommendations.json"
```

### 导入接口

`POST /api/v1/admin/recommendations/import`（需管理员），无参数。

- 读取 `engine.recommendations_file` 指向的 JSON，格式为 `{ "<user_id>": [<resource_id>, ...] }`（每用户 top-N）
- 一次性查询平台库用户与资源存在性，仅导入**数值 ID 相同**的用户与资源
- 写入/覆盖该用户的 `Recommendation` 缓存行，返回统计：`imported_users / skipped_users / imported_resources / skipped_resources`

### ID 映射限制（重要）

engine 基于其模拟数据集（`dataset/sim`，1965 用户、494 资源，ID 从 0 开始）训练，推荐中的 ID 是**模拟数据的内部编号**，与 platform 数据库的真实 ID（MySQL auto-increment）**不是同一套**。因此：

- 导入时只匹配数值 ID 相同的用户与资源，其余跳过（安全但不产生真实个性化效果）
- 当前 engine（sim 数据）导入到真实平台时，匹配量可能很少

**要让推荐真正个性化，需要 engine 改为消费 platform 真实数据**（用户、行为、资源）并输出平台侧 ID——这是 engine 侧的数据管线工作，超出 platform 收尾范围，留作后续。

### 触发方式

- 当前为管理员手动调用导入接口
- 后续可扩展为定时任务 / 发布流程中自动触发

## 路线 A 展望（后续）

远期按设计文档回到「独立微服务 + REST」：

1. engine 增加 HTTP serving 层（如 FastAPI），加载训练好的模型，暴露推荐接口
2. platform 后端定义 engine client，`GET /recommendations` 缓存未命中时实时调用 engine
3. 引擎侧接入 platform 真实数据管线，使推荐基于真实用户行为

## 相关文档

- [design.md](./design.md) —— 决策记录 #12
- [api-design.md](./api-design.md) —— 6.1 推荐接口、6.2 导入接口
- [README.md](../README.md) —— 推荐数据流速览
