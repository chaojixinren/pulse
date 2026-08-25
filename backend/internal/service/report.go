package service

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

// IdentityStat 报告中的单个身份统计。
type IdentityStat struct {
	IdentityID    string `json:"identity_id"`
	Name          string `json:"name"`
	SessionCount  int    `json:"session_count"`
	TotalDuration int    `json:"total_duration"`
}

// DailyReport 日报结构。
type DailyReport struct {
	Date          string         `json:"date"`
	SessionCount  int            `json:"session_count"`
	TotalDuration int            `json:"total_duration"`
	ByIdentity    []IdentityStat `json:"by_identity"`
	Todos         []string       `json:"todos"`
	Notes         []string       `json:"notes"`
}

// DailyPoint 按天的一个数据点（图表用）。
type DailyPoint struct {
	Date          string `json:"date"`
	SessionCount  int    `json:"session_count"`
	TotalDuration int    `json:"total_duration"`
}

// WeeklyReport 周报结构。
type WeeklyReport struct {
	Week            string         `json:"week"`
	SessionCount    int            `json:"session_count"`
	TotalDuration   int            `json:"total_duration"`
	ByIdentity      []IdentityStat `json:"by_identity"`
	TopTodos        []string       `json:"top_todos"`
	CommitmentsDone int            `json:"commitments_done"`
	DailyTrend      []DailyPoint   `json:"daily_trend"`
}

// StatsReport 统计汇总（图表数据）。
type StatsReport struct {
	From          string         `json:"from"`
	To            string         `json:"to"`
	SessionCount  int            `json:"session_count"`
	TotalDuration int            `json:"total_duration"`
	ByIdentity    []IdentityStat `json:"by_identity"`
	DailyTrend    []DailyPoint   `json:"daily_trend"`
}

type ReportService struct {
	sessions   *repository.AudioSessionRepo
	identities *repository.IdentityRepo
}

func NewReportService(sessions *repository.AudioSessionRepo, identities *repository.IdentityRepo) *ReportService {
	return &ReportService{sessions: sessions, identities: identities}
}

// reportLocation 报告按 Asia/Shanghai 时区聚合；无法加载时回退 UTC。
func reportLocation() *time.Location {
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		return loc
	}
	return time.UTC
}

// identityStats 聚合区间内身份分布，并返回会话总数与总时长。
func (s *ReportService) identityStats(ctx context.Context, userID string, from, to time.Time) (int, int, []IdentityStat, error) {
	rows, err := s.sessions.StatsByUser(ctx, userID, from, to)
	if err != nil {
		return 0, 0, nil, apperrors.WrapInternal(err)
	}

	identityNames := map[string]string{}
	if ids, err := s.identities.ListByUser(ctx, userID); err == nil {
		for _, id := range ids {
			identityNames[id.ID] = id.Name
		}
	}

	var sessionCount, totalDuration int
	byIdentity := make([]IdentityStat, 0, len(rows))
	for _, row := range rows {
		sessionCount += row.SessionCount
		totalDuration += row.TotalDuration
		stat := IdentityStat{
			IdentityID:    row.IdentityID,
			Name:          identityNames[row.IdentityID],
			SessionCount:  row.SessionCount,
			TotalDuration: row.TotalDuration,
		}
		if stat.IdentityID == "" {
			stat.Name = "未分配"
		}
		byIdentity = append(byIdentity, stat)
	}
	return sessionCount, totalDuration, byIdentity, nil
}

func (s *ReportService) Daily(ctx context.Context, userID, date string) (*DailyReport, error) {
	d, err := time.ParseInLocation("2006-01-02", date, reportLocation())
	if err != nil {
		return nil, apperrors.NewBadRequest("日期格式应为 YYYY-MM-DD")
	}
	from := d
	to := d.AddDate(0, 0, 1)

	report := &DailyReport{
		Date:       d.Format("2006-01-02"),
		ByIdentity: make([]IdentityStat, 0),
		Todos:      make([]string, 0),
		Notes:      make([]string, 0),
	}

	sessionCount, totalDuration, byIdentity, err := s.identityStats(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	report.SessionCount = sessionCount
	report.TotalDuration = totalDuration
	report.ByIdentity = byIdentity
	return report, nil
}

// mondayOf 返回 t 所在周的周一（周一=1，周日=7）。
func mondayOf(t time.Time) time.Time {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	weekday := int(day.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return day.AddDate(0, 0, -(weekday - 1))
}

// reportAgg 聚合周报/统计共用的区间数据。
type reportAgg struct {
	sessionCount    int
	totalDuration   int
	byIdentity      []IdentityStat
	topTodos        []string
	commitmentsDone int
	dailyTrend      []DailyPoint
}

func (s *ReportService) aggregate(ctx context.Context, userID string, from, to time.Time) (*reportAgg, error) {
	sessionCount, totalDuration, byIdentity, err := s.identityStats(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}

	trendRows, err := s.sessions.DailyTrendByUser(ctx, userID, from, to)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	dailyTrend := make([]DailyPoint, 0, len(trendRows))
	for _, row := range trendRows {
		dailyTrend = append(dailyTrend, DailyPoint{
			Date:          row.Date,
			SessionCount:  row.SessionCount,
			TotalDuration: row.TotalDuration,
		})
	}

	extracted, err := s.sessions.ExtractedDataInRange(ctx, userID, from, to)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	topTodos, commitmentsDone := aggregateExtracted(extracted)

	return &reportAgg{
		sessionCount:    sessionCount,
		totalDuration:   totalDuration,
		byIdentity:      byIdentity,
		topTodos:        topTodos,
		commitmentsDone: commitmentsDone,
		dailyTrend:      dailyTrend,
	}, nil
}

// aggregateExtracted 从多份 extracted_data JSON 中聚合高频待办（前 5）与承诺条数。
func aggregateExtracted(extracted []string) ([]string, int) {
	todoCount := map[string]int{}
	commitments := 0
	for _, raw := range extracted {
		if raw == "" || raw == "{}" {
			continue
		}
		var d model.ExtractedData
		if err := json.Unmarshal([]byte(raw), &d); err != nil {
			continue
		}
		for _, todo := range d.Todos {
			if todo.Text != "" {
				todoCount[todo.Text]++
			}
		}
		commitments += len(d.Commitments)
	}

	type kv struct {
		text string
		n    int
	}
	list := make([]kv, 0, len(todoCount))
	for text, n := range todoCount {
		list = append(list, kv{text: text, n: n})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].n != list[j].n {
			return list[i].n > list[j].n
		}
		return list[i].text < list[j].text
	})

	top := make([]string, 0, len(list))
	for i, it := range list {
		if i >= 5 {
			break
		}
		top = append(top, it.text)
	}
	return top, commitments
}

// Weekly 计算以 week（YYYY-MM-DD）所在周一为起点的周报；week 为空时取当前周。
func (s *ReportService) Weekly(ctx context.Context, userID, week string) (*WeeklyReport, error) {
	loc := reportLocation()
	var d time.Time
	var err error
	if week == "" {
		d = time.Now().In(loc)
	} else {
		d, err = time.ParseInLocation("2006-01-02", week, loc)
		if err != nil {
			return nil, apperrors.NewBadRequest("周报日期格式应为 YYYY-MM-DD")
		}
	}
	monday := mondayOf(d)
	agg, err := s.aggregate(ctx, userID, monday, monday.AddDate(0, 0, 7))
	if err != nil {
		return nil, err
	}
	return &WeeklyReport{
		Week:            monday.Format("2006-01-02"),
		SessionCount:    agg.sessionCount,
		TotalDuration:   agg.totalDuration,
		ByIdentity:      agg.byIdentity,
		TopTodos:        agg.topTodos,
		CommitmentsDone: agg.commitmentsDone,
		DailyTrend:      agg.dailyTrend,
	}, nil
}

// Stats 汇总 [from, to]（含两端）区间内的统计图表数据。
func (s *ReportService) Stats(ctx context.Context, userID, from, to string) (*StatsReport, error) {
	loc := reportLocation()
	fromT, err := time.ParseInLocation("2006-01-02", from, loc)
	if err != nil {
		return nil, apperrors.NewBadRequest("起始日期格式应为 YYYY-MM-DD")
	}
	toT, err := time.ParseInLocation("2006-01-02", to, loc)
	if err != nil {
		return nil, apperrors.NewBadRequest("结束日期格式应为 YYYY-MM-DD")
	}
	if fromT.After(toT) {
		return nil, apperrors.NewBadRequest("起始日期不能晚于结束日期")
	}

	agg, err := s.aggregate(ctx, userID, fromT, toT.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	report := &StatsReport{
		From:          fromT.Format("2006-01-02"),
		To:            toT.Format("2006-01-02"),
		SessionCount:  agg.sessionCount,
		TotalDuration: agg.totalDuration,
		ByIdentity:    agg.byIdentity,
		DailyTrend:    agg.dailyTrend,
	}
	return report, nil
}
