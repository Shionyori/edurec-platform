package service_test

import (
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

func resourceWithTags(id uint, categoryID uint, tags string) model.Resource {
	r := model.Resource{Tags: tags, CategoryID: categoryID}
	r.ID = id
	return r
}

func TestParseResourceTags(t *testing.T) {
	t.Run("正常数组", func(t *testing.T) {
		got := service.ParseResourceTags(`["机器学习","回归"]`)
		if len(got) != 2 || got[0] != "机器学习" {
			t.Errorf("ParseResourceTags = %v", got)
		}
	})

	// 标签来自爬虫与数据集导入，是外部数据：null、类型不对、JSON 损坏都不能 panic，
	// 也不能漏出空字符串标签。断言 len==0 而不是「逐个查空」——后者在返回空切片时
	// 恒真，等于没断言。
	t.Run("空与损坏一律返回空", func(t *testing.T) {
		for _, raw := range []string{"", "  ", "null", "not json", `{"a":1}`, `[1,2]`, `["ok",3]`} {
			if got := service.ParseResourceTags(raw); len(got) != 0 {
				t.Errorf("ParseResourceTags(%q) = %v, want 空切片", raw, got)
			}
		}
	})

	t.Run("数组里的空白标签被剔除", func(t *testing.T) {
		got := service.ParseResourceTags(`["机器学习","  ",""]`)
		if len(got) != 1 || got[0] != "机器学习" {
			t.Errorf("got %v, want [机器学习]", got)
		}
	})
}

func TestAggregateKnowledgePointsFromTags(t *testing.T) {
	t.Run("按词频聚合，低于阈值与停用词被剔除", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["机器学习","教程"]`),
			resourceWithTags(2, 1, `["机器学习","回归"]`),
			resourceWithTags(3, 2, `["机器学习"]`),
			resourceWithTags(4, 2, `["深度学习"]`),
		}

		got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{
			Stoplist: []string{"教程"}, MinCount: 2, Max: 60,
		})

		if len(got) != 1 {
			t.Fatalf("聚合出 %d 个知识点，want 1: %+v", len(got), got)
		}
		if got[0].Name != "机器学习" || got[0].Count != 3 {
			t.Errorf("got %+v, want 机器学习 x3", got[0])
		}
		// 出现最多的分类成为知识点的归属分类（这里分类 1 与 2 各 1 次，取 id 小的）
		if got[0].CategoryID != 1 {
			t.Errorf("CategoryID = %d, want 1", got[0].CategoryID)
		}
	})

	t.Run("同一资源内重复标签只计一次", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["机器学习","机器学习"]`),
			resourceWithTags(2, 1, `["机器学习"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{MinCount: 2, Max: 60})
		if len(got) != 1 || got[0].Count != 2 {
			t.Fatalf("got %+v, want 机器学习 x2（重复标签不该把词频刷到 3）", got)
		}
	})

	t.Run("短标签被剔除", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["C"]`),
			resourceWithTags(2, 1, `["C"]`),
		}
		if got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{MinCount: 2, Max: 60}); len(got) != 0 {
			t.Errorf("单字符标签不该成为知识点: %+v", got)
		}
	})

	t.Run("损坏标签不打断整批聚合", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `null`),
			resourceWithTags(2, 1, `not json`),
			resourceWithTags(3, 1, `["机器学习"]`),
			resourceWithTags(4, 1, `["机器学习"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{MinCount: 2, Max: 60})
		if len(got) != 1 || got[0].Name != "机器学习" {
			t.Fatalf("got %+v, want 仅 机器学习", got)
		}
	})

	t.Run("超出上限时按词频截断，同频按名字稳定排序", func(t *testing.T) {
		// 注意：fixture 里的词必须是 2 个 rune 以上。单个汉字只占 1 个 rune，
		// 会被 MinKnowledgePointNameRunes=2 过滤掉（见上一个用例的 "C"），
		// 用单字当 fixture 会让这个用例恒为 0 个候选、断言永不成立。
		resources := []model.Resource{
			resourceWithTags(1, 1, `["甲甲","乙乙","丙丙"]`),
			resourceWithTags(2, 1, `["甲甲","乙乙","丙丙"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{MinCount: 2, Max: 2})
		if len(got) != 2 {
			t.Fatalf("got %d 个，want 2", len(got))
		}
		// 三次词频相同 → 按名字升序，保证多次运行结果一致（map 迭代本身无序）
		if got[0].Name > got[1].Name {
			t.Errorf("同频未按名字升序: %+v", got)
		}
	})

	t.Run("无分类的资源不影响聚合", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 0, `["机器学习"]`),
			resourceWithTags(2, 0, `["机器学习"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{MinCount: 2, Max: 60})
		if len(got) != 1 || got[0].CategoryID != 0 {
			t.Fatalf("got %+v, want CategoryID=0", got)
		}
	})
}

func TestAggregate_CaseFoldingAndSynonyms(t *testing.T) {
	t.Run("拉丁文大小写归并到同一条，展示名取出现最多的写法", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["Python"]`),
			resourceWithTags(2, 1, `["Python"]`),
			resourceWithTags(3, 1, `["python"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{MinCount: 2, Max: 60})

		if len(got) != 1 {
			t.Fatalf("got %d 条，want 1（Python/python 应归并）: %+v", len(got), got)
		}
		if got[0].Name != "Python" {
			t.Errorf("展示名 = %q, want Python（出现次数多的写法胜出）", got[0].Name)
		}
		if got[0].Count != 3 {
			t.Errorf("Count = %d, want 3（归并后应累加）", got[0].Count)
		}
		if len(got[0].Aliases) != 1 || got[0].Aliases[0] != "python" {
			t.Errorf("Aliases = %v, want [python]（折叠进来的原始写法要留作别名）", got[0].Aliases)
		}
	})

	t.Run("同频时展示名取字典序最小，保证确定", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["Go"]`),
			resourceWithTags(2, 1, `["go"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{MinCount: 2, Max: 60})
		// "Go" < "go"（大写字母字节序在前）
		if len(got) != 1 || got[0].Name != "Go" {
			t.Fatalf("got %+v, want 展示名 Go", got)
		}
	})

	t.Run("synonyms 把别名归到规范名", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["Golang"]`),
			resourceWithTags(2, 1, `["Go语言"]`),
			resourceWithTags(3, 1, `["Go"]`),
			resourceWithTags(4, 1, `["go"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{
			MinCount: 2, Max: 60,
			Synonyms: []service.SynonymGroup{{Canonical: "Go", Aliases: []string{"Golang", "Go语言", "go"}}},
		})

		if len(got) != 1 {
			t.Fatalf("got %d 条，want 1（四种写法应归到 Go）: %+v", len(got), got)
		}
		if got[0].Name != "Go" || got[0].Count != 4 {
			t.Errorf("got %+v, want Go 计数 4", got[0])
		}
		// 别名要去重、且不含规范名本身；sort.Strings 后顺序确定。
		// 字节序：'G'(0x47) < 'g'(0x67)，且 "Golang" 的第 3 字节 'l'(0x6C) < "Go语言" 的第 3 字节 0xE8
		want := []string{"Golang", "Go语言", "go"}
		if len(got[0].Aliases) != len(want) {
			t.Fatalf("Aliases = %v, want %v", got[0].Aliases, want)
		}
		for i := range want {
			if got[0].Aliases[i] != want[i] {
				t.Errorf("Aliases = %v, want %v", got[0].Aliases, want)
				break
			}
		}
	})

	t.Run("synonyms 的别名也受停用词与长度规则约束", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["Golang"]`),
			resourceWithTags(2, 1, `["Golang"]`),
		}
		// 别名 "Golang" 被停用 → 不应有任何知识点产出
		got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{
			MinCount: 2, Max: 60, Stoplist: []string{"Golang"},
			Synonyms: []service.SynonymGroup{{Canonical: "Go", Aliases: []string{"Golang"}}},
		})
		if len(got) != 0 {
			t.Errorf("停用词应作用于别名，got %+v", got)
		}
	})
}
