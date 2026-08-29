package service

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

func newAccountService(t *testing.T) (*AccountService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	svc := NewAccountService(
		repository.NewUserRepo(db),
		repository.NewIdentityRepo(db),
		repository.NewDeviceRepo(db),
		repository.NewAudioSessionRepo(db),
		repository.NewRefreshTokenRepo(db),
		nil,
	)
	return svc, mock
}

func TestAccountExport(t *testing.T) {
	svc, mock := newAccountService(t)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash, name, avatar_url, settings, created_at, updated_at, deleted_at")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(userCols).AddRow("u1", "a@b.com", "hash", "Alice", nil, "{}", now, now, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name, description, color, icon, is_default, created_at, updated_at, deleted_at")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(identityCols).AddRow("i1", "u1", "Work", nil, "#000000", "person", true, now, now, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, device_id, name, device_type, firmware_version, battery_level, last_seen_at, is_active, device_token_hash, created_at, updated_at")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(deviceCols))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, identity_id, device_id, audio_mime, transcript, duration, file_size, status, error_message, extracted_data, ai_confidence, recorded_at, processed_at, created_at, updated_at")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(sessionListCols))

	export, err := svc.Export(context.Background(), "u1")
	require.NoError(t, err)
	require.NotNil(t, export)
	assert.Equal(t, "Alice", export.User.Name)
	assert.Equal(t, "a@b.com", export.User.Email)

	// password_hash 通过 json:"-" 排除，序列化后不应出现。
	b, err := json.Marshal(export)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "password_hash")
	assert.NotContains(t, string(b), "hash")

	require.Len(t, export.Identities, 1)
	assert.Equal(t, "Work", export.Identities[0].Name)
	assert.Empty(t, export.Devices)
	assert.Empty(t, export.Sessions)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountExportUserNotFound(t *testing.T) {
	svc, mock := newAccountService(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows(userCols))

	_, err := svc.Export(context.Background(), "missing")
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 40400, appErr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountDelete(t *testing.T) {
	svc, mock := newAccountService(t)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET deleted_at")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE refresh_tokens SET revoked_at")).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnResult(sqlmock.NewResult(0, 2))

	require.NoError(t, svc.Delete(context.Background(), "u1"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAccountExportJSONExcludesSensitiveFields 验收「敏感字段不泄露」：
// password_hash / deleted_at / device_token_hash / audio_data 均通过 json:"-" 排除，
// 序列化导出结构后这些字段名及其值都不应出现。
func TestAccountExportJSONExcludesSensitiveFields(t *testing.T) {
	now := time.Now().UTC()
	deleted := now
	transcript := "转录文本"
	export := &AccountExport{
		User: model.User{
			ID: "u1", Email: "a@b.com", PasswordHash: "bcrypt-hash-value",
			Name: "Alice", CreatedAt: now, UpdatedAt: now, DeletedAt: &deleted,
		},
		Identities: []model.Identity{{ID: "i1", UserID: "u1", Name: "Work"}},
		Devices: []model.Device{{
			ID: "d1", UserID: "u1", DeviceID: "dev-1", Name: "手表",
			DeviceTokenHash: "device-token-hash-value",
		}},
		Sessions: []model.AudioSession{{
			ID: "s1", UserID: "u1", AudioData: []byte("top-secret-audio-bytes"),
			Transcript: &transcript, Status: model.StatusCompleted,
		}},
	}

	b, err := json.Marshal(export)
	require.NoError(t, err)
	body := string(b)

	// 字段名与敏感值都不应泄露。
	for _, forbidden := range []string{
		"password_hash", "bcrypt-hash-value",
		"deleted_at",
		"device_token_hash", "device-token-hash-value",
		"audio_data", "top-secret-audio-bytes",
	} {
		assert.NotContains(t, body, forbidden, "导出 JSON 不应包含 %q", forbidden)
	}

	// 非敏感字段仍应正常输出。
	assert.Contains(t, body, "a@b.com")
	assert.Contains(t, body, "Alice")
	assert.Contains(t, body, "转录文本")
}
