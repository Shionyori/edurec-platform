// evaluate_recommendations 对平台缓存的推荐结果做离线评估，并与「热门基线」对比。
//
// 目的：让「推荐到底有没有用」在平台侧可度量（engine 侧的离线指标是另一条独立证据）。
//
//	按用户时间序留出最近的交互作为测试集（默认后 20%，至少 1 条）
//	推荐列表 = recommendations 表里该用户的缓存（严格按缓存顺序）
//	热门基线 = 训练集里被交互次数最多的资源排序
//	指标：Recall@K / HitRate@K / NDCG@K（K = 5/10/20）
//
// 用法（在 backend/ 下）：
//
//	CONFIG_PATH=configs/config.yaml go run ./cmd/evaluate_recommendations
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/database"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// recallAtK 命中数占相关集的比例
func recallAtK(relevant map[uint]struct{}, ranked []uint, k int) float64 {
	if len(relevant) == 0 {
		return 0
	}
	hit := 0
	for i, id := range ranked {
		if i >= k {
			break
		}
		if _, ok := relevant[id]; ok {
			hit++
		}
	}
	return float64(hit) / float64(len(relevant))
}

// hitRateAtK 前 K 个里至少有一个相关即为 1
func hitRateAtK(relevant map[uint]struct{}, ranked []uint, k int) float64 {
	for i, id := range ranked {
		if i >= k {
			break
		}
		if _, ok := relevant[id]; ok {
			return 1
		}
	}
	return 0
}

// ndcgAtK 位置加权的排序质量
func ndcgAtK(relevant map[uint]struct{}, ranked []uint, k int) float64 {
	dcg := 0.0
	for i, id := range ranked {
		if i >= k {
			break
		}
		if _, ok := relevant[id]; ok {
			dcg += 1 / math.Log2(float64(i)+2)
		}
	}
	n := len(relevant)
	if n > k {
		n = k
	}
	ideal := 0.0
	for i := 0; i < n; i++ {
		ideal += 1 / math.Log2(float64(i)+2)
	}
	if ideal == 0 {
		return 0
	}
	return dcg / ideal
}

func main() {
	minBehaviors := flag.Int("min-behaviors", 5, "参与评估所需的最少交互数")
	testRatio := flag.Float64("test-ratio", 0.2, "留作测试集的交互比例")
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

	train, test := splitBehaviors(db, *minBehaviors, *testRatio)
	recByUser := loadRecommendations(db)
	popularityRank := popularityBaseline(train)

	ks := []int{5, 10, 20}
	report := evaluate(test, recByUser, popularityRank, ks)

	printReport(report, len(train), len(test))
}

// splitBehaviors 按用户时间序把交互切成 训练 / 测试。
// 测试集取每个用户最近的 testRatio 比例（至少 1 条），其余为训练。
// 交互数不足 minBehaviors 的用户整体留作训练（避免测试信号过弱）。
func splitBehaviors(db *gorm.DB, minBehaviors int, testRatio float64) (
	map[uint]map[uint]struct{}, map[uint]map[uint]struct{},
) {
	var behaviors []model.UserBehavior
	db.Order("user_id, created_at, id").Find(&behaviors)

	byUser := map[uint][]uint{}
	for _, b := range behaviors {
		byUser[b.UserID] = append(byUser[b.UserID], b.ResourceID)
	}

	train := map[uint]map[uint]struct{}{}
	test := map[uint]map[uint]struct{}{}
	for uid, ids := range byUser {
		train[uid] = map[uint]struct{}{}
		if len(ids) < minBehaviors {
			for _, id := range ids {
				train[uid][id] = struct{}{}
			}
			continue
		}
		nTest := int(float64(len(ids)) * testRatio)
		if nTest < 1 {
			nTest = 1
		}
		cut := len(ids) - nTest
		for i, id := range ids {
			if i < cut {
				train[uid][id] = struct{}{}
			} else {
				if test[uid] == nil {
					test[uid] = map[uint]struct{}{}
				}
				test[uid][id] = struct{}{}
			}
		}
	}
	return train, test
}

func loadRecommendations(db *gorm.DB) map[uint][]uint {
	var recs []model.Recommendation
	db.Find(&recs)
	out := make(map[uint][]uint, len(recs))
	for _, r := range recs {
		var ids []uint
		if err := json.Unmarshal([]byte(r.ResourceIDs), &ids); err != nil {
			continue
		}
		out[r.UserID] = ids
	}
	return out
}

// popularityBaseline 训练集里被交互次数最多的资源排序（热门兜底基线）
func popularityBaseline(train map[uint]map[uint]struct{}) []uint {
	counts := map[uint]int{}
	for _, ids := range train {
		for id := range ids {
			counts[id]++
		}
	}
	ranked := make([]uint, 0, len(counts))
	for id := range counts {
		ranked = append(ranked, id)
	}
	sort.Slice(ranked, func(i, j int) bool {
		if counts[ranked[i]] != counts[ranked[j]] {
			return counts[ranked[i]] > counts[ranked[j]]
		}
		return ranked[i] < ranked[j]
	})
	return ranked
}

type methodMetrics map[string]map[int]float64 // 方法 → 指标名 → K → 值

// evaluate 对每个有测试集的用户分别计算「实际推荐」与「热门基线」的指标并求均值。
// 没有推荐缓存行的用户，其推荐列表按空处理（计 0）——这是诚实的口径。
func evaluate(
	test map[uint]map[uint]struct{},
	recByUser map[uint][]uint,
	popularityRank []uint,
	ks []int,
) map[string]methodMetrics {
	rec := methodMetrics{}
	pop := methodMetrics{}
	for _, k := range ks {
		rec["recall"] = ensure(rec["recall"], k)
		rec["hitrate"] = ensure(rec["hitrate"], k)
		rec["ndcg"] = ensure(rec["ndcg"], k)
		pop["recall"] = ensure(pop["recall"], k)
		pop["hitrate"] = ensure(pop["hitrate"], k)
		pop["ndcg"] = ensure(pop["ndcg"], k)
	}

	n := 0
	for uid, relevant := range test {
		n++
		recRanked := recByUser[uid]
		for _, k := range ks {
			rec["recall"][k] += recallAtK(relevant, recRanked, k)
			rec["hitrate"][k] += hitRateAtK(relevant, recRanked, k)
			rec["ndcg"][k] += ndcgAtK(relevant, recRanked, k)
			pop["recall"][k] += recallAtK(relevant, popularityRank, k)
			pop["hitrate"][k] += hitRateAtK(relevant, popularityRank, k)
			pop["ndcg"][k] += ndcgAtK(relevant, popularityRank, k)
		}
	}
	if n == 0 {
		return map[string]methodMetrics{"recommendation": rec, "popularity": pop}
	}
	for _, m := range []methodMetrics{rec, pop} {
		for _, byK := range m {
			for k := range byK {
				byK[k] /= float64(n)
			}
		}
	}
	return map[string]methodMetrics{"recommendation": rec, "popularity": pop}
}

func ensure(m map[int]float64, k int) map[int]float64 {
	if m == nil {
		return map[int]float64{k: 0}
	}
	m[k] = 0
	return m
}

func printReport(report map[string]methodMetrics, trainUsers, testUsers int) {
	fmt.Printf("[evaluate_recommendations] 训练用户 %d，评估用户 %d\n", trainUsers, testUsers)
	if testUsers == 0 {
		fmt.Println("没有可评估的测试集（交互太少），请积累更多行为后重试")
		return
	}
	header := fmt.Sprintf("%-16s", "方法")
	for _, k := range []int{5, 10, 20} {
		header += fmt.Sprintf("  %-12s", fmt.Sprintf("NDCG@%d", k))
	}
	for _, k := range []int{5, 10, 20} {
		header += fmt.Sprintf("  %-12s", fmt.Sprintf("HitRate@%d", k))
	}
	fmt.Println(header)
	for _, name := range []string{"recommendation", "popularity"} {
		m := report[name]
		line := fmt.Sprintf("%-16s", name)
		for _, k := range []int{5, 10, 20} {
			line += fmt.Sprintf("  %-12.4f", m["ndcg"][k])
		}
		for _, k := range []int{5, 10, 20} {
			line += fmt.Sprintf("  %-12.4f", m["hitrate"][k])
		}
		fmt.Println(line)
	}
	fmt.Println("\n注：recommendation = 平台缓存的实际推荐结果；popularity = 训练集热门排序基线。")
}
