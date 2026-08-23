package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/chaojixinren/pulse/internal/model"
)

type IdentityRepo struct {
	db *sql.DB
}

func NewIdentityRepo(db *sql.DB) *IdentityRepo { return &IdentityRepo{db: db} }

const identityColumns = "id, user_id, name, description, color, icon, is_default, created_at, updated_at, deleted_at"

func scanIdentity(row interface{ Scan(...interface{}) error }) (*model.Identity, error) {
	var i model.Identity
	if err := row.Scan(&i.ID, &i.UserID, &i.Name, &i.Description, &i.Color, &i.Icon, &i.IsDefault, &i.CreatedAt, &i.UpdatedAt, &i.DeletedAt); err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *IdentityRepo) Create(ctx context.Context, i *model.Identity) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO identities (id, user_id, name, description, color, icon, is_default) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		i.ID, i.UserID, i.Name, i.Description, i.Color, i.Icon, i.IsDefault)
	return err
}

func (r *IdentityRepo) GetByID(ctx context.Context, id, userID string) (*model.Identity, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+identityColumns+` FROM identities WHERE id = ? AND user_id = ? AND deleted_at IS NULL`, id, userID)
	i, err := scanIdentity(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return i, err
}

func (r *IdentityRepo) ListByUser(ctx context.Context, userID string) ([]model.Identity, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+identityColumns+` FROM identities WHERE user_id = ? AND deleted_at IS NULL ORDER BY is_default DESC, created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.Identity, 0)
	for rows.Next() {
		i, err := scanIdentity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *i)
	}
	return out, rows.Err()
}

func (r *IdentityRepo) CountByUser(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM identities WHERE user_id = ? AND deleted_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *IdentityRepo) GetDefault(ctx context.Context, userID string) (*model.Identity, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+identityColumns+` FROM identities WHERE user_id = ? AND is_default = TRUE AND deleted_at IS NULL`, userID)
	i, err := scanIdentity(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return i, err
}

func (r *IdentityRepo) Update(ctx context.Context, i *model.Identity) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE identities SET name = ?, description = ?, color = ?, icon = ?, updated_at = ? WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		i.Name, i.Description, i.Color, i.Icon, time.Now().UTC(), i.ID, i.UserID)
	return err
}

// SetDefault 在事务中先将该用户其它身份取消默认，再设置目标身份为默认。
func (r *IdentityRepo) SetDefault(ctx context.Context, userID, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE identities SET is_default = FALSE, updated_at = ? WHERE user_id = ? AND deleted_at IS NULL`, time.Now().UTC(), userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE identities SET is_default = TRUE, updated_at = ? WHERE id = ? AND user_id = ? AND deleted_at IS NULL`, time.Now().UTC(), id, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// Delete 软删除（默认身份禁止删除，由 service 层校验）。
func (r *IdentityRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE identities SET deleted_at = ?, updated_at = ? WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		time.Now().UTC(), time.Now().UTC(), id, userID)
	return err
}
