package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/database"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

// defaultFilePath 相对 backend/ 目录，与 crawler/run.py 的输出目录一致
const defaultFilePath = "data/bilibili/latest.json"

func main() {
	filePath := flag.String("file", defaultFilePath, "爬虫输出的 JSON 文件路径")
	dryRun := flag.Bool("dry-run", false, "只解析并统计，不写库")
	flag.Parse()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	db, err := database.InitMySQL(cfg.Database)
	if err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}

	svc := service.NewBilibiliImportService(
		repository.NewResourceRepository(db),
		repository.NewCategoryRepository(db),
		*filePath,
	)

	var result *service.BilibiliImportResult
	if *dryRun {
		result, err = svc.Preview()
	} else {
		result, err = svc.Import()
	}
	if err != nil {
		log.Fatalf("导入失败: %v", err)
	}

	prefix := "[import_bilibili]"
	if *dryRun {
		fmt.Printf("%s --dry-run：未写库\n", prefix)
	}
	fmt.Printf("%s 文件：%s\n", prefix, *filePath)
	fmt.Printf("%s 新增资源 %d 条，刷新资源 %d 条，跳过 %d 条，新建分类 %d 个\n",
		prefix, result.CreatedResources, result.UpdatedResources,
		result.SkippedResources, result.CreatedCategories)
}
