package service

import (
	"context"
	"time"

	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

// TimelineItem 时间线条目。
type TimelineItem struct {
	SessionID  string    `json:"session_id"`
	IdentityID string    `json:"identity_id,omitempty"`
	Transcript string    `json:"transcript"`
	Duration   int       `json:"duration"`
	Status     string    `json:"status"`
	RecordedAt time.Time `json:"recorded_at"`
}

// TimelineFilter 时间线过滤条件。
type TimelineFilter struct {
	IdentityID *string
	From       *time.Time
	To         *time.Time
	Status     *string
}

type TimelineService struct {
	sessions *repository.AudioSessionRepo
}

func NewTimelineService(sessions *repository.AudioSessionRepo) *TimelineService {
	return &TimelineService{sessions: sessions}
}

func (s *TimelineService) List(ctx context.Context, userID string, f TimelineFilter, page, size int) ([]TimelineItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	sessions, total, err := s.sessions.ListByUser(ctx, userID, repository.SessionFilter{
		IdentityID: f.IdentityID,
		From:       f.From,
		To:         f.To,
		Status:     f.Status,
	}, page, size)
	if err != nil {
		return nil, 0, apperrors.WrapInternal(err)
	}

	items := make([]TimelineItem, 0, len(sessions))
	for _, sess := range sessions {
		item := TimelineItem{
			SessionID:  sess.ID,
			Duration:   sess.Duration,
			Status:     sess.Status,
			RecordedAt: sess.RecordedAt,
		}
		if sess.IdentityID != nil {
			item.IdentityID = *sess.IdentityID
		}
		if sess.Transcript != nil {
			item.Transcript = *sess.Transcript
		}
		items = append(items, item)
	}
	return items, total, nil
}
