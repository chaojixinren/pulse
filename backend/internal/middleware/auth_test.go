package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/pkg/utils"
)

func TestExtractToken(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"empty header", "", ""},
		{"bearer token", "Bearer abc123", "abc123"},
		{"lowercase bearer", "bearer abc123", "abc123"},
		{"extra spaces", "Bearer    abc123", "abc123"},
		{"basic scheme", "Basic abc123", ""},
		{"no scheme", "abc123", ""},
		{"only scheme", "Bearer", ""},
		{"empty token", "Bearer ", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			if tc.header != "" {
				c.Request.Header.Set("Authorization", tc.header)
			}
			assert.Equal(t, tc.want, extractToken(c))
		})
	}
}

// buildAuthedEngine 复现生产路由的中间件链：ErrorHandler + Auth + 受保护 handler。
func buildAuthedEngine(cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/", Auth(cfg), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetString(CtxUserID)})
	})
	return r
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	cfg := &config.Config{JWTSecret: "secret"}
	r := buildAuthedEngine(cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddlewareRejectsInvalidToken(t *testing.T) {
	cfg := &config.Config{JWTSecret: "secret"}
	r := buildAuthedEngine(cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddlewareRejectsWrongSecretToken(t *testing.T) {
	cfg := &config.Config{JWTSecret: "secret"}
	tok, err := utils.GenerateAccessToken("user-1", "another-secret", 3600e9)
	assert.NoError(t, err)

	r := buildAuthedEngine(cfg)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddlewareAcceptsValidToken(t *testing.T) {
	cfg := &config.Config{JWTSecret: "secret"}
	tok, err := utils.GenerateAccessToken("user-1", "secret", 3600e9)
	assert.NoError(t, err)

	r := buildAuthedEngine(cfg)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-1")
}
