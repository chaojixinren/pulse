//go:build e2e

// 真实基础设施端到端测试：需要可用的 MySQL 与 Redis。
// 运行方式：
//
//	export TEST_DATABASE_DSN='user:pass@tcp(localhost:3306)/pulse?charset=utf8mb4&parseTime=True&loc=Local'
//	export TEST_REDIS_URL='redis://localhost:6379'
//	go test -tags e2e -run TestLiveE2E ./test/ -v
//
// 前提：目标库已执行 migrations（go run cmd/migrate/main.go）。
package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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

type liveClient struct {
	base  string
	token string
}

func (c *liveClient) do(t *testing.T, method, path string, body io.Reader, contentType string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, c.base+path, body)
	require.NoError(t, err)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, b
}

func (c *liveClient) json(t *testing.T, method, path string, payload interface{}) (*http.Response, map[string]interface{}) {
	t.Helper()
	var body io.Reader
	ct := ""
	if payload != nil {
		b, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(b)
		ct = "application/json"
	}
	resp, raw := c.do(t, method, path, body, ct)
	var m map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &m)
	}
	return resp, m
}

func (c *liveClient) upload(t *testing.T, filename string, content []byte, fields map[string]string) (*http.Response, map[string]interface{}) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = fw.Write(content)
	require.NoError(t, err)
	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}
	require.NoError(t, w.Close())

	resp, raw := c.do(t, http.MethodPost, "/api/v1/audio/upload", &buf, w.FormDataContentType())
	var m map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &m)
	}
	return resp, m
}

func TestLiveE2EFullFlow(t *testing.T) {
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

	email := fmt.Sprintf("e2e-%d@example.com", time.Now().UnixNano())
	c := &liveClient{base: srv.URL}

	// 清理遗留数据（幂等）
	defer func() { _, _ = db.Exec("DELETE FROM users WHERE email = ?", email) }()

	// 1) 健康检查
	resp, health := c.json(t, http.MethodGet, "/health", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, float64(0), health["code"])

	// 2) 注册
	resp, reg := c.json(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": email, "password": "secret123", "name": "E2E User",
	})
	require.Equal(t, http.StatusOK, resp.StatusCode, "register: %v", reg)
	regData := reg["data"].(map[string]interface{})
	userID := regData["id"].(string)
	require.NotEmpty(t, userID)

	// 3) 登录
	resp, login := c.json(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": email, "password": "secret123",
	})
	require.Equal(t, http.StatusOK, resp.StatusCode, "login: %v", login)
	loginData := login["data"].(map[string]interface{})
	c.token = loginData["access_token"].(string)
	require.NotEmpty(t, c.token)

	// 4) 当前用户
	resp, me := c.json(t, http.MethodGet, "/api/v1/auth/me", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, email, me["data"].(map[string]interface{})["email"])

	// 5) 上传音频
	resp, up := c.upload(t, "clip.wav", []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'A', 'V', 'E', 'f', 'm', 't', ' '}, map[string]string{"duration": "10"})
	require.Equal(t, http.StatusOK, resp.StatusCode, "upload: %v", up)
	sessionID := up["data"].(map[string]interface{})["session_id"].(string)
	require.NotEmpty(t, sessionID)

	// 6) 时间线（含上传的会话）
	resp, tl := c.json(t, http.MethodGet, "/api/v1/timeline?page=1&size=20", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "timeline: %v", tl)
	tlData := tl["data"].(map[string]interface{})
	assert.Equal(t, float64(1), tlData["total"])
	items := tlData["items"].([]interface{})
	require.Len(t, items, 1)
	assert.Equal(t, sessionID, items[0].(map[string]interface{})["session_id"])

	// 7) 创建身份（首个身份自动默认）
	resp, ident := c.json(t, http.MethodPost, "/api/v1/identities", map[string]string{"name": "Work"})
	require.Equal(t, http.StatusOK, resp.StatusCode, "create identity: %v", ident)
	identData := ident["data"].(map[string]interface{})
	assert.Equal(t, true, identData["is_default"])

	// 8) 日报（当前日期）
	resp, report := c.json(t, http.MethodGet, "/api/v1/reports/daily?date="+time.Now().Format("2006-01-02"), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "report: %v", report)
	reportData := report["data"].(map[string]interface{})
	assert.Equal(t, float64(1), reportData["session_count"])
	assert.Equal(t, float64(10), reportData["total_duration"])
}
