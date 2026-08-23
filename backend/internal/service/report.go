package service

import (
	"context"
	"time"

	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

// IdentityStat 日报中的单个身份统计。
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

type ReportService struct {
	sessions   *repository.AudioSessionRepo
	identities *repository.IdentityRepo
}

func NewReportService(sessions *repository.AudioSessionRepo, identities *repository.IdentityRepo) *ReportService {
	return &ReportService{sessions: sessions, identities: identities}
}

// reportLocation 日报按 Asia/Shanghai 时区聚合；无法加载时回退 UTC。
func reportLocation() *time.Location {
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		return loc
	}
	return time.UTC
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

	rows, err := s.sessions.StatsByUser(ctx, userID, from, to)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}

	identityNames := map[string]string{}
	if ids, err := s.identities.ListByUser(ctx, userID); err == nil {
		for _, id := range ids {
			identityNames[id.ID] = id.Name
		}
	}

	for _, row := range rows {
		report.SessionCount += row.SessionCount
		report.TotalDuration += row.TotalDuration
		stat := IdentityStat{
			IdentityID:    row.IdentityID,
			Name:          identityNames[row.IdentityID],
			SessionCount:  row.SessionCount,
			TotalDuration: row.TotalDuration,
		}
		if stat.IdentityID == "" {
			stat.Name = "未分配"
		}
		report.ByIdentity = append(report.ByIdentity, stat)
	}
	return report, nil
}
