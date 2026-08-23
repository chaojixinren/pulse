package repository

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// newMockDB 返回由 sqlmock 驱动的 *sql.DB，用于无真实 MySQL 的仓库/服务层测试。
func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// 各表列名，与仓库层 SELECT 顺序保持一致。
var (
	userCols         = []string{"id", "email", "password_hash", "name", "avatar_url", "settings", "created_at", "updated_at", "deleted_at"}
	refreshTokenCols = []string{"id", "user_id", "token_hash", "expires_at", "created_at", "revoked_at"}
	identityCols     = []string{"id", "user_id", "name", "description", "color", "icon", "is_default", "created_at", "updated_at", "deleted_at"}
	sessionCols      = []string{"id", "user_id", "identity_id", "device_id", "audio_data", "audio_mime", "transcript", "duration", "file_size", "status", "error_message", "extracted_data", "ai_confidence", "recorded_at", "processed_at", "created_at", "updated_at"}
	sessionListCols  = []string{"id", "user_id", "identity_id", "device_id", "audio_mime", "transcript", "duration", "file_size", "status", "error_message", "extracted_data", "ai_confidence", "recorded_at", "processed_at", "created_at", "updated_at"}
)
