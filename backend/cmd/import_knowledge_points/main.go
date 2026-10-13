package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/database"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

const prefix = "[import_knowledge_points]"

// defaultPreviewLimit --preview 未指定 --limit 时展示的知识点条数。
// 预览是用来核对聚合质量的，看几十条就够，全量刷屏反而找不着要点。
const defaultPreviewLimit = 30

func main() {
	fromTags := flag.Bool("from-tags", false, "从 resources.tags 聚合知识点（本轮的唯一来源）")
	filePath := flag.String("file", "", "从 JSON 文件导入知识点（手工/LLM 产出，格式见 -h 与 spec §8.3）")
	limit := flag.Int("limit", 0, "最多处理多少个知识点；--preview 未指定时默认 30，导入时不限")
	preview := flag.Bool("preview", false, "只打印将要导入的知识点，不连数据库（优先于 --dry-run）")
	dryRun := flag.Bool("dry-run", false, "连库走完整聚合，但不写库")
	flag.Parse()

	if !*fromTags && *filePath == "" {
		flag.Usage()
		log.Fatalf("%s 必须指定 --from-tags 或 --file 之一", prefix)
	}
	if *fromTags && *filePath != "" {
		log.Fatalf("%s --from-tags 与 --file 互斥，只能选一个", prefix)
	}

	limitSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "limit" {
			limitSet = true
		}
	})
	effectiveLimit := *limit
	if !limitSet && *preview {
		effectiveLimit = defaultPreviewLimit
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("%s 加载配置失败: %v", prefix, err)
	}

	if *preview {
		// 注意：--from-tags 必须读 resources 表才能聚合，所以 --preview「只读库、不写库」，
		// 与 import_dataset 的 --preview（解析本地文件、完全不连库）语义不同。
		fmt.Printf("%s --preview：只读数据库、不写库\n", prefix)
	}

	db, err := database.InitMySQL(cfg.Database)
	if err != nil {
		log.Fatalf("%s MySQL 初始化失败: %v", prefix, err)
	}

	importer := service.NewKnowledgePointImportService(
		repository.NewResourceRepository(db),
		repository.NewKnowledgePointRepository(db),
	)

	if *filePath != "" {
		log.Fatalf("%s --file 导入本轮尚未实现，请用 --from-tags", prefix)
	}

	result, candidates, err := importer.ImportFromTags(cfg.KnowledgePoints, !*dryRun && !*preview)
	if err != nil {
		log.Fatalf("%s 聚合失败: %v", prefix, err)
	}

	printCandidates(candidates, effectiveLimit)

	if *dryRun {
		fmt.Printf("%s --dry-run：未写库\n", prefix)
	}
	fmt.Printf("%s 聚合出 %d 个候选知识点，新增 %d，更新 %d\n",
		prefix, result.Candidates, result.Created, result.Updated)
	if result.Candidates == 0 {
		fmt.Printf("%s 提示：一个知识点都没聚合出来，检查 resources 表是否有数据、"+
			"以及 min_resource_count 是否过高（当前 %d）\n",
			prefix, cfg.KnowledgePoints.KnowledgePointsOrDefault().MinResourceCount)
	}
}

// printCandidates 打印候选知识点，供人工检视聚合质量（spec §8.4 的阶段 0 闸门）
func printCandidates(candidates []service.TagAggregate, limit int) {
	shown := candidates
	if limit > 0 && len(shown) > limit {
		shown = shown[:limit]
	}
	for i, c := range shown {
		encoded, err := json.Marshal(c)
		if err != nil {
			log.Fatalf("%s 序列化第 %d 个候选失败: %v", prefix, i+1, err)
		}
		fmt.Printf("%s [%d/%d] %s\n", prefix, i+1, len(shown), encoded)
	}
	if len(shown) < len(candidates) {
		fmt.Printf("%s 上面只展示前 %d 个（--limit 可改，共 %d 个）\n",
			prefix, len(shown), len(candidates))
	}
}
