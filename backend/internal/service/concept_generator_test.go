package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

func TestHTTPConceptRunnerParsesConcepts(t *testing.T) {
	var got struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		PointCode   string `json:"point_code"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/knowledge/generate" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		// 响应样本来自 engine 的契约 fixture（scripts/check-contract-sync.sh 同步）
		body, err := os.ReadFile("testdata/kg-generate.contract.json")
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()

	runner := service.NewHTTPConceptRunner(server.URL, 5*time.Second)
	concepts, err := runner.Generate(context.Background(), &model.KnowledgePoint{
		Code: "mathematical-analysis", Name: "数学分析", Description: "极限与积分",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(concepts) != 2 || concepts[0].Name != "积分" {
		t.Fatalf("concepts = %+v", concepts)
	}
	if got.Title != "数学分析" || got.PointCode != "mathematical-analysis" {
		t.Errorf("请求体 = %+v", got)
	}
}

func TestHTTPConceptRunnerReturnsEngineError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "Wikifier 超时"})
	}))
	defer server.Close()

	runner := service.NewHTTPConceptRunner(server.URL, 5*time.Second)
	if _, err := runner.Generate(context.Background(), &model.KnowledgePoint{Name: "数学分析"}); err == nil {
		t.Fatal("engine 500 时应当报错")
	}
}

func TestGeneratedConceptCodeIsStableAndUnique(t *testing.T) {
	a1 := service.GeneratedConceptCode("mathematical-analysis", "积分")
	a2 := service.GeneratedConceptCode("mathematical-analysis", "积分")
	b := service.GeneratedConceptCode("mathematical-analysis", "导数")
	c := service.GeneratedConceptCode("abstract-algebra", "积分")

	if a1 != a2 {
		t.Errorf("同一输入应得到稳定 code")
	}
	if a1 == b || a1 == c {
		t.Errorf("不同概念/学科应得到不同 code")
	}
}
