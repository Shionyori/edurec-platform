package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/router"
	"github.com/gin-gonic/gin"
)

// 知识点/学科/学习画像路由必须注册。
// 未携带 token 时，已注册路由由鉴权中间件返回 401；只有未注册的路由才是 404。
// 这条测试可以防止路由被误删或分组挂错。
func TestKnowledgeRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := router.New(&config.Config{}, nil, nil)

	routes := []string{
		"/api/v1/knowledge-points",
		"/api/v1/knowledge-points/calculus",
		"/api/v1/knowledge-points/calculus/resources",
		"/api/v1/disciplines",
		"/api/v1/disciplines/math",
		"/api/v1/learning-profile",
	}

	for _, route := range routes {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, route, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("GET %s = %d, want 401（404 表示路由未注册）", route, rec.Code)
		}
	}
}
