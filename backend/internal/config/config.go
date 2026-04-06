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
	// Additional OAuth2 providers
	AmazonClientID         string
	AmazonClientSecret     string
	Auth0ClientID          string
	Auth0ClientSecret      string
	Auth0Domain            string
	AutodeskClientID       string
	AutodeskClientSecret   string
	BitlyClientID          string
	BitlyClientSecret      string
	BoxClientID            string
	BoxClientSecret        string
	DailymotionClientID    string
	DailymotionClientSecret string
	DisqusClientID         string
	DisqusClientSecret     string
	DropboxClientID        string
	DropboxClientSecret    string
	EtsyClientID           string
	EtsyClientSecret       string
	FigmaClientID          string
	FigmaClientSecret      string
	HubspotClientID        string
	HubspotClientSecret    string
	KakaoClientID          string
	KakaoClientSecret      string
	LineClientID           string
	LineClientSecret       string
	MailchimpClientID      string
	MailchimpClientSecret  string
	NotionClientID         string
	NotionClientSecret     string
	OktaClientID           string
	OktaClientSecret       string
	OktaDomain             string
	OIDCClientID           string
	OIDCClientSecret       string
	OIDCAuthURL            string
	OIDCTokenURL           string
	OIDCUserInfoURL        string
	PatreonClientID        string
	PatreonClientSecret    string
	PayPalClientID         string
	PayPalClientSecret     string
	PodioClientID          string
	PodioClientSecret      string
	RedditClientID         string
	RedditClientSecret     string
	SalesforceClientID     string
	SalesforceClientSecret string
	TradeshiftClientID     string
	TradeshiftClientSecret string
	WordPressClientID      string
	WordPressClientSecret  string
	YahooClientID          string
	YahooClientSecret      string
	YammerClientID         string
	YammerClientSecret     string
	YandexClientID         string
	YandexClientSecret     string
	ZohoClientID           string
	ZohoClientSecret       string
	ZoomClientID           string
	ZoomClientSecret       string
	// Twilio SMS
	TwilioSID   string
	TwilioToken string
	TwilioFrom  string
	// FCM push notifications
	FCMServerKey string
	// Mailgun
	MailgunAPIKey string
	MailgunDomain string
	// Resend
	ResendAPIKey string
	// Vonage SMS
	VonageAPIKey    string
	VonageAPISecret string
	VonageFrom      string
	// MSG91 SMS
	MSG91AuthKey  string
	MSG91SenderID string
	// APNS push notifications
	APNSKeyID    string
	APNSTeamID   string
	APNSKeyPath  string
	APNSBundleID string
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
		AmazonClientID:         getEnv("OAUTH_AMAZON_CLIENT_ID", ""),
		AmazonClientSecret:     getEnv("OAUTH_AMAZON_CLIENT_SECRET", ""),
		Auth0ClientID:          getEnv("OAUTH_AUTH0_CLIENT_ID", ""),
		Auth0ClientSecret:      getEnv("OAUTH_AUTH0_CLIENT_SECRET", ""),
		Auth0Domain:            getEnv("OAUTH_AUTH0_DOMAIN", ""),
		AutodeskClientID:       getEnv("OAUTH_AUTODESK_CLIENT_ID", ""),
		AutodeskClientSecret:   getEnv("OAUTH_AUTODESK_CLIENT_SECRET", ""),
		BitlyClientID:          getEnv("OAUTH_BITLY_CLIENT_ID", ""),
		BitlyClientSecret:      getEnv("OAUTH_BITLY_CLIENT_SECRET", ""),
		BoxClientID:            getEnv("OAUTH_BOX_CLIENT_ID", ""),
		BoxClientSecret:        getEnv("OAUTH_BOX_CLIENT_SECRET", ""),
		DailymotionClientID:    getEnv("OAUTH_DAILYMOTION_CLIENT_ID", ""),
		DailymotionClientSecret: getEnv("OAUTH_DAILYMOTION_CLIENT_SECRET", ""),
		DisqusClientID:         getEnv("OAUTH_DISQUS_CLIENT_ID", ""),
		DisqusClientSecret:     getEnv("OAUTH_DISQUS_CLIENT_SECRET", ""),
		DropboxClientID:        getEnv("OAUTH_DROPBOX_CLIENT_ID", ""),
		DropboxClientSecret:    getEnv("OAUTH_DROPBOX_CLIENT_SECRET", ""),
		EtsyClientID:           getEnv("OAUTH_ETSY_CLIENT_ID", ""),
		EtsyClientSecret:       getEnv("OAUTH_ETSY_CLIENT_SECRET", ""),
		FigmaClientID:          getEnv("OAUTH_FIGMA_CLIENT_ID", ""),
		FigmaClientSecret:      getEnv("OAUTH_FIGMA_CLIENT_SECRET", ""),
		HubspotClientID:        getEnv("OAUTH_HUBSPOT_CLIENT_ID", ""),
		HubspotClientSecret:    getEnv("OAUTH_HUBSPOT_CLIENT_SECRET", ""),
		KakaoClientID:          getEnv("OAUTH_KAKAO_CLIENT_ID", ""),
		KakaoClientSecret:      getEnv("OAUTH_KAKAO_CLIENT_SECRET", ""),
		LineClientID:           getEnv("OAUTH_LINE_CLIENT_ID", ""),
		LineClientSecret:       getEnv("OAUTH_LINE_CLIENT_SECRET", ""),
		MailchimpClientID:      getEnv("OAUTH_MAILCHIMP_CLIENT_ID", ""),
		MailchimpClientSecret:  getEnv("OAUTH_MAILCHIMP_CLIENT_SECRET", ""),
		NotionClientID:         getEnv("OAUTH_NOTION_CLIENT_ID", ""),
		NotionClientSecret:     getEnv("OAUTH_NOTION_CLIENT_SECRET", ""),
		OktaClientID:           getEnv("OAUTH_OKTA_CLIENT_ID", ""),
		OktaClientSecret:       getEnv("OAUTH_OKTA_CLIENT_SECRET", ""),
		OktaDomain:             getEnv("OAUTH_OKTA_DOMAIN", ""),
		OIDCClientID:           getEnv("OAUTH_OIDC_CLIENT_ID", ""),
		OIDCClientSecret:       getEnv("OAUTH_OIDC_CLIENT_SECRET", ""),
		OIDCAuthURL:            getEnv("OAUTH_OIDC_AUTH_URL", ""),
		OIDCTokenURL:           getEnv("OAUTH_OIDC_TOKEN_URL", ""),
		OIDCUserInfoURL:        getEnv("OAUTH_OIDC_USERINFO_URL", ""),
		PatreonClientID:        getEnv("OAUTH_PATREON_CLIENT_ID", ""),
		PatreonClientSecret:    getEnv("OAUTH_PATREON_CLIENT_SECRET", ""),
		PayPalClientID:         getEnv("OAUTH_PAYPAL_CLIENT_ID", ""),
		PayPalClientSecret:     getEnv("OAUTH_PAYPAL_CLIENT_SECRET", ""),
		PodioClientID:          getEnv("OAUTH_PODIO_CLIENT_ID", ""),
		PodioClientSecret:      getEnv("OAUTH_PODIO_CLIENT_SECRET", ""),
		RedditClientID:         getEnv("OAUTH_REDDIT_CLIENT_ID", ""),
		RedditClientSecret:     getEnv("OAUTH_REDDIT_CLIENT_SECRET", ""),
		SalesforceClientID:     getEnv("OAUTH_SALESFORCE_CLIENT_ID", ""),
		SalesforceClientSecret: getEnv("OAUTH_SALESFORCE_CLIENT_SECRET", ""),
		TradeshiftClientID:     getEnv("OAUTH_TRADESHIFT_CLIENT_ID", ""),
		TradeshiftClientSecret: getEnv("OAUTH_TRADESHIFT_CLIENT_SECRET", ""),
		WordPressClientID:      getEnv("OAUTH_WORDPRESS_CLIENT_ID", ""),
		WordPressClientSecret:  getEnv("OAUTH_WORDPRESS_CLIENT_SECRET", ""),
		YahooClientID:          getEnv("OAUTH_YAHOO_CLIENT_ID", ""),
		YahooClientSecret:      getEnv("OAUTH_YAHOO_CLIENT_SECRET", ""),
		YammerClientID:         getEnv("OAUTH_YAMMER_CLIENT_ID", ""),
		YammerClientSecret:     getEnv("OAUTH_YAMMER_CLIENT_SECRET", ""),
		YandexClientID:         getEnv("OAUTH_YANDEX_CLIENT_ID", ""),
		YandexClientSecret:     getEnv("OAUTH_YANDEX_CLIENT_SECRET", ""),
		ZohoClientID:           getEnv("OAUTH_ZOHO_CLIENT_ID", ""),
		ZohoClientSecret:       getEnv("OAUTH_ZOHO_CLIENT_SECRET", ""),
		ZoomClientID:           getEnv("OAUTH_ZOOM_CLIENT_ID", ""),
		ZoomClientSecret:       getEnv("OAUTH_ZOOM_CLIENT_SECRET", ""),
		TwilioSID:            getEnv("TWILIO_SID", ""),
		TwilioToken:          getEnv("TWILIO_TOKEN", ""),
		TwilioFrom:           getEnv("TWILIO_FROM", ""),
		FCMServerKey:         getEnv("FCM_SERVER_KEY", ""),
		MailgunAPIKey:        getEnv("MAILGUN_API_KEY", ""),
		MailgunDomain:        getEnv("MAILGUN_DOMAIN", ""),
		ResendAPIKey:         getEnv("RESEND_API_KEY", ""),
		VonageAPIKey:         getEnv("VONAGE_API_KEY", ""),
		VonageAPISecret:      getEnv("VONAGE_API_SECRET", ""),
		VonageFrom:           getEnv("VONAGE_FROM", ""),
		MSG91AuthKey:         getEnv("MSG91_AUTH_KEY", ""),
		MSG91SenderID:        getEnv("MSG91_SENDER_ID", ""),
		APNSKeyID:            getEnv("APNS_KEY_ID", ""),
		APNSTeamID:           getEnv("APNS_TEAM_ID", ""),
		APNSKeyPath:          getEnv("APNS_KEY_PATH", ""),
		APNSBundleID:         getEnv("APNS_BUNDLE_ID", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
