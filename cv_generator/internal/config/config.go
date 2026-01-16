package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration.
type Config struct {
	// Server
	Port string
	Host string

	// Auth Service
	AuthServiceURL string

	// Gemini AI
	GeminiAPIKey      string
	GeminiModel       string
	GeminiTemperature float32

	// PDF Generation
	ChromePath string // Path to Chrome/Chromium (optional)

	// Logging
	LogLevel  string
	LogFormat string
}

// Load loads configuration from environment variables.
func Load() *Config {
	return &Config{
		Port:              GetEnv("PORT", "8083"),
		Host:              GetEnv("HOST", "0.0.0.0"),
		AuthServiceURL:    GetEnv("AUTH_SERVICE_URL", "http://localhost:8082"),
		GeminiAPIKey:      GetEnv("GEMINI_API_KEY", ""),
		GeminiModel:       GetEnv("GEMINI_MODEL", "gemini-2.0-flash"),
		GeminiTemperature: GetEnvFloat32("GEMINI_TEMPERATURE", 0.7),
		ChromePath:        GetEnv("CHROME_PATH", ""),
		LogLevel:          GetEnv("LOG_LEVEL", "INFO"),
		LogFormat:         GetEnv("LOG_FORMAT", "json"),
	}
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

// GetEnvFloat32 returns the float32 value of an environment variable.
func GetEnvFloat32(key string, defaultValue float32) float32 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 32); err == nil {
			return float32(floatVal)
		}
	}
	return defaultValue
}
