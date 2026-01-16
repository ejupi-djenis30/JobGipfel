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

	"analytics_service/internal/analytics"
	"analytics_service/internal/api"
	"analytics_service/internal/config"
	"analytics_service/internal/db"
	"analytics_service/internal/logger"
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
			fmt.Printf("analytics_service %s (%s)\n", version, commit)
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
	fmt.Println(`Analytics Service - Application Tracking & Market Insights

Usage:
  analytics_service [command]

Commands:
  (none)    Start the HTTP server
  version   Show version information
  help      Show this help message

Environment Variables:
  DATABASE_URL          PostgreSQL connection string (required)
  AUTH_SERVICE_URL      URL of auth_service (default: http://localhost:8082)
  PORT                  Server port (default: 8087)`)
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

	// Analytics service
	analyticsService := analytics.NewService(dbConn)
	slog.Info("Analytics service initialized")

	// Router
	router := api.SetupRouter(cfg, analyticsService)

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
		slog.Info("Starting analytics service", "address", addr, "version", version)
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
