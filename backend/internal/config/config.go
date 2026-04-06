package config

import "os"

type Config struct {
	Port                 string
	DatabaseDSN          string
	RedisAddr            string
	JWTSecret            string
	StoragePath          string
	AppEnv               string
	SMTPHost             string
	SMTPPort             string
	SMTPUser             string
	SMTPPass             string
	SMTPFrom             string
	ConsoleSignupEnabled string // "auto" (default), "true", or "false"
	// OAuth2 providers
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	AppleClientID      string
	AppleClientSecret  string
	OAuthRedirectURL   string // e.g., https://yourdomain.com/v1/account/sessions/oauth/callback
	// Twilio SMS
	TwilioSID   string
	TwilioToken string
	TwilioFrom  string
	// FCM push notifications
	FCMServerKey string
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
		SMTPFrom:             getEnv("SMTP_FROM", "noreply@applad.local"),
		ConsoleSignupEnabled: getEnv("CONSOLE_SIGNUP_ENABLED", "auto"),
		GoogleClientID:       getEnv("OAUTH_GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:   getEnv("OAUTH_GOOGLE_CLIENT_SECRET", ""),
		GitHubClientID:       getEnv("OAUTH_GITHUB_CLIENT_ID", ""),
		GitHubClientSecret:   getEnv("OAUTH_GITHUB_CLIENT_SECRET", ""),
		AppleClientID:        getEnv("OAUTH_APPLE_CLIENT_ID", ""),
		AppleClientSecret:    getEnv("OAUTH_APPLE_CLIENT_SECRET", ""),
		OAuthRedirectURL:     getEnv("OAUTH_REDIRECT_URL", ""),
		TwilioSID:            getEnv("TWILIO_SID", ""),
		TwilioToken:          getEnv("TWILIO_TOKEN", ""),
		TwilioFrom:           getEnv("TWILIO_FROM", ""),
		FCMServerKey:         getEnv("FCM_SERVER_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
