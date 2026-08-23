package service

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/utils"
)

func newAuthService(t *testing.T) (*AuthService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	cfg := &config.Config{JWTSecret: "test-secret", JWTExpiresIn: time.Hour, RefreshTokenTTL: 7 * 24 * time.Hour}
	svc := NewAuthService(cfg, repository.NewUserRepo(db), repository.NewRefreshTokenRepo(db))
	return svc, mock
}

var userCols = []string{"id", "email", "password_hash", "name", "avatar_url", "settings", "created_at", "updated_at", "deleted_at"}
var refreshTokenCols = []string{"id", "user_id", "token_hash", "expires_at", "created_at", "revoked_at"}

func TestAuthRegister(t *testing.T) {
	t.Run("success_hashes_password", func(t *testing.T) {
		svc, mock := newAuthService(t)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
			WithArgs("a@b.com").
			WillReturnRows(sqlmock.NewRows(userCols))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users")).
			WithArgs(sqlmock.AnyArg(), "a@b.com", sqlmock.AnyArg(), "Alice", nil, "{}").
			WillReturnResult(sqlmock.NewResult(1, 1))

		u, err := svc.Register(context.Background(), "a@b.com", "secret123", "Alice")
		require.NoError(t, err)
		require.NotNil(t, u)

		// 验收：密码在库里为 bcrypt 哈希，不存明文。
		assert.NotEqual(t, "secret123", u.PasswordHash)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("secret123")))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate_email", func(t *testing.T) {
		svc, mock := newAuthService(t)
		now := time.Now().UTC()

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
			WithArgs("a@b.com").
			WillReturnRows(sqlmock.NewRows(userCols).AddRow("u1", "a@b.com", "hash", "Alice", nil, "{}", now, now, nil))

		_, err := svc.Register(context.Background(), "a@b.com", "secret123", "Alice")
		require.Error(t, err)
		appErr, ok := apperrors.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, 40000, appErr.Code)
	})
}

func TestAuthLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mock := newAuthService(t)
		now := time.Now().UTC()
		hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
		require.NoError(t, err)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
			WithArgs("a@b.com").
			WillReturnRows(sqlmock.NewRows(userCols).AddRow("u1", "a@b.com", string(hash), "Alice", nil, "{}", now, now, nil))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO refresh_tokens")).
			WithArgs(sqlmock.AnyArg(), "u1", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		pair, err := svc.Login(context.Background(), "a@b.com", "secret123")
		require.NoError(t, err)
		assert.NotEmpty(t, pair.AccessToken)
		assert.NotEmpty(t, pair.RefreshToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wrong_password", func(t *testing.T) {
		svc, mock := newAuthService(t)
		now := time.Now().UTC()
		hash, err := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
		require.NoError(t, err)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
			WithArgs("a@b.com").
			WillReturnRows(sqlmock.NewRows(userCols).AddRow("u1", "a@b.com", string(hash), "Alice", nil, "{}", now, now, nil))

		_, err = svc.Login(context.Background(), "a@b.com", "wrong-password")
		require.Error(t, err)
		appErr, ok := apperrors.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, 40100, appErr.Code)
	})

	t.Run("unknown_email", func(t *testing.T) {
		svc, mock := newAuthService(t)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
			WithArgs("nobody@b.com").
			WillReturnRows(sqlmock.NewRows(userCols))

		_, err := svc.Login(context.Background(), "nobody@b.com", "secret123")
		require.Error(t, err)
		appErr, ok := apperrors.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, 40100, appErr.Code)
	})
}

func TestAuthRefresh(t *testing.T) {
	t.Run("success_rotates_token", func(t *testing.T) {
		svc, mock := newAuthService(t)
		now := time.Now().UTC()
		hash := utils.SHA256Hex("raw-refresh-token")

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, token_hash")).
			WithArgs(hash).
			WillReturnRows(sqlmock.NewRows(refreshTokenCols).AddRow("t1", "u1", hash, now.Add(time.Hour), now, nil))
		mock.ExpectExec(regexp.QuoteMeta("UPDATE refresh_tokens SET revoked_at")).
			WithArgs(sqlmock.AnyArg(), hash).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO refresh_tokens")).
			WithArgs(sqlmock.AnyArg(), "u1", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		pair, err := svc.Refresh(context.Background(), "raw-refresh-token")
		require.NoError(t, err)
		assert.NotEmpty(t, pair.AccessToken)
		assert.NotEmpty(t, pair.RefreshToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("revoked_token", func(t *testing.T) {
		svc, mock := newAuthService(t)
		now := time.Now().UTC()
		revokedAt := now.Add(-time.Minute)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, token_hash")).
			WithArgs(utils.SHA256Hex("raw-refresh-token")).
			WillReturnRows(sqlmock.NewRows(refreshTokenCols).AddRow("t1", "u1", utils.SHA256Hex("raw-refresh-token"), now.Add(time.Hour), now, revokedAt))

		_, err := svc.Refresh(context.Background(), "raw-refresh-token")
		require.Error(t, err)
		appErr, ok := apperrors.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, 40100, appErr.Code)
	})
}

func TestAuthLogout(t *testing.T) {
	svc, mock := newAuthService(t)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE refresh_tokens SET revoked_at")).
		WithArgs(sqlmock.AnyArg(), utils.SHA256Hex("raw-refresh-token")).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, svc.Logout(context.Background(), "raw-refresh-token"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthGetUser(t *testing.T) {
	svc, mock := newAuthService(t)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(userCols).AddRow("u1", "a@b.com", "hash", "Alice", nil, "{}", now, now, nil))

	u, err := svc.GetUser(context.Background(), "u1")
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, "Alice", u.Name)

	b, err := json.Marshal(u)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "password_hash", "对外返回的 User JSON 不应包含密码哈希字段")
	assert.NotContains(t, string(b), "hash", "对外返回的 User JSON 不应包含密码哈希值")
}

// 用负 TTL 证明 issueTokens 真正使用 cfg.JWTExpiresIn（而非硬编码 1h）。
func TestAuthAccessTokenUsesConfigTTL(t *testing.T) {
	svc, mock := newAuthService(t)
	svc.cfg.JWTExpiresIn = -time.Minute // 立即过期
	svc.cfg.RefreshTokenTTL = time.Hour

	now := time.Now().UTC()
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password_hash")).
		WithArgs("a@b.com").
		WillReturnRows(sqlmock.NewRows(userCols).AddRow("u1", "a@b.com", string(hash), "Alice", nil, "{}", now, now, nil))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO refresh_tokens")).
		WithArgs(sqlmock.AnyArg(), "u1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	pair, err := svc.Login(context.Background(), "a@b.com", "secret123")
	require.NoError(t, err)

	// access token 使用负 TTL，解析时应判定为已过期。
	_, err = utils.ParseAccessToken(pair.AccessToken, "test-secret")
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

var _ = model.User{}
