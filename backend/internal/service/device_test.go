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

	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

var deviceCols = []string{"id", "user_id", "device_id", "name", "device_type", "firmware_version", "battery_level", "last_seen_at", "is_active", "device_token_hash", "created_at", "updated_at"}

func newDeviceService(t *testing.T) (*DeviceService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewDeviceService(repository.NewDeviceRepo(db)), mock
}

func deviceRow(id, deviceID string) []driver.Value {
	now := time.Now().UTC()
	return []driver.Value{id, "u1", deviceID, "手表", "wearable", nil, nil, nil, true, "hash", now, now}
}

func TestHeartbeat(t *testing.T) {
	svc, mock := newDeviceService(t)
	firmware := "1.0"
	battery := 80

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("d1", "u1").
		WillReturnRows(sqlmock.NewRows(deviceCols).AddRow(deviceRow("d1", "dev-1")...))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE devices SET firmware_version")).
		WithArgs("1.0", 80, sqlmock.AnyArg(), sqlmock.AnyArg(), "d1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("d1", "u1").
		WillReturnRows(sqlmock.NewRows(deviceCols).AddRow(deviceRow("d1", "dev-1")...))

	got, err := svc.Heartbeat(context.Background(), "u1", "d1", &firmware, &battery)
	require.NoError(t, err)
	assert.Equal(t, "d1", got.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHeartbeatInvalidBattery(t *testing.T) {
	svc, _ := newDeviceService(t)
	battery := 150

	_, err := svc.Heartbeat(context.Background(), "u1", "d1", nil, &battery)
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 40000, appErr.Code)
}

func TestIssueCommand(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		svc, mock := newDeviceService(t)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
			WithArgs("d1", "u1").
			WillReturnRows(sqlmock.NewRows(deviceCols).AddRow(deviceRow("d1", "dev-1")...))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO device_commands")).
			WithArgs(sqlmock.AnyArg(), "d1", "u1", "start_recording", "pending").
			WillReturnResult(sqlmock.NewResult(1, 1))

		cmd, err := svc.IssueCommand(context.Background(), "u1", "d1", "start_recording")
		require.NoError(t, err)
		assert.Equal(t, "start_recording", cmd.Command)
		assert.Equal(t, "pending", cmd.Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid_command", func(t *testing.T) {
		svc, _ := newDeviceService(t)
		_, err := svc.IssueCommand(context.Background(), "u1", "d1", "bogus")
		require.Error(t, err)
		appErr, ok := apperrors.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, 40000, appErr.Code)
	})
}

func TestUnbind(t *testing.T) {
	svc, mock := newDeviceService(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("d1", "u1").
		WillReturnRows(sqlmock.NewRows(deviceCols).AddRow(deviceRow("d1", "dev-1")...))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM devices")).
		WithArgs("d1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, svc.Unbind(context.Background(), "u1", "d1"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListDevices(t *testing.T) {
	svc, mock := newDeviceService(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(deviceCols).AddRow(deviceRow("d1", "dev-1")...))

	list, err := svc.List(context.Background(), "u1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "dev-1", list[0].DeviceID)
	assert.NoError(t, mock.ExpectationsWereMet())
}
