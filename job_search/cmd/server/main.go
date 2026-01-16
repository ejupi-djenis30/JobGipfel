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

	"job_search/internal/api"
	"job_search/internal/config"
	"job_search/internal/db"
	"job_search/internal/gemini"
	"job_search/internal/logger"
	"job_search/internal/search"
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
			fmt.Printf("job_search %s (%s)\n", version, commit)
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
	fmt.Println(`Job Search Service - Advanced Job Search with AI

Usage:
  job_search [command]

Commands:
  (none)    Start the HTTP server
  version   Show version information
  help      Show this help message

Environment Variables:
  DATABASE_URL          PostgreSQL connection string (required)
  GEMINI_API_KEY        Gemini API key (required for AI search)
  AUTH_SERVICE_URL      URL of auth_service (default: http://localhost:8082)
  PORT                  Server port (default: 8085)
  DEFAULT_PAGE_SIZE     Default results per page (default: 20)
  MAX_PAGE_SIZE         Maximum results per page (default: 100)`)
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
			slog.Warn("Failed to initialize Gemini client, AI search disabled", "error", err)
		} else {
			defer geminiClient.Close()
			slog.Info("Gemini client initialized (AI search enabled)")
		}
	} else {
		slog.Info("GEMINI_API_KEY not set, AI search disabled")
	}

	// Search service
	searchService := search.NewService(dbConn, geminiClient)
	slog.Info("Search service initialized")

	// Router
	router := api.SetupRouter(cfg, searchService)

	// Server
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Starting job search service", "address", addr, "version", version)
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
