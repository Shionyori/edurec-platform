package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/database"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

// generate_concepts：批量/单个重建学科概念（调用 engine 的 kg_server.py）。
//
// 生成能力在 engine 侧，本命令只是「遍历学科 -> 调 engine -> 落库」的编排器，
// 不直接执行 Python。engine 服务需先启动：
//
//	cd EMBEDD-ER && .venv\Scripts\python.exe kg_server.py --port 8090
//
// 用法（在 backend 目录下）：
//
//	CONFIG_PATH=configs/config.local.yaml go run ./cmd/generate_concepts -all
//	CONFIG_PATH=configs/config.local.yaml go run ./cmd/generate_concepts -code mathematical-analysis
//
// 已有概念按名称去重：MOOCCube 的概念只更新 pageRank，生成的新概念标记 source=generated。
func main() {
	all := flag.Bool("all", false, "为全部学科重新生成概念")
	code := flag.String("code", "", "只生成指定学科（code）")
	url := flag.String("url", "", "覆盖 engine 生成服务地址")
	timeout := flag.Int("timeout", 0, "单次请求超时秒数（默认取配置）")
	flag.Parse()

	if !*all && *code == "" {
		log.Fatal("请用 -all 或 -code 指定要生成的学科")
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	baseURL := cfg.Knowledge.GeneratorURL
	if *url != "" {
		baseURL = *url
	}
	timeoutSeconds := cfg.Knowledge.GeneratorTimeoutSeconds
	if *timeout > 0 {
		timeoutSeconds = *timeout
	}

	db, err := database.InitMySQL(cfg.Database)
	if err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}
	knowledgeRepo := repository.NewKnowledgeRepository(db)
	runner := service.NewHTTPConceptRunner(service.HTTPConceptRunnerConfig{
		BaseURL:          baseURL,
		Timeout:          time.Duration(timeoutSeconds) * time.Second,
		Token:            cfg.Knowledge.GeneratorToken,
		MaxConcurrency:   cfg.Knowledge.GeneratorMaxConcurrency,
		FailureThreshold: cfg.Knowledge.GeneratorFailureThreshold,
		CircuitCooldown:  time.Duration(cfg.Knowledge.GeneratorCircuitCooldownSeconds) * time.Second,
	})
	generator := service.NewConceptGenerator(knowledgeRepo, runner, time.Hour)

	codes := []string{*code}
	if *all {
		points, err := knowledgeRepo.ListPoints()
		if err != nil {
			log.Fatalf("读取知识点失败: %v", err)
		}
		codes = codes[:0]
		for i := range points {
			codes = append(codes, points[i].Code)
		}
		fmt.Printf("[generate_concepts] 共 %d 个学科，engine=%s\n", len(codes), baseURL)
	}

	failed := 0
	for index, pointCode := range codes {
		status, err := generator.GenerateNow(context.Background(), pointCode)
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "[%d/%d] %s 失败: %v\n", index+1, len(codes), pointCode, err)
			continue
		}
		fmt.Printf("[%d/%d] %-24s 生成 %d 个概念，累计 %d\n",
			index+1, len(codes), pointCode, status.Generated, status.Total)
	}

	if failed > 0 {
		log.Fatalf("完成，但有 %d 个学科生成失败", failed)
	}
	fmt.Printf("[generate_concepts] 全部完成（%d 个学科）\n", len(codes))
}
