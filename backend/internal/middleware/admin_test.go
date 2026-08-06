package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/gin-gonic/gin"
)

type stubUserRepository struct {
	isAdmin bool
	hasErr  bool
}

func (s *stubUserRepository) Create(_ *model.User) error {
	return nil
}

func (s *stubUserRepository) FindByUsername(_ string) (*model.User, error) {
	return nil, nil
}

func (s *stubUserRepository) FindByEmail(_ string) (*model.User, error) {
	return nil, nil
}

func (s *stubUserRepository) FindByID(_ uint) (*model.User, error) {
	return nil, nil
}

func (s *stubUserRepository) Update(_ *model.User) error {
	return nil
}

func (s *stubUserRepository) IsAdmin(_ uint) (bool, error) {
	if s.hasErr {
		return false, errStub
	}
	return s.isAdmin, nil
}

var errStub = &stubError{}

type stubError struct{}

func (e *stubError) Error() string {
	return "stub error"
}

func TestAdminRequiredAllowsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(userIDKey, uint(1))
		c.Next()
	})
	router.Use(AdminRequired(&stubUserRepository{isAdmin: true}))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminRequiredRejectsNormalUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(userIDKey, uint(1))
		c.Next()
	})
	router.Use(AdminRequired(&stubUserRepository{isAdmin: false}))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if !hasErrorCode(w, apperror.CodeForbidden) {
		t.Fatalf("response = %s, want code %d", w.Body.String(), apperror.CodeForbidden)
	}
}

func TestAdminRequiredRejectsMissingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AdminRequired(&stubUserRepository{isAdmin: true}))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if !hasErrorCode(w, apperror.CodeUnauthorized) {
		t.Fatalf("response = %s, want code %d", w.Body.String(), apperror.CodeUnauthorized)
	}
}

func hasErrorCode(w *httptest.ResponseRecorder, wantCode int) bool {
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		return false
	}
	return body.Code == wantCode
}
