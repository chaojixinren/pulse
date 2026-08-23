package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndParseAccessToken(t *testing.T) {
	tok, err := GenerateAccessToken("user-123", "secret", time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)

	userID, err := ParseAccessToken(tok, "secret")
	require.NoError(t, err)
	assert.Equal(t, "user-123", userID)
}

func TestParseAccessTokenWrongSecret(t *testing.T) {
	tok, err := GenerateAccessToken("user-123", "secret", time.Hour)
	require.NoError(t, err)

	_, err = ParseAccessToken(tok, "wrong-secret")
	assert.Error(t, err, "使用错误密钥应解析失败")
}

func TestParseAccessTokenExpired(t *testing.T) {
	tok, err := GenerateAccessToken("user-123", "secret", -time.Minute)
	require.NoError(t, err)

	_, err = ParseAccessToken(tok, "secret")
	assert.Error(t, err, "过期 token 应解析失败")
}

func TestParseAccessTokenMalformed(t *testing.T) {
	for _, bad := range []string{"", "not-a-jwt", "a.b.c", "...."} {
		_, err := ParseAccessToken(bad, "secret")
		assert.Error(t, err, "畸形 token %q 应解析失败", bad)
	}
}

func TestParseAccessTokenMissingSubject(t *testing.T) {
	tok, err := GenerateAccessToken("", "secret", time.Hour)
	require.NoError(t, err)

	_, err = ParseAccessToken(tok, "secret")
	assert.Error(t, err, "subject 为空的 token 应被拒绝")
}

func TestParseAccessTokenRejectsNonHMAC(t *testing.T) {
	claims := jwt.RegisteredClaims{
		Subject:   "user-123",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	// 使用无密钥签名方法，构造非 HMAC 的 token。
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokStr, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = ParseAccessToken(tokStr, "secret")
	assert.Error(t, err, "非 HMAC 签名的 token 应被拒绝")
}
