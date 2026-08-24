package errors

import (
	stderrors "errors"
	"net/http"
)

// AppError 是业务错误，携带 HTTP 状态码与业务码。
type AppError struct {
	HTTPStatus int
	Code       int
	Message    string
}

func (e *AppError) Error() string { return e.Message }

var (
	ErrUnauthorized    = &AppError{HTTPStatus: http.StatusUnauthorized, Code: 40100, Message: "未授权"}
	ErrForbidden       = &AppError{HTTPStatus: http.StatusForbidden, Code: 40300, Message: "禁止访问"}
	ErrNotFound        = &AppError{HTTPStatus: http.StatusNotFound, Code: 40400, Message: "资源不存在"}
	ErrBadRequest      = &AppError{HTTPStatus: http.StatusBadRequest, Code: 40000, Message: "请求参数错误"}
	ErrTooManyRequests = &AppError{HTTPStatus: http.StatusTooManyRequests, Code: 42900, Message: "请求过于频繁"}
	ErrInternal        = &AppError{HTTPStatus: http.StatusInternalServerError, Code: 50000, Message: "服务器内部错误"}
)

func NewBadRequest(msg string) *AppError {
	return &AppError{HTTPStatus: http.StatusBadRequest, Code: 40000, Message: msg}
}

func NewUnauthorized(msg string) *AppError {
	return &AppError{HTTPStatus: http.StatusUnauthorized, Code: 40100, Message: msg}
}

func NewForbidden(msg string) *AppError {
	return &AppError{HTTPStatus: http.StatusForbidden, Code: 40300, Message: msg}
}

func NewNotFound(msg string) *AppError {
	return &AppError{HTTPStatus: http.StatusNotFound, Code: 40400, Message: msg}
}

func NewInternal(msg string) *AppError {
	return &AppError{HTTPStatus: http.StatusInternalServerError, Code: 50000, Message: msg}
}

func NewTooManyRequests(msg string) *AppError {
	return &AppError{HTTPStatus: http.StatusTooManyRequests, Code: 42900, Message: msg}
}

func WrapInternal(err error) *AppError {
	if err == nil {
		return ErrInternal
	}
	return &AppError{HTTPStatus: http.StatusInternalServerError, Code: 50000, Message: err.Error()}
}

// AsAppError 判断 error 是否为 *AppError。
func AsAppError(err error) (*AppError, bool) {
	var e *AppError
	ok := stderrors.As(err, &e)
	return e, ok
}
