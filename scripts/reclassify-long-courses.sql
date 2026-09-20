-- =============================================================================
-- 把「长合集 / 系统课程」类 B 站内容归入 course 类型
--
-- 背景：B 站采集统一按 type=video 落库（BilibiliImportOptions.ResourceType），
-- 因为搜索结果不区分单集与系列课程，导致 150 小时的全套课程与十几分钟的单集
-- 类型相同、无法按类型筛选。
--
-- ⚠️ 自 2026-09-19 起，「长合集→course」已由**导入时自动判定**（每次新内容都会判一次）：
--   规则见 config.go 的 ContentRulesConfig.IsLongCourse，口径在 config.yaml 的
--   content_rules 段可调。本脚本只用于**首次数据迁移**与**口径调整后重新对齐历史数据**。
--
-- 判定口径（与代码一致，两档都与时长同时成立）：
--   强标记：标题含 课程 / 精讲 / 合集 / 全集 / 全套 / 系列 / 全N集 / 共N讲
--           且时长 >= 120 分钟
--   弱关键词：标题含 教程 / 讲解 且时长 >= 600 分钟（10 小时）
--   —— 弱词门槛更高，因为「教程」在 B 站短片标题里极常见。
--
-- ⚠️ duration 是「分钟:秒」，不是「时:分」！
--   采集器 collect.py 的 _format_duration 把秒数格式化成 f"{seconds//60}:{seconds%60:02d}"，
--   所以 9003:20 表示 9003 分钟。因此取第 1 段即为分钟数。
--
-- 用法：
--   mysql -h 127.0.0.1 -P 3308 -u root -p edurec < scripts/reclassify-long-courses.sql
--
-- 可反复执行（幂等：只改 type）；采集重复导入不会把它改回 video ——
-- 判重命中时只刷新 view_count 与 metadata，不覆盖 type（见 service/crawl_import.go）。
-- =============================================================================

-- ① 先看会改哪些行（建议先跑这一步确认）
SELECT id,
       CAST(SUBSTRING_INDEX(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.duration')), ':', 1) AS UNSIGNED) AS minutes,
       CASE WHEN title REGEXP '教程|讲解' AND title NOT REGEXP '课程|精讲|合集|全集|全套|系列|(全|共)[0-9]+[集讲]'
            THEN '弱关键词档' ELSE '强标记档' END AS matched_by,
       LEFT(title, 50) AS title
FROM resources
WHERE (
        (title REGEXP '课程|精讲|合集|全集|全套|系列|(全|共)[0-9]+[集讲]'
         AND CAST(SUBSTRING_INDEX(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.duration')), ':', 1) AS UNSIGNED) >= 120)
     OR (title REGEXP '教程|讲解'
         AND CAST(SUBSTRING_INDEX(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.duration')), ':', 1) AS UNSIGNED) >= 600)
      )
ORDER BY minutes DESC;

-- ② 执行改判
UPDATE resources
SET type = 'course'
WHERE (
        (title REGEXP '课程|精讲|合集|全集|全套|系列|(全|共)[0-9]+[集讲]'
         AND CAST(SUBSTRING_INDEX(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.duration')), ':', 1) AS UNSIGNED) >= 120)
     OR (title REGEXP '教程|讲解'
         AND CAST(SUBSTRING_INDEX(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.duration')), ':', 1) AS UNSIGNED) >= 600)
      );

-- ③ 核对结果（2026-09-19 该口径下为 course 63 / video 206）
SELECT type, COUNT(*) AS n FROM resources GROUP BY type;

-- 回滚：把不符合上述口径的 course 行改回 video
-- UPDATE resources
-- SET type = 'video'
-- WHERE type = 'course'
--   AND NOT (
--         (title REGEXP '课程|精讲|合集|全集|全套|系列|(全|共)[0-9]+[集讲]'
--          AND CAST(SUBSTRING_INDEX(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.duration')), ':', 1) AS UNSIGNED) >= 120)
--      OR (title REGEXP '教程|讲解'
--          AND CAST(SUBSTRING_INDEX(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.duration')), ':', 1) AS UNSIGNED) >= 600)
--       );
