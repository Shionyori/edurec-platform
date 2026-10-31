package service_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

// 学科种子必须覆盖主要学科，且每个知识点都挂在存在的学科下
func TestDefaultDisciplinesConsistent(t *testing.T) {
	disciplines := make(map[string]struct{}, len(service.DefaultDisciplines))
	for _, discipline := range service.DefaultDisciplines {
		if discipline.Code == "" || discipline.Name == "" {
			t.Errorf("学科缺少 code/name: %+v", discipline)
		}
		if _, exists := disciplines[discipline.Code]; exists {
			t.Errorf("学科 code 重复: %s", discipline.Code)
		}
		disciplines[discipline.Code] = struct{}{}
		if len(discipline.Keywords) == 0 {
			t.Errorf("学科 %s 缺少关键词（自动爬取需要）", discipline.Code)
		}
	}

	for _, point := range service.DefaultKnowledgePoints {
		if point.DisciplineCode == "" {
			t.Errorf("知识点 %s 未归入学科", point.Code)
			continue
		}
		if _, ok := disciplines[point.DisciplineCode]; !ok {
			t.Errorf("知识点 %s 引用了不存在的学科 %s", point.Code, point.DisciplineCode)
		}
	}

	if len(disciplines) < 15 {
		t.Errorf("主要学科数量过少: %d", len(disciplines))
	}
}

// fakeKnowledgeRepo 只实现学科详情用到的查询；其余方法由嵌入接口兜底（不应被调用）。
type fakeKnowledgeRepo struct {
	repository.KnowledgeRepository
	discipline *model.Discipline
	points     []model.KnowledgePoint
	resources  []model.Resource
}

func (f *fakeKnowledgeRepo) FindDisciplineByCode(string) (*model.Discipline, error) {
	return f.discipline, nil
}

func (f *fakeKnowledgeRepo) ListPointsByDiscipline(string) ([]model.KnowledgePoint, error) {
	return f.points, nil
}

func (f *fakeKnowledgeRepo) CountResourcesByPoint() (map[uint]int64, error) {
	return map[uint]int64{}, nil
}

func (f *fakeKnowledgeRepo) CountConceptsByPoint() (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (f *fakeKnowledgeRepo) ListResourcesByDiscipline(string, int, int) ([]model.Resource, int64, error) {
	return f.resources, int64(len(f.resources)), nil
}

// 学科无资源时 resources 必须序列化为 []，不能是 null：
// 前端按数组访问（detail.resources.length），null 会导致整页渲染崩溃。
func TestDisciplineDetailEmptyResourcesSerializesAsArray(t *testing.T) {
	svc := service.NewDisciplineService(&fakeKnowledgeRepo{
		discipline: &model.Discipline{Code: "math", Name: "数学"},
	})

	detail, err := svc.Detail("math", 12)
	if err != nil {
		t.Fatalf("Detail() error = %v", err)
	}
	if detail.Resources == nil {
		t.Fatal("Resources = nil，应为空切片")
	}

	body, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(body), `"resources":[]`) {
		t.Fatalf("resources 未序列化为数组: %s", body)
	}
}
