package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/chaojixinren/pulse/internal/api"
	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/pkg/logger"
	"github.com/chaojixinren/pulse/pkg/utils"
)

func TestMain(m *testing.M) {
	logger.Init("test", "error")
	os.Exit(m.Run())
}

var (
	e2eUserCols        = []string{"id", "email", "password_hash", "name", "avatar_url", "settings", "created_at", "updated_at", "deleted_at"}
	e2eSessionListCols = []string{"id", "user_id", "identity_id", "device_id", "audio_mime", "transcript", "duration", "file_size", "status", "error_message", "extracted_data", "ai_confidence", "recorded_at", "processed_at", "created_at", "updated_at"}
	e2eIdentityCols    = []string{"id", "user_id", "name", "description", "color", "icon", "is_default", "created_at", "updated_at", "deleted_at"}
	e2eDeviceCols      = []string{"id", "user_id", "device_id", "name", "device_type", "firmware_version", "battery_level", "last_seen_at", "is_active", "device_token_hash", "created_at", "updated_at"}
	e2eBindCodeCols    = []string{"id", "user_id", "code", "expires_at", "used_at", "created_at"}
)

const e2eSecret = "e2e-secret"

// newE2ERouter 构建完整路由（含所有中间件/服务/仓库），DB 由 sqlmock 驱动。
func newE2ERouter(t *testing.T) (*gin.Engine, sqlmock.Sqlmock, *config.Config) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	cfg := &config.Config{JWTSecret: e2eSecret, GINMode: gin.TestMode, MaxAudioSize: 1024 * 1024, JWTExpiresIn: time.Hour, RefreshTokenTTL: 7 * 24 * time.Hour}
	router, _ := api.NewRouter(cfg, db)
	return router, mock, cfg
}

func perform(router http.Handler, method, path string, body io.Reader, contentType, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func jsonReq(t *testing.T, v interface{}) (io.Reader, string) {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b), "application/json"
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m), "响应应为 JSON: %s", w.Body.String())
	return m
}

func dataOf(m map[string]interface{}) map[string]interface{} {
	d, _ := m["data"].(map[string]interface{})
	return d
}

func newMultipart(t *testing.T, filename string, content []byte, fields map[string]string) (io.Reader, string) {
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
	return &buf, w.FormDataContentType()
}

func e2eToken(t *testing.T, userID string) string {
	t.Helper()
	tok, err := utils.GenerateAccessToken(userID, e2eSecret, time.Hour)
	require.NoError(t, err)
	return tok
}

func e2eWAV() []byte {
	return []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'A', 'V', 'E', 'f', 'm', 't', ' '}
}

func TestE2EHealth(t *testing.T) {
	router, _, _ := newE2ERouter(t)

	w := perform(router, http.MethodGet, "/health", nil, "", "")
	assert.Equal(t, http.StatusOK, w.Code)
	resp := decode(t, w)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "ok", resp["message"])
}

func TestE2ERegister(t *testing.T) {
	router, mock, _ := newE2ERouter(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
		WithArgs("a@b.com").
		WillReturnRows(sqlmock.NewRows(e2eUserCols))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users")).
		WithArgs(sqlmock.AnyArg(), "a@b.com", sqlmock.AnyArg(), "Alice", nil, "{}").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, ct := jsonReq(t, map[string]string{"email": "a@b.com", "password": "secret123", "name": "Alice"})
	w := perform(router, http.MethodPost, "/api/v1/auth/register", body, ct, "")

	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())
	resp := decode(t, w)
	assert.Equal(t, float64(0), resp["code"])

	data := dataOf(resp)
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "a@b.com", data["email"])
	assert.Equal(t, "Alice", data["name"])

	// 验收：注册响应不泄露密码。
	assert.NotContains(t, w.Body.String(), "secret123")
	assert.NotContains(t, w.Body.String(), "password_hash")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestE2ELogin(t *testing.T) {
	router, mock, _ := newE2ERouter(t)
	now := time.Now().UTC()
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
		WithArgs("a@b.com").
		WillReturnRows(sqlmock.NewRows(e2eUserCols).AddRow("u1", "a@b.com", string(hash), "Alice", nil, "{}", now, now, nil))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO refresh_tokens")).
		WithArgs(sqlmock.AnyArg(), "u1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, ct := jsonReq(t, map[string]string{"email": "a@b.com", "password": "secret123"})
	w := perform(router, http.MethodPost, "/api/v1/auth/login", body, ct, "")

	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())
	data := dataOf(decode(t, w))
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestE2EMe(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		router, _, _ := newE2ERouter(t)
		w := perform(router, http.MethodGet, "/api/v1/auth/me", nil, "", "")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("authorized", func(t *testing.T) {
		router, mock, _ := newE2ERouter(t)
		now := time.Now().UTC()

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
			WithArgs("u1").
			WillReturnRows(sqlmock.NewRows(e2eUserCols).AddRow("u1", "a@b.com", "hash", "Alice", nil, "{}", now, now, nil))

		w := perform(router, http.MethodGet, "/api/v1/auth/me", nil, "", e2eToken(t, "u1"))
		require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())
		data := dataOf(decode(t, w))
		assert.Equal(t, "Alice", data["name"])
		assert.NotContains(t, w.Body.String(), "password_hash")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestE2EUploadTimelineReport(t *testing.T) {
	router, mock, _ := newE2ERouter(t)
	token := e2eToken(t, "u1")
	now := time.Now().UTC()

	// 1) 上传音频
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audio_sessions")).
		WithArgs(sqlmock.AnyArg(), "u1", nil, nil, sqlmock.AnyArg(), sqlmock.AnyArg(), nil, 10, sqlmock.AnyArg(), "pending", nil, "{}", nil, sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, ct := newMultipart(t, "clip.wav", e2eWAV(), map[string]string{"duration": "10"})
	w := perform(router, http.MethodPost, "/api/v1/audio/upload", body, ct, token)

	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())
	uploadData := dataOf(decode(t, w))
	sessionID := uploadData["session_id"].(string)
	assert.NotEmpty(t, sessionID)
	assert.Equal(t, "pending", uploadData["status"])

	// 2) 时间线
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM audio_sessions")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, identity_id, device_id, audio_mime")).
		WithArgs("u1", 20, 0).
		WillReturnRows(sqlmock.NewRows(e2eSessionListCols).
			AddRow(sessionID, "u1", nil, nil, "audio/wav", nil, 10, int64(13), "pending", nil, "{}", nil, now, nil, now, now))

	w = perform(router, http.MethodGet, "/api/v1/timeline?page=1&size=20", nil, "", token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())
	tl := dataOf(decode(t, w))
	assert.Equal(t, float64(1), tl["total"])
	items := tl["items"].([]interface{})
	require.Len(t, items, 1)
	assert.Equal(t, sessionID, items[0].(map[string]interface{})["session_id"])

	// 3) 日报
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(identity_id, '')")).
		WithArgs("u1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"identity_id", "cnt", "total_duration"}).AddRow("", 1, 10))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(e2eIdentityCols))

	w = perform(router, http.MethodGet, "/api/v1/reports/daily?date=2024-01-02", nil, "", token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())
	report := dataOf(decode(t, w))
	assert.Equal(t, "2024-01-02", report["date"])
	assert.Equal(t, float64(1), report["session_count"])
	assert.Equal(t, float64(10), report["total_duration"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestE2EUploadInvalidRecordedAt(t *testing.T) {
	router, _, _ := newE2ERouter(t)
	token := e2eToken(t, "u1")

	body, ct := newMultipart(t, "clip.wav", e2eWAV(), map[string]string{"recorded_at": "not-a-time"})
	w := perform(router, http.MethodPost, "/api/v1/audio/upload", body, ct, token)

	require.Equal(t, http.StatusBadRequest, w.Code, "响应: %s", w.Body.String())
	assert.Equal(t, float64(40000), decode(t, w)["code"])
}

func TestE2EIdentityCRUD(t *testing.T) {
	router, mock, _ := newE2ERouter(t)
	token := e2eToken(t, "u1")
	now := time.Now().UTC()

	// 创建首个身份 → 自动成为默认身份
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO identities")).
		WithArgs(sqlmock.AnyArg(), "u1", "Work", nil, "#000000", "person", false).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM identities")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE identities SET is_default = FALSE")).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE identities SET is_default = TRUE")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnRows(sqlmock.NewRows(e2eIdentityCols).AddRow("i1", "u1", "Work", nil, "#000000", "person", true, now, now, nil))

	body, ct := jsonReq(t, map[string]string{"name": "Work"})
	w := perform(router, http.MethodPost, "/api/v1/identities", body, ct, token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())
	identity := dataOf(decode(t, w))
	assert.Equal(t, "Work", identity["name"])
	assert.Equal(t, true, identity["is_default"])

	// 列出身份
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(e2eIdentityCols).AddRow("i1", "u1", "Work", nil, "#000000", "person", true, now, now, nil))

	w = perform(router, http.MethodGet, "/api/v1/identities", nil, "", token)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, decode(t, w)["data"].([]interface{}), 1)

	// 删除默认身份 → 400
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("i1", "u1").
		WillReturnRows(sqlmock.NewRows(e2eIdentityCols).AddRow("i1", "u1", "Work", nil, "#000000", "person", true, now, now, nil))

	w = perform(router, http.MethodDelete, "/api/v1/identities/i1", nil, "", token)
	require.Equal(t, http.StatusBadRequest, w.Code, "响应: %s", w.Body.String())
	assert.Equal(t, float64(40000), decode(t, w)["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestE2EDeviceBindFlow(t *testing.T) {
	router, mock, _ := newE2ERouter(t)
	token := e2eToken(t, "u1")
	now := time.Now().UTC()

	// 1. 生成绑定码
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO device_bind_codes")).
		WithArgs(sqlmock.AnyArg(), "u1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	w := perform(router, http.MethodPost, "/api/v1/devices/bind-code", nil, "application/json", token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())
	code := dataOf(decode(t, w))["code"].(string)
	assert.Len(t, code, 6)

	// 2. 绑定设备
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, code")).
		WithArgs(code).
		WillReturnRows(sqlmock.NewRows(e2eBindCodeCols).AddRow("c1", "u1", code, now.Add(time.Hour), nil, now))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("dev-1").
		WillReturnRows(sqlmock.NewRows(e2eDeviceCols))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE device_bind_codes SET used_at")).
		WithArgs(sqlmock.AnyArg(), code).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO devices")).
		WithArgs(sqlmock.AnyArg(), "u1", "dev-1", "手表", "wearable", nil, nil, nil, true, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, ct := jsonReq(t, map[string]string{"device_id": "dev-1", "name": "手表", "bind_code": code})
	w = perform(router, http.MethodPost, "/api/v1/devices/bind", body, ct, token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())
	bindData := dataOf(decode(t, w))
	device := bindData["device"].(map[string]interface{})
	assert.Equal(t, "dev-1", device["device_id"])
	assert.NotEmpty(t, bindData["device_token"], "绑定应返回设备 token")
	deviceID := device["id"].(string)

	// 3. 心跳
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs(deviceID, "u1").
		WillReturnRows(sqlmock.NewRows(e2eDeviceCols).AddRow(deviceID, "u1", "dev-1", "手表", "wearable", nil, nil, nil, true, "hash", now, now))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE devices SET firmware_version")).
		WithArgs(nil, 80, sqlmock.AnyArg(), sqlmock.AnyArg(), deviceID, "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs(deviceID, "u1").
		WillReturnRows(sqlmock.NewRows(e2eDeviceCols).AddRow(deviceID, "u1", "dev-1", "手表", "wearable", nil, nil, nil, true, "hash", now, now))

	body, ct = jsonReq(t, map[string]int{"battery_level": 80})
	w = perform(router, http.MethodPost, "/api/v1/devices/"+deviceID+"/heartbeat", body, ct, token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())

	// 4. 解绑
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs(deviceID, "u1").
		WillReturnRows(sqlmock.NewRows(e2eDeviceCols).AddRow(deviceID, "u1", "dev-1", "手表", "wearable", nil, nil, nil, true, "hash", now, now))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM devices")).
		WithArgs(deviceID, "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w = perform(router, http.MethodDelete, "/api/v1/devices/"+deviceID, nil, "", token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())

	assert.NoError(t, mock.ExpectationsWereMet())
}
