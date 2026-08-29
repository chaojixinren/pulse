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

// ========== Device CRUD ==========

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

func (r *DeviceRepo) CreateCommand(ctx context.Context, c *model.DeviceCommand) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO device_commands (id, device_id, user_id, command, status) VALUES (?, ?, ?, ?, ?)`,
		c.ID, c.DeviceID, c.UserID, c.Command, c.Status)
	return err
}

// GetByTokenHash 按设备 token 的 SHA256 反查设备（设备级鉴权用，不限用户）。
// 只返回启用中的设备，解绑（DELETE）或停用后 token 立即失效。
func (r *DeviceRepo) GetByTokenHash(ctx context.Context, hash string) (*model.Device, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+deviceColumns+` FROM devices WHERE device_token_hash = ? AND is_active = TRUE`, hash)
	d, err := scanDevice(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

// RotateToken 重置设备 token（设备重新配对时调用），不改动设备名。
func (r *DeviceRepo) RotateToken(ctx context.Context, id, hash string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE devices SET device_token_hash = ?, is_active = TRUE, updated_at = ? WHERE id = ?`,
		hash, now, id)
	return err
}

// UpdateHeartbeatByID 与 UpdateHeartbeat 相同，但不带 user_id 条件。
// 设备级鉴权已经通过 token 确定了设备归属，无需再用 user_id 二次限定。
func (r *DeviceRepo) UpdateHeartbeatByID(ctx context.Context, id string, firmware *string, battery *int) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE devices SET firmware_version = COALESCE(?, firmware_version), battery_level = COALESCE(?, battery_level), last_seen_at = ?, updated_at = ? WHERE id = ? AND is_active = TRUE`,
		firmware, battery, now, now, id)
	return err
}

// GetByIDOnly 按主键查设备，不限用户（设备级鉴权路径用）。
func (r *DeviceRepo) GetByIDOnly(ctx context.Context, id string) (*model.Device, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+deviceColumns+` FROM devices WHERE id = ?`, id)
	d, err := scanDevice(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

const commandColumns = "id, device_id, user_id, command, status, created_at, updated_at"

func scanCommand(row interface{ Scan(...interface{}) error }) (*model.DeviceCommand, error) {
	var c model.DeviceCommand
	if err := row.Scan(&c.ID, &c.DeviceID, &c.UserID, &c.Command, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

// ListPendingCommands 取出设备待执行的指令，按下发时间升序。
// 指令保持 pending 直到设备回执，因此断连期间的指令不会丢失（至少一次投递）。
func (r *DeviceRepo) ListPendingCommands(ctx context.Context, deviceUUID string) ([]model.DeviceCommand, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+commandColumns+` FROM device_commands WHERE device_id = ? AND status = 'pending' ORDER BY created_at ASC`,
		deviceUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.DeviceCommand, 0)
	for rows.Next() {
		c, err := scanCommand(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// ExpireStaleCommands 把下发过久仍未执行的 pending 指令置为 expired。
// 没有这一步，一条设备永远收不到的指令会在每次心跳里被反复投递。
func (r *DeviceRepo) ExpireStaleCommands(ctx context.Context, deviceUUID string, before time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE device_commands SET status = 'expired', updated_at = ? WHERE device_id = ? AND status = 'pending' AND created_at < ?`,
		time.Now().UTC(), deviceUUID, before)
	return err
}

// AckCommand 设备回执，把指令置为终态。返回 false 表示指令不存在、不属于该设备或已是终态。
func (r *DeviceRepo) AckCommand(ctx context.Context, id, deviceUUID, status string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE device_commands SET status = ?, updated_at = ? WHERE id = ? AND device_id = ? AND status = 'pending'`,
		status, time.Now().UTC(), id, deviceUUID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

