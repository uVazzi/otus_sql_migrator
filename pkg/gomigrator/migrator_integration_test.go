//go:build integration

package gomigrator

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq" // DB driver
	"github.com/stretchr/testify/suite"
)

type MigratorSuiteTest struct {
	suite.Suite
	db       *sql.DB
	dir      string
	migrator Migrator
}

type migrationData struct {
	tableName string
	prefix    string
}

func TestMigratorIntegrationSuite(t *testing.T) {
	suite.Run(t, new(MigratorSuiteTest))
}

func (s *MigratorSuiteTest) SetupSuite() {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		s.T().Fatal("missing TEST_DB_DSN")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		s.T().Fatal(err)
	}
	s.db = db

}

func (s *MigratorSuiteTest) TearDownSuite() {
	s.db.Close()
}

func (s *MigratorSuiteTest) SetupTest() {
	s.dir = s.T().TempDir()
	s.migrator = NewMigrator(s.db, s.dir)
}

func (s *MigratorSuiteTest) TearDownTest() {
	_, err := s.db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`)
	if err != nil {
		s.T().Fatal(err)
	}
}

func (s *MigratorSuiteTest) TestUp() {
	tablesData := []migrationData{
		migrationData{"test_1", "20250515101157"},
		migrationData{"test_2", "20250515101158"},
		migrationData{"test_3", "20250515101159"},
	}

	// one migration
	err := createMigrationFile(s.dir, tablesData[0].prefix, tablesData[0].tableName, getCreateTableTemplate(tablesData[0].tableName))
	s.NoError(err)

	err = s.migrator.Up()
	s.NoError(err)

	// two migration
	err = createMigrationFile(s.dir, tablesData[1].prefix, tablesData[1].tableName, getCreateTableTemplate(tablesData[1].tableName))
	s.NoError(err)
	err = createMigrationFile(s.dir, tablesData[2].prefix, tablesData[2].tableName, getCreateTableTemplate(tablesData[2].tableName))
	s.NoError(err)

	err = s.migrator.Up()
	s.NoError(err)

	// check tables and migration_schema
	for _, tableData := range tablesData {
		s.True(existTable(s, tableData.tableName))
		s.True(existMigration(s, tableData.prefix+"_"+tableData.tableName))
	}
}

func (s *MigratorSuiteTest) TestDown() {
	tablesData := []migrationData{
		migrationData{"test_1", "20250515101157"},
		migrationData{"test_2", "20250515101158"},
	}

	for _, tableData := range tablesData {
		template := getCreateTableTemplate(tableData.tableName)
		err := createMigrationFile(s.dir, tableData.prefix, tableData.tableName, template)
		s.NoError(err)

		err = upMigrationByTemplate(s, template, tableData.prefix+"_"+tableData.tableName)
		s.NoError(err)
	}

	for _, tableData := range tablesData {
		s.True(existTable(s, tableData.tableName))
		s.True(existMigration(s, tableData.prefix+"_"+tableData.tableName))
	}

	err := s.migrator.Down()
	s.NoError(err)

	s.False(existMigration(s, tablesData[1].prefix+"_"+tablesData[1].tableName))
	s.False(existTable(s, tablesData[1].tableName))

	err = s.migrator.Down()
	s.NoError(err)

	s.False(existMigration(s, tablesData[0].prefix+"_"+tablesData[0].tableName))
	s.False(existTable(s, tablesData[0].tableName))
}

func (s *MigratorSuiteTest) TestRedo() {
	prefix := "20250515101157"
	tableName := "test"

	template := getCreateTableTemplate(tableName)
	s.NoError(createMigrationFile(s.dir, prefix, tableName, template))
	s.NoError(upMigrationByTemplate(s, template, prefix+"_"+tableName))

	var beforeRedoTime time.Time
	err := s.db.QueryRow(`SELECT applied_at FROM migration_schema WHERE name = $1`, prefix+"_"+tableName).Scan(&beforeRedoTime)
	s.Require().NoError(err)

	time.Sleep(1 * time.Second)
	s.NoError(s.migrator.Redo())

	s.True(existTable(s, tableName))
	s.True(existMigration(s, prefix+"_"+tableName))

	var afterRedoTime time.Time
	err = s.db.QueryRow(`SELECT applied_at FROM migration_schema WHERE name = $1`, prefix+"_"+tableName).Scan(&afterRedoTime)
	s.NoError(err)
	s.True(afterRedoTime.After(beforeRedoTime))
}

func (s *MigratorSuiteTest) TestStatus() {
	tablesData := []migrationData{
		migrationData{"test_1", "20250515101157"},
		migrationData{"test_2", "20250515101158"},
	}

	template := getCreateTableTemplate(tablesData[0].tableName)
	s.NoError(createMigrationFile(s.dir, tablesData[0].prefix, tablesData[0].tableName, template))
	s.NoError(upMigrationByTemplate(s, template, tablesData[0].prefix+"_"+tablesData[0].tableName))

	output := getStdoutByFunc(func() {
		s.NoError(s.migrator.Status())
	})

	s.Contains(output, tablesData[0].prefix+"_"+tablesData[0].tableName)
	s.Contains(output, "APPLIED")

	template = getCreateTableTemplate(tablesData[1].tableName)
	s.NoError(createMigrationFile(s.dir, tablesData[1].prefix, tablesData[1].tableName, template))

	output = getStdoutByFunc(func() {
		s.NoError(s.migrator.Status())
	})

	s.Contains(output, tablesData[0].prefix+"_"+tablesData[0].tableName)
	s.Contains(output, tablesData[1].prefix+"_"+tablesData[1].tableName)
	s.Contains(output, "PENDING")
}

func (s *MigratorSuiteTest) TestVersionDB() {
	tablesData := []migrationData{
		migrationData{"test_1", "20250515101157"},
		migrationData{"test_2", "20250515101158"},
	}

	template := getCreateTableTemplate(tablesData[0].tableName)
	s.NoError(createMigrationFile(s.dir, tablesData[0].prefix, tablesData[0].tableName, template))
	s.NoError(upMigrationByTemplate(s, template, tablesData[0].prefix+"_"+tablesData[0].tableName))

	output := getStdoutByFunc(func() {
		s.NoError(s.migrator.Status())
	})
	s.Contains(output, tablesData[0].prefix+"_"+tablesData[0].tableName)

	template = getCreateTableTemplate(tablesData[1].tableName)
	s.NoError(createMigrationFile(s.dir, tablesData[1].prefix, tablesData[1].tableName, template))
	s.NoError(upMigrationByTemplate(s, template, tablesData[1].prefix+"_"+tablesData[1].tableName))

	output = getStdoutByFunc(func() {
		s.NoError(s.migrator.Status())
	})
	s.Contains(output, tablesData[1].prefix+"_"+tablesData[1].tableName)
}

func getCreateTableTemplate(tableName string) string {
	dataTemplate := []string{
		up,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (uuid UUID PRIMARY KEY NOT NULL);`, tableName),
		"",
		down,
		fmt.Sprintf(`DROP TABLE IF EXISTS %s;`, tableName),
		"",
	}
	return strings.Join(dataTemplate[0:], "\n")
}

func createMigrationFile(dir string, prefix string, tableName, template string) error {
	path := filepath.Join(dir, prefix+"_"+tableName+".sql")

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(template)
	if err != nil {
		return err
	}

	return nil
}

func upMigrationByTemplate(s *MigratorSuiteTest, template, migrationName string) error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS migration_schema (
			name VARCHAR PRIMARY KEY,
			is_success BOOLEAN,
			applied_at TIMESTAMPTZ NOT NULL,
		    updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	upSQL, err := getSQLByTemplate(template, up)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(upSQL)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		"INSERT INTO migration_schema (name, is_success, applied_at, updated_at) VALUES ($1, $2, NOW(), NOW())",
		migrationName,
		true,
	)
	return err
}

func existTable(s *MigratorSuiteTest, tableName string) bool {
	query := `
	SELECT EXISTS (
		SELECT 1 
		FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_name = $1
	)`
	var exists bool
	err := s.db.QueryRow(query, tableName).Scan(&exists)
	s.NoError(err)

	return exists
}

func existMigration(s *MigratorSuiteTest, migrationName string) bool {
	row := s.db.QueryRow(`SELECT name FROM migration_schema WHERE name = $1`, migrationName)
	var name string
	err := row.Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}

	s.NoError(err)
	return true
}

func getStdoutByFunc(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
