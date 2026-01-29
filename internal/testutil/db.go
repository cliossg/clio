package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// NewTestDB creates a new in-memory SQLite database with all migrations applied.
func NewTestDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}

	if err := ApplyMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot apply migrations: %w", err)
	}

	return db, nil
}

// ApplyMigrations applies all SQL migrations to the database.
func ApplyMigrations(db *sql.DB) error {
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		return fmt.Errorf("migrations directory not found")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("cannot read migrations directory: %w", err)
	}

	var migrations []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrations = append(migrations, entry.Name())
		}
	}
	sort.Strings(migrations)

	for _, migration := range migrations {
		content, err := os.ReadFile(filepath.Join(migrationsDir, migration))
		if err != nil {
			return fmt.Errorf("cannot read migration %s: %w", migration, err)
		}

		upSQL := extractUpMigration(string(content))
		if upSQL == "" {
			continue
		}

		if _, err := db.Exec(upSQL); err != nil {
			return fmt.Errorf("cannot execute migration %s: %w", migration, err)
		}
	}

	return nil
}

func findMigrationsDir() string {
	paths := []string{
		"assets/migrations/sqlite",
		"../assets/migrations/sqlite",
		"../../assets/migrations/sqlite",
		"../../../assets/migrations/sqlite",
		"../../../../assets/migrations/sqlite",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

func extractUpMigration(content string) string {
	lines := strings.Split(content, "\n")
	var upLines []string
	inUp := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "+migrate Up") {
			inUp = true
			continue
		}
		if strings.Contains(trimmed, "+migrate Down") {
			break
		}
		if inUp {
			upLines = append(upLines, line)
		}
	}

	return strings.Join(upLines, "\n")
}

// TestDBProvider implements DBProvider for testing.
type TestDBProvider struct {
	DB *sql.DB
}

func (p *TestDBProvider) GetDB() *sql.DB {
	return p.DB
}
