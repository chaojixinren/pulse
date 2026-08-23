package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/chaojixinren/pulse/internal/model"
)

type DeviceRepo struct {
	db *sql.DB
}

func NewDeviceRepo(db *sql.DB) *DeviceRepo { return &DeviceRepo{db: db} }

const deviceColumns = "id, user_id, device_id, name, device_type, firmware_version, battery_level, last_seen_at, is_active, device_token_hash, created_at, updated_at"

func scanDevice(row interface{ Scan(...interface{}) error }) (*model.Device, error) {
	var d model.Device
	if err := row.Scan(
		&d.ID, &d.UserID, &d.DeviceID, &d.Name, &d.DeviceType, &d.FirmwareVersion, &d.BatteryLevel,
		&d.LastSeenAt, &d.IsActive, &d.DeviceTokenHash, &d.CreatedAt, &d.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeviceRepo) Create(ctx context.Context, d *model.Device) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO devices (id, user_id, device_id, name, device_type, firmware_version, battery_level, last_seen_at, is_active, device_token_hash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.UserID, d.DeviceID, d.Name, d.DeviceType, d.FirmwareVersion, d.BatteryLevel, d.LastSeenAt, d.IsActive, d.DeviceTokenHash)
	return err
}

func (r *DeviceRepo) GetByID(ctx context.Context, id, userID string) (*model.Device, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+deviceColumns+` FROM devices WHERE id = ? AND user_id = ?`, id, userID)
	d, err := scanDevice(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

// GetByDeviceID 按硬件唯一标识查询（跨用户，用于判断是否已被绑定）。
func (r *DeviceRepo) GetByDeviceID(ctx context.Context, deviceID string) (*model.Device, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+deviceColumns+` FROM devices WHERE device_id = ?`, deviceID)
	d, err := scanDevice(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (r *DeviceRepo) ListByUser(ctx context.Context, userID string) ([]model.Device, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+deviceColumns+` FROM devices WHERE user_id = ? AND is_active = TRUE ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.Device, 0)
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *DeviceRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM devices WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

// UpdateHeartbeat 更新心跳：last_seen_at 与 updated_at 置为当前时间，
// firmware_version / battery_level 仅在传入非 nil 时覆盖（COALESCE 保留旧值）。
func (r *DeviceRepo) UpdateHeartbeat(ctx context.Context, id, userID string, firmware *string, battery *int) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE devices SET firmware_version = COALESCE(?, firmware_version), battery_level = COALESCE(?, battery_level), last_seen_at = ?, updated_at = ? WHERE id = ? AND user_id = ? AND is_active = TRUE`,
		firmware, battery, now, now, id, userID)
	return err
}

const bindCodeColumns = "id, user_id, code, expires_at, used_at, created_at"

func scanBindCode(row interface{ Scan(...interface{}) error }) (*model.DeviceBindCode, error) {
	var c model.DeviceBindCode
	if err := row.Scan(&c.ID, &c.UserID, &c.Code, &c.ExpiresAt, &c.UsedAt, &c.CreatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *DeviceRepo) CreateBindCode(ctx context.Context, c *model.DeviceBindCode) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO device_bind_codes (id, user_id, code, expires_at) VALUES (?, ?, ?, ?)`,
		c.ID, c.UserID, c.Code, c.ExpiresAt)
	return err
}

func (r *DeviceRepo) GetBindCodeByCode(ctx context.Context, code string) (*model.DeviceBindCode, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+bindCodeColumns+` FROM device_bind_codes WHERE code = ?`, code)
	c, err := scanBindCode(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// MarkBindCodeUsed 将绑定码标记为已使用（一次性）。
func (r *DeviceRepo) MarkBindCodeUsed(ctx context.Context, code string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE device_bind_codes SET used_at = ? WHERE code = ? AND used_at IS NULL`, time.Now().UTC(), code)
	return err
}

func (r *DeviceRepo) CreateCommand(ctx context.Context, c *model.DeviceCommand) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO device_commands (id, device_id, user_id, command, status) VALUES (?, ?, ?, ?, ?)`,
		c.ID, c.DeviceID, c.UserID, c.Command, c.Status)
	return err
}
