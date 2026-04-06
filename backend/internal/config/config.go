package config

import "os"

type Config struct {
	Port        string
	DatabaseDSN string
	RedisAddr   string
	JWTSecret   string
	StoragePath string
	AppEnv      string
	SMTPHost    string
	SMTPPort    string
	SMTPUser    string
	SMTPPass    string
	SMTPFrom    string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseDSN: getEnv("DATABASE_DSN", "applad:applad@tcp(mariadb:3306)/applad?parseTime=true"),
		RedisAddr:   getEnv("REDIS_ADDR", "redis:6379"),
		JWTSecret:   getEnv("JWT_SECRET", "change-me-in-production"),
		StoragePath: getEnv("STORAGE_PATH", "/var/applad/storage"),
		AppEnv:      getEnv("APP_ENV", "development"),
		SMTPHost:    getEnv("SMTP_HOST", ""),
		SMTPPort:    getEnv("SMTP_PORT", "587"),
		SMTPUser:    getEnv("SMTP_USER", ""),
		SMTPPass:    getEnv("SMTP_PASS", ""),
		SMTPFrom:    getEnv("SMTP_FROM", "noreply@applad.local"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
