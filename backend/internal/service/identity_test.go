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

var identityCols = []string{"id", "user_id", "name", "description", "color", "icon", "is_default", "created_at", "updated_at", "deleted_at"}

func newIdentityService(t *testing.T) (*IdentityService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewIdentityService(repository.NewIdentityRepo(db)), mock
}

func identityRow(id, name string, isDefault bool) ([]string, []driver.Value) {
	now := time.Now().UTC()
	return identityCols, []driver.Value{id, "u1", name, nil, "#000000", "person", isDefault, now, now, nil}
}

func TestIdentityCreateEmptyName(t *testing.T) {
	svc, _ := newIdentityService(t)

	_, err := svc.Create(context.Background(), "u1", IdentityInput{Name: "   "})
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 40000, appErr.Code)
}

func TestIdentityCreateFirstBecomesDefault(t *testing.T) {
	svc, mock := newIdentityService(t)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO identities")).
		WithArgs(sqlmock.AnyArg(), "u1", "Work", nil, "#000000", "person", false).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM identities")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE identities SET is_default = FALSE")).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE identities SET is_default = TRUE")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	cols, row := identityRow("i1", "Work", true)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(row...))

	got, err := svc.Create(context.Background(), "u1", IdentityInput{Name: "Work"})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Work", got.Name)
	assert.True(t, got.IsDefault, "首个身份应自动成为默认身份")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIdentityUpdate(t *testing.T) {
	svc, mock := newIdentityService(t)
	cols, row := identityRow("i1", "Work", false)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("i1", "u1").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(row...))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE identities SET name")).
		WithArgs("Home", nil, "#222222", "home", sqlmock.AnyArg(), "i1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	_, row2 := identityRow("i1", "Home", false)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("i1", "u1").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(row2...))

	got, err := svc.Update(context.Background(), "u1", "i1", IdentityInput{Name: "Home", Color: "#222222", Icon: "home"})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Home", got.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIdentityDeleteDefaultBlocked(t *testing.T) {
	svc, mock := newIdentityService(t)
	cols, row := identityRow("i1", "Work", true)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("i1", "u1").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(row...))

	err := svc.Delete(context.Background(), "u1", "i1")
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 40000, appErr.Code, "默认身份应禁止删除")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIdentityDeleteNonDefault(t *testing.T) {
	svc, mock := newIdentityService(t)
	cols, row := identityRow("i2", "Home", false)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("i2", "u1").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(row...))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE identities SET deleted_at")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "i2", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, svc.Delete(context.Background(), "u1", "i2"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIdentitySetDefault(t *testing.T) {
	svc, mock := newIdentityService(t)
	cols, row := identityRow("i2", "Home", false)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("i2", "u1").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(row...))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE identities SET is_default = FALSE")).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE identities SET is_default = TRUE")).
		WithArgs(sqlmock.AnyArg(), "i2", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, svc.SetDefault(context.Background(), "u1", "i2"))
	assert.NoError(t, mock.ExpectationsWereMet())
}
