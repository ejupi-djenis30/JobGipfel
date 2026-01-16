package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	// Server
	Port string
	Host string

	// Database
	DatabaseURL string

	// Auth Service
	AuthServiceURL string

	// CV Generator Service
	CVGeneratorURL string

	// Gemini AI
	GeminiAPIKey      string
	GeminiModel       string
	GeminiTemperature float32

	// Email - SMTP (platform fallback)
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	// Gmail OAuth
	GoogleClientID     string
	GoogleClientSecret string

	// Web Automation
	ChromePath      string
	AutomationDelay time.Duration // Delay between actions
	MaxRetries      int

	// Rate Limiting
	RateLimitPerHour int

	// Logging
	LogLevel  string
	LogFormat string
}

// Load loads configuration from environment variables.
func Load() *Config {
	return &Config{
		Port:               GetEnv("PORT", "8084"),
		Host:               GetEnv("HOST", "0.0.0.0"),
		DatabaseURL:        GetEnv("DATABASE_URL", ""),
		AuthServiceURL:     GetEnv("AUTH_SERVICE_URL", "http://localhost:8082"),
		CVGeneratorURL:     GetEnv("CV_GENERATOR_URL", "http://localhost:8083"),
		GeminiAPIKey:       GetEnv("GEMINI_API_KEY", ""),
		GeminiModel:        GetEnv("GEMINI_MODEL", "gemini-2.0-flash"),
		GeminiTemperature:  GetEnvFloat32("GEMINI_TEMPERATURE", 0.7),
		SMTPHost:           GetEnv("SMTP_HOST", ""),
		SMTPPort:           GetEnvInt("SMTP_PORT", 587),
		SMTPUsername:       GetEnv("SMTP_USERNAME", ""),
		SMTPPassword:       GetEnv("SMTP_PASSWORD", ""),
		SMTPFrom:           GetEnv("SMTP_FROM", ""),
		GoogleClientID:     GetEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: GetEnv("GOOGLE_CLIENT_SECRET", ""),
		ChromePath:         GetEnv("CHROME_PATH", ""),
		AutomationDelay:    time.Duration(GetEnvInt("AUTOMATION_DELAY_MS", 2000)) * time.Millisecond,
		MaxRetries:         GetEnvInt("MAX_RETRIES", 3),
		RateLimitPerHour:   GetEnvInt("RATE_LIMIT_PER_HOUR", 20),
		LogLevel:           GetEnv("LOG_LEVEL", "INFO"),
		LogFormat:          GetEnv("LOG_FORMAT", "json"),
	}
}

// IsSMTPEnabled returns true if SMTP is configured.
func (c *Config) IsSMTPEnabled() bool {
	return c.SMTPHost != "" && c.SMTPUsername != ""
}

// IsGeminiEnabled returns true if Gemini is configured.
func (c *Config) IsGeminiEnabled() bool {
	return c.GeminiAPIKey != ""
}

// GetEnv returns the value of an environment variable or a default.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt returns the integer value of an environment variable.
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetEnvFloat32 returns the float32 value of an environment variable.
func GetEnvFloat32(key string, defaultValue float32) float32 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 32); err == nil {
			return float32(floatVal)
		}
	}
	return defaultValue
}
