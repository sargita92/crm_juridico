package database_test

import (
	"strings"
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
		if arg == "-test.short" || arg == "-short" || strings.HasPrefix(arg, "-test.short=") || strings.HasPrefix(arg, "-short=") {
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

// TestPortugueseCharacters_RoundTrip guarantees that accented and special
// Portuguese characters (é, ç, ã, ô, ñ, emoji) survive a write/read cycle
// through MySQL. This catches regressions in server charset, connection
// charset, and table collation configuration.
func TestPortugueseCharacters_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)

	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS pt_charset_probe (
			id INT AUTO_INCREMENT PRIMARY KEY,
			label VARCHAR(255) NOT NULL,
			body TEXT NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`).Error; err != nil {
		t.Fatalf("failed to create probe table: %v", err)
	}
	t.Cleanup(func() { _ = db.Exec("DROP TABLE IF EXISTS pt_charset_probe").Error })

	samples := []struct {
		label string
		body  string
	}{
		{"acentos agudos", "José, café, análise, período"},
		{"cedilha", "ação, coração, maçã, serviço, preferência"},
		{"nasal e circunflexo", "São João, mãe, pãezinhos, canção, câmara"},
		{"juridico", "Ação Trabalhista – Advocacia Família, Cível & Previdenciária"},
		{"emoji e simbolos", "⚖️ Contrato ✅ € — £ 😀 ¹²³"},
	}

	for _, s := range samples {
		if err := db.Exec(
			"INSERT INTO pt_charset_probe (label, body) VALUES (?, ?)",
			s.label, s.body,
		).Error; err != nil {
			t.Fatalf("insert %q failed: %v", s.label, err)
		}
	}

	type row struct {
		Label string
		Body  string
	}
	var rows []row
	if err := db.Raw("SELECT label, body FROM pt_charset_probe ORDER BY id").Scan(&rows).Error; err != nil {
		t.Fatalf("select failed: %v", err)
	}

	if len(rows) != len(samples) {
		t.Fatalf("expected %d rows, got %d", len(samples), len(rows))
	}

	for i, s := range samples {
		if rows[i].Label != s.label {
			t.Errorf("row %d label: want %q, got %q", i, s.label, rows[i].Label)
		}
		if rows[i].Body != s.body {
			t.Errorf("row %d body: want %q, got %q", i, s.body, rows[i].Body)
		}
	}

	// LIKE with accented chars should also match (collation-dependent).
	var count int64
	if err := db.Raw(
		"SELECT COUNT(*) FROM pt_charset_probe WHERE body LIKE ?", "%ação%",
	).Scan(&count).Error; err != nil {
		t.Fatalf("like search failed: %v", err)
	}
	if count == 0 {
		t.Errorf("expected LIKE '%%ação%%' to match, got 0 rows")
	}
}
