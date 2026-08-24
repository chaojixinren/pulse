package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func newRateReq() *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	return req
}

func TestRateLimitNilRedisPasses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", RateLimitByIP(nil, "auth", 1, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRateReq())
	assert.Equal(t, http.StatusOK, w.Code, "Redis 未配置时应放行")
}

func TestRateLimitByIPBlocksOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb := newTestRedis(t)
	r := gin.New()
	r.GET("/x", RateLimitByIP(rdb, "auth", 2, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newRateReq())
		assert.Equal(t, http.StatusOK, w.Code, "第 %d 次应通过", i+1)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRateReq())
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "超过限额应返回 429")
	assert.Contains(t, w.Body.String(), "42900")
}

func TestRateLimitByUserUsesUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb := newTestRedis(t)
	r := gin.New()
	r.GET("/x",
		func(c *gin.Context) { c.Set(CtxUserID, "u1"); c.Next() },
		RateLimitByUser(rdb, "upload", 1, time.Minute),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, newRateReq())
	assert.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, newRateReq())
	assert.Equal(t, http.StatusTooManyRequests, w2.Code, "同一 user_id 共享配额")
}
