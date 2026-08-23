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

func TestIdentityRepoCreate(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewIdentityRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO identities")).
		WithArgs("i1", "u1", "Work", nil, "#000000", "person", false).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), &model.Identity{ID: "i1", UserID: "u1", Name: "Work", Color: "#000000", Icon: "person"})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIdentityRepoGetByID(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewIdentityRepo(db)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("i1", "u1").
		WillReturnRows(sqlmock.NewRows(identityCols).AddRow("i1", "u1", "Work", nil, "#000000", "person", true, now, now, nil))

	i, err := repo.GetByID(context.Background(), "i1", "u1")
	require.NoError(t, err)
	require.NotNil(t, i)
	assert.Equal(t, "Work", i.Name)
	assert.True(t, i.IsDefault)
}

func TestIdentityRepoListByUser(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewIdentityRepo(db)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(identityCols).
			AddRow("i1", "u1", "Work", nil, "#000000", "person", true, now, now, nil).
			AddRow("i2", "u1", "Home", nil, "#111111", "home", false, now, now, nil))

	list, err := repo.ListByUser(context.Background(), "u1")
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "i1", list[0].ID, "默认身份应排在最前")
}

func TestIdentityRepoCountByUser(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewIdentityRepo(db)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM identities")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(3))

	n, err := repo.CountByUser(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, 3, n)
}

func TestIdentityRepoGetDefault(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewIdentityRepo(db)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(identityCols).AddRow("i1", "u1", "Work", nil, "#000000", "person", true, now, now, nil))

	i, err := repo.GetDefault(context.Background(), "u1")
	require.NoError(t, err)
	require.NotNil(t, i)
	assert.True(t, i.IsDefault)
}

func TestIdentityRepoUpdate(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewIdentityRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE identities SET name")).
		WithArgs("Work2", nil, "#222222", "person", sqlmock.AnyArg(), "i1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(context.Background(), &model.Identity{ID: "i1", UserID: "u1", Name: "Work2", Color: "#222222", Icon: "person"})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIdentityRepoSetDefault(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewIdentityRepo(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE identities SET is_default = FALSE")).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE identities SET is_default = TRUE")).
		WithArgs(sqlmock.AnyArg(), "i2", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.SetDefault(context.Background(), "u1", "i2"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIdentityRepoDelete(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewIdentityRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE identities SET deleted_at")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "i1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Delete(context.Background(), "i1", "u1"))
	assert.NoError(t, mock.ExpectationsWereMet())
}
