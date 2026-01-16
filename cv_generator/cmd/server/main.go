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

	"cv_generator/internal/api"
	"cv_generator/internal/auth"
	"cv_generator/internal/config"
	"cv_generator/internal/generator"
	"cv_generator/internal/logger"
	"cv_generator/internal/pdf"
)

var (
	version = "1.0.0"
	commit  = "dev"
)

func main() {
	// Load .env file if present
	if err := godotenv.Load(); err != nil {
		// Ignore error if .env doesn't exist
	}

	// Check for version command
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("cv_generator %s (%s)\n", version, commit)
			return
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log := logger.New(&logger.Config{
		Level:  logger.Level(cfg.LogLevel),
		Format: logger.Format(cfg.LogFormat),
	})
	log.SetDefault()

	// Run server
	runServer(cfg)
}

func printUsage() {
	fmt.Println(`CV Generator Service - Generate professional CVs/Resumes

Usage:
  cv_generator [command]

Commands:
  (none)    Start the HTTP server
  version   Show version information
  help      Show this help message

Environment Variables:
  GEMINI_API_KEY        Gemini API key (required)
  AUTH_SERVICE_URL      URL of auth_service (default: http://localhost:8082)
  PORT                  Server port (default: 8083)
  CHROME_PATH           Path to Chrome/Chromium (optional, auto-detected)`)
}

func runServer(cfg *config.Config) {
	ctx := context.Background()

	// Validate required config
	if cfg.GeminiAPIKey == "" {
		slog.Error("GEMINI_API_KEY is required")
		os.Exit(1)
	}

	// Initialize auth client
	authClient := auth.NewClient(cfg.AuthServiceURL)
	slog.Info("Auth service client initialized", "url", cfg.AuthServiceURL)

	// Initialize Gemini client
	geminiClient, err := generator.NewClient(ctx, generator.ClientConfig{
		APIKey:      cfg.GeminiAPIKey,
		Model:       cfg.GeminiModel,
		Temperature: cfg.GeminiTemperature,
	})
	if err != nil {
		slog.Error("Failed to initialize Gemini client", "error", err)
		os.Exit(1)
	}
	defer geminiClient.Close()
	slog.Info("Gemini client initialized", "model", cfg.GeminiModel)

	// Initialize PDF converter
	pdfConverter := pdf.NewConverter(cfg.ChromePath)
	slog.Info("PDF converter initialized")

	// Setup router
	router := api.SetupRouter(cfg, authClient, geminiClient, pdfConverter)

	// Create server
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  60 * time.Second,  // Longer timeout for PDF generation
		WriteTimeout: 120 * time.Second, // Even longer for large PDFs
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("Starting CV generator service", "address", addr, "version", version)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server stopped")
}
