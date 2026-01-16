package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrator handles database migrations.
type Migrator struct {
	m   *migrate.Migrate
	url string
}

// NewMigrator creates a new Migrator instance.
// It looks for migrations in ./migrations relative to the executable.
func NewMigrator(databaseURL string) (*Migrator, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	// Find migrations directory
	migrationsPath, err := findMigrationsDir()
	if err != nil {
		return nil, fmt.Errorf("failed to find migrations directory: %w", err)
	}

	// Create migrate instance with file source
	sourceURL := fmt.Sprintf("file://%s", filepath.ToSlash(migrationsPath))
	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	return &Migrator{
		m:   m,
		url: databaseURL,
	}, nil
}

// findMigrationsDir looks for the migrations directory in common locations.
func findMigrationsDir() (string, error) {
	// Try current directory first
	candidates := []string{
		"migrations",
		"./migrations",
	}

	// Try relative to executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "migrations"),
			filepath.Join(exeDir, "..", "migrations"),
		)
	}

	// Try relative to working directory
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "migrations"),
		)
	}

	for _, path := range candidates {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			return absPath, nil
		}
	}

	return "", fmt.Errorf("migrations directory not found in any of: %v", candidates)
}

// Up runs all available migrations.
func (migrator *Migrator) Up() error {
	if err := migrator.m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up failed: %w", err)
	}
	return nil
}

// Down rolls back all migrations.
func (migrator *Migrator) Down() error {
	if err := migrator.m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration down failed: %w", err)
	}
	return nil
}

// Steps runs n migrations. If n > 0, runs n up migrations. If n < 0, runs -n down migrations.
func (migrator *Migrator) Steps(n int) error {
	if err := migrator.m.Steps(n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration steps failed: %w", err)
	}
	return nil
}

// Version returns the current migration version and dirty state.
func (migrator *Migrator) Version() (uint, bool, error) {
	version, dirty, err := migrator.m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	return version, dirty, nil
}

// Force sets the migration version without running migrations.
// This is useful when recovering from a failed migration.
func (migrator *Migrator) Force(version int) error {
	if err := migrator.m.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version: %w", err)
	}
	return nil
}

// Close closes the migrator and releases resources.
func (migrator *Migrator) Close() error {
	sourceErr, dbErr := migrator.m.Close()
	if sourceErr != nil {
		return fmt.Errorf("failed to close source: %w", sourceErr)
	}
	if dbErr != nil {
		return fmt.Errorf("failed to close database: %w", dbErr)
	}
	return nil
}
