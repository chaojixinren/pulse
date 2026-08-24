package config

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitCSV(t *testing.T) {
	assert.Nil(t, splitCSV(""))
	assert.Equal(t, []string{"a", "b", "c"}, splitCSV("a,b,c"))
	assert.Equal(t, []string{"a", "b"}, splitCSV(" a , b , ,"))
	assert.Equal(t, []string{"http://localhost:3000", "http://localhost:5173"}, splitCSV("http://localhost:3000,http://localhost:5173"))
}

func TestGetEnvFallback(t *testing.T) {
	t.Setenv("TEST_GETENV_KEY", "hello")
	assert.Equal(t, "hello", getEnv("TEST_GETENV_KEY", "fallback"))
	assert.Equal(t, "fallback", getEnv("TEST_GETENV_MISSING", "fallback"))
	// 空字符串应回退默认值
	t.Setenv("TEST_GETENV_EMPTY", "")
	assert.Equal(t, "fallback", getEnv("TEST_GETENV_EMPTY", "fallback"))
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/pulse")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "debug", cfg.GINMode)
	assert.Equal(t, "redis://localhost:6379", cfg.RedisURL)
	assert.Equal(t, "test-secret", cfg.JWTSecret)
	assert.Equal(t, time.Hour, cfg.JWTExpiresIn, "access token 默认 1h")
	assert.Equal(t, 168*time.Hour, cfg.RefreshTokenTTL, "refresh token 默认 7 天")
	assert.Equal(t, "https://api.stepfun.com/v1", cfg.StepFunBaseURL)
	assert.Equal(t, "stepaudio-2.5-asr", cfg.StepFunSTTModel)
	assert.Equal(t, "https://api.openai.com/v1", cfg.AIBaseURL)
	assert.Equal(t, "gpt-4o-mini", cfg.AIModel)
	assert.Equal(t, 0.6, cfg.AIConfidenceThreshold)
	assert.Equal(t, int64(52428800), cfg.MaxAudioSize)
	assert.Equal(t, 30, cfg.AudioRetentionDays)
	assert.Equal(t, "info", cfg.LogLevel)
}

// 无 .env 文件且缺少关键环境变量时，Load 应返回含「.env」提示的明确错误。
func TestLoadMissingEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/pulse")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ".env")
}

// validate 直接校验 Config，覆盖「.env 存在但缺项」时返回具体缺失项的分支。
func TestValidate(t *testing.T) {
	assert.Contains(t, (&Config{}).validate().Error(), "JWT_SECRET")
	assert.Contains(t, (&Config{JWTSecret: "s"}).validate().Error(), "DATABASE_DSN")
	assert.NoError(t, (&Config{JWTSecret: "s", MySQLDSN: "d"}).validate())
}

func TestLoadInvalidJWTExpiresIn(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/pulse")
	t.Setenv("JWT_EXPIRES_IN", "not-a-duration")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_EXPIRES_IN")
}

func TestLoadInvalidMaxAudioSize(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/pulse")
	t.Setenv("MAX_AUDIO_SIZE", "huge")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MAX_AUDIO_SIZE")
}

func TestLoadCustomValues(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/pulse")
	t.Setenv("PORT", "9090")
	t.Setenv("GIN_MODE", "release")
	t.Setenv("JWT_EXPIRES_IN", "1h")
	t.Setenv("REFRESH_TOKEN_TTL", "2h")
	t.Setenv("MAX_AUDIO_SIZE", "1024")
	t.Setenv("ALLOWED_ORIGINS", "http://a.com,http://b.com")
	t.Setenv("AI_BASE_URL", "https://llm.example.com/v1")
	t.Setenv("AI_MODEL", "my-model")
	t.Setenv("AI_CONFIDENCE_THRESHOLD", "0.8")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "release", cfg.GINMode)
	assert.Equal(t, time.Hour, cfg.JWTExpiresIn)
	assert.Equal(t, 2*time.Hour, cfg.RefreshTokenTTL)
	assert.Equal(t, int64(1024), cfg.MaxAudioSize)
	assert.Equal(t, []string{"http://a.com", "http://b.com"}, cfg.AllowedOrigins)
	assert.Equal(t, "https://llm.example.com/v1", cfg.AIBaseURL)
	assert.Equal(t, "my-model", cfg.AIModel)
	assert.Equal(t, 0.8, cfg.AIConfidenceThreshold)
}

func TestLoadInvalidAIConfidenceThreshold(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/pulse")
	t.Setenv("AI_CONFIDENCE_THRESHOLD", "high")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AI_CONFIDENCE_THRESHOLD")
}

func TestDecodeEncryptionKey(t *testing.T) {
	// 空 → nil（关闭加密）。
	key, err := decodeEncryptionKey("")
	require.NoError(t, err)
	assert.Nil(t, key)

	// 32 字节 base64 → 原样解码。
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i)
	}
	key, err = decodeEncryptionKey(base64.StdEncoding.EncodeToString(raw))
	require.NoError(t, err)
	assert.Equal(t, raw, key)

	// 非法 base64 → 报错。
	_, err = decodeEncryptionKey("!!!not-base64!!!")
	require.Error(t, err)

	// 长度不足 32 字节 → 报错。
	_, err = decodeEncryptionKey(base64.StdEncoding.EncodeToString([]byte("too-short")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32 字节")
}
