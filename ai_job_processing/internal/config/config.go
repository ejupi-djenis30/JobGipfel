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

	// Gemini AI
	GeminiAPIKey      string
	GeminiModel       string
	GeminiTemperature float32

	// Languages
	TargetLanguages []string
	SourceLanguage  string

	// Logging
	LogLevel  string
	LogFormat string

	// Rate limiting
	RateLimitRPM int
}

// Load loads configuration from environment variables.
func Load() *Config {
	// Parse target languages
	targetLangs := GetEnv("TARGET_LANGUAGES", "de,fr,it,en")
	languages := parseLanguages(targetLangs)

	return &Config{
		Port:              GetEnv("PORT", "8081"),
		Host:              GetEnv("HOST", "0.0.0.0"),
		DatabaseURL:       GetEnv("DATABASE_URL", ""),
		GeminiAPIKey:      GetEnv("GEMINI_API_KEY", ""),
		GeminiModel:       GetEnv("GEMINI_MODEL", "gemini-2.0-flash"),
		GeminiTemperature: GetEnvFloat32("GEMINI_TEMPERATURE", 0.3),
		TargetLanguages:   languages,
		SourceLanguage:    GetEnv("SOURCE_LANGUAGE", ""),
		LogLevel:          GetEnv("LOG_LEVEL", "INFO"),
		LogFormat:         GetEnv("LOG_FORMAT", "json"),
		RateLimitRPM:      GetEnvInt("RATE_LIMIT_RPM", 60),
	}
}

// parseLanguages parses comma-separated language codes.
func parseLanguages(s string) []string {
	if s == "" {
		return []string{"de", "fr", "it", "en"}
	}

	parts := strings.Split(s, ",")
	languages := make([]string, 0, len(parts))
	for _, p := range parts {
		lang := strings.TrimSpace(strings.ToLower(p))
		if lang != "" {
			languages = append(languages, lang)
		}
	}
	return languages
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

// LanguageNames maps ISO codes to human-readable names.
var LanguageNames = map[string]string{
	"de": "German",
	"fr": "French",
	"it": "Italian",
	"en": "English",
	"rm": "Romansh",
	"es": "Spanish",
	"pt": "Portuguese",
}

// GetLanguageName returns the human-readable name for a language code.
func GetLanguageName(code string) string {
	if name, ok := LanguageNames[code]; ok {
		return name
	}
	return code
}
