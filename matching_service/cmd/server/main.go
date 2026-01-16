package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"matching_service/internal/api"
	"matching_service/internal/auth"
	"matching_service/internal/config"
	"matching_service/internal/db"
	"matching_service/internal/gemini"
	"matching_service/internal/logger"
	"matching_service/internal/matcher"
)

var (
	version = "1.0.0"
	commit  = "dev"
)

func main() {
	godotenv.Load()

	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("matching_service %s (%s)\n", version, commit)
			return
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}

	cfg := config.Load()
	log := logger.New(&logger.Config{
		Level:  logger.Level(cfg.LogLevel),
		Format: logger.Format(cfg.LogFormat),
	})
	log.SetDefault()

	runServer(cfg)
}

func printUsage() {
	fmt.Println(`Matching Service - AI-Powered Job Recommendations

Usage:
  matching_service [command]

Commands:
  (none)    Start the HTTP server
  version   Show version information
  help      Show this help message

Environment Variables:
  DATABASE_URL          PostgreSQL connection string (required)
  GEMINI_API_KEY        Gemini API key (required for AI scoring)
  AUTH_SERVICE_URL      URL of auth_service (default: http://localhost:8082)
  PORT                  Server port (default: 8086)`)
}

func runServer(cfg *config.Config) {
	ctx := context.Background()

	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	// Database
	slog.Info("Connecting to database...")
	dbConn, err := db.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbConn.Close()
	slog.Info("Database connected")

	// Auth client
	authClient := auth.NewClient(cfg.AuthServiceURL)
	slog.Info("Auth client initialized", "url", cfg.AuthServiceURL)

	// Gemini client (optional)
	var geminiClient *gemini.Client
	if cfg.GeminiAPIKey != "" {
		geminiClient, err = gemini.NewClient(ctx, gemini.ClientConfig{
			APIKey:         cfg.GeminiAPIKey,
			Model:          cfg.GeminiModel,
			EmbeddingModel: cfg.EmbeddingModel,
			Temperature:    cfg.GeminiTemperature,
		})
		if err != nil {
			slog.Warn("Failed to initialize Gemini, AI features disabled", "error", err)
		} else {
			defer geminiClient.Close()
			slog.Info("Gemini client initialized (AI matching enabled)")
		}
	} else {
		slog.Info("GEMINI_API_KEY not set, using rule-based matching")
	}

	// Matcher service
	matcherService := matcher.NewService(dbConn, authClient, geminiClient)
	slog.Info("Matcher service initialized")

	// Router
	router := api.SetupRouter(cfg, matcherService)

	// Server
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Starting matching service", "address", addr, "version", version)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}
	slog.Info("Server stopped")
}
