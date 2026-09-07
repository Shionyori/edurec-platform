package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/database"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// demo_seed：把 edurec-engine 的模拟数据集（dataset/sim）播种进平台库，
// 用户/资源/类目 ID 与 engine 推荐产物（recommendations.json）保持一致，
// 使导入后首页个性化推荐可完整演示。
//
// 用法（在 backend 目录下）：
//
//	CONFIG_PATH=configs/config.yaml go run ./cmd/demo_seed -with-behaviors
//
// 播种账号：demo<id>（如 demo1）/ demo123456；另创建管理员 demo_admin / demo123456。
func main() {
	withBehaviors := flag.Bool("with-behaviors", false, "同时播种 behaviors/ratings（默认仅用户/资源/类目）")
	admin := flag.Bool("admin", true, "确保存在管理员 demo_admin")
	password := flag.String("password", "demo123456", "演示账号统一密码")
	flag.Parse()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Engine.DatasetDir == "" {
		log.Fatal("未配置 engine.dataset_dir（engine 的 dataset/sim 目录）")
	}
	db, err := database.InitMySQL(cfg.Database)
	if err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}
	// 允许显式插入 id=0（MySQL 默认会为 0 自动分配自增值）
	db.Exec("SET SESSION sql_mode = CONCAT(@@SESSION.sql_mode, ',NO_AUTO_VALUE_ON_ZERO')")

	dir := cfg.Engine.DatasetDir
	resRows := readCSV(filepath.Join(dir, "resources.csv"))
	userRows := readCSV(filepath.Join(dir, "users.csv"))
	if len(resRows) == 0 || len(userRows) == 0 {
		log.Fatalf("engine 数据集为空（%s），请先运行 engine 的 gen_sim_data", dir)
	}

	now := time.Now()

	// 类目：resources.csv 中出现过的 category_id
	catSeen := map[string]bool{}
	catRows := [][]any{}
	for _, r := range resRows {
		cid := r["category_id"]
		if !catSeen[cid] {
			catSeen[cid] = true
			catRows = append(catRows, []any{mustInt(cid), "模拟类别" + cid})
		}
	}
	nCats := insertIgnore(db, "categories", []string{"id", "name"}, catRows, 200)

	// 资源（保留原始 ID）
	resOut := make([][]any, 0, len(resRows))
	for _, r := range resRows {
		// engine 的 sim 数据集多标签以 "|" 分隔（与 io.save_bundle 一致）
		tags := strings.Split(strings.TrimSpace(r["tags"]), "|")
		tagJSON, _ := json.Marshal(tags)
		resOut = append(resOut, []any{
			mustInt(r["resource_id"]), "模拟资源" + r["resource_id"], r["type"],
			mustInt(r["category_id"]), string(tagJSON), "{}", 0, 0, now,
		})
	}
	nRes := insertIgnore(db, "resources",
		[]string{"id", "title", "type", "category_id", "tags", "metadata",
			"avg_rating", "view_count", "created_at"}, resOut, 200)

	// 用户（保留原始 ID，统一密码）
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("生成密码哈希失败: %v", err)
	}
	usrOut := make([][]any, 0, len(userRows))
	for _, u := range userRows {
		uid := u["user_id"]
		usrOut = append(usrOut, []any{
			mustInt(uid), "demo" + uid, "demo" + uid + "@sim.local", string(hash),
			"演示用户" + uid, now,
		})
	}
	nUsers := insertIgnore(db, "users",
		[]string{"id", "username", "email", "password_hash", "display_name", "created_at"},
		usrOut, 300)

	nBeh, nRat := 0, 0
	if *withBehaviors {
		behOut := make([][]any, 0)
		for _, b := range readCSV(filepath.Join(dir, "behaviors.csv")) {
			behOut = append(behOut, []any{
				mustInt(b["user_id"]), mustInt(b["resource_id"]), b["action"],
				mustInt(b["ts"]),
			})
		}
		nBeh = insertIgnore(db, "user_behaviors",
			[]string{"user_id", "resource_id", "action", "created_at"},
			behOut, 2000, "FROM_UNIXTIME")
		ratOut := make([][]any, 0)
		for _, r := range readCSV(filepath.Join(dir, "ratings.csv")) {
			ratOut = append(ratOut, []any{
				mustInt(r["user_id"]), mustInt(r["resource_id"]), mustInt(r["score"]),
				mustInt(r["ts"]),
			})
		}
		nRat = insertIgnore(db, "ratings",
			[]string{"user_id", "resource_id", "score", "created_at"},
			ratOut, 2000, "FROM_UNIXTIME")
	}

	nAdmin := 0
	if *admin {
		nAdmin = ensureAdmin(db, string(hash))
	}

	fmt.Printf("[demo_seed] 类目=%d 资源=%d 用户=%d 行为=%d 评分=%d 管理员=%d\n",
		nCats, nRes, nUsers, nBeh, nRat, nAdmin)
	fmt.Printf("[demo_seed] 演示账号: demo<id> / %s（如 demo1）；管理员: demo_admin / %s\n",
		*password, *password)
}

// insertIgnore 分批 INSERT IGNORE；tsMode=FROM_UNIXTIME 时最后一列时间参数
// 用 FROM_UNIXTIME(?) 包裹。返回受影响行数（含忽略）。
func insertIgnore(db *gorm.DB, table string, cols []string, rows [][]any,
	chunk int, tsMode ...string) int {
	if len(rows) == 0 {
		return 0
	}
	useUnix := len(tsMode) > 0 && tsMode[0] == "FROM_UNIXTIME"
	colStr := "`" + strings.Join(cols, "`,`") + "`"
	var n int
	for start := 0; start < len(rows); start += chunk {
		end := start + chunk
		if end > len(rows) {
			end = len(rows)
		}
		part := rows[start:end]
		args := make([]any, 0, len(part)*len(cols))
		valueStrs := make([]string, 0, len(part))
		for _, row := range part {
			marks := make([]string, len(cols))
			for ci, v := range row {
				if useUnix && ci == len(cols)-1 {
					marks[ci] = "FROM_UNIXTIME(?)"
				} else {
					marks[ci] = "?"
				}
				args = append(args, v)
			}
			valueStrs = append(valueStrs, "("+strings.Join(marks, ",")+")")
		}
		sql := fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES %s",
			table, colStr, strings.Join(valueStrs, ","))
		if err := db.Exec(sql, args...).Error; err != nil {
			log.Fatalf("写入 %s 失败: %v", table, err)
		}
		n += len(part)
	}
	return n
}

func ensureAdmin(db *gorm.DB, passwordHash string) int {
	var id uint
	err := db.Raw("SELECT id FROM users WHERE username = ? LIMIT 1", "demo_admin").Scan(&id).Error
	if err != nil || id == 0 {
		res := db.Exec(
			"INSERT IGNORE INTO users (username, email, password_hash, display_name, created_at) VALUES (?, ?, ?, ?, ?)",
			"demo_admin", "demo_admin@sim.local", passwordHash, "演示管理员", time.Now())
		if res.Error != nil {
			log.Fatalf("创建 demo_admin 失败: %v", res.Error)
		}
		var id2 uint
		if err := db.Raw("SELECT id FROM users WHERE username = ? LIMIT 1", "demo_admin").Scan(&id2).Error; err != nil {
			log.Fatalf("查询 demo_admin 失败: %v", err)
		}
		id = id2
	}
	res := db.Exec("INSERT IGNORE INTO admins (user_id) VALUES (?)", id)
	if res.Error != nil {
		log.Fatalf("设置管理员失败: %v", res.Error)
	}
	return int(id)
}

func readCSV(path string) []map[string]string {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("打开 %s 失败: %v", path, err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		log.Fatalf("解析 %s 失败: %v", path, err)
	}
	if len(rows) == 0 {
		return nil
	}
	header := rows[0]
	out := make([]map[string]string, 0, len(rows)-1)
	for _, r := range rows[1:] {
		m := map[string]string{}
		for i, h := range header {
			if i < len(r) {
				m[h] = r[i]
			}
		}
		out = append(out, m)
	}
	return out
}

func mustInt(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("非法整数 %q: %v", s, err)
	}
	return v
}
