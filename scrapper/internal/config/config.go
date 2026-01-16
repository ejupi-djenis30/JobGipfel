package config

import (
	"os"
	"strconv"
)

// AIProcessingMode defines how jobs should be processed.
type AIProcessingMode string

const (
	AIProcessingModeNone      AIProcessingMode = "none"      // Don't call AI service
	AIProcessingModeProcess   AIProcessingMode = "process"   // Normalize + translate
	AIProcessingModeNormalize AIProcessingMode = "normalize" // Normalize only
	AIProcessingModeTranslate AIProcessingMode = "translate" // Translate only
)

// Config holds all application configuration.
type Config struct {
	// Database
	DatabaseURL string

	// Logging
	LogLevel  string
	LogFormat string

	// Scraper defaults
	ScraperDelayMinMs      int
	ScraperDelayMaxMs      int
	ScraperDefaultMaxPages int
	ScraperDefaultDaysBack int

	// AI Processing Service Integration
	AIServiceURL     string           // URL of the ai_job_processing microservice
	AIProcessingMode AIProcessingMode // How to process jobs: none, process, normalize, translate
}

// Load loads configuration from environment variables.
func Load() *Config {
	return &Config{
		DatabaseURL:            GetEnv("DATABASE_URL", ""),
		LogLevel:               GetEnv("LOG_LEVEL", "INFO"),
		LogFormat:              GetEnv("LOG_FORMAT", "text"),
		ScraperDelayMinMs:      GetEnvInt("SCRAPER_DELAY_MIN_MS", 2000),
		ScraperDelayMaxMs:      GetEnvInt("SCRAPER_DELAY_MAX_MS", 5000),
		ScraperDefaultMaxPages: GetEnvInt("SCRAPER_DEFAULT_MAX_PAGES", 0),
		ScraperDefaultDaysBack: GetEnvInt("SCRAPER_DEFAULT_DAYS_BACK", 60),
		AIServiceURL:           GetEnv("AI_SERVICE_URL", ""),
		AIProcessingMode:       AIProcessingMode(GetEnv("AI_PROCESSING_MODE", "none")),
	}
}

// IsAIEnabled returns true if AI processing is configured.
func (c *Config) IsAIEnabled() bool {
	return c.AIServiceURL != "" && c.AIProcessingMode != AIProcessingModeNone
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

// GetEnvBool returns the boolean value of an environment variable or a default value.
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
