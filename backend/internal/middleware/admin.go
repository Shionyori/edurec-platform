package middleware

import (
	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/gin-gonic/gin"
)

// AdminRequired 在 AuthRequired 之后校验当前用户是否为管理员
func AdminRequired(users repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := UserID(c)
		if !ok {
			response.Error(c, 401, apperror.CodeUnauthorized, "未认证或 token 无效")
			c.Abort()
			return
		}

		isAdmin, err := users.IsAdmin(userID)
		if err != nil {
			response.Error(c, 500, apperror.CodeInternal, "服务器内部错误")
			c.Abort()
			return
		}
		if !isAdmin {
			response.Error(c, 403, apperror.CodeForbidden, "无权限")
			c.Abort()
			return
		}

		c.Next()
	}
}
