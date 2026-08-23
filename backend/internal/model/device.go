package model

import "time"

// Device 对应 devices 表。
type Device struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	DeviceID        string     `json:"device_id"`
	Name            string     `json:"name"`
	DeviceType      string     `json:"device_type"`
	FirmwareVersion *string    `json:"firmware_version,omitempty"`
	BatteryLevel    *int       `json:"battery_level,omitempty"`
	LastSeenAt      *time.Time `json:"last_seen_at,omitempty"`
	IsActive        bool       `json:"is_active"`
	DeviceTokenHash string     `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// DeviceBindCode 对应 device_bind_codes 表（一次性绑定码）。
type DeviceBindCode struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Code      string     `json:"code"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// DeviceCommand 对应 device_commands 表（下发给硬件的指令，先落库）。
type DeviceCommand struct {
	ID        string    `json:"id"`
	DeviceID  string    `json:"device_id"`
	UserID    string    `json:"user_id"`
	Command   string    `json:"command"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
