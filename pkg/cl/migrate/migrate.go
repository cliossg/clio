package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/google/uuid"
)

// Migration represents a database migration.
type Migration struct {
	Datetime string
	Name     string
	Up       string
	Down     string
}

// Migrator handles database migrations with version tracking.
type Migrator struct {
	db       *sql.DB
	log      logger.Logger
	assetsFS embed.FS
	engine   string
	path     string
}

// New creates a new Migrator.
func New(assetsFS embed.FS, engine string, log logger.Logger) *Migrator {
	return &Migrator{
		assetsFS: assetsFS,
		engine:   engine,
		log:      log,
	}
}

// SetDB sets the database connection.
func (m *Migrator) SetDB(db *sql.DB) {
	m.db = db
}

// SetPath sets a custom migration path.
func (m *Migrator) SetPath(path string) {
	m.path = path
}

// Run executes pending migrations in order.
// Creates migrations table if it doesn't exist.
// Returns error if any migration fails (transactional per migration).
func (m *Migrator) Run(ctx context.Context) error {
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("cannot create migrations table: %w", err)
	}

	fileMigrations, err := m.loadFileMigrations()
	if err != nil {
		return fmt.Errorf("cannot load file migrations: %w", err)
	}

	dbMigrations, err := m.loadDBMigrations()
	if err != nil {
		return fmt.Errorf("cannot load database migrations: %w", err)
	}

	pending := m.findPendingMigrations(fileMigrations, dbMigrations)

	if len(pending) == 0 {
		m.log.Info("No pending migrations")
		return nil
	}

	m.log.Infof("Running %d pending migration(s)", len(pending))

	for _, migration := range pending {
		if err := m.runMigration(ctx, migration); err != nil {
			return fmt.Errorf("migration %s-%s failed: %w", migration.Datetime, migration.Name, err)
		}
		m.log.Infof("Applied migration: %s-%s", migration.Datetime, migration.Name)
	}

	return nil
}

func (m *Migrator) ensureMigrationsTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS migrations (
		id TEXT PRIMARY KEY,
		datetime TEXT NOT NULL,
		name TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := m.db.ExecContext(ctx, query)
	return err
}

func (m *Migrator) loadFileMigrations() ([]Migration, error) {
	var migrations []Migration
	migrationPath := m.path
	if migrationPath == "" {
		migrationPath = fmt.Sprintf("assets/migrations/%s", m.engine)
	}

	err := fs.WalkDir(m.assetsFS, migrationPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".sql") {
			filename := filepath.Base(path)
			parts := strings.SplitN(filename, "-", 2)
			if len(parts) < 2 {
				return fmt.Errorf("invalid migration filename: %s", filename)
			}

			content, err := m.assetsFS.ReadFile(path)
			if err != nil {
				return fmt.Errorf("cannot read migration file %s: %w", path, err)
			}

			sections := strings.Split(string(content), "-- +migrate ")
			var upSection, downSection string
			for _, section := range sections {
				if strings.HasPrefix(section, "Up") {
					upSection = strings.TrimPrefix(section, "Up\n")
				} else if strings.HasPrefix(section, "Down") {
					downSection = strings.TrimPrefix(section, "Down\n")
				}
			}

			migrations = append(migrations, Migration{
				Datetime: parts[0],
				Name:     strings.TrimSuffix(parts[1], ".sql"),
				Up:       upSection,
				Down:     downSection,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Datetime < migrations[j].Datetime
	})

	return migrations, nil
}

func (m *Migrator) loadDBMigrations() ([]Migration, error) {
	rows, err := m.db.Query("SELECT datetime, name FROM migrations ORDER BY datetime")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var migration Migration
		if err := rows.Scan(&migration.Datetime, &migration.Name); err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}
	return migrations, nil
}

func (m *Migrator) findPendingMigrations(fileMigrations, dbMigrations []Migration) []Migration {
	dbMigrationsMap := make(map[string]struct{})
	for _, dbMigration := range dbMigrations {
		dbMigrationsMap[dbMigration.Datetime+dbMigration.Name] = struct{}{}
	}

	var pendingMigrations []Migration
	for _, fileMigration := range fileMigrations {
		if _, exists := dbMigrationsMap[fileMigration.Datetime+fileMigration.Name]; !exists {
			pendingMigrations = append(pendingMigrations, fileMigration)
		}
	}
	return pendingMigrations
}

func (m *Migrator) runMigration(ctx context.Context, migration Migration) error {
	if migration.Up == "" {
		return fmt.Errorf("no Up section found in migration %s-%s", migration.Datetime, migration.Name)
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, migration.Up); err != nil {
		return err
	}

	// Use UUID for SQLite (no gen_random_uuid())
	id := uuid.New().String()
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO migrations (id, datetime, name, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)",
		id, migration.Datetime, migration.Name); err != nil {
		return err
	}

	return tx.Commit()
}
