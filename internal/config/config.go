// Package config 负责从环境变量加载应用配置。
//
// 所有可变配置（DSN、JWT secret、CORS、端口、上传目录等）都通过 env 注入，
// 不再硬编码在源码中。生产环境（ENV=production）缺少关键配置时启动失败。
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 应用配置
type Config struct {
	Env          string        // development / production
	Port         string        // 监听端口
	DSN          string        // PostgreSQL 连接串
	JWTSecret    string        // JWT 签名密钥
	JWTExpire    time.Duration // token 有效期
	CORSOrigins  []string      // 允许的跨域来源
	UploadDir    string        // 上传文件根目录
	MaxUploadMB  int           // 单文件上传大小上限（MB）
}

// Load 从环境变量加载配置；生产环境缺失关键项时 fatal。
func Load() *Config {
	env := getEnv("ENV", "development")
	isProd := env == "production"

	cfg := &Config{
		Env:         env,
		Port:        getEnv("SERVER_PORT", "8080"),
		DSN:         buildDSN(),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		JWTExpire:   getDuration("JWT_EXPIRE_HOURS", 168) * time.Hour, // 默认 7 天
		CORSOrigins: getStringSlice("CORS_ORIGINS", defaultCORSOrigins()),
		UploadDir:   getEnv("UPLOAD_DIR", "./uploads"),
		MaxUploadMB: getInt("MAX_UPLOAD_MB", 20),
	}

	// 生产环境必须显式配置关键项
	if isProd {
		if cfg.JWTSecret == "" {
			log.Fatal("ENV=production 但未设置 JWT_SECRET，拒绝启动")
		}
		if cfg.DSN == "" {
			log.Fatal("ENV=production 但未配置数据库连接，拒绝启动")
		}
		if len(cfg.CORSOrigins) == 0 {
			log.Fatal("ENV=production 但未设置 CORS_ORIGINS，拒绝启动")
		}
	} else {
		// 开发环境兜底默认值
		if cfg.JWTSecret == "" {
			cfg.JWTSecret = "dev-secret-key-change-in-production"
		}
	}

	return cfg
}

// buildDSN 根据分散的 DB_* 环境变量组装 DSN，优先使用直接的 DSN 变量。
func buildDSN() string {
	if dsn := os.Getenv("DSN"); dsn != "" {
		return dsn
	}
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "anan_diary")
	sslmode := getEnv("DB_SSLMODE", "disable")
	tz := getEnv("DB_TIMEZONE", "Asia/Shanghai")
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		host, port, user, password, dbName, sslmode, tz)
}

func defaultCORSOrigins() []string {
	return []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"http://0.0.0.0:3000",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getDuration(key string, fallbackHours int) time.Duration {
	return time.Duration(getInt(key, fallbackHours)) * time.Hour
}

func getStringSlice(key string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

// IsProduction 是否为生产环境
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// AllowedOrigin 判断给定 origin 是否在白名单中
func (c *Config) AllowedOrigin(origin string) bool {
	for _, o := range c.CORSOrigins {
		if o == origin {
			return true
		}
	}
	return false
}