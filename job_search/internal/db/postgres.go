package db

import (
	"context"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// NewDB creates a new PostgreSQL connection.
func NewDB(ctx context.Context, url string) (*sqlx.DB, error) {
	if url == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	db, err := sqlx.ConnectContext(ctx, "pgx", url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(30 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable pgvector extension
	_, err = db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		// Log warning but don't fail - extension might already exist or need superuser
		fmt.Printf("Warning: Could not create vector extension: %v\n", err)
	}

	return db, nil
}
