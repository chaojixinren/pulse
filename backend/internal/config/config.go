package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config 汇总所有运行配置。
type Config struct {
	Port     string
	GINMode  string
	MySQLDSN string
	RedisURL string

	StepFunAPIKey   string
	StepFunBaseURL  string
	StepFunSTTModel string

	AIAPIKey              string
	AIBaseURL             string
	AIModel               string
	AIConfidenceThreshold float64

	JWTSecret       string
	JWTExpiresIn    time.Duration
	RefreshTokenTTL time.Duration

	AllowedOrigins []string
	LogLevel       string

	MaxAudioSize       int64
	AudioRetentionDays int

	// AudioEncryptionKey 为 AES-256-GCM 的 32 字节密钥（base64 解码后）；空表示关闭加密存储。
	AudioEncryptionKey []byte

	// 限流配额（次/分钟）
	RateLimitAuthPerMin   int
	RateLimitUploadPerMin int
}

// Load 从 .env 与环境变量读取配置并校验。
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		if os.Getenv("JWT_SECRET") == "" || os.Getenv("DATABASE_DSN") == "" {
			return nil, fmt.Errorf("未找到 .env 文件且未设置环境变量：请复制 backend/.env.example 为 backend/.env 并填写配置")
		}
	}

	expires, err := time.ParseDuration(getEnv("JWT_EXPIRES_IN", "1h"))
	if err != nil {
		return nil, fmt.Errorf("JWT_EXPIRES_IN 配置无效: %w", err)
	}

	refreshTTL, err := time.ParseDuration(getEnv("REFRESH_TOKEN_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("REFRESH_TOKEN_TTL 配置无效: %w", err)
	}

	maxAudioSize, err := strconv.ParseInt(getEnv("MAX_AUDIO_SIZE", "52428800"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("MAX_AUDIO_SIZE 配置无效: %w", err)
	}

	retention, err := strconv.Atoi(getEnv("AUDIO_RETENTION_DAYS", "30"))
	if err != nil {
		return nil, fmt.Errorf("AUDIO_RETENTION_DAYS 配置无效: %w", err)
	}

	aiConfidence, err := strconv.ParseFloat(getEnv("AI_CONFIDENCE_THRESHOLD", "0.6"), 64)
	if err != nil {
		return nil, fmt.Errorf("AI_CONFIDENCE_THRESHOLD 配置无效: %w", err)
	}

	encKey, err := decodeEncryptionKey(os.Getenv("AUDIO_ENCRYPTION_KEY"))
	if err != nil {
		return nil, err
	}

	authRate, err := strconv.Atoi(getEnv("RATE_LIMIT_AUTH_PER_MIN", "20"))
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_AUTH_PER_MIN 配置无效: %w", err)
	}
	uploadRate, err := strconv.Atoi(getEnv("RATE_LIMIT_UPLOAD_PER_MIN", "30"))
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_UPLOAD_PER_MIN 配置无效: %w", err)
	}

	cfg := &Config{
		Port:                  getEnv("PORT", "8080"),
		GINMode:               getEnv("GIN_MODE", "debug"),
		MySQLDSN:              os.Getenv("DATABASE_DSN"),
		RedisURL:              getEnv("REDIS_URL", "redis://localhost:6379"),
		StepFunAPIKey:         os.Getenv("STEPFUN_API_KEY"),
		StepFunBaseURL:        getEnv("STEPFUN_API_BASE", "https://api.stepfun.com/v1"),
		StepFunSTTModel:       getEnv("STEPFUN_STT_MODEL", "stepaudio-2.5-asr"),
		AIAPIKey:              os.Getenv("AI_API_KEY"),
		AIBaseURL:             getEnv("AI_BASE_URL", "https://api.openai.com/v1"),
		AIModel:               getEnv("AI_MODEL", "gpt-4o-mini"),
		AIConfidenceThreshold: aiConfidence,
		JWTSecret:             os.Getenv("JWT_SECRET"),
		JWTExpiresIn:          expires,
		RefreshTokenTTL:       refreshTTL,
		AllowedOrigins:        splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		LogLevel:              getEnv("LOG_LEVEL", "info"),
		MaxAudioSize:          maxAudioSize,
		AudioRetentionDays:    retention,
		AudioEncryptionKey:    encKey,
		RateLimitAuthPerMin:   authRate,
		RateLimitUploadPerMin: uploadRate,
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// decodeEncryptionKey 将 base64 编码的 32 字节密钥解码为 []byte；空值返回 nil（关闭加密）。
func decodeEncryptionKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	key, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("AUDIO_ENCRYPTION_KEY 无效：应为 base64 编码的 32 字节密钥: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("AUDIO_ENCRYPTION_KEY 长度应为 32 字节（AES-256），当前 %d 字节", len(key))
	}
	return key, nil
}

func (c *Config) validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET 未设置：请在 .env 中配置")
	}
	if c.MySQLDSN == "" {
		return fmt.Errorf("DATABASE_DSN 未设置：请在 .env 中配置")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
