package gomigrator

import (
	"database/sql"
	"errors"
	"time"
)

type Migration struct {
	db *sql.DB
}

func NewMigration(db *sql.DB) *Migration {
	return &Migration{
		db: db,
	}
}

var ErrNotMigrationToRollback = errors.New("not migration to rollback")

type MigrationSchema struct {
	Name      string
	IsSuccess bool
	AppliedAt time.Time
	UpdatedAt time.Time
}

func (m *Migration) createMigrationTable() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS migration_schema (
			name VARCHAR PRIMARY KEY,
			is_success BOOLEAN,
			applied_at TIMESTAMPTZ NOT NULL,
		    updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	return err
}

func (m *Migration) getAppliedMigrations() (map[string]MigrationSchema, error) {
	rows, err := m.db.Query("SELECT name, is_success, applied_at, updated_at FROM migration_schema")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]MigrationSchema)
	for rows.Next() {
		var migrationSchema MigrationSchema
		err = rows.Scan(
			&migrationSchema.Name,
			&migrationSchema.IsSuccess,
			&migrationSchema.AppliedAt,
			&migrationSchema.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		applied[migrationSchema.Name] = migrationSchema
	}

	return applied, rows.Err()
}

func (m *Migration) applyMigration(migrationName string, isSuccess bool) error {
	_, err := m.db.Exec(
		"INSERT INTO migration_schema (name, is_success, applied_at, updated_at) VALUES ($1, $2, NOW(), NOW())",
		migrationName,
		isSuccess,
	)
	return err
}

func (m *Migration) rollbackMigration(migrationName string) error {
	_, err := m.db.Exec("DELETE FROM migration_schema WHERE name = $1", migrationName)
	return err
}

func (m *Migration) updateSuccessMigration(migrationName string, isSuccess bool) error {
	_, err := m.db.Exec(
		"UPDATE migration_schema SET is_success = $1, updated_at = NOW() WHERE name = $2",
		isSuccess,
		migrationName,
	)
	return err
}

func (m *Migration) getLastApplyMigration() (string, error) {
	row := m.db.QueryRow("SELECT name FROM migration_schema ORDER BY applied_at DESC LIMIT 1")
	var lastID string
	err := row.Scan(&lastID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotMigrationToRollback
		}
		return "", err
	}

	return lastID, nil
}
