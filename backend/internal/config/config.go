package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration loaded from the environment.
type Config struct {
	AppEnv      string
	HTTPAddr    string
	FrontendURL string

	DB DBConfig

	JWT JWTConfig

	// Cross-origin allowed for the SPA origin only by default.
	CORSAllowedOrigins []string

	// Rate limiting
	FormRatePerHour int
	APIRatePerMin   int

	// MAX messenger integration
	Max MaxConfig

	// AI agent
	AI AIConfig

	log *slog.Logger
}

// SetLogger injects the structured logger used by the HTTP layer for
// operational warnings (MAX persistence failures etc). Called once from
// cmd/server.
func (c *Config) SetLogger(l *slog.Logger) { c.log = l }

// Log returns the configured slog logger, or a default JSON logger when none
// has been set yet, so handlers can always log without nil checks.
func (c *Config) Log() *slog.Logger {
	if c.log == nil {
		return slog.Default()
	}
	return c.log
}

type DBConfig struct {
	URL          string
	MaxOpenConns int32
	MaxIdleConns int32
	MaxIdleTime  time.Duration
}

type JWTConfig struct {
	Secret string
	TTL    time.Duration
	Issuer string
}

type MaxConfig struct {
	// Enabled toggles real outbound messaging. When false, notifications are
	// logged only (dry-run), keeping local development safe.
	Enabled       bool
	APIToken      string
	APIBase       string
	BotHandle     string
	WebhookSecret string
	// DeepLinkPrefix is the prefix to build a "write in MAX" deep link,
	// e.g. https://max.ru/ or tg://.
	DeepLinkPrefix string
	// NotifyChatID is the admin chat where new leads/appointments are pushed.
	NotifyChatID string
}

type AIConfig struct {
	// Enabled toggles the LLM layer. When false, only the rule-based bot runs.
	Enabled  bool
	Provider string // "openai" | "stub" | ""
	APIKey   string
	Model    string
	BaseURL  string
	// MaxTokens caps completion length to keep responses bounded.
	MaxTokens int
}

// Load reads .env (if present) and environment variables.
func Load() (*Config, error) {
	_ = godotenv.Load() // .env is optional in production

	c := &Config{
		AppEnv:             getenv("APP_ENV", "dev"),
		HTTPAddr:           getenv("HTTP_ADDR", ":8080"),
		FrontendURL:        getenv("FRONTEND_URL", "http://localhost:5173"),
		CORSAllowedOrigins: splitCSV(getenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:4173")),
		FormRatePerHour:    getenvInt("FORM_RATE_PER_HOUR", 10),
		APIRatePerMin:      getenvInt("API_RATE_PER_MIN", 120),
		DB: DBConfig{
			URL:          getenv("DATABASE_URL", "postgres://tm:tm@localhost:5432/timemachine?sslmode=disable"),
			MaxOpenConns: int32(getenvInt("DB_MAX_OPEN_CONNS", 10)),
			MaxIdleConns: int32(getenvInt("DB_MAX_IDLE_CONNS", 2)),
			MaxIdleTime:  getdur("DB_MAX_IDLE_TIME", 15*time.Minute),
		},
		JWT: JWTConfig{
			Secret: getenv("JWT_SECRET", ""),
			TTL:    getdur("JWT_TTL", 12*time.Hour),
			Issuer: getenv("JWT_ISSUER", "timemachine"),
		},
		Max: MaxConfig{
			Enabled:        getenv("MAX_ENABLED", "false") == "true",
			APIToken:       getenv("MAX_API_TOKEN", ""),
			APIBase:        getenv("MAX_API_BASE", "https://botapi.max.ru"),
			BotHandle:      getenv("MAX_BOT_HANDLE", "timemachine_bot"),
			WebhookSecret:  getenv("MAX_WEBHOOK_SECRET", ""),
			DeepLinkPrefix: getenv("MAX_DEEPLINK_PREFIX", "https://max.ru/"),
			NotifyChatID:   getenv("MAX_NOTIFY_CHAT_ID", ""),
		},
		AI: AIConfig{
			Enabled:   getenv("AI_ENABLED", "false") == "true",
			Provider:  getenv("AI_PROVIDER", "stub"),
			APIKey:    getenv("AI_API_KEY", ""),
			Model:     getenv("AI_MODEL", "gpt-4o-mini"),
			BaseURL:   getenv("AI_BASE_URL", "https://api.openai.com/v1"),
			MaxTokens: getenvInt("AI_MAX_TOKENS", 600),
		},
	}

	if c.AppEnv == "production" {
		if c.JWT.Secret == "" {
			return nil, fmt.Errorf("JWT_SECRET must be set in production")
		}
		if len(c.JWT.Secret) < 32 {
			return nil, fmt.Errorf("JWT_SECRET must be at least 32 chars in production")
		}
		if c.Max.Enabled {
			if c.Max.WebhookSecret == "" {
				return nil, fmt.Errorf("MAX_WEBHOOK_SECRET must be set when MAX_ENABLED=true in production")
			}
			if c.Max.APIToken == "" {
				return nil, fmt.Errorf("MAX_API_TOKEN must be set when MAX_ENABLED=true in production")
			}
			if c.Max.NotifyChatID == "" {
				return nil, fmt.Errorf("MAX_NOTIFY_CHAT_ID must be set when MAX_ENABLED=true in production")
			}
		}
		if c.AI.Enabled && c.AI.Provider == "openai" && c.AI.APIKey == "" {
			return nil, fmt.Errorf("AI_API_KEY must be set when AI_PROVIDER=openai in production")
		}
	}
	if c.JWT.Secret == "" {
		// Dev fallback so the app boots without a configured secret.
		c.JWT.Secret = "dev-only-insecure-secret-change-me"
	}
	return c, nil
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getdur(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (c *Config) IsProd() bool { return c.AppEnv == "production" }
