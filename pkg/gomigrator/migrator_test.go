package gomigrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	tmpDir := t.TempDir()
	m := &migrator{
		dir:  tmpDir,
		logg: NewLogger(),
	}

	name := "test"
	err := m.Create(name)
	assert.NoError(t, err)

	files, err := os.ReadDir(tmpDir)
	assert.NoError(t, err)

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	createdFile := files[0].Name()
	if !strings.HasSuffix(createdFile, name+".sql") {
		t.Errorf("unexpected file name: %s", createdFile)
	}

	contents, err := os.ReadFile(filepath.Join(tmpDir, createdFile))
	assert.NoError(t, err)

	dataTemplate := []string{up, "", down, ""}
	expected := strings.Join(dataTemplate[0:], "\n")

	assert.Equal(t, expected, string(contents))
}

func TestGetSQLByTemplate(t *testing.T) {
	createTableSQL := `CREATE TABLE IF NOT EXISTS "user" (id INT);`
	dropTableSQL := `DROP TABLE IF EXISTS "user";`

	dataTemplate := []string{up, createTableSQL, "", down, dropTableSQL, ""}
	template := strings.Join(dataTemplate[0:], "\n")

	t.Run("get up", func(t *testing.T) {
		sql, err := getSQLByTemplate(template, up)
		assert.NoError(t, err)

		expected := strings.Join([]string{up, createTableSQL, "\n"}, "\n")
		assert.Equal(t, expected, sql)
	})

	t.Run("get down", func(t *testing.T) {
		sql, err := getSQLByTemplate(template, down)
		assert.NoError(t, err)

		expected := strings.Join([]string{"", dropTableSQL, ""}, "\n")
		assert.Equal(t, expected, sql)
	})

	t.Run("invalid type", func(t *testing.T) {
		_, err := getSQLByTemplate(template, "-- +invalid")
		assert.ErrorIs(t, err, ErrIncorrectType)
	})
}
