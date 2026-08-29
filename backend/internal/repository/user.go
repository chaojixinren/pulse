package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/chaojixinren/pulse/internal/model"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo { return &UserRepo{db: db} }

const userColumns = "id, email, password_hash, name, avatar_url, settings, created_at, updated_at, deleted_at"

func scanUser(row interface{ Scan(...interface{}) error }) (*model.User, error) {
	var u model.User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AvatarURL, &u.Settings, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, name, avatar_url, settings) VALUES (?, ?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.PasswordHash, u.Name, u.AvatarURL, u.Settings)
	return err
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE email = ? AND deleted_at IS NULL`, email)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = ? AND deleted_at IS NULL`, id)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// SoftDelete 软删除用户（置 deleted_at），用于注销；已删除用户无法再登录。
func (r *UserRepo) SoftDelete(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, now, now, id)
	return err
}

// UpdateSettings 覆盖写入用户 settings JSON。
func (r *UserRepo) UpdateSettings(ctx context.Context, id, settings string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET settings = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, settings, now, id)
	return err
}
