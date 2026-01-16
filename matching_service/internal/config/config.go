package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration.
type Config struct {
	Port string
	Host string

	DatabaseURL    string
	AuthServiceURL string

	GeminiAPIKey      string
	GeminiModel       string
	EmbeddingModel    string
	GeminiTemperature float32

	CacheEnabled bool
	CacheTTL     int // seconds

	LogLevel  string
	LogFormat string
}

func Load() *Config {
	return &Config{
		Port:              GetEnv("PORT", "8086"),
		Host:              GetEnv("HOST", "0.0.0.0"),
		DatabaseURL:       GetEnv("DATABASE_URL", ""),
		AuthServiceURL:    GetEnv("AUTH_SERVICE_URL", "http://localhost:8082"),
		GeminiAPIKey:      GetEnv("GEMINI_API_KEY", ""),
		GeminiModel:       GetEnv("GEMINI_MODEL", "gemini-2.0-flash"),
		EmbeddingModel:    GetEnv("EMBEDDING_MODEL", "text-embedding-004"),
		GeminiTemperature: GetEnvFloat32("GEMINI_TEMPERATURE", 0.3),
		CacheEnabled:      GetEnvBool("CACHE_ENABLED", true),
		CacheTTL:          GetEnvInt("CACHE_TTL", 3600),
		LogLevel:          GetEnv("LOG_LEVEL", "INFO"),
		LogFormat:         GetEnv("LOG_FORMAT", "json"),
	}
}

func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func GetEnvFloat32(key string, defaultValue float32) float32 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 32); err == nil {
			return float32(floatVal)
		}
	}
	return defaultValue
}

func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
