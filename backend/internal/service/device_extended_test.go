package service

import (
	"context"
	"regexp"
	"testing"

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
