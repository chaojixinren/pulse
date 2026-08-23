package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/chaojixinren/pulse/internal/model"
)

type ReminderRepo struct {
	db *sql.DB
}

func NewReminderRepo(db *sql.DB) *ReminderRepo { return &ReminderRepo{db: db} }

const reminderColumns = "id, user_id, session_id, identity_id, type, content, due_at, status, created_at, updated_at"

func scanReminder(row interface{ Scan(...interface{}) error }) (*model.Reminder, error) {
	var r model.Reminder
	if err := row.Scan(&r.ID, &r.UserID, &r.SessionID, &r.IdentityID, &r.Type, &r.Content, &r.DueAt, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *ReminderRepo) Create(ctx context.Context, m *model.Reminder) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO reminders (id, user_id, session_id, identity_id, type, content, due_at, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.UserID, m.SessionID, m.IdentityID, m.Type, m.Content, m.DueAt, m.Status)
	return err
}

func (r *ReminderRepo) GetByID(ctx context.Context, id, userID string) (*model.Reminder, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+reminderColumns+` FROM reminders WHERE id = ? AND user_id = ?`, id, userID)
	m, err := scanReminder(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

// ListByUser 列出用户提醒；status 为空时不过滤状态。
func (r *ReminderRepo) ListByUser(ctx context.Context, userID, status string) ([]model.Reminder, error) {
	query := `SELECT ` + reminderColumns + ` FROM reminders WHERE user_id = ?`
	args := []interface{}{userID}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.Reminder, 0)
	for rows.Next() {
		m, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

// UpdateStatus 仅允许 pending → done/dismissed，其它状态不更新。
func (r *ReminderRepo) UpdateStatus(ctx context.Context, id, userID, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE reminders SET status = ?, updated_at = ? WHERE id = ? AND user_id = ? AND status = ?`,
		status, time.Now().UTC(), id, userID, model.ReminderStatusPending)
	return err
}

// ListPendingTodosByIdentity 返回某身份下未完成的待办提醒（身份切换提醒内容引用）。
func (r *ReminderRepo) ListPendingTodosByIdentity(ctx context.Context, userID, identityID string) ([]model.Reminder, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+reminderColumns+` FROM reminders WHERE user_id = ? AND identity_id = ? AND type = ? AND status = ? ORDER BY created_at ASC`,
		userID, identityID, model.ReminderTypeTodo, model.ReminderStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.Reminder, 0)
	for rows.Next() {
		m, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}
