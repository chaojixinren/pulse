package api_test

import (
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAggQueries 为周报/统计接口铺设 4 条聚合查询（身份分布、身份列表、按天趋势、提取数据）。
func mockAggQueries(mock sqlmock.Sqlmock, userID string) {
	now := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(identity_id, '')")).
		WithArgs(userID, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"identity_id", "cnt", "total_duration"}).
			AddRow("i1", 2, 80))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows(e2eIdentityCols).
			AddRow("i1", userID, "Work", nil, "#000000", "person", true, now, now, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DATE_FORMAT(recorded_at")).
		WithArgs(userID, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"d", "cnt", "total_duration"}).
			AddRow("2024-01-01", 1, 30).
			AddRow("2024-01-02", 1, 50))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT extracted_data")).
		WithArgs(userID, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"extracted_data"}).
			AddRow(`{"todos":[{"text":"买牛奶"}],"commitments":[],"notes":[]}`))
}

func TestE2EWeeklyReport(t *testing.T) {
	router, mock, _ := newE2ERouter(t)
	token := e2eToken(t, "u1")
	mockAggQueries(mock, "u1")

	w := perform(router, http.MethodGet, "/api/v1/reports/weekly?week=2024-01-02", nil, "", token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())

	report := dataOf(decode(t, w))
	assert.Equal(t, "2024-01-01", report["week"], "2024-01-02 为周二，周一起点为 2024-01-01")
	assert.Equal(t, float64(2), report["session_count"])
	assert.Equal(t, float64(80), report["total_duration"])

	byIdentity := report["by_identity"].([]interface{})
	require.Len(t, byIdentity, 1)
	assert.Equal(t, "Work", byIdentity[0].(map[string]interface{})["name"])

	topTodos := report["top_todos"].([]interface{})
	require.Len(t, topTodos, 1)
	assert.Equal(t, "买牛奶", topTodos[0])

	trend := report["daily_trend"].([]interface{})
	require.Len(t, trend, 2)
	assert.Equal(t, "2024-01-01", trend[0].(map[string]interface{})["date"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestE2EStatsReport(t *testing.T) {
	router, mock, _ := newE2ERouter(t)
	token := e2eToken(t, "u1")
	mockAggQueries(mock, "u1")

	w := perform(router, http.MethodGet, "/api/v1/reports/stats?from=2024-01-01&to=2024-01-07", nil, "", token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())

	report := dataOf(decode(t, w))
	assert.Equal(t, "2024-01-01", report["from"])
	assert.Equal(t, "2024-01-07", report["to"])
	assert.Equal(t, float64(2), report["session_count"])
	assert.Equal(t, float64(80), report["total_duration"])
	require.Len(t, report["by_identity"].([]interface{}), 1)
	require.Len(t, report["daily_trend"].([]interface{}), 2)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestE2EAccountExport(t *testing.T) {
	router, mock, _ := newE2ERouter(t)
	token := e2eToken(t, "u1")
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(e2eUserCols).AddRow("u1", "a@b.com", "hash", "Alice", nil, "{}", now, now, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(e2eIdentityCols))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(e2eDeviceCols).
			AddRow("d1", "u1", "dev-1", "手表", "wearable", nil, nil, nil, true, "device-token-hash-value", now, now))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, identity_id, device_id, audio_mime")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(e2eSessionListCols))

	w := perform(router, http.MethodGet, "/api/v1/account/export", nil, "", token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())

	data := dataOf(decode(t, w))
	assert.Equal(t, "a@b.com", data["user"].(map[string]interface{})["email"])

	// 验收：导出不泄露敏感字段。
	body := w.Body.String()
	for _, forbidden := range []string{
		"password_hash", "hash",
		"deleted_at",
		"device_token_hash", "device-token-hash-value",
		"audio_data",
	} {
		assert.NotContains(t, body, forbidden, "导出不应包含 %q", forbidden)
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestE2EAccountDelete(t *testing.T) {
	router, mock, _ := newE2ERouter(t)
	token := e2eToken(t, "u1")

	// 注销：软删除用户 + 吊销全部 refresh token。
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET deleted_at")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE refresh_tokens SET revoked_at")).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnResult(sqlmock.NewResult(0, 2))

	w := perform(router, http.MethodDelete, "/api/v1/account", nil, "", token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())
	resp := decode(t, w)
	assert.Equal(t, "账户已注销", resp["message"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestE2EAccountExportUnauthorized(t *testing.T) {
	router, _, _ := newE2ERouter(t)
	w := perform(router, http.MethodGet, "/api/v1/account/export", nil, "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
