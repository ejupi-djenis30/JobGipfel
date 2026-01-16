package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// NewDB creates a new PostgreSQL connection using sqlx.
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

	return db, nil
}
