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
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

func TestAudioSessionRepoCreate(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewAudioSessionRepo(db)
	now := time.Now().UTC()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audio_sessions")).
		WithArgs("s1", "u1", nil, nil, []byte("audio"), nil, nil, 10, int64(5), "pending", nil, "{}", nil, now, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), &model.AudioSession{
		ID: "s1", UserID: "u1", AudioData: []byte("audio"), Duration: 10,
		FileSize: int64Ptr(5), Status: model.StatusPending, ExtractedData: "{}", RecordedAt: now,
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAudioSessionRepoGetByID(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db, mock := newMockDB(t)
		repo := NewAudioSessionRepo(db)
		now := time.Now().UTC()

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, identity_id, device_id, audio_data")).
			WithArgs("s1").
			WillReturnRows(sqlmock.NewRows(sessionCols).
				AddRow("s1", "u1", nil, nil, []byte("audio"), "audio/wav", nil, 10, int64(5), "pending", nil, "{}", nil, now, nil, now, now))

		s, err := repo.GetByID(context.Background(), "s1")
		require.NoError(t, err)
		require.NotNil(t, s)
		assert.Equal(t, []byte("audio"), s.AudioData)
		assert.Equal(t, model.StatusPending, s.Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not_found", func(t *testing.T) {
		db, mock := newMockDB(t)
		repo := NewAudioSessionRepo(db)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, identity_id, device_id, audio_data")).
			WithArgs("missing").
			WillReturnRows(sqlmock.NewRows(sessionCols))

		s, err := repo.GetByID(context.Background(), "missing")
		require.NoError(t, err)
		assert.Nil(t, s)
	})
}

func TestAudioSessionRepoGetByIDAndUser(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewAudioSessionRepo(db)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, identity_id, device_id, audio_mime")).
		WithArgs("s1", "u1").
		WillReturnRows(sqlmock.NewRows(sessionListCols).
			AddRow("s1", "u1", nil, nil, "audio/wav", nil, 10, int64(5), "pending", nil, "{}", nil, now, nil, now, now))

	s, err := repo.GetByIDAndUser(context.Background(), "s1", "u1")
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, "s1", s.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAudioSessionRepoUpdateStatusSuccess(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewAudioSessionRepo(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM audio_sessions WHERE id = ? FOR UPDATE")).
		WithArgs("s1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("pending"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET status = ?, error_message = ?")).
		WithArgs("processing", nil, sqlmock.AnyArg(), "processing", "completed", sqlmock.AnyArg(), "s1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.UpdateStatus(context.Background(), "s1", "processing", ""))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAudioSessionRepoUpdateStatusIllegalTransition(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewAudioSessionRepo(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM audio_sessions WHERE id = ? FOR UPDATE")).
		WithArgs("s1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("completed"))
	mock.ExpectRollback()

	err := repo.UpdateStatus(context.Background(), "s1", "failed", "boom")
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok, "非法流转应返回 *AppError")
	assert.Equal(t, 40000, appErr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAudioSessionRepoUpdateStatusNotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewAudioSessionRepo(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM audio_sessions WHERE id = ? FOR UPDATE")).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"status"}))
	mock.ExpectRollback()

	err := repo.UpdateStatus(context.Background(), "missing", "processing", "")
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 40400, appErr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAudioSessionRepoUpdateTranscript(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewAudioSessionRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET transcript")).
		WithArgs("hello", sqlmock.AnyArg(), "s1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateTranscript(context.Background(), "s1", "hello"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAudioSessionRepoClaimProcessing(t *testing.T) {
	t.Run("claimed", func(t *testing.T) {
		db, mock := newMockDB(t)
		repo := NewAudioSessionRepo(db)

		mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET status = ?")).
			WithArgs("processing", sqlmock.AnyArg(), "s1", "pending", "processing").
			WillReturnResult(sqlmock.NewResult(0, 1))

		claimed, err := repo.ClaimProcessing(context.Background(), "s1")
		require.NoError(t, err)
		assert.True(t, claimed)
	})

	t.Run("already_claimed", func(t *testing.T) {
		db, mock := newMockDB(t)
		repo := NewAudioSessionRepo(db)

		mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET status = ?")).
			WithArgs("processing", sqlmock.AnyArg(), "s1", "pending", "processing").
			WillReturnResult(sqlmock.NewResult(0, 0))

		claimed, err := repo.ClaimProcessing(context.Background(), "s1")
		require.NoError(t, err)
		assert.False(t, claimed)
	})
}

func TestAudioSessionRepoListProcessable(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		db, mock := newMockDB(t)
		repo := NewAudioSessionRepo(db)

		mock.ExpectQuery(regexp.QuoteMeta("FROM audio_sessions WHERE status IN (?, ?) ORDER BY created_at ASC LIMIT ?")).
			WithArgs("pending", "processing", 5).
			WillReturnRows(sqlmock.NewRows(sessionCols))

		list, err := repo.ListProcessable(context.Background(), 5)
		require.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("non_empty", func(t *testing.T) {
		db, mock := newMockDB(t)
		repo := NewAudioSessionRepo(db)
		now := time.Now().UTC()

		mock.ExpectQuery(regexp.QuoteMeta("FROM audio_sessions WHERE status IN (?, ?) ORDER BY created_at ASC LIMIT ?")).
			WithArgs("pending", "processing", 5).
			WillReturnRows(sqlmock.NewRows(sessionCols).
				AddRow("s1", "u1", nil, nil, []byte("a"), "audio/wav", nil, 1, int64(1), "pending", nil, "{}", nil, now, nil, now, now))

		list, err := repo.ListProcessable(context.Background(), 5)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, "s1", list[0].ID)
		assert.Equal(t, []byte("a"), list[0].AudioData)
	})
}

func TestAudioSessionRepoListByUser(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		db, mock := newMockDB(t)
		repo := NewAudioSessionRepo(db)
		now := time.Now().UTC()

		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM audio_sessions")).
			WithArgs("u1").
			WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(1)))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, identity_id, device_id, audio_mime")).
			WithArgs("u1", 20, 0).
			WillReturnRows(sqlmock.NewRows(sessionListCols).
				AddRow("s1", "u1", nil, nil, "audio/wav", "hello", 10, int64(5), "completed", nil, "{}", nil, now, nil, now, now))

		list, total, err := repo.ListByUser(context.Background(), "u1", SessionFilter{}, 1, 20)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, int64(1), total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("filtered", func(t *testing.T) {
		db, mock := newMockDB(t)
		repo := NewAudioSessionRepo(db)
		identityID := "i1"
		status := "completed"

		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM audio_sessions")).
			WithArgs("u1", "i1", "completed").
			WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(0)))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, identity_id, device_id, audio_mime")).
			WithArgs("u1", "i1", "completed", 20, 0).
			WillReturnRows(sqlmock.NewRows(sessionListCols))

		list, total, err := repo.ListByUser(context.Background(), "u1", SessionFilter{
			IdentityID: &identityID,
			Status:     &status,
		}, 1, 20)
		require.NoError(t, err)
		assert.Empty(t, list)
		assert.Equal(t, int64(0), total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAudioSessionRepoStatsByUser(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewAudioSessionRepo(db)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 1)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(identity_id, '')")).
		WithArgs("u1", from, to).
		WillReturnRows(sqlmock.NewRows([]string{"identity_id", "cnt", "total_duration"}).
			AddRow("i1", 3, 120).
			AddRow("", 1, 30))

	rows, err := repo.StatsByUser(context.Background(), "u1", from, to)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "i1", rows[0].IdentityID)
	assert.Equal(t, 3, rows[0].SessionCount)
	assert.Equal(t, 120, rows[0].TotalDuration)
	assert.Equal(t, "", rows[1].IdentityID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func int64Ptr(v int64) *int64 { return &v }
