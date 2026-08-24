package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

// mockAggQueries 为 Weekly/Stats 铺设 4 条聚合查询的期望（身份分布、身份列表、按天趋势、提取数据）。
func mockAggQueries(mock sqlmock.Sqlmock, userID string) {
	now := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(identity_id, '')")).
		WithArgs(userID, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"identity_id", "cnt", "total_duration"}).
			AddRow("i1", 2, 80))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows(identityCols).
			AddRow("i1", userID, "Work", nil, "#000000", "person", true, now, now, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DATE_FORMAT(recorded_at")).
		WithArgs(userID, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"d", "cnt", "total_duration"}).
			AddRow("2024-01-01", 1, 30).
			AddRow("2024-01-02", 1, 50))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT extracted_data")).
		WithArgs(userID, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"extracted_data"}).
			AddRow(`{"todos":[{"text":"买牛奶"},{"text":"写周报"}],"commitments":[{"text":"x","from":"a","to":"b"}],"notes":["n1"]}`).
			AddRow(`{"todos":[{"text":"买牛奶"}],"commitments":[],"notes":[]}`))
}

func TestReportWeeklyAggregates(t *testing.T) {
	svc, mock := newReportService(t)
	mockAggQueries(mock, "u1")

	report, err := svc.Weekly(context.Background(), "u1", "2024-01-02")
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, "2024-01-01", report.Week, "2024-01-02 为周二，周一起点为 2024-01-01")
	assert.Equal(t, 2, report.SessionCount)
	assert.Equal(t, 80, report.TotalDuration)

	require.Len(t, report.ByIdentity, 1)
	assert.Equal(t, "Work", report.ByIdentity[0].Name)

	assert.Equal(t, []string{"买牛奶", "写周报"}, report.TopTodos)
	assert.Equal(t, 1, report.CommitmentsDone)

	require.Len(t, report.DailyTrend, 2)
	assert.Equal(t, "2024-01-01", report.DailyTrend[0].Date)
	assert.Equal(t, 30, report.DailyTrend[0].TotalDuration)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReportWeeklyInvalidDate(t *testing.T) {
	svc, _ := newReportService(t)
	_, err := svc.Weekly(context.Background(), "u1", "2024-13-99")
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 40000, appErr.Code)
}

func TestReportStatsAggregates(t *testing.T) {
	svc, mock := newReportService(t)
	mockAggQueries(mock, "u1")

	report, err := svc.Stats(context.Background(), "u1", "2024-01-01", "2024-01-07")
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, "2024-01-01", report.From)
	assert.Equal(t, "2024-01-07", report.To)
	assert.Equal(t, 2, report.SessionCount)
	assert.Equal(t, 80, report.TotalDuration)
	require.Len(t, report.ByIdentity, 1)
	require.Len(t, report.DailyTrend, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReportStatsInvalidRange(t *testing.T) {
	svc, _ := newReportService(t)
	_, err := svc.Stats(context.Background(), "u1", "2024-01-08", "2024-01-01")
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 40000, appErr.Code)
}

func TestMondayOf(t *testing.T) {
	loc := time.UTC
	// 2024-01-02 是周二 → 周一为 2024-01-01。
	d := time.Date(2024, 1, 2, 15, 0, 0, 0, loc)
	assert.Equal(t, "2024-01-01", mondayOf(d).Format("2006-01-02"))
	// 周日 2024-01-07 → 周一 2024-01-01。
	sun := time.Date(2024, 1, 7, 0, 0, 0, 0, loc)
	assert.Equal(t, "2024-01-01", mondayOf(sun).Format("2006-01-02"))
	// 周一本身不变。
	mon := time.Date(2024, 1, 1, 0, 0, 0, 0, loc)
	assert.Equal(t, "2024-01-01", mondayOf(mon).Format("2006-01-02"))
}

func TestAggregateExtracted(t *testing.T) {
	rows := []string{
		`{"todos":[{"text":"A"},{"text":"B"}],"commitments":[{"text":"c","from":"x","to":"y"}]}`,
		`{"todos":[{"text":"A"}],"commitments":[],"notes":["n"]}`,
		"{}",
		"",
		"not-json",
	}
	top, commitments := aggregateExtracted(rows)
	assert.Equal(t, []string{"A", "B"}, top, "A 出现 2 次排前，B 出现 1 次")
	assert.Equal(t, 1, commitments, "仅第一份数据含 1 条承诺")
}
