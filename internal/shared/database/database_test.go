package database_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
)

var sharedContainer *testhelper.MySQLContainer

func TestMain(m *testing.M) {
	short := false
	for _, arg := range os.Args {
		if arg == "-test.short" || arg == "-short" {
			short = true
			break
		}
	}

	if !short {
		ctx := context.Background()
		sharedContainer = testhelper.NewMySQLContainerForMain(ctx)
		code := m.Run()
		_ = sharedContainer.Container.Terminate(ctx)
		os.Exit(code)
	}

	os.Exit(m.Run())
}

func migrationsSource() string {
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	return "file://" + filepath.Join(projectRoot, "migrations")
}

func TestDatabaseConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}
}

func TestRunMigrations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	if err := database.RunMigrations(db, log, migrationsSource()); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	var tableName string
	result := db.Raw("SHOW TABLES LIKE 'schema_info'").Scan(&tableName)
	if result.Error != nil {
		t.Fatalf("failed to check schema_info table: %v", result.Error)
	}

	if tableName != "schema_info" {
		t.Errorf("expected table schema_info to exist, got %q", tableName)
	}
}

func TestRunMigrations_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	if err := database.RunMigrations(db, log, migrationsSource()); err != nil {
		t.Fatalf("first migration run failed: %v", err)
	}

	if err := database.RunMigrations(db, log, migrationsSource()); err != nil {
		t.Fatalf("second migration run should be idempotent: %v", err)
	}
}
