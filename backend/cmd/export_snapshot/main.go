package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/database"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

const contractVersion = 1

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Engine.SnapshotDir == "" {
		log.Fatal("未配置 engine.snapshot_dir")
	}
	db, err := database.InitMySQL(cfg.Database)
	if err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}

	runID := time.Now().Format("20060102_150405")
	dir := filepath.Join(cfg.Engine.SnapshotDir, runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("创建快照目录失败: %v", err)
	}

	// users：只导出 ID，不含个人字段
	var users []model.User
	db.Order("id").Find(&users)
	userRows := make([][]string, 0, len(users))
	for _, u := range users {
		userRows = append(userRows, []string{strconv.FormatUint(uint64(u.ID), 10)})
	}
	writeCSV(filepath.Join(dir, "users.csv"), []string{"user_id"}, userRows)

	// resources
	var res []model.Resource
	db.Order("id").Find(&res)
	resRows := make([][]string, 0, len(res))
	for _, r := range res {
		tags := strings.TrimSpace(r.Tags)
		if tags == "" {
			tags = "[]"
		}
		resRows = append(resRows, []string{
			strconv.FormatUint(uint64(r.ID), 10),
			r.Title,
			r.Type,
			strconv.FormatUint(uint64(r.CategoryID), 10),
			tags,
			r.Metadata,
			strconv.FormatFloat(float64(r.AvgRating), 'f', 2, 32),
			strconv.FormatUint(uint64(r.ViewCount), 10),
			strconv.FormatInt(r.CreatedAt.Unix(), 10),
		})
	}
	writeCSV(filepath.Join(dir, "resources.csv"),
		[]string{"resource_id", "title", "type", "category_id", "tags_json",
			"metadata_json", "avg_rating", "view_count", "created_at"}, resRows)

	// categories
	var cats []model.Category
	db.Order("id").Find(&cats)
	catRows := make([][]string, 0, len(cats))
	for _, c := range cats {
		catRows = append(catRows, []string{strconv.FormatUint(uint64(c.ID), 10), c.Name})
	}
	writeCSV(filepath.Join(dir, "categories.csv"), []string{"category_id", "name"}, catRows)

	// behaviors / ratings：分页流式
	var maxTs int64
	behRows := [][]string{}
	var behBatch []model.UserBehavior
	err = db.Model(&model.UserBehavior{}).Order("id").FindInBatches(&behBatch, 5000,
		func(_ *gorm.DB, _ int) error {
			for _, b := range behBatch {
				if t := b.CreatedAt.Unix(); t > maxTs {
					maxTs = t
				}
				behRows = append(behRows, []string{
					strconv.FormatUint(uint64(b.UserID), 10),
					strconv.FormatUint(uint64(b.ResourceID), 10),
					b.Action,
					strconv.FormatInt(b.CreatedAt.Unix(), 10),
				})
			}
			return nil
		}).Error
	if err != nil {
		log.Fatalf("导出 behaviors 失败: %v", err)
	}
	writeCSV(filepath.Join(dir, "behaviors.csv"),
		[]string{"user_id", "resource_id", "action", "ts"}, behRows)

	ratRows := [][]string{}
	var ratBatch []model.Rating
	err = db.Model(&model.Rating{}).Order("id").FindInBatches(&ratBatch, 5000,
		func(_ *gorm.DB, _ int) error {
			for _, r := range ratBatch {
				if t := r.CreatedAt.Unix(); t > maxTs {
					maxTs = t
				}
				ratRows = append(ratRows, []string{
					strconv.FormatUint(uint64(r.UserID), 10),
					strconv.FormatUint(uint64(r.ResourceID), 10),
					strconv.FormatUint(uint64(r.Score), 10),
					strconv.FormatInt(r.CreatedAt.Unix(), 10),
				})
			}
			return nil
		}).Error
	if err != nil {
		log.Fatalf("导出 ratings 失败: %v", err)
	}
	writeCSV(filepath.Join(dir, "ratings.csv"),
		[]string{"user_id", "resource_id", "score", "ts"}, ratRows)

	// meta + sha256
	files := []string{"users.csv", "resources.csv", "categories.csv",
		"behaviors.csv", "ratings.csv"}
	sha := map[string]string{}
	for _, f := range files {
		sum, err := fileSHA(filepath.Join(dir, f))
		if err != nil {
			log.Fatalf("计算 %s sha256 失败: %v", f, err)
		}
		sha[f] = sum
	}
	meta := map[string]any{
		"contract_version": contractVersion,
		"run_id":           runID,
		"exported_at":      time.Now().Unix(),
		"behavior_window":  map[string]int64{"start": 0, "end": maxTs},
		"users_count":      len(users), "resources_count": len(res),
		"behaviors_count": len(behRows), "ratings_count": len(ratRows),
		"categories_count": len(cats),
		"files_sha256":     sha,
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), metaJSON, 0o644); err != nil {
		log.Fatalf("写 meta.json 失败: %v", err)
	}
	fmt.Printf("[export_snapshot] 快照导出完成 -> %s\n", dir)
}

func writeCSV(path string, header []string, rows [][]string) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("创建 %s 失败: %v", path, err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write(header); err != nil {
		log.Fatalf("写 %s 失败: %v", path, err)
	}
	if err := w.WriteAll(rows); err != nil {
		log.Fatalf("写 %s 失败: %v", path, err)
	}
}

func fileSHA(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
