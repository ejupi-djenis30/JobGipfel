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

	"auth_service/internal/api"
	"auth_service/internal/auth"
	"auth_service/internal/config"
	"auth_service/internal/db"
	"auth_service/internal/gemini"
	"auth_service/internal/logger"
	"auth_service/internal/store"
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

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log := logger.New(&logger.Config{
		Level:  logger.Level(cfg.LogLevel),
		Format: logger.Format(cfg.LogFormat),
	})
	log.SetDefault()

	// Check for subcommand
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "migrate":
			runMigrate(cfg, os.Args[2:])
			return
		case "version":
			fmt.Printf("auth_service %s (%s)\n", version, commit)
			return
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}

	// Run server
	runServer(cfg)
}

func printUsage() {
	fmt.Println(`Auth Service - Authentication and Profile Management

Usage:
  auth_service [command]

Commands:
  (none)    Start the HTTP server
  migrate   Run database migrations
  version   Show version information
  help      Show this help message

Run 'auth_service migrate --help' for migration options.

Environment Variables:
  DATABASE_URL          PostgreSQL connection string (required)
  JWT_SECRET            Secret key for JWT signing (required, min 32 chars)
  GOOGLE_CLIENT_ID      Google OAuth client ID
  GOOGLE_CLIENT_SECRET  Google OAuth client secret
  LINKEDIN_CLIENT_ID    LinkedIn OAuth client ID
  LINKEDIN_CLIENT_SECRET LinkedIn OAuth client secret
  GEMINI_API_KEY        Gemini API key for CV parsing
  PORT                  Server port (default: 8082)
  FRONTEND_URL          Frontend URL for CORS (default: http://localhost:3000)`)
}

func runServer(cfg *config.Config) {
	ctx := context.Background()

	// Validate required config
	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	if cfg.JWTSecret == "" {
		slog.Error("JWT_SECRET is required")
		os.Exit(1)
	}

	// Initialize database
	slog.Info("Connecting to database...")
	dbConn, err := db.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbConn.Close()
	slog.Info("Database connected")

	// Initialize store
	storeInstance := store.NewStore(dbConn)

	// Initialize JWT manager
	jwtManager, err := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiryHours, cfg.RefreshExpiryDays)
	if err != nil {
		slog.Error("Failed to initialize JWT manager", "error", err)
		os.Exit(1)
	}

	// Initialize OAuth providers
	var googleProvider *auth.GoogleProvider
	if cfg.IsGoogleOAuthEnabled() {
		googleProvider = auth.NewGoogleProvider(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)
		slog.Info("Google OAuth enabled")
	}

	var linkedInProvider *auth.LinkedInProvider
	if cfg.IsLinkedInOAuthEnabled() {
		linkedInProvider = auth.NewLinkedInProvider(cfg.LinkedInClientID, cfg.LinkedInClientSecret, cfg.LinkedInRedirectURL)
		slog.Info("LinkedIn OAuth enabled")
	}

	// Initialize Gemini client
	var geminiClient *gemini.Client
	if cfg.IsGeminiEnabled() {
		geminiClient, err = gemini.NewClient(ctx, gemini.ClientConfig{
			APIKey:      cfg.GeminiAPIKey,
			Model:       cfg.GeminiModel,
			Temperature: cfg.GeminiTemperature,
		})
		if err != nil {
			slog.Warn("Failed to initialize Gemini client", "error", err)
		} else {
			defer geminiClient.Close()
			slog.Info("Gemini CV parsing enabled")
		}
	}

	// Setup router
	router := api.SetupRouter(cfg, storeInstance, jwtManager, googleProvider, linkedInProvider, geminiClient)

	// Create server
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("Starting auth service", "address", addr, "version", version)
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

func runMigrate(cfg *config.Config, args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	databaseURL := fs.String("database", cfg.DatabaseURL, "PostgreSQL connection string")
	direction := fs.String("direction", "up", "Migration direction: up, down")
	steps := fs.Int("steps", 0, "Number of migrations to run (0 = all)")
	force := fs.Int("force", -1, "Force migration version (for recovery)")
	fs.Parse(args)

	if *databaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL is required. Set via environment variable or --database flag.")
		os.Exit(1)
	}

	fmt.Printf("Running migrations (%s)...\n", *direction)

	migrator, err := db.NewMigrator(*databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create migrator: %v\n", err)
		os.Exit(1)
	}
	defer migrator.Close()

	// Force version if specified
	if *force >= 0 {
		fmt.Printf("Forcing migration version to %d...\n", *force)
		if err := migrator.Force(*force); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to force migration version: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Migration version forced successfully.")
		return
	}

	// Run migrations
	if *steps != 0 {
		n := *steps
		if *direction == "down" {
			n = -n
		}
		if err := migrator.Steps(n); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Migration failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		switch *direction {
		case "up":
			if err := migrator.Up(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Migration up failed: %v\n", err)
				os.Exit(1)
			}
		case "down":
			if err := migrator.Down(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Migration down failed: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "Error: Invalid direction '%s'. Use 'up' or 'down'.\n", *direction)
			os.Exit(1)
		}
	}

	// Show current version
	ver, dirty, err := migrator.Version()
	if err != nil {
		fmt.Printf("Migrations applied successfully.\n")
	} else {
		dirtyStr := ""
		if dirty {
			dirtyStr = " (dirty)"
		}
		fmt.Printf("Migrations applied successfully. Current version: %d%s\n", ver, dirtyStr)
	}
}
