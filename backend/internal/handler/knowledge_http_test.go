package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/handler"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// fakeKnowledgeRepo 只实现 HTTP 契约测试用到的查询；其余方法由嵌入接口兜底。
// 所有列表方法都返回 nil，用来验证服务层会把它们规范成 JSON 数组。
type fakeKnowledgeRepo struct {
	repository.KnowledgeRepository
	point      *model.KnowledgePoint
	discipline *model.Discipline
}

func (f *fakeKnowledgeRepo) ListPoints() ([]model.KnowledgePoint, error) { return nil, nil }

func (f *fakeKnowledgeRepo) ListEdges() ([]model.KnowledgePointEdge, error) { return nil, nil }

func (f *fakeKnowledgeRepo) CountResourcesByPoint() (map[uint]int64, error) {
	return map[uint]int64{}, nil
}

func (f *fakeKnowledgeRepo) CountConceptsByPoint() (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (f *fakeKnowledgeRepo) FindPointByCode(string) (*model.KnowledgePoint, error) {
	return f.point, nil
}

func (f *fakeKnowledgeRepo) ListResourcesByPoint(uint, int, int) ([]model.Resource, int64, error) {
	return nil, 0, nil
}

func (f *fakeKnowledgeRepo) ListConceptsByPoint(string, int) ([]model.KnowledgeConcept, error) {
	return nil, nil
}

func (f *fakeKnowledgeRepo) ListConceptEdgesAmong([]uint) ([]model.KnowledgeConceptEdge, error) {
	return nil, nil
}

func (f *fakeKnowledgeRepo) FindDisciplineByCode(string) (*model.Discipline, error) {
	return f.discipline, nil
}

func (f *fakeKnowledgeRepo) ListPointsByDiscipline(string) ([]model.KnowledgePoint, error) {
	return nil, nil
}

func (f *fakeKnowledgeRepo) ListResourcesByDiscipline(string, int, int) ([]model.Resource, int64, error) {
	return nil, 0, nil
}

type envelope struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

func serve(t *testing.T, register func(*gin.Engine), target string) envelope {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	register(r)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("%s 状态码 = %d, want 200; body=%s", target, rec.Code, rec.Body.String())
	}

	var body envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败: %v; body=%s", err, rec.Body.String())
	}
	return body
}

func assertArrayFields(t *testing.T, data json.RawMessage, fields ...string) {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatalf("data 不是对象: %v; data=%s", err, data)
	}
	for _, field := range fields {
		raw, ok := object[field]
		if !ok {
			t.Errorf("缺少字段 %s", field)
			continue
		}
		if string(raw) != "[]" {
			t.Errorf("字段 %s = %s, want []（前端按数组渲染）", field, raw)
		}
	}
}

// 知识点列表：即使仓储返回 nil，也必须序列化为 []。
func TestKnowledgeListEmptyIsArray(t *testing.T) {
	repo := &fakeKnowledgeRepo{}
	h := handler.NewKnowledgeHandler(service.NewKnowledgeService(repo))

	body := serve(t, func(r *gin.Engine) {
		r.GET("/knowledge-points", h.List)
	}, "/knowledge-points")

	if string(body.Data) != "[]" {
		t.Fatalf("data = %s, want []", body.Data)
	}
}

// 知识点详情：资源/概念/先修边为空时全部是数组。
func TestKnowledgeDetailEmptyListsAreArrays(t *testing.T) {
	repo := &fakeKnowledgeRepo{
		point: &model.KnowledgePoint{Code: "calculus", Name: "高等数学"},
	}
	h := handler.NewKnowledgeHandler(service.NewKnowledgeService(repo))

	body := serve(t, func(r *gin.Engine) {
		r.GET("/knowledge-points/:code", h.Detail)
	}, "/knowledge-points/calculus")

	assertArrayFields(t, body.Data, "nodes", "edges", "resources", "concepts", "concept_edges")
}

// 学科详情：无资源时 resources 必须是 []，不能是 null（曾导致前端整页渲染崩溃）。
func TestDisciplineDetailEmptyResourcesIsArray(t *testing.T) {
	repo := &fakeKnowledgeRepo{
		discipline: &model.Discipline{Code: "math", Name: "数学"},
	}
	h := handler.NewDisciplineHandler(service.NewDisciplineService(repo), nil)

	body := serve(t, func(r *gin.Engine) {
		r.GET("/disciplines/:code", h.Detail)
	}, "/disciplines/math")

	assertArrayFields(t, body.Data, "points", "resources")
}
