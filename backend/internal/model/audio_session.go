package model

import "time"

// 语音会话状态取值。
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

var validStatus = map[string]bool{
	StatusPending:    true,
	StatusProcessing: true,
	StatusCompleted:  true,
	StatusFailed:     true,
}

// IsValidStatus 判断状态字符串是否合法。
func IsValidStatus(s string) bool { return validStatus[s] }

// 状态机：pending → processing → completed；processing/pending → failed；failed → processing/pending（重试）。
var allowedTransitions = map[string]map[string]bool{
	StatusPending:    {StatusProcessing: true, StatusFailed: true},
	StatusProcessing: {StatusCompleted: true, StatusFailed: true},
	StatusFailed:     {StatusProcessing: true, StatusPending: true},
	StatusCompleted:  {},
}

// CanTransition 判断状态流转是否合法。
func CanTransition(from, to string) bool {
	toStates, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	return toStates[to]
}

// AudioSession 对应 audio_sessions 表。音频二进制直接存于 MySQL。
type AudioSession struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	IdentityID    *string    `json:"identity_id,omitempty"`
	DeviceID      *string    `json:"device_id,omitempty"`
	AudioData     []byte     `json:"-"`
	AudioMime     *string    `json:"audio_mime,omitempty"`
	Transcript    *string    `json:"transcript,omitempty"`
	Duration      int        `json:"duration"`
	FileSize      *int64     `json:"file_size,omitempty"`
	Status        string     `json:"status"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
	ExtractedData string     `json:"extracted_data,omitempty"`
	AIConfidence  *float64   `json:"ai_confidence,omitempty"`
	RecordedAt    time.Time  `json:"recorded_at"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
