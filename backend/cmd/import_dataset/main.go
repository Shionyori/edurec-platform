package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/database"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

const prefix = "[import_dataset]"

// defaultPreviewLimit --preview 未指定 --limit 时展示的条数。
// 预览是用来核对字段映射的，看几条就够，全量刷屏反而找不着要点。
const defaultPreviewLimit = 5

func main() {
	name := flag.String("dataset", "", "datasets.<name> 的配置名（必填）")
	filePath := flag.String("file", "", "覆盖配置里的 file，便于拿同一套映射试不同文件")
	format := flag.String("format", "", "覆盖配置里的 format：auto | json | jsonl | csv")
	limit := flag.Int("limit", 0, "最多处理多少条；--preview 未指定时默认 5 条，导入时不限")
	preview := flag.Bool("preview", false, "只解析并打印映射结果，不连数据库（优先于 --dry-run）")
	dryRun := flag.Bool("dry-run", false, "连库走完整判重与分类逻辑，但不写库")
	flag.Parse()

	if *name == "" {
		flag.Usage()
		log.Fatalf("%s 必须用 --dataset 指定数据集名，如 --dataset mooc", prefix)
	}

	// --limit 是否显式给出，决定 --preview 用默认 5 条还是照常全量
	limitSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "limit" {
			limitSet = true
		}
	})

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("%s 加载配置失败: %v", prefix, err)
	}

	dataset, ok := cfg.Datasets[*name]
	if !ok {
		log.Fatalf("%s 配置里没有 datasets.%s，可用的有：%v", prefix, *name, datasetNames(cfg))
	}
	// 只覆盖这两项：字段映射、兜底分类、落库类型仍以 YAML 为准，否则同一次导入
	// 究竟用了哪套映射就说不清了
	if *filePath != "" {
		dataset.File = *filePath
	}
	if *format != "" {
		dataset.Format = *format
	}
	// 数据集名已知，此时才校验：报错信息里带上名字才查得动 YAML
	// （config.Load 也校验过全部数据集，这里是为了让 --file 覆盖后的路径也受检）
	if err := dataset.Validate(*name); err != nil {
		log.Fatalf("%s %v", prefix, err)
	}

	effectiveLimit := *limit
	if !limitSet && *preview {
		effectiveLimit = defaultPreviewLimit
	}

	loaded, err := service.LoadDataset(dataset)
	if err != nil {
		log.Fatalf("%s %v", prefix, err)
	}
	items := loaded.Items
	if effectiveLimit > 0 && len(items) > effectiveLimit {
		items = items[:effectiveLimit]
	}

	if *preview {
		printPreview(*name, dataset, loaded, items, effectiveLimit)
		return
	}

	db, err := database.InitMySQL(cfg.Database)
	if err != nil {
		log.Fatalf("%s MySQL 初始化失败: %v", prefix, err)
	}

	// filePath 传空：items 已在上面解析好，不经过爬虫交接文件
	// 白名单与内容归类规则都传空/零值：数据集条目没有 typename，
	// 分区白名单与「长合集→course」判定都只作用于 B 站条目，对数据集不生效。
	importer := service.NewCrawlImportService(
		repository.NewResourceRepository(db),
		repository.NewCategoryRepository(db),
		"",
		nil,
		config.ContentRulesConfig{},
	)
	// 不给 SourceURLTemplate：source_url 在 LoadDataset 阶段就已确定，取不到的行
	// 已被跳过。这里再兜底反而会拼出错误的判重键，让同一门课反复建行。
	result, _, err := importer.ImportItems(items, service.CrawlImportOptions{
		ResourceType: dataset.ResourceType,
	}, !*dryRun)
	if err != nil {
		log.Fatalf("%s 导入失败: %v", prefix, err)
	}

	if *dryRun {
		fmt.Printf("%s --dry-run：未写库\n", prefix)
	}
	fmt.Printf("%s 数据集：%s（%s，%s）\n", prefix,
		dataset.File, dataset.FormatOrDefault(), dataset.ResourceType)
	if effectiveLimit > 0 && len(loaded.Items) > effectiveLimit {
		fmt.Printf("%s 按 --limit 只处理了前 %d 条（共映射出 %d 条）\n",
			prefix, effectiveLimit, len(loaded.Items))
	}
	// 解析与落库分两阶段报数：两阶段都有「跳过」，不分开标注会让人以为数目对不上
	fmt.Printf("%s 解析：共 %d 条，映射成功 %d 条，跳过 %d 条\n",
		prefix, loaded.Total, len(loaded.Items), loaded.Skipped())
	fmt.Printf("%s 落库：新增资源 %d 条，刷新资源 %d 条，跳过 %d 条，新建分类 %d 个\n",
		prefix, result.CreatedResources, result.UpdatedResources,
		result.SkippedResources, result.CreatedCategories)
	printSkipReasons("解析阶段跳过原因：", loaded.SkipReasons)
}

// printPreview 不连数据库地展示映射结果，用于拿到陌生数据集时先把 YAML 调对
func printPreview(
	name string,
	dataset config.DatasetConfig,
	loaded *service.DatasetLoadResult,
	items []service.CrawlItem,
	effectiveLimit int,
) {
	fmt.Printf("%s --preview：未连数据库，未写库\n", prefix)
	fmt.Printf("%s datasets.%s → %s（%s，落库类型 %s，兜底分类 %q）\n", prefix,
		name, dataset.File, dataset.FormatOrDefault(), dataset.ResourceType, dataset.DefaultCategory)
	fmt.Printf("%s 共 %d 条，映射成功 %d 条，跳过 %d 条",
		prefix, loaded.Total, len(loaded.Items), loaded.Skipped())
	if len(items) < len(loaded.Items) {
		fmt.Printf("；下面只展示前 %d 条（--limit 可改）", len(items))
	}
	fmt.Println()

	for i, item := range items {
		encoded, err := json.MarshalIndent(item, "", "  ")
		if err != nil {
			log.Fatalf("%s 序列化第 %d 条失败: %v", prefix, i+1, err)
		}
		fmt.Printf("\n%s --- [%d/%d] ---\n%s\n", prefix, i+1, len(items), encoded)
	}
	if len(items) > 0 {
		fmt.Println()
	}

	if loaded.Total == 0 {
		fmt.Printf("%s 提示：数据集里没有记录，检查 items_path 是否指到了正确的数组（当前 %q）\n",
			prefix, dataset.ItemsPath)
	} else if len(loaded.Items) == 0 {
		// 一条都没映射出来通常不是「数据太脏」，而是映射压根没对上
		fmt.Printf("%s 提示：%d 条全被跳过，多半是 fields 没对上——"+
			"先看下面的跳过原因，再对照数据集里的实际字段名改 YAML\n", prefix, loaded.Total)
	}
	printSkipReasons("跳过原因：", loaded.SkipReasons)
}

// printSkipReasons 按影响面从大到小打印跳过原因，调映射时优先解决排第一的那条
func printSkipReasons(label string, reasons map[string]int) {
	if len(reasons) == 0 {
		return
	}
	fmt.Printf("%s %s\n", prefix, label)
	for _, entry := range sortedSkipReasons(reasons) {
		fmt.Printf("%s   %d 条  %s\n", prefix, entry.count, entry.reason)
	}
}

// reasonCount 跳过原因及其条数
type reasonCount struct {
	reason string
	count  int
}

// sortedSkipReasons 按条数倒序排列；条数相同时按文案排序，保证多次运行输出一致
func sortedSkipReasons(reasons map[string]int) []reasonCount {
	counts := make([]reasonCount, 0, len(reasons))
	for reason, count := range reasons {
		counts = append(counts, reasonCount{reason: reason, count: count})
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].count != counts[j].count {
			return counts[i].count > counts[j].count
		}
		return counts[i].reason < counts[j].reason
	})
	return counts
}

// datasetNames 列出已配置的数据集名，便于 --dataset 写错时直接看到可选项
func datasetNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Datasets))
	for name := range cfg.Datasets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
