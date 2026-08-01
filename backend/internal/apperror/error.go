package apperror

import (
	"errors"
	"fmt"
)

// 业务错误码，统一响应中的 code 字段
const (
	CodeBadRequest   = 10001
	CodeUnauthorized = 10002
	CodeForbidden    = 10003
	CodeNotFound     = 10004
	CodeConflict     = 10005
	CodeInternal     = 10006
)

// Error 表示带 HTTP 状态码和业务错误码的接口错误
type Error struct {
	HTTPStatus int
	Code       int
	Message    string
	Err        error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

func BadRequest(message string) *Error {
	return &Error{HTTPStatus: 400, Code: CodeBadRequest, Message: message}
}

func Unauthorized(message string) *Error {
	return &Error{HTTPStatus: 401, Code: CodeUnauthorized, Message: message}
}

func Forbidden(message string) *Error {
	return &Error{HTTPStatus: 403, Code: CodeForbidden, Message: message}
}

func NotFound(message string) *Error {
	return &Error{HTTPStatus: 404, Code: CodeNotFound, Message: message}
}

func Conflict(message string) *Error {
	return &Error{HTTPStatus: 409, Code: CodeConflict, Message: message}
}

func Internal(err error) *Error {
	return &Error{HTTPStatus: 500, Code: CodeInternal, Message: "服务器内部错误", Err: err}
}

// IsBusinessError 判断错误是否已经是带业务状态码的接口错误
func IsBusinessError(err error) bool {
	var appErr *Error
	return errors.As(err, &appErr)
}
