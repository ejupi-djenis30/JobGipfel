package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port string
	Host string

	DatabaseURL    string
	AuthServiceURL string

	LogLevel  string
	LogFormat string
}

func Load() *Config {
	return &Config{
		Port:           GetEnv("PORT", "8087"),
		Host:           GetEnv("HOST", "0.0.0.0"),
		DatabaseURL:    GetEnv("DATABASE_URL", ""),
		AuthServiceURL: GetEnv("AUTH_SERVICE_URL", "http://localhost:8082"),
		LogLevel:       GetEnv("LOG_LEVEL", "INFO"),
		LogFormat:      GetEnv("LOG_FORMAT", "json"),
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
