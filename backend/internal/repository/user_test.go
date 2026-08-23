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

func TestUserRepoCreate(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewUserRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users")).
		WithArgs("u1", "a@b.com", "hash", "Alice", nil, "{}").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), &model.User{
		ID: "u1", Email: "a@b.com", PasswordHash: "hash", Name: "Alice", Settings: "{}",
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoGetByEmail(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db, mock := newMockDB(t)
		repo := NewUserRepo(db)
		now := time.Now().UTC()

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
			WithArgs("a@b.com").
			WillReturnRows(sqlmock.NewRows(userCols).AddRow("u1", "a@b.com", "hash", "Alice", nil, "{}", now, now, nil))

		u, err := repo.GetByEmail(context.Background(), "a@b.com")
		require.NoError(t, err)
		require.NotNil(t, u)
		assert.Equal(t, "u1", u.ID)
		assert.Equal(t, "Alice", u.Name)
		assert.Equal(t, "hash", u.PasswordHash)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not_found", func(t *testing.T) {
		db, mock := newMockDB(t)
		repo := NewUserRepo(db)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
			WithArgs("missing@b.com").
			WillReturnRows(sqlmock.NewRows(userCols))

		u, err := repo.GetByEmail(context.Background(), "missing@b.com")
		require.NoError(t, err)
		assert.Nil(t, u)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepoGetByID(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewUserRepo(db)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(userCols).AddRow("u1", "a@b.com", "hash", "Alice", nil, "{}", now, now, nil))

	u, err := repo.GetByID(context.Background(), "u1")
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, "u1", u.ID)
}
