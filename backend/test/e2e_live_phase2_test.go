//go:build e2e

package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/api"
	"github.com/chaojixinren/pulse/internal/config"
)

// TestLiveE2EPhase2Devices 覆盖 Phase 2 设备管理的真实基础设施端到端流程。
// 运行方式见 e2e_live_test.go；需先执行 migrations 并设置 TEST_DATABASE_DSN / TEST_REDIS_URL。
func TestLiveE2EPhase2Devices(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	redisURL := os.Getenv("TEST_REDIS_URL")
	if dsn == "" || redisURL == "" {
		t.Skip("跳过真实基础设施 e2e：需设置 TEST_DATABASE_DSN 与 TEST_REDIS_URL")
	}

	db, err := config.InitDB(dsn)
	require.NoError(t, err)
	defer db.Close()

	rdb, err := config.InitRedis(redisURL)
	require.NoError(t, err)
	defer rdb.Close()

	cfg := &config.Config{JWTSecret: "live-e2e-secret", GINMode: gin.TestMode, MaxAudioSize: 1024 * 1024, JWTExpiresIn: time.Hour, RefreshTokenTTL: 7 * 24 * time.Hour}
	router, _ := api.NewRouter(cfg, db, rdb)

	srv := httptest.NewServer(router)
	defer srv.Close()

	email := fmt.Sprintf("e2e-phase2-%d@example.com", time.Now().UnixNano())
	c := &liveClient{base: srv.URL}
	defer func() { _, _ = db.Exec("DELETE FROM users WHERE email = ?", email) }()

	// 注册 + 登录
	resp, reg := c.json(t, http.MethodPost, "/api/v1/auth/register", map[string]string{"email": email, "password": "secret123", "name": "E2E Phase2"})
	require.Equal(t, http.StatusOK, resp.StatusCode, "register: %v", reg)

	resp, login := c.json(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": "secret123"})
	require.Equal(t, http.StatusOK, resp.StatusCode, "login: %v", login)
	c.token = login["data"].(map[string]interface{})["access_token"].(string)

	// 1) 生成绑定码
	resp, bc := c.json(t, http.MethodPost, "/api/v1/devices/bind-code", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "bind-code: %v", bc)
	bindCode := bc["data"].(map[string]interface{})["code"].(string)
	require.Len(t, bindCode, 6)

	// 2) 绑定设备
	deviceID := fmt.Sprintf("live-dev-%d", time.Now().UnixNano())
	resp, bd := c.json(t, http.MethodPost, "/api/v1/devices/bind", map[string]string{"device_id": deviceID, "name": "手表", "bind_code": bindCode})
	require.Equal(t, http.StatusOK, resp.StatusCode, "bind: %v", bd)
	bindData := bd["data"].(map[string]interface{})
	device := bindData["device"].(map[string]interface{})
	devID := device["id"].(string)
	assert.NotEmpty(t, bindData["device_token"])
	require.NotEmpty(t, devID)

	// 3) 心跳
	resp, _ = c.json(t, http.MethodPost, "/api/v1/devices/"+devID+"/heartbeat", map[string]int{"battery_level": 80})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 4) 指令下发
	resp, _ = c.json(t, http.MethodPost, "/api/v1/devices/"+devID+"/command", map[string]string{"command": "start_recording"})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 5) 设备列表
	resp, dl := c.json(t, http.MethodGet, "/api/v1/devices", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "devices: %v", dl)
	assert.GreaterOrEqual(t, len(dl["data"].([]interface{})), 1)

	// 6) 解绑
	resp, _ = c.json(t, http.MethodDelete, "/api/v1/devices/"+devID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
