package api

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/chaojixinren/pulse/internal/middleware"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

func TestParseIntDefault(t *testing.T) {
	assert.Equal(t, 5, parseIntDefault("", 5))
	assert.Equal(t, 10, parseIntDefault("10", 5))
	assert.Equal(t, -3, parseIntDefault("-3", 5))
	assert.Equal(t, 5, parseIntDefault("abc", 5))
	assert.Equal(t, 5, parseIntDefault("10abc", 5))
	assert.Equal(t, 0, parseIntDefault("0", 5))
}

func TestCurrentUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	assert.Empty(t, currentUserID(c), "未设置 user_id 时应返回空串")

	c.Set(middleware.CtxUserID, "user-123")
	assert.Equal(t, "user-123", currentUserID(c))

	c2, _ := gin.CreateTestContext(w)
	c2.Set(middleware.CtxUserID, 123) // 非字符串类型
	assert.Empty(t, currentUserID(c2), "非字符串 user_id 应返回空串")
}

func TestFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	fail(c, apperrors.ErrUnauthorized)

	assert.True(t, c.IsAborted(), "fail 应中止后续处理")
	requireLastErr(t, c, apperrors.ErrUnauthorized)
}

func requireLastErr(t *testing.T, c *gin.Context, want error) {
	t.Helper()
	if assert.NotEmpty(t, c.Errors) {
		assert.Equal(t, want, c.Errors.Last().Err)
	}
}
