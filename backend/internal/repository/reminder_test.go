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

var reminderRepoCols = []string{"id", "user_id", "session_id", "identity_id", "type", "content", "due_at", "status", "created_at", "updated_at"}

func TestReminderRepoCreate(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewReminderRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO reminders")).
		WithArgs("r1", "u1", nil, nil, "todo", "买菜", nil, "pending").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), &model.Reminder{ID: "r1", UserID: "u1", Type: model.ReminderTypeTodo, Content: "买菜", Status: model.ReminderStatusPending})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReminderRepoListByUser(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewReminderRepo(db)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, session_id")).
		WithArgs("u1", "pending").
		WillReturnRows(sqlmock.NewRows(reminderRepoCols).AddRow("r1", "u1", nil, nil, "todo", "买菜", nil, "pending", now, now))

	list, err := repo.ListByUser(context.Background(), "u1", model.ReminderStatusPending)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "r1", list[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReminderRepoUpdateStatus(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewReminderRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE reminders SET status")).
		WithArgs("done", sqlmock.AnyArg(), "r1", "u1", "pending").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateStatus(context.Background(), "r1", "u1", model.ReminderStatusDone))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReminderRepoListPendingTodosByIdentity(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewReminderRepo(db)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, session_id")).
		WithArgs("u1", "i2", "todo", "pending").
		WillReturnRows(sqlmock.NewRows(reminderRepoCols).AddRow("r1", "u1", nil, "i2", "todo", "买菜", nil, "pending", now, now))

	list, err := repo.ListPendingTodosByIdentity(context.Background(), "u1", "i2")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, model.ReminderTypeTodo, list[0].Type)
	assert.NoError(t, mock.ExpectationsWereMet())
}
