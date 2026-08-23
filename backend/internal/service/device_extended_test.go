package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

// assertAppCode 断言 err 为业务错误且业务码匹配。
func assertAppCode(t *testing.T, err error, code int) {
	t.Helper()
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok, "应为业务错误，got %v", err)
	assert.Equal(t, code, appErr.Code)
}

func TestBindExpiredCode(t *testing.T) {
	svc, mock := newDeviceService(t)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, code")).
		WithArgs("123456").
		WillReturnRows(sqlmock.NewRows(bindCodeCols).AddRow("c1", "u1", "123456", now.Add(-time.Minute), nil, now))

	_, _, err := svc.Bind(context.Background(), "u1", "dev-1", "手表", "123456")
	assertAppCode(t, err, 40000)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBindCodeOtherUser(t *testing.T) {
	svc, mock := newDeviceService(t)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, code")).
		WithArgs("123456").
		WillReturnRows(sqlmock.NewRows(bindCodeCols).AddRow("c1", "u2", "123456", now.Add(time.Hour), nil, now))

	_, _, err := svc.Bind(context.Background(), "u1", "dev-1", "手表", "123456")
	assertAppCode(t, err, 40000)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBindValidation(t *testing.T) {
	t.Run("空 device_id", func(t *testing.T) {
		svc, _ := newDeviceService(t)
		_, _, err := svc.Bind(context.Background(), "u1", "  ", "手表", "123456")
		assertAppCode(t, err, 40000)
	})

	t.Run("空绑定码", func(t *testing.T) {
		svc, _ := newDeviceService(t)
		_, _, err := svc.Bind(context.Background(), "u1", "dev-1", "手表", "  ")
		assertAppCode(t, err, 40000)
	})
}

func TestGenerateBindCodeExpiry(t *testing.T) {
	svc, mock := newDeviceService(t)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO device_bind_codes")).
		WithArgs(sqlmock.AnyArg(), "u1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	before := time.Now().UTC()
	code, err := svc.GenerateBindCode(context.Background(), "u1")
	after := time.Now().UTC()
	require.NoError(t, err)
	assert.True(t, code.ExpiresAt.After(before.Add(9*time.Minute)) && code.ExpiresAt.Before(after.Add(11*time.Minute)),
		"绑定码有效期应约为 10 分钟，got %v", code.ExpiresAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRandomBindCode(t *testing.T) {
	for i := 0; i < 20; i++ {
		code, err := randomBindCode(6)
		require.NoError(t, err)
		assert.Len(t, code, 6)
		for _, c := range code {
			assert.True(t, c >= '0' && c <= '9', "绑定码应为纯数字，got %q", code)
		}
	}
}

func TestDeviceNotFound(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		svc, mock := newDeviceService(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
			WithArgs("d1", "u1").
			WillReturnRows(sqlmock.NewRows(deviceCols))
		_, err := svc.Get(context.Background(), "u1", "d1")
		assertAppCode(t, err, 40400)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("unbind", func(t *testing.T) {
		svc, mock := newDeviceService(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
			WithArgs("d1", "u1").
			WillReturnRows(sqlmock.NewRows(deviceCols))
		assertAppCode(t, svc.Unbind(context.Background(), "u1", "d1"), 40400)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("heartbeat", func(t *testing.T) {
		svc, mock := newDeviceService(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
			WithArgs("d1", "u1").
			WillReturnRows(sqlmock.NewRows(deviceCols))
		_, err := svc.Heartbeat(context.Background(), "u1", "d1", nil, nil)
		assertAppCode(t, err, 40400)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("command", func(t *testing.T) {
		svc, mock := newDeviceService(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
			WithArgs("d1", "u1").
			WillReturnRows(sqlmock.NewRows(deviceCols))
		_, err := svc.IssueCommand(context.Background(), "u1", "d1", "start_recording")
		assertAppCode(t, err, 40400)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
