package service

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/utils"
)

var commandCols = []string{"id", "device_id", "user_id", "command", "status", "created_at", "updated_at"}

func commandRow(id, command string) []driver.Value {
	now := time.Now().UTC()
	return []driver.Value{id, "d1", "u1", command, "pending", now, now}
}

// expectBindCodeLookup 预置一次绑定码查询，code 归属固定为 u1。
func expectBindCodeLookup(mock sqlmock.Sqlmock, code string) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, code")).
		WithArgs(code).
		WillReturnRows(sqlmock.NewRows(bindCodeCols).AddRow(bindCodeRow(code, nil)...))
}

func TestClaimCreatesDeviceWithoutUserAuth(t *testing.T) {
	svc, mock := newDeviceService(t)

	expectBindCodeLookup(mock, "123456")
	// 设备尚未存在
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("dev-1").
		WillReturnRows(sqlmock.NewRows(deviceCols))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE device_bind_codes SET used_at")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO devices")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	device, token, err := svc.Claim(context.Background(), "dev-1", "手表", "123456")
	require.NoError(t, err)
	// 用户身份完全由绑定码反解，设备侧无需任何预置凭据。
	assert.Equal(t, "u1", device.UserID)
	assert.NotEmpty(t, token)
	// 库里只存哈希，明文 token 仅此一次返回。
	assert.Equal(t, utils.SHA256Hex(token), device.DeviceTokenHash)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimRepairRotatesToken(t *testing.T) {
	svc, mock := newDeviceService(t)

	expectBindCodeLookup(mock, "123456")
	// 同一用户的设备已存在（恢复出厂后重新配对）
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("dev-1").
		WillReturnRows(sqlmock.NewRows(deviceCols).AddRow(deviceRow("d1", "dev-1")...))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE device_bind_codes SET used_at")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE devices SET device_token_hash")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("d1").
		WillReturnRows(sqlmock.NewRows(deviceCols).AddRow(deviceRow("d1", "dev-1")...))

	device, token, err := svc.Claim(context.Background(), "dev-1", "", "123456")
	require.NoError(t, err)
	assert.Equal(t, "d1", device.ID)
	assert.NotEmpty(t, token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimRejectsDeviceOwnedByAnotherUser(t *testing.T) {
	svc, mock := newDeviceService(t)

	expectBindCodeLookup(mock, "123456")
	// 该 device_id 已属于 u2，绑定码却是 u1 的 —— 不能借此劫持。
	other := deviceRow("d9", "dev-1")
	other[1] = "u2"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("dev-1").
		WillReturnRows(sqlmock.NewRows(deviceCols).AddRow(other...))

	_, _, err := svc.Claim(context.Background(), "dev-1", "", "123456")
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 400, appErr.HTTPStatus)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimRejectsExpiredBindCode(t *testing.T) {
	svc, mock := newDeviceService(t)

	expired := bindCodeRow("123456", nil)
	expired[3] = time.Now().UTC().Add(-time.Minute) // expires_at 已过
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, code")).
		WithArgs("123456").
		WillReturnRows(sqlmock.NewRows(bindCodeCols).AddRow(expired...))

	_, _, err := svc.Claim(context.Background(), "dev-1", "", "123456")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimRejectsUsedBindCode(t *testing.T) {
	svc, mock := newDeviceService(t)

	used := time.Now().UTC().Add(-time.Minute)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, code")).
		WithArgs("123456").
		WillReturnRows(sqlmock.NewRows(bindCodeCols).AddRow(bindCodeRow("123456", used)...))

	_, _, err := svc.Claim(context.Background(), "dev-1", "", "123456")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Bind 不允许重新配对：已被绑定的设备一律拒绝，行为与改造前一致。
func TestBindStillRejectsAlreadyBoundDevice(t *testing.T) {
	svc, mock := newDeviceService(t)

	expectBindCodeLookup(mock, "123456")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("dev-1").
		WillReturnRows(sqlmock.NewRows(deviceCols).AddRow(deviceRow("d1", "dev-1")...))

	_, _, err := svc.Bind(context.Background(), "u1", "dev-1", "", "123456")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthenticateDeviceSuccess(t *testing.T) {
	svc, mock := newDeviceService(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs(utils.SHA256Hex("tok-abc")).
		WillReturnRows(sqlmock.NewRows(deviceCols).AddRow(deviceRow("d1", "dev-1")...))

	d, err := svc.AuthenticateDevice(context.Background(), "tok-abc")
	require.NoError(t, err)
	assert.Equal(t, "d1", d.ID)
	assert.Equal(t, "u1", d.UserID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthenticateDeviceUnknownToken(t *testing.T) {
	svc, mock := newDeviceService(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs(utils.SHA256Hex("bad")).
		WillReturnRows(sqlmock.NewRows(deviceCols))

	_, err := svc.AuthenticateDevice(context.Background(), "bad")
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 401, appErr.HTTPStatus)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthenticateDeviceEmptyToken(t *testing.T) {
	svc, _ := newDeviceService(t)
	_, err := svc.AuthenticateDevice(context.Background(), "   ")
	require.Error(t, err)
}

func TestDeviceHeartbeatReturnsPendingCommands(t *testing.T) {
	svc, mock := newDeviceService(t)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE devices SET firmware_version")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE device_commands SET status = 'expired'")).
		WillReturnResult(sqlmock.NewResult(1, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, device_id, user_id, command")).
		WithArgs("d1").
		WillReturnRows(sqlmock.NewRows(commandCols).AddRow(commandRow("cmd-1", "start_recording")...))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("d1").
		WillReturnRows(sqlmock.NewRows(deviceCols).AddRow(deviceRow("d1", "dev-1")...))

	battery := 87
	device, cmds, err := svc.DeviceHeartbeat(context.Background(), "d1", nil, &battery)
	require.NoError(t, err)
	assert.Equal(t, "d1", device.ID)
	require.Len(t, cmds, 1)
	assert.Equal(t, "start_recording", cmds[0].Command)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeviceHeartbeatRejectsInvalidBattery(t *testing.T) {
	svc, _ := newDeviceService(t)
	bad := 150
	_, _, err := svc.DeviceHeartbeat(context.Background(), "d1", nil, &bad)
	require.Error(t, err)
}

func TestAckCommand(t *testing.T) {
	svc, mock := newDeviceService(t)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE device_commands SET status = ?")).
		WithArgs("done", sqlmock.AnyArg(), "cmd-1", "d1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, svc.AckCommand(context.Background(), "d1", "cmd-1", "done"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAckCommandRejectsUnknownStatus(t *testing.T) {
	svc, _ := newDeviceService(t)
	require.Error(t, svc.AckCommand(context.Background(), "d1", "cmd-1", "whatever"))
}

// 指令不存在、不属于该设备或已是终态时，影响行数为 0。
func TestAckCommandNotFound(t *testing.T) {
	svc, mock := newDeviceService(t)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE device_commands SET status = ?")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := svc.AckCommand(context.Background(), "d1", "cmd-x", "done")
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 404, appErr.HTTPStatus)
	assert.NoError(t, mock.ExpectationsWereMet())
}
