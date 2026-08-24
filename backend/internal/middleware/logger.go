package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/chaojixinren/pulse/pkg/logger"
)

// Logger 记录每个请求的 method、path、status 与耗时。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
		}
		if rid := c.GetString(CtxRequestID); rid != "" {
			fields = append(fields, zap.String("request_id", rid))
		}
		logger.Log.Info("http_request", fields...)
	}
}
