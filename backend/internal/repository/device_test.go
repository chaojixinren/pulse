package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/model"
)

var deviceRepoCols = []string{"id", "user_id", "device_id", "name", "device_type", "firmware_version", "battery_level", "last_seen_at", "is_active", "device_token_hash", "created_at", "updated_at"}
var bindCodeRepoCols = []string{"id", "user_id", "code", "expires_at", "used_at", "created_at"}

func TestDeviceRepoCreate(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDeviceRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO devices")).
		WithArgs("d1", "u1", "dev-1", "手表", "wearable", nil, nil, nil, true, "hash").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), &model.Device{ID: "d1", UserID: "u1", DeviceID: "dev-1", Name: "手表", DeviceType: "wearable", IsActive: true, DeviceTokenHash: "hash"})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeviceRepoGetByDeviceID(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDeviceRepo(db)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id")).
		WithArgs("dev-1").
		WillReturnRows(sqlmock.NewRows(deviceRepoCols).AddRow("d1", "u1", "dev-1", "手表", "wearable", nil, nil, nil, true, "hash", now, now))

	d, err := repo.GetByDeviceID(context.Background(), "dev-1")
	require.NoError(t, err)
	require.NotNil(t, d)
	assert.Equal(t, "d1", d.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeviceRepoUpdateHeartbeat(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDeviceRepo(db)
	firmware := "1.0"
	battery := 80

	mock.ExpectExec(regexp.QuoteMeta("UPDATE devices SET firmware_version")).
		WithArgs("1.0", 80, sqlmock.AnyArg(), sqlmock.AnyArg(), "d1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateHeartbeat(context.Background(), "d1", "u1", &firmware, &battery))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeviceRepoCreateBindCodeAndMarkUsed(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDeviceRepo(db)
	now := time.Now().UTC()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO device_bind_codes")).
		WithArgs("c1", "u1", "123456", now).
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, repo.CreateBindCode(context.Background(), &model.DeviceBindCode{ID: "c1", UserID: "u1", Code: "123456", ExpiresAt: now}))

	mock.ExpectExec(regexp.QuoteMeta("UPDATE device_bind_codes SET used_at")).
		WithArgs(sqlmock.AnyArg(), "123456").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.MarkBindCodeUsed(context.Background(), "123456"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeviceRepoCreateCommand(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDeviceRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO device_commands")).
		WithArgs("cmd1", "d1", "u1", "start_recording", "pending").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.CreateCommand(context.Background(), &model.DeviceCommand{ID: "cmd1", DeviceID: "d1", UserID: "u1", Command: "start_recording", Status: "pending"}))
	assert.NoError(t, mock.ExpectationsWereMet())
}
