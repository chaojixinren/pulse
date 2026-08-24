package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// CtxRequestID 是 gin context 中存放 request_id 的键。
const CtxRequestID = "request_id"

// RequestIDHeader 是请求/响应中透传 request_id 的 header 名。
const RequestIDHeader = "X-Request-ID"

// Trace 生成或透传 request_id，写入 context 与响应头，用于链路串联。
func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(RequestIDHeader)
		if rid == "" {
			rid = newRequestID()
		}
		c.Set(CtxRequestID, rid)
		c.Header(RequestIDHeader, rid)
		c.Next()
	}
}

// newRequestID 生成 32 位十六进制随机 ID。
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
