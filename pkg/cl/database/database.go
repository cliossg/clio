package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"github.com/cliossg/clio/pkg/cl/config"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/cliossg/clio/pkg/cl/migrate"
)

// Database manages the SQLite database connection and lifecycle.
type Database struct {
	DB            *sql.DB
	assetsFS      embed.FS
	migrationPath string
	cfg           *config.Config
	log           logger.Logger
}

// New creates a new Database instance.
func New(assetsFS embed.FS, cfg *config.Config, log logger.Logger) *Database {
	return &Database{
		assetsFS: assetsFS,
		cfg:      cfg,
		log:      log,
	}
}

// SetMigrationPath sets a custom migration path.
func (d *Database) SetMigrationPath(path string) {
	d.migrationPath = path
}

// Start opens the database connection and runs migrations.
func (d *Database) Start(ctx context.Context) error {
	// Ensure directory exists
	dbDir := filepath.Dir(d.cfg.Database.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("cannot create database directory: %w", err)
	}

	// Open SQLite database with WAL mode for better concurrency
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON", d.cfg.Database.Path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("cannot open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("cannot ping database: %w", err)
	}

	d.DB = db
	d.log.Info("Database connection established")

	// Run migrations
	migrator := migrate.New(d.assetsFS, "sqlite", d.log)
	migrator.SetDB(d.DB)
	if d.migrationPath != "" {
		migrator.SetPath(d.migrationPath)
	}
	if err := migrator.Run(ctx); err != nil {
		return fmt.Errorf("cannot run migrations: %w", err)
	}

	return nil
}

// Stop closes the database connection.
func (d *Database) Stop(ctx context.Context) error {
	if d.DB != nil {
		d.log.Info("Closing database connection")
		return d.DB.Close()
	}
	return nil
}

// GetDB returns the underlying sql.DB.
func (d *Database) GetDB() *sql.DB {
	return d.DB
}
