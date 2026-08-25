//go:build e2e

// 真实基础设施端到端测试（Phase 3）：报告、数据导出/注销、音频加密。
// 运行方式与前置条件见 e2e_live_test.go 顶部说明。
package test

import (
	"database/sql"
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
	"github.com/chaojixinren/pulse/internal/service"
)

// setupPhase3 打开真实 DB 并构造路由。
func setupPhase3(t *testing.T, dsn string) (*httptest.Server, *sql.DB) {
	t.Helper()
	db, err := config.InitDB(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	cfg := &config.Config{
		JWTSecret:       "live-e2e-phase3-secret",
		GINMode:         gin.TestMode,
		MaxAudioSize:    1024 * 1024,
		JWTExpiresIn:    time.Hour,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}
	router, _ := api.NewRouter(cfg, db)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv, db
}

func TestLiveE2EPhase3Flow(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("跳过真实基础设施 e2e：需设置 TEST_DATABASE_DSN")
	}
	srv, db := setupPhase3(t, dsn)

	email := fmt.Sprintf("e2e-p3-%d@example.com", time.Now().UnixNano())
	c := &liveClient{base: srv.URL}
	defer func() { _, _ = db.Exec("DELETE FROM users WHERE email = ?", email) }()

	// 注册 + 登录
	_, _ = c.json(t, http.MethodPost, "/api/v1/auth/register", map[string]string{"email": email, "password": "secret123", "name": "P3"})
	_, login := c.json(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": "secret123"})
	c.token = login["data"].(map[string]interface{})["access_token"].(string)
	require.NotEmpty(t, c.token, "login: %v", login)

	// 周报
	resp, weekly := c.json(t, http.MethodGet, "/api/v1/reports/weekly?week="+time.Now().Format("2006-01-02"), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "weekly: %v", weekly)

	// 统计
	from := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().Format("2006-01-02")
	resp, stats := c.json(t, http.MethodGet, "/api/v1/reports/stats?from="+from+"&to="+to, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "stats: %v", stats)

	// 导出：应包含用户信息且不含 password_hash
	resp, exp := c.json(t, http.MethodGet, "/api/v1/account/export", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "export: %v", exp)
	expUser := exp["data"].(map[string]interface{})["user"].(map[string]interface{})
	assert.Equal(t, email, expUser["email"])
	assert.NotContains(t, expUser, "password_hash", "导出不应包含 password_hash")

	// 注销
	resp, del := c.json(t, http.MethodDelete, "/api/v1/account", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "delete: %v", del)

	// 注销后无法登录
	resp, _ = c.json(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": "secret123"})
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "注销后应无法登录")
}

// TestLiveE2EPhase3Encryption 验收「audio_data 中存储的是密文」：
// 配置 AUDIO_ENCRYPTION_KEY 后上传音频，直接读取数据库中的 audio_data 应为密文且可解密回原文。
func TestLiveE2EPhase3Encryption(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("跳过真实基础设施 e2e：需设置 TEST_DATABASE_DSN")
	}

	db, err := config.InitDB(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	cfg := &config.Config{
		JWTSecret:          "live-e2e-phase3-enc-secret",
		GINMode:            gin.TestMode,
		MaxAudioSize:       1024 * 1024,
		JWTExpiresIn:       time.Hour,
		RefreshTokenTTL:    7 * 24 * time.Hour,
		AudioEncryptionKey: key,
	}
	router, _ := api.NewRouter(cfg, db)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	email := fmt.Sprintf("e2e-p3-enc-%d@example.com", time.Now().UnixNano())
	c := &liveClient{base: srv.URL}
	defer func() { _, _ = db.Exec("DELETE FROM users WHERE email = ?", email) }()

	_, _ = c.json(t, http.MethodPost, "/api/v1/auth/register", map[string]string{"email": email, "password": "secret123", "name": "Enc"})
	_, login := c.json(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": "secret123"})
	c.token = login["data"].(map[string]interface{})["access_token"].(string)
	require.NotEmpty(t, c.token, "login: %v", login)

	plain := []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'A', 'V', 'E', 'f', 'm', 't', ' '}
	resp, up := c.upload(t, "clip.wav", plain, map[string]string{"duration": "10"})
	require.Equal(t, http.StatusOK, resp.StatusCode, "upload: %v", up)
	sessionID := up["data"].(map[string]interface{})["session_id"].(string)
	require.NotEmpty(t, sessionID)

	// 直接读取数据库中落库的 audio_data。
	var stored []byte
	err = db.QueryRow("SELECT audio_data FROM audio_sessions WHERE id = ?", sessionID).Scan(&stored)
	require.NoError(t, err)
	assert.NotEqual(t, plain, stored, "audio_data 应存储密文而非明文")

	dec, err := service.DecryptAudio(stored, key)
	require.NoError(t, err)
	assert.Equal(t, plain, dec, "密文应可解密回原文")
}
