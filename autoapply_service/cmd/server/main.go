package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"autoapply_service/internal/api"
	"autoapply_service/internal/auth"
	"autoapply_service/internal/config"
	"autoapply_service/internal/cvgen"
	"autoapply_service/internal/db"
	"autoapply_service/internal/email"
	"autoapply_service/internal/gemini"
	"autoapply_service/internal/logger"
	"autoapply_service/internal/selenium"
	"autoapply_service/internal/store"
)

var (
	version = "1.0.0"
	commit  = "dev"
)

func main() {
	godotenv.Load()

	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "migrate":
			runMigrate(os.Args[2:])
			return
		case "version":
			fmt.Printf("autoapply_service %s (%s)\n", version, commit)
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
	fmt.Println(`AutoApply Service - Automated Job Applications

Usage:
  autoapply_service [command]

Commands:
  (none)    Start the HTTP server
  migrate   Run database migrations
  version   Show version information
  help      Show this help message

Environment Variables:
  DATABASE_URL          PostgreSQL connection string (required)
  GEMINI_API_KEY        Gemini API key (required)
  AUTH_SERVICE_URL      URL of auth_service (default: http://localhost:8082)
  CV_GENERATOR_URL      URL of cv_generator (default: http://localhost:8083)
  SMTP_HOST             SMTP server host
  SMTP_PORT             SMTP server port (default: 587)
  SMTP_USERNAME         SMTP username
  SMTP_PASSWORD         SMTP password
  SMTP_FROM             Sender email address
  PORT                  Server port (default: 8084)
  RATE_LIMIT_PER_HOUR   Max applications per hour (default: 20)`)
}

func runServer(cfg *config.Config) {
	ctx := context.Background()

	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	if cfg.GeminiAPIKey == "" {
		slog.Error("GEMINI_API_KEY is required")
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

	// Store
	storeInstance := store.NewStore(dbConn)

	// Auth client
	authClient := auth.NewClient(cfg.AuthServiceURL)
	slog.Info("Auth client initialized", "url", cfg.AuthServiceURL)

	// CV generator client
	cvgenClient := cvgen.NewClient(cfg.CVGeneratorURL)
	slog.Info("CV generator client initialized", "url", cfg.CVGeneratorURL)

	// Email sender
	var smtpConfig *email.SMTPConfig
	if cfg.IsSMTPEnabled() {
		smtpConfig = &email.SMTPConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
		}
		slog.Info("SMTP configured", "host", cfg.SMTPHost)
	}
	emailSender := email.NewSender(smtpConfig)

	// Gemini client
	geminiClient, err := gemini.NewClient(ctx, gemini.ClientConfig{
		APIKey:      cfg.GeminiAPIKey,
		Model:       cfg.GeminiModel,
		Temperature: cfg.GeminiTemperature,
	})
	if err != nil {
		slog.Error("Failed to initialize Gemini client", "error", err)
		os.Exit(1)
	}
	defer geminiClient.Close()
	slog.Info("Gemini client initialized")

	// Web automation
	automation := selenium.NewAutomation(cfg.ChromePath, cfg.AutomationDelay)
	slog.Info("Web automation initialized", "delay", cfg.AutomationDelay)

	// Router
	router := api.SetupRouter(cfg, storeInstance, authClient, cvgenClient, emailSender, geminiClient, automation)

	// Server
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 300 * time.Second, // Long timeout for automation
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Starting autoapply service", "address", addr, "version", version)
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

func runMigrate(args []string) {
	cfg := config.Load()

	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	databaseURL := fs.String("database", cfg.DatabaseURL, "PostgreSQL connection string")
	direction := fs.String("direction", "up", "Migration direction: up, down")
	fs.Parse(args)

	if *databaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL is required")
		os.Exit(1)
	}

	fmt.Printf("Running migrations (%s)...\n", *direction)

	migrator, err := db.NewMigrator(*databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer migrator.Close()

	switch *direction {
	case "up":
		if err := migrator.Up(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "down":
		if err := migrator.Down(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("Migrations applied successfully.")
}
