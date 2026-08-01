package middleware

import (
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	jwtutil "github.com/Shionyori/edurec-platform/backend/internal/util/jwt"
	"github.com/gin-gonic/gin"
)

type contextKey string

const userIDKey contextKey = "user_id"

// AuthRequired 校验 Bearer Access Token，并把用户 ID 写入请求上下文
func AuthRequired(jwtManager *jwtutil.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			response.Error(c, 401, apperror.CodeUnauthorized, "未认证或 token 无效")
			c.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		userID, err := jwtManager.ParseAccessToken(token)
		if err != nil {
			response.Error(c, 401, apperror.CodeUnauthorized, "未认证或 token 无效")
			c.Abort()
			return
		}

		c.Set(userIDKey, userID)
		c.Next()
	}
}

// UserID 从请求上下文读取当前登录用户 ID
func UserID(c *gin.Context) (uint, bool) {
	value, exists := c.Get(userIDKey)
	if !exists {
		return 0, false
	}
	userID, ok := value.(uint)
	return userID, ok
}
