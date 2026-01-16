package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type Migrator struct {
	m   *migrate.Migrate
	url string
}

func NewMigrator(databaseURL string) (*Migrator, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	migrationsPath, err := findMigrationsDir()
	if err != nil {
		return nil, fmt.Errorf("failed to find migrations directory: %w", err)
	}

	sourceURL := fmt.Sprintf("file://%s", filepath.ToSlash(migrationsPath))
	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	return &Migrator{m: m, url: databaseURL}, nil
}

func findMigrationsDir() (string, error) {
	candidates := []string{"migrations", "./migrations"}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(exeDir, "migrations"))
	}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "migrations"))
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

	return "", fmt.Errorf("migrations directory not found")
}

func (m *Migrator) Up() error {
	if err := m.m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up failed: %w", err)
	}
	return nil
}

func (m *Migrator) Down() error {
	if err := m.m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration down failed: %w", err)
	}
	return nil
}

func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	return version, dirty, nil
}

func (m *Migrator) Force(version int) error {
	return m.m.Force(version)
}

func (m *Migrator) Close() error {
	sourceErr, dbErr := m.m.Close()
	if sourceErr != nil {
		return sourceErr
	}
	return dbErr
}
