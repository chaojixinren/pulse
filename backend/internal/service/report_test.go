package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

func newReportService(t *testing.T) (*ReportService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewReportService(repository.NewAudioSessionRepo(db), repository.NewIdentityRepo(db), nil), mock
}

func TestReportLocation(t *testing.T) {
	loc := reportLocation()
	require.NotNil(t, loc)
	d, err := time.ParseInLocation("2006-01-02", "2024-01-02", loc)
	require.NoError(t, err)
	assert.Equal(t, 2024, d.Year())
	assert.Equal(t, time.January, d.Month())
	assert.Equal(t, 2, d.Day())
}

func TestReportDailyInvalidDate(t *testing.T) {
	svc, _ := newReportService(t)

	_, err := svc.Daily(context.Background(), "u1", "2024-13-99")
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 40000, appErr.Code)
}

func TestReportDailyAggregates(t *testing.T) {
	svc, mock := newReportService(t)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(identity_id, '')")).
		WithArgs("u1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"identity_id", "cnt", "total_duration"}).
			AddRow("i1", 3, 120).
			AddRow("", 1, 30))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(identityCols).
			AddRow("i1", "u1", "Work", nil, "#000000", "person", true, now, now, nil))

	report, err := svc.Daily(context.Background(), "u1", "2024-01-02")
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, "2024-01-02", report.Date)
	assert.Equal(t, 4, report.SessionCount, "会话数应累加")
	assert.Equal(t, 150, report.TotalDuration, "总时长应累加")

	require.Len(t, report.ByIdentity, 2)
	assert.Equal(t, "i1", report.ByIdentity[0].IdentityID)
	assert.Equal(t, "Work", report.ByIdentity[0].Name, "身份名应映射")
	assert.Equal(t, 3, report.ByIdentity[0].SessionCount)
	assert.Equal(t, "未分配", report.ByIdentity[1].Name, "无身份应标记为未分配")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReportDailyEmptyReturnsEmptyReport(t *testing.T) {
	svc, mock := newReportService(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(identity_id, '')")).
		WithArgs("u1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"identity_id", "cnt", "total_duration"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(identityCols))

	report, err := svc.Daily(context.Background(), "u1", "2024-01-02")
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, 0, report.SessionCount)
	assert.Equal(t, 0, report.TotalDuration)
	assert.NotNil(t, report.ByIdentity)
	assert.Empty(t, report.ByIdentity)
	assert.NotNil(t, report.Todos)
	assert.NotNil(t, report.Notes)
	assert.NoError(t, mock.ExpectationsWereMet())
}
