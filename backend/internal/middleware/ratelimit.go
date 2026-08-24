package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/response"
)

// RateLimit 基于 Redis 固定窗口计数限流。
// purpose 用于区分场景；keyFn 生成限流键；limit 为窗口内最大请求数。
// Redis 不可用或未配置时 fail-open（放行），避免限流组件拖垮正常请求。
func RateLimit(rdb *redis.Client, purpose string, limit int, window time.Duration, keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil || limit <= 0 {
			c.Next()
			return
		}
		key := "ratelimit:" + purpose + ":" + keyFn(c)
		ctx := c.Request.Context()

		n, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}
		if n == 1 {
			_ = rdb.Expire(ctx, key, window)
		}
		if n > int64(limit) {
			response.Error(c, apperrors.NewTooManyRequests("请求过于频繁，请稍后再试"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitByIP 按客户端 IP 限流。
func RateLimitByIP(rdb *redis.Client, purpose string, limit int, window time.Duration) gin.HandlerFunc {
	return RateLimit(rdb, purpose, limit, window, func(c *gin.Context) string { return c.ClientIP() })
}

// RateLimitByUser 按登录用户限流（需在 Auth 之后挂载）；取不到 user_id 时回退按 IP。
func RateLimitByUser(rdb *redis.Client, purpose string, limit int, window time.Duration) gin.HandlerFunc {
	return RateLimit(rdb, purpose, limit, window, func(c *gin.Context) string {
		if v, ok := c.Get(CtxUserID); ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
		return c.ClientIP()
	})
}
