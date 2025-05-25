package gomigrator

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	up   = "-- +up"
	down = "-- +down"
)

type Migrator interface {
	Create(name string) error
	Up() error
	Down() error
	Redo() error
	Status() error
	VersionDB() error
}

type migrator struct {
	logg      *Logger
	db        *sql.DB
	locker    *Locker
	migration *Migration
	dir       string
}

func NewMigrator(db *sql.DB, dir string) Migrator {
	return &migrator{
		logg:      NewLogger(),
		locker:    NewLocker(db),
		migration: NewMigration(db),
		db:        db,
		dir:       dir,
	}
}

var (
	ErrCreateFile        = errors.New("failed create file")
	ErrWriteFile         = errors.New("failed writing file")
	ErrIncorrectTemplate = errors.New("incorrect template file")
	ErrIncorrectType     = errors.New("incorrect migration type")
)

func (m *migrator) Create(name string) error {
	timestamp := time.Now().UTC().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", timestamp, name)
	path := filepath.Join(m.dir, filename)

	dataTemplate := []string{up, "", down, ""}
	template := strings.Join(dataTemplate[0:], "\n")

	file, err := os.Create(path)
	if err != nil {
		m.logg.Error(fmt.Sprintf("Failed create migration file: %s : %s", path, err.Error()))
		return ErrCreateFile
	}
	defer file.Close()

	_, err = file.WriteString(template)
	if err != nil {
		m.logg.Error(fmt.Sprintf("Error writing migration: %s : %s", path, err.Error()))
		return ErrWriteFile
	}

	m.logg.Info(fmt.Sprintf("Created new migration: %s", path))
	return nil
}

func (m *migrator) Up() error {
	err := m.locker.Lock()
	if err != nil {
		m.logg.Error(fmt.Sprintf("Failed pg_advisory_lock: %s", err.Error()))
		return err
	}
	defer m.locker.Unlock()

	err = m.migration.createMigrationTable()
	if err != nil {
		m.logg.Error(fmt.Sprintf("Failed create migration table: %s", err.Error()))
		return err
	}

	applied, err := m.migration.getAppliedMigrations()
	if err != nil {
		m.logg.Error(fmt.Sprintf("Failed get applied migrations: %s", err.Error()))
		return err
	}

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		m.logg.Error(fmt.Sprintf("Failed read migration dir: %s", err.Error()))
		return err
	}

	migrations := make([]string, 0)
	for _, file := range entries {
		name := file.Name()
		if !file.IsDir() && filepath.Ext(name) == ".sql" {
			migrations = append(migrations, name)
		}
	}
	sort.Strings(migrations)

	for _, file := range migrations {
		name := strings.TrimSuffix(file, ".sql")
		_, ok := applied[name]
		if ok {
			continue
		}

		content, err := os.ReadFile(filepath.Join(m.dir, file))
		if err != nil {
			m.logg.Error(fmt.Sprintf("Failed read migration %s: %s", file, err.Error()))
			return err
		}

		upSQL, err := getSQLByTemplate(string(content), up)
		if err != nil {
			m.logg.Error(fmt.Sprintf("Failed parse template %s: %s", file, err.Error()))
			return err
		}

		_, err = m.db.Exec(upSQL)
		if err != nil {
			errApply := m.migration.applyMigration(name, false)
			if errApply != nil {
				m.logg.Error(fmt.Sprintf(
					"Failed execute up AND writing migration_schema %s: %s | %s",
					name,
					err.Error(),
					errApply.Error(),
				))
				return fmt.Errorf("execute up and writing migration_schema: %w | %w", err, errApply)
			}

			m.logg.Error(fmt.Sprintf("Failed execute up migration %s: %s", file, err.Error()))
			return err
		}

		err = m.migration.applyMigration(name, true)
		if err != nil {
			m.logg.Error(fmt.Sprintf(
				"Failed writing migration_schema during apply migration %s: %s",
				name,
				err.Error(),
			))
			return err
		}

		m.logg.Info(fmt.Sprintf("Migration has been applied: %s", name))
	}

	return nil
}

func (m *migrator) Down() error {
	err := m.locker.Lock()
	if err != nil {
		m.logg.Error(fmt.Sprintf("Failed pg_advisory_lock: %s", err.Error()))
		return err
	}
	defer m.locker.Unlock()

	lastMigration, err := m.migration.getLastApplyMigration()
	if err != nil {
		m.logg.Error(fmt.Sprintf("Failed get last migration: %s", err.Error()))
		return err
	}

	file := filepath.Join(m.dir, lastMigration+".sql")
	content, err := os.ReadFile(file)
	if err != nil {
		m.logg.Error(fmt.Sprintf("Failed read migration file: %s", err.Error()))
		return err
	}

	downSQL, err := getSQLByTemplate(string(content), down)
	if err != nil {
		m.logg.Error(fmt.Sprintf("Failed parse template %s: %s", file, err.Error()))
		return err
	}

	_, err = m.db.Exec(downSQL)
	if err != nil {
		errUpdate := m.migration.updateSuccessMigration(lastMigration, false)
		if errUpdate != nil {
			m.logg.Error(fmt.Sprintf(
				"Failed execute down AND writing migration_schema %s: %s | %s",
				lastMigration,
				err.Error(),
				errUpdate.Error(),
			))
			return fmt.Errorf("execute down and writing migration_schema: %w | %w", err, errUpdate)
		}

		m.logg.Error(fmt.Sprintf("Failed execute down migration %s: %s", file, err.Error()))
		return err
	}

	err = m.migration.rollbackMigration(lastMigration)
	if err != nil {
		m.logg.Error(fmt.Sprintf(
			"Failed delete record migration_schema during rollback migration %s: %s",
			lastMigration,
			err.Error()),
		)
		return err
	}

	m.logg.Info(fmt.Sprintf("Migration has been rollback: %s", lastMigration))
	return nil
}

func (m *migrator) Redo() error {
	err := m.Down()
	if err != nil {
		return err
	}

	return m.Up()
}

func (m *migrator) Status() error {
	applied, err := m.migration.getAppliedMigrations()
	if err != nil {
		m.logg.Error(fmt.Sprintf("Failed get applied migrations: %s", err.Error()))
		return err
	}

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		m.logg.Error(fmt.Sprintf("Failed read migration dir: %s", err.Error()))
		return err
	}

	migrationNames := make([]string, 0)
	for _, file := range entries {
		name := file.Name()
		if !file.IsDir() && filepath.Ext(name) == ".sql" {
			migrationNames = append(migrationNames, strings.TrimSuffix(name, ".sql"))
		}
	}
	sort.Strings(migrationNames)

	formatForPrint := "%-40s %-10s %-25s %-25s \n"
	fmt.Printf(formatForPrint, "Migration name", "Status", "Updated at", "Applied at")
	for _, name := range migrationNames {
		migrationSchema, ok := applied[name]
		if !ok {
			fmt.Printf(formatForPrint, name, "PENDING", "-", "-")
		} else {
			if !migrationSchema.IsSuccess {
				fmt.Printf(
					formatForPrint,
					name,
					"ERROR",
					migrationSchema.UpdatedAt.Format(time.RFC3339),
					migrationSchema.AppliedAt.Format(time.RFC3339),
				)
			} else {
				fmt.Printf(
					formatForPrint,
					name,
					"APPLIED",
					migrationSchema.UpdatedAt.Format(time.RFC3339),
					migrationSchema.AppliedAt.Format(time.RFC3339),
				)
			}
		}
	}

	return nil
}

func (m *migrator) VersionDB() error {
	lastMigration, err := m.migration.getLastApplyMigration()
	if err != nil {
		m.logg.Error(fmt.Sprintf("Failed get last migration: %s", err.Error()))
		return err
	}

	fmt.Printf("Latest migration: %s\n", lastMigration)
	return nil
}

func getSQLByTemplate(content, migrationType string) (string, error) {
	upMarkerIndex := strings.Index(content, up)
	downMarkerIndex := strings.Index(content, down)
	if upMarkerIndex == -1 || downMarkerIndex == -1 || upMarkerIndex >= downMarkerIndex {
		return "", ErrIncorrectTemplate
	}

	splitContent := strings.SplitN(content, down, 2)

	if migrationType == up {
		return splitContent[0], nil
	}
	if migrationType == down {
		return splitContent[1], nil
	}

	return "", ErrIncorrectType
}
