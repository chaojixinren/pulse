package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestTraceGeneratesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Trace())
	r.GET("/x", func(c *gin.Context) {
		rid, ok := c.Get(CtxRequestID)
		assert.True(t, ok)
		assert.NotEmpty(t, rid)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get(RequestIDHeader), "响应头应回传 request_id")
}

func TestTracePropagatesExistingRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Trace())
	r.GET("/x", func(c *gin.Context) {
		assert.Equal(t, "my-req-id", c.GetString(CtxRequestID))
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(RequestIDHeader, "my-req-id")
	r.ServeHTTP(w, req)

	assert.Equal(t, "my-req-id", w.Header().Get(RequestIDHeader), "应透传已有 request_id")
}

func TestNewRequestID(t *testing.T) {
	id := newRequestID()
	assert.Len(t, id, 32, "request_id 应为 32 位十六进制")
	assert.NotEqual(t, id, newRequestID(), "两次生成的 ID 应不同")
}
