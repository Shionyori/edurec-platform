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

		got := service.AggregateKnowledgePointsFromTags(resources, []string{"教程"}, 2, 60)

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
		got := service.AggregateKnowledgePointsFromTags(resources, nil, 2, 60)
		if len(got) != 1 || got[0].Count != 2 {
			t.Fatalf("got %+v, want 机器学习 x2（重复标签不该把词频刷到 3）", got)
		}
	})

	t.Run("短标签被剔除", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["C"]`),
			resourceWithTags(2, 1, `["C"]`),
		}
		if got := service.AggregateKnowledgePointsFromTags(resources, nil, 2, 60); len(got) != 0 {
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
		got := service.AggregateKnowledgePointsFromTags(resources, nil, 2, 60)
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
		got := service.AggregateKnowledgePointsFromTags(resources, nil, 2, 2)
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
		got := service.AggregateKnowledgePointsFromTags(resources, nil, 2, 60)
		if len(got) != 1 || got[0].CategoryID != 0 {
			t.Fatalf("got %+v, want CategoryID=0", got)
		}
	})
}
