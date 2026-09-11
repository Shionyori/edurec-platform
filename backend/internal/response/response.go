package response

import (
	"github.com/gin-gonic/gin"
)

// Body 统一接口响应格式
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// Page 分页响应数据
type Page struct {
	List     any   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	HasMore  bool  `json:"has_more"` // 无限滚动：是否还有下一页（本地耗尽后由在线爬取决定）
}

func OK(c *gin.Context, data any) {
	Write(c, 200, 0, "ok", data)
}

func Created(c *gin.Context, data any) {
	Write(c, 201, 0, "ok", data)
}

func Error(c *gin.Context, httpStatus int, code int, message string) {
	Write(c, httpStatus, code, message, nil)
}

func Write(c *gin.Context, httpStatus int, code int, message string, data any) {
	c.AbortWithStatusJSON(httpStatus, Body{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
