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

	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

var reminderCols = []string{"id", "user_id", "session_id", "identity_id", "type", "content", "due_at", "status", "created_at", "updated_at"}

func newReminderService(t *testing.T) (*ReminderService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewReminderService(repository.NewReminderRepo(db)), mock
}

func reminderRow(id, typ, content, status string) []driver.Value {
	now := time.Now().UTC()
	return []driver.Value{id, "u1", nil, nil, typ, content, nil, status, now, now}
}

func TestGenerateFromAnalysisTodosAndCommitments(t *testing.T) {
	svc, mock := newReminderService(t)
	i1 := "i1"
	result := &model.AnalysisResult{
		IdentityID: &i1,
		Confidence: 0.9,
		Extracted: model.ExtractedData{
			Todos:       []model.Todo{{Text: "买菜"}},
			Commitments: []model.Commitment{{Text: "帮小王改方案", From: "我", To: "小王"}},
			Notes:       []string{},
		},
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO reminders")).
		WithArgs(sqlmock.AnyArg(), "u1", "s1", "i1", "todo", "买菜", nil, "pending").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO reminders")).
		WithArgs(sqlmock.AnyArg(), "u1", "s1", "i1", "commitment", "我 → 小王：帮小王改方案", nil, "pending").
		WillReturnResult(sqlmock.NewResult(1, 1))

	got, err := svc.GenerateFromAnalysis(context.Background(), "u1", "s1", result, nil)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, model.ReminderTypeTodo, got[0].Type)
	assert.Equal(t, model.ReminderTypeCommitment, got[1].Type)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerateFromAnalysisIdentitySwitch(t *testing.T) {
	svc, mock := newReminderService(t)
	prev := "i1"
	i2 := "i2"
	result := &model.AnalysisResult{
		IdentityID: &i2,
		Confidence: 0.9,
		Extracted: model.ExtractedData{
			Todos:       []model.Todo{{Text: "买菜"}},
			Commitments: []model.Commitment{},
			Notes:       []string{},
		},
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO reminders")).
		WithArgs(sqlmock.AnyArg(), "u1", "s1", "i2", "todo", "买菜", nil, "pending").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, session_id")).
		WithArgs("u1", "i2", "todo", "pending").
		WillReturnRows(sqlmock.NewRows(reminderCols).AddRow(reminderRow("r1", "todo", "买菜", "pending")...))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO reminders")).
		WithArgs(sqlmock.AnyArg(), "u1", "s1", "i2", "identity_switch", "身份切换，上次未完成事项：买菜", nil, "pending").
		WillReturnResult(sqlmock.NewResult(1, 1))

	got, err := svc.GenerateFromAnalysis(context.Background(), "u1", "s1", result, &prev)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, model.ReminderTypeIdentitySwitch, got[1].Type)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerateFromAnalysisNoSwitchWhenSameIdentity(t *testing.T) {
	svc, mock := newReminderService(t)
	prev := "i2"
	i2 := "i2"
	result := &model.AnalysisResult{
		IdentityID: &i2,
		Extracted: model.ExtractedData{
			Todos:       []model.Todo{},
			Commitments: []model.Commitment{},
			Notes:       []string{},
		},
	}

	// 无 todo/commitment，身份相同 → 不产生任何提醒。
	got, err := svc.GenerateFromAnalysis(context.Background(), "u1", "s1", result, &prev)
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkDone(t *testing.T) {
	svc, mock := newReminderService(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, session_id")).
		WithArgs("r1", "u1").
		WillReturnRows(sqlmock.NewRows(reminderCols).AddRow(reminderRow("r1", "todo", "买菜", "pending")...))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE reminders SET status")).
		WithArgs("done", sqlmock.AnyArg(), "r1", "u1", "pending").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, svc.MarkDone(context.Background(), "u1", "r1"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDismiss(t *testing.T) {
	svc, mock := newReminderService(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, session_id")).
		WithArgs("r1", "u1").
		WillReturnRows(sqlmock.NewRows(reminderCols).AddRow(reminderRow("r1", "todo", "买菜", "pending")...))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE reminders SET status")).
		WithArgs("dismissed", sqlmock.AnyArg(), "r1", "u1", "pending").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, svc.Dismiss(context.Background(), "u1", "r1"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateStatusNonPendingBlocked(t *testing.T) {
	svc, mock := newReminderService(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, session_id")).
		WithArgs("r1", "u1").
		WillReturnRows(sqlmock.NewRows(reminderCols).AddRow(reminderRow("r1", "todo", "买菜", "done")...))

	err := svc.MarkDone(context.Background(), "u1", "r1")
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 40000, appErr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCommitmentContent(t *testing.T) {
	assert.Equal(t, "我 → 小王：改方案", commitmentContent(model.Commitment{Text: "改方案", From: "我", To: "小王"}))
	assert.Equal(t, "我：改方案", commitmentContent(model.Commitment{Text: "改方案", From: "我"}))
	assert.Equal(t, "改方案", commitmentContent(model.Commitment{Text: "改方案"}))
	assert.Equal(t, "", commitmentContent(model.Commitment{Text: "  "}))
}

func TestIdentitySwitchContent(t *testing.T) {
	assert.Contains(t, identitySwitchContent(nil), "暂无未完成待办")
	items := []model.Reminder{{Content: "买菜"}, {Content: "交报告"}}
	assert.Equal(t, "身份切换，上次未完成事项：买菜；交报告", identitySwitchContent(items))
}
