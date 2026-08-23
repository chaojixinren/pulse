package model

import "time"

// 提醒类型取值。
const (
	ReminderTypeTodo           = "todo"
	ReminderTypeCommitment     = "commitment"
	ReminderTypeIdentitySwitch = "identity_switch"
)

// 提醒状态取值。
const (
	ReminderStatusPending   = "pending"
	ReminderStatusDone      = "done"
	ReminderStatusDismissed = "dismissed"
)

// Reminder 对应 reminders 表。
type Reminder struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	SessionID  *string    `json:"session_id,omitempty"`
	IdentityID *string    `json:"identity_id,omitempty"`
	Type       string     `json:"type"`
	Content    string     `json:"content"`
	DueAt      *time.Time `json:"due_at,omitempty"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
