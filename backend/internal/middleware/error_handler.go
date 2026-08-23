package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/pkg/response"
)

// ErrorHandler 统一将 handler 通过 c.Error 抛出的错误映射为 JSON 响应。
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			response.Error(c, c.Errors.Last().Err)
		}
	}
}
