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

func TestE2EDeviceCommandAndList(t *testing.T) {
	router, mock, _ := newE2ERouter(t)
	token := e2eToken(t, "u1")
	now := time.Now().UTC()

	// 设备列表
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(e2eDeviceCols).AddRow("d1", "u1", "dev-1", "手表", "wearable", nil, nil, nil, true, "hash", now, now))

	w := perform(router, http.MethodGet, "/api/v1/devices", nil, "", token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())
	assert.Len(t, decode(t, w)["data"].([]interface{}), 1)

	// 设备详情
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("d1", "u1").
		WillReturnRows(sqlmock.NewRows(e2eDeviceCols).AddRow("d1", "u1", "dev-1", "手表", "wearable", nil, nil, nil, true, "hash", now, now))

	w = perform(router, http.MethodGet, "/api/v1/devices/d1", nil, "", token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())
	assert.Equal(t, "dev-1", dataOf(decode(t, w))["device_id"])

	// 下发指令
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("d1", "u1").
		WillReturnRows(sqlmock.NewRows(e2eDeviceCols).AddRow("d1", "u1", "dev-1", "手表", "wearable", nil, nil, nil, true, "hash", now, now))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO device_commands")).
		WithArgs(sqlmock.AnyArg(), "d1", "u1", "start_recording", "pending").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, ct := jsonReq(t, map[string]string{"command": "start_recording"})
	w = perform(router, http.MethodPost, "/api/v1/devices/d1/command", body, ct, token)
	require.Equal(t, http.StatusOK, w.Code, "响应: %s", w.Body.String())
	cmd := dataOf(decode(t, w))
	assert.Equal(t, "start_recording", cmd["command"])
	assert.Equal(t, "pending", cmd["status"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestE2EDeviceCommandInvalid(t *testing.T) {
	router, _, _ := newE2ERouter(t)
	token := e2eToken(t, "u1")

	body, ct := jsonReq(t, map[string]string{"command": "bogus"})
	w := perform(router, http.MethodPost, "/api/v1/devices/d1/command", body, ct, token)
	require.Equal(t, http.StatusBadRequest, w.Code, "响应: %s", w.Body.String())
	assert.Equal(t, float64(40000), decode(t, w)["code"])
}

func TestE2EDeviceHeartbeatInvalidBattery(t *testing.T) {
	router, _, _ := newE2ERouter(t)
	token := e2eToken(t, "u1")

	body, ct := jsonReq(t, map[string]int{"battery_level": 150})
	w := perform(router, http.MethodPost, "/api/v1/devices/d1/heartbeat", body, ct, token)
	require.Equal(t, http.StatusBadRequest, w.Code, "响应: %s", w.Body.String())
	assert.Equal(t, float64(40000), decode(t, w)["code"])
}

func TestE2EDeviceBindInvalidCode(t *testing.T) {
	router, mock, _ := newE2ERouter(t)
	token := e2eToken(t, "u1")
	now := time.Now().UTC()

	// 绑定码已使用（used_at 非空）→ 一次性校验失败
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, code")).
		WithArgs("123456").
		WillReturnRows(sqlmock.NewRows(e2eBindCodeCols).AddRow("c1", "u1", "123456", now.Add(time.Hour), now, now))

	body, ct := jsonReq(t, map[string]string{"device_id": "dev-1", "name": "手表", "bind_code": "123456"})
	w := perform(router, http.MethodPost, "/api/v1/devices/bind", body, ct, token)
	require.Equal(t, http.StatusBadRequest, w.Code, "响应: %s", w.Body.String())
	assert.Equal(t, float64(40000), decode(t, w)["code"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
