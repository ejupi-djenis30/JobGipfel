package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	// Server
	Port string
	Host string

	// Database
	DatabaseURL string

	// JWT
	JWTSecret         string
	JWTExpiryHours    int
	RefreshExpiryDays int

	// Google OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// LinkedIn OAuth
	LinkedInClientID     string
	LinkedInClientSecret string
	LinkedInRedirectURL  string

	// Gemini AI (for CV parsing)
	GeminiAPIKey      string
	GeminiModel       string
	GeminiTemperature float32

	// Frontend
	FrontendURL string

	// Logging
	LogLevel  string
	LogFormat string
}

// Load loads configuration from environment variables.
func Load() *Config {
	return &Config{
		Port:                 GetEnv("PORT", "8082"),
		Host:                 GetEnv("HOST", "0.0.0.0"),
		DatabaseURL:          GetEnv("DATABASE_URL", ""),
		JWTSecret:            GetEnv("JWT_SECRET", ""),
		JWTExpiryHours:       GetEnvInt("JWT_EXPIRY_HOURS", 24),
		RefreshExpiryDays:    GetEnvInt("REFRESH_EXPIRY_DAYS", 7),
		GoogleClientID:       GetEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:   GetEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:    GetEnv("GOOGLE_REDIRECT_URL", "http://localhost:8082/api/v1/auth/google/callback"),
		LinkedInClientID:     GetEnv("LINKEDIN_CLIENT_ID", ""),
		LinkedInClientSecret: GetEnv("LINKEDIN_CLIENT_SECRET", ""),
		LinkedInRedirectURL:  GetEnv("LINKEDIN_REDIRECT_URL", "http://localhost:8082/api/v1/auth/linkedin/callback"),
		GeminiAPIKey:         GetEnv("GEMINI_API_KEY", ""),
		GeminiModel:          GetEnv("GEMINI_MODEL", "gemini-2.0-flash"),
		GeminiTemperature:    GetEnvFloat32("GEMINI_TEMPERATURE", 0.3),
		FrontendURL:          GetEnv("FRONTEND_URL", "http://localhost:3000"),
		LogLevel:             GetEnv("LOG_LEVEL", "INFO"),
		LogFormat:            GetEnv("LOG_FORMAT", "json"),
	}
}

// IsGoogleOAuthEnabled returns true if Google OAuth is configured.
func (c *Config) IsGoogleOAuthEnabled() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

// IsLinkedInOAuthEnabled returns true if LinkedIn OAuth is configured.
func (c *Config) IsLinkedInOAuthEnabled() bool {
	return c.LinkedInClientID != "" && c.LinkedInClientSecret != ""
}

// IsGeminiEnabled returns true if Gemini is configured.
func (c *Config) IsGeminiEnabled() bool {
	return c.GeminiAPIKey != ""
}

// GetEnv returns the value of an environment variable or a default value.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt returns the integer value of an environment variable or a default value.
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetEnvFloat32 returns the float32 value of an environment variable or a default value.
func GetEnvFloat32(key string, defaultValue float32) float32 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 32); err == nil {
			return float32(floatVal)
		}
	}
	return defaultValue
}

// GetEnvBool returns the boolean value of an environment variable or a default value.
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// GetEnvSlice returns a slice from a comma-separated environment variable.
func GetEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return defaultValue
}
