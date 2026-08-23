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

func TestRefreshTokenRepoCreate(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewRefreshTokenRepo(db)
	expires := time.Now().UTC().Add(7 * 24 * time.Hour)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO refresh_tokens")).
		WithArgs("t1", "u1", "hash", expires).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), &model.RefreshToken{ID: "t1", UserID: "u1", TokenHash: "hash", ExpiresAt: expires})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepoGetByHash(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db, mock := newMockDB(t)
		repo := NewRefreshTokenRepo(db)
		now := time.Now().UTC()
		expires := now.Add(time.Hour)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, token_hash")).
			WithArgs("hash").
			WillReturnRows(sqlmock.NewRows(refreshTokenCols).AddRow("t1", "u1", "hash", expires, now, nil))

		tok, err := repo.GetByHash(context.Background(), "hash")
		require.NoError(t, err)
		require.NotNil(t, tok)
		assert.Equal(t, "u1", tok.UserID)
		assert.Nil(t, tok.RevokedAt)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not_found", func(t *testing.T) {
		db, mock := newMockDB(t)
		repo := NewRefreshTokenRepo(db)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, token_hash")).
			WithArgs("missing").
			WillReturnRows(sqlmock.NewRows(refreshTokenCols))

		tok, err := repo.GetByHash(context.Background(), "missing")
		require.NoError(t, err)
		assert.Nil(t, tok)
	})
}

func TestRefreshTokenRepoRevoke(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewRefreshTokenRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE refresh_tokens SET revoked_at")).
		WithArgs(sqlmock.AnyArg(), "hash").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Revoke(context.Background(), "hash"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepoRevokeAllForUser(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewRefreshTokenRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE refresh_tokens SET revoked_at")).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnResult(sqlmock.NewResult(0, 2))

	require.NoError(t, repo.RevokeAllForUser(context.Background(), "u1"))
	assert.NoError(t, mock.ExpectationsWereMet())
}
