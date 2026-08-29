package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/model"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

// fakeAuthenticator 按 token 返回预置设备，未命中则返回 401。
type fakeAuthenticator struct {
	token  string
	device *model.Device
}

func (f *fakeAuthenticator) AuthenticateDevice(_ context.Context, token string) (*model.Device, error) {
	if f.device != nil && token == f.token {
		return f.device, nil
	}
	return nil, apperrors.NewUnauthorized("设备凭据无效")
}

// buildDeviceEngine 复现生产路由的设备态中间件链。
func buildDeviceEngine(a DeviceAuthenticator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())
	r.POST("/device/ping", DeviceAuth(a), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":    c.GetString(CtxUserID),
			"device_id":  c.GetString(CtxDeviceUUID),
			"device_biz": c.GetString(CtxDeviceBizID),
		})
	})
	return r
}

func deviceFixture() *model.Device {
	return &model.Device{ID: "d1", UserID: "u1", DeviceID: "dev-1", IsActive: true}
}

func TestDeviceAuthAcceptsDeviceScheme(t *testing.T) {
	r := buildDeviceEngine(&fakeAuthenticator{token: "tok", device: deviceFixture()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/device/ping", nil)
	req.Header.Set("Authorization", "Device tok")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	// 设备归属的 user_id 写进与用户态相同的键，下游 handler 无需感知鉴权来源。
	assert.Equal(t, "u1", body["user_id"])
	assert.Equal(t, "d1", body["device_id"])
	assert.Equal(t, "dev-1", body["device_biz"])
}

// 用户 JWT 用的 Bearer 不应能走设备通道，反之亦然 —— 两套凭据必须彼此隔离。
func TestDeviceAuthRejectsBearerScheme(t *testing.T) {
	r := buildDeviceEngine(&fakeAuthenticator{token: "tok", device: deviceFixture()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/device/ping", nil)
	req.Header.Set("Authorization", "Bearer tok")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeviceAuthRejectsMissingHeader(t *testing.T) {
	r := buildDeviceEngine(&fakeAuthenticator{token: "tok", device: deviceFixture()})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/device/ping", nil))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeviceAuthRejectsUnknownToken(t *testing.T) {
	r := buildDeviceEngine(&fakeAuthenticator{token: "tok", device: deviceFixture()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/device/ping", nil)
	req.Header.Set("Authorization", "Device wrong")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestExtractSchemeToken(t *testing.T) {
	cases := []struct {
		name   string
		scheme string
		header string
		want   string
	}{
		{"device scheme", "Device", "Device abc123", "abc123"},
		{"lowercase device", "Device", "device abc123", "abc123"},
		{"scheme mismatch", "Device", "Bearer abc123", ""},
		{"bearer unaffected", "Bearer", "Bearer abc123", "abc123"},
		{"only scheme", "Device", "Device", ""},
		{"empty token", "Device", "Device ", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			c.Request.Header.Set("Authorization", tc.header)
			assert.Equal(t, tc.want, extractSchemeToken(c, tc.scheme))
		})
	}
}
