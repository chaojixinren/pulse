package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/internal/middleware"
)

// currentUserID 从 gin context 读取认证中间件写入的 user_id。
func currentUserID(c *gin.Context) string {
	v, exists := c.Get(middleware.CtxUserID)
	if !exists {
		return ""
	}
	id, _ := v.(string)
	return id
}

// fail 通过 gin 的错误通道抛出业务错误，由 ErrorHandler 中间件统一映射为响应。
func fail(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
