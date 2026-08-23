package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

// Response 是统一 JSON 响应结构，code 为 0 表示成功。
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// OK 返回成功响应。
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: "ok", Data: data})
}

// OKMessage 返回带自定义消息的成功响应。
func OKMessage(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: message})
}

// Error 将 error 映射为统一错误响应；未知错误按 500 处理。
func Error(c *gin.Context, err error) {
	if appErr, ok := apperrors.AsAppError(err); ok {
		c.JSON(appErr.HTTPStatus, Response{Code: appErr.Code, Message: appErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, Response{Code: 50000, Message: "服务器内部错误"})
}
