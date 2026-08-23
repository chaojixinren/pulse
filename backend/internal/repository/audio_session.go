package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/chaojixinren/pulse/internal/model"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

type AudioSessionRepo struct {
	db *sql.DB
}

func NewAudioSessionRepo(db *sql.DB) *AudioSessionRepo { return &AudioSessionRepo{db: db} }

// sessionColumns 含音频二进制数据（用于需要读取音频的场景）。
const sessionColumns = "id, user_id, identity_id, device_id, audio_data, audio_mime, transcript, duration, file_size, status, error_message, extracted_data, ai_confidence, recorded_at, processed_at, created_at, updated_at"

// sessionListColumns 不含音频二进制数据（用于列表/摘要场景）。
const sessionListColumns = "id, user_id, identity_id, device_id, audio_mime, transcript, duration, file_size, status, error_message, extracted_data, ai_confidence, recorded_at, processed_at, created_at, updated_at"

func scanSession(row interface{ Scan(...interface{}) error }) (*model.AudioSession, error) {
	var s model.AudioSession
	err := row.Scan(
		&s.ID, &s.UserID, &s.IdentityID, &s.DeviceID, &s.AudioData, &s.AudioMime, &s.Transcript, &s.Duration,
		&s.FileSize, &s.Status, &s.ErrorMessage, &s.ExtractedData, &s.AIConfidence,
		&s.RecordedAt, &s.ProcessedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func scanSessionSummary(row interface{ Scan(...interface{}) error }) (*model.AudioSession, error) {
	var s model.AudioSession
	err := row.Scan(
		&s.ID, &s.UserID, &s.IdentityID, &s.DeviceID, &s.AudioMime, &s.Transcript, &s.Duration,
		&s.FileSize, &s.Status, &s.ErrorMessage, &s.ExtractedData, &s.AIConfidence,
		&s.RecordedAt, &s.ProcessedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *AudioSessionRepo) Create(ctx context.Context, s *model.AudioSession) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audio_sessions
			(id, user_id, identity_id, device_id, audio_data, audio_mime, transcript, duration, file_size, status, error_message, extracted_data, ai_confidence, recorded_at, processed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.UserID, s.IdentityID, s.DeviceID, s.AudioData, s.AudioMime, s.Transcript, s.Duration,
		s.FileSize, s.Status, s.ErrorMessage, s.ExtractedData, s.AIConfidence, s.RecordedAt, s.ProcessedAt)
	return err
}

func (r *AudioSessionRepo) GetByID(ctx context.Context, id string) (*model.AudioSession, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+sessionColumns+` FROM audio_sessions WHERE id = ?`, id)
	s, err := scanSession(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func (r *AudioSessionRepo) GetByIDAndUser(ctx context.Context, id, userID string) (*model.AudioSession, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+sessionListColumns+` FROM audio_sessions WHERE id = ? AND user_id = ?`, id, userID)
	s, err := scanSessionSummary(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

// UpdateStatus 按状态机校验后进行流转，并更新 updated_at / processed_at。
func (r *AudioSessionRepo) UpdateStatus(ctx context.Context, id, status, errMsg string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var current string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM audio_sessions WHERE id = ? FOR UPDATE`, id).Scan(&current); err != nil {
		if err == sql.ErrNoRows {
			return apperrors.NewNotFound("语音会话不存在")
		}
		return err
	}
	if !model.CanTransition(current, status) {
		return apperrors.NewBadRequest(fmt.Sprintf("非法的状态流转: %s -> %s", current, status))
	}

	var errMsgVal interface{}
	if errMsg == "" {
		errMsgVal = nil
	} else {
		errMsgVal = errMsg
	}

	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx,
		`UPDATE audio_sessions
		 SET status = ?, error_message = ?, updated_at = ?,
		     processed_at = IF(? = ?, ?, processed_at)
		 WHERE id = ?`,
		status, errMsgVal, now, status, model.StatusCompleted, now, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *AudioSessionRepo) UpdateTranscript(ctx context.Context, id, transcript string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE audio_sessions SET transcript = ?, updated_at = ? WHERE id = ?`,
		transcript, time.Now().UTC(), id)
	return err
}

// ClaimProcessing 原子地将 pending 会话置为 processing，避免多 worker 重复处理。
func (r *AudioSessionRepo) ClaimProcessing(ctx context.Context, id string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE audio_sessions SET status = ?, updated_at = ? WHERE id = ? AND status = ?`,
		model.StatusProcessing, time.Now().UTC(), id, model.StatusPending)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *AudioSessionRepo) ListPending(ctx context.Context, limit int) ([]model.AudioSession, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+sessionColumns+` FROM audio_sessions WHERE status = ? ORDER BY created_at ASC LIMIT ?`,
		model.StatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.AudioSession, 0)
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

// SessionFilter 时间线 / 列表过滤条件。
type SessionFilter struct {
	IdentityID *string
	From       *time.Time
	To         *time.Time
	Status     *string
}

func (r *AudioSessionRepo) ListByUser(ctx context.Context, userID string, f SessionFilter, page, size int) ([]model.AudioSession, int64, error) {
	where := []string{"user_id = ?"}
	args := []interface{}{userID}

	if f.IdentityID != nil && *f.IdentityID != "" {
		where = append(where, "identity_id = ?")
		args = append(args, *f.IdentityID)
	}
	if f.Status != nil && *f.Status != "" {
		where = append(where, "status = ?")
		args = append(args, *f.Status)
	}
	if f.From != nil {
		where = append(where, "recorded_at >= ?")
		args = append(args, *f.From)
	}
	if f.To != nil {
		where = append(where, "recorded_at < ?")
		args = append(args, *f.To)
	}
	whereClause := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM audio_sessions WHERE `+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	query := `SELECT ` + sessionListColumns + ` FROM audio_sessions WHERE ` + whereClause + ` ORDER BY recorded_at DESC LIMIT ? OFFSET ?`
	listArgs := append(append([]interface{}{}, args...), size, offset)

	rows, err := r.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]model.AudioSession, 0)
	for rows.Next() {
		s, err := scanSessionSummary(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *s)
	}
	return out, total, rows.Err()
}

// IdentityStatRow 日报聚合的一行结果。
type IdentityStatRow struct {
	IdentityID    string
	SessionCount  int
	TotalDuration int
}

func (r *AudioSessionRepo) StatsByUser(ctx context.Context, userID string, from, to time.Time) ([]IdentityStatRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT COALESCE(identity_id, '') AS identity_id,
		       COUNT(*) AS cnt,
		       COALESCE(CAST(SUM(duration) AS SIGNED), 0) AS total_duration
		FROM audio_sessions
		WHERE user_id = ? AND recorded_at >= ? AND recorded_at < ?
		GROUP BY identity_id
		ORDER BY cnt DESC`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]IdentityStatRow, 0)
	for rows.Next() {
		var r IdentityStatRow
		if err := rows.Scan(&r.IdentityID, &r.SessionCount, &r.TotalDuration); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
