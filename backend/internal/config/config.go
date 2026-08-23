package config

import (
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

	JWTSecret    string
	JWTExpiresIn time.Duration

	AllowedOrigins []string
	LogLevel       string

	MaxAudioSize       int64
	AudioRetentionDays int
}

// Load 从 .env 与环境变量读取配置并校验。
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		if os.Getenv("JWT_SECRET") == "" || os.Getenv("DATABASE_DSN") == "" {
			return nil, fmt.Errorf("未找到 .env 文件且未设置环境变量：请复制 backend/.env.example 为 backend/.env 并填写配置")
		}
	}

	expires, err := time.ParseDuration(getEnv("JWT_EXPIRES_IN", "168h"))
	if err != nil {
		return nil, fmt.Errorf("JWT_EXPIRES_IN 配置无效: %w", err)
	}

	maxAudioSize, err := strconv.ParseInt(getEnv("MAX_AUDIO_SIZE", "52428800"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("MAX_AUDIO_SIZE 配置无效: %w", err)
	}

	retention, err := strconv.Atoi(getEnv("AUDIO_RETENTION_DAYS", "30"))
	if err != nil {
		return nil, fmt.Errorf("AUDIO_RETENTION_DAYS 配置无效: %w", err)
	}

	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		GINMode:            getEnv("GIN_MODE", "debug"),
		MySQLDSN:           os.Getenv("DATABASE_DSN"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
		StepFunAPIKey:      os.Getenv("STEPFUN_API_KEY"),
		StepFunBaseURL:     getEnv("STEPFUN_API_BASE", "https://api.stepfun.com/v1"),
		StepFunSTTModel:    getEnv("STEPFUN_STT_MODEL", "stepaudio-2.5-asr"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		JWTExpiresIn:       expires,
		AllowedOrigins:     splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		MaxAudioSize:       maxAudioSize,
		AudioRetentionDays: retention,
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
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
