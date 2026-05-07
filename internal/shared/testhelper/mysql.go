package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	dbName     = "crm_juridico_test"
	dbUser     = "test"
	dbPassword = "test_secret"
	dbRoot     = "root_secret"
)

type MySQLContainer struct {
	Container testcontainers.Container
	DSN       string
	Host      string
	Port      string
}

func NewMySQLContainer(ctx context.Context, t *testing.T) *MySQLContainer {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": dbRoot,
			"MYSQL_DATABASE":     dbName,
			"MYSQL_USER":         dbUser,
			"MYSQL_PASSWORD":     dbPassword,
		},
		Cmd: []string{
			"--character-set-server=utf8mb4",
			"--collation-server=utf8mb4_unicode_ci",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("ready for connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
			wait.ForSQL("3306/tcp", "mysql", func(host string, port string) string {
				return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True",
					dbUser, dbPassword, host, port, dbName,
				)
			}).WithStartupTimeout(60*time.Second),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start mysql container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	mappedPort, err := container.MappedPort(ctx, "3306")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true",
		dbUser, dbPassword, host, mappedPort.Port(), dbName,
	)

	return &MySQLContainer{
		Container: container,
		DSN:       dsn,
		Host:      host,
		Port:      mappedPort.Port(),
	}
}

func (mc *MySQLContainer) DB(t *testing.T) *gorm.DB {
	t.Helper()

	var db *gorm.DB
	var err error

	for i := range 10 {
		db, err = gorm.Open(mysql.Open(mc.DSN), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		})
		if err == nil {
			sqlDB, pingErr := db.DB()
			if pingErr == nil && sqlDB.Ping() == nil {
				return db
			}
		}
		t.Logf("retry %d/10: waiting for mysql connection...", i+1)
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}

	t.Fatalf("failed to connect to test database after retries: %v", err)
	return nil
}

func (mc *MySQLContainer) Logger() *zap.Logger {
	log, _ := zap.NewDevelopment()
	return log
}

func projectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}

func MigrationsPath() string {
	return filepath.Join(projectRoot(), "migrations")
}

func TemplatesPath() string {
	return filepath.Join(projectRoot(), "web", "templates", "**", "*.html")
}

// TemplateGlobs returns the set of globs needed to load all HTML templates,
// since Go's filepath.Glob does not support `**` as recursive matching.
func TemplateGlobs() []string {
	root := filepath.Join(projectRoot(), "web", "templates")
	return []string{
		filepath.Join(root, "*.html"),
		filepath.Join(root, "*", "*.html"),
		filepath.Join(root, "*", "*", "*.html"),
	}
}

func TemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"add":                 func(a, b int) int { return a + b },
		"sub":                 func(a, b int) int { return a - b },
		"aiPlaygroundEnabled": func() bool { return false },
		"typeIcon":            func(t string) string { return "🔔" },
		"typeLabel":           func(t string) string { return "" },
		"relativeTime":        func(t time.Time) string { return "" },
		"dict": func(values ...interface{}) map[string]interface{} {
			m := make(map[string]interface{})
			for i := 0; i+1 < len(values); i += 2 {
				key, _ := values[i].(string)
				m[key] = values[i+1]
			}
			return m
		},
		"formatFileSize": func(size int64) string {
			const (
				kb = 1024
				mb = kb * 1024
			)
			switch {
			case size >= mb:
				return fmt.Sprintf("%.1f MB", float64(size)/float64(mb))
			case size >= kb:
				return fmt.Sprintf("%.1f KB", float64(size)/float64(kb))
			default:
				return fmt.Sprintf("%d B", size)
			}
		},
		"formatValor": func(c *int64) string {
			if c == nil {
				return ""
			}
			return fmt.Sprintf("%.2f", float64(*c)/100.0)
		},
		"uint8Or": func(p *uint8, fallback uint8) uint8 {
			if p == nil {
				return fallback
			}
			return *p
		},
		"dateOr": func(p *time.Time) string {
			if p == nil {
				return ""
			}
			return p.Format("2006-01-02")
		},
		"deref": func(p *string) string {
			if p == nil {
				return ""
			}
			return *p
		},
		"prettyJSON": func(v interface{}) string {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return fmt.Sprintf("%v", v)
			}
			return string(b)
		},
	}
}

func ParseTemplates() *template.Template {
	tmpl := template.New("").Funcs(TemplateFuncMap())
	for _, pattern := range TemplateGlobs() {
		matches, _ := filepath.Glob(pattern)
		if len(matches) == 0 {
			continue
		}
		tmpl = template.Must(tmpl.ParseGlob(pattern))
	}
	return tmpl
}

func NewMySQLContainerForMain(ctx context.Context) *MySQLContainer {
	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": dbRoot,
			"MYSQL_DATABASE":     dbName,
			"MYSQL_USER":         dbUser,
			"MYSQL_PASSWORD":     dbPassword,
		},
		Cmd: []string{
			"--character-set-server=utf8mb4",
			"--collation-server=utf8mb4_unicode_ci",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("ready for connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
			wait.ForSQL("3306/tcp", "mysql", func(host string, port string) string {
				return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True",
					dbUser, dbPassword, host, port, dbName,
				)
			}).WithStartupTimeout(60*time.Second),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to start mysql container: %v", err))
	}

	host, err := container.Host(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to get container host: %v", err))
	}

	mappedPort, err := container.MappedPort(ctx, "3306")
	if err != nil {
		panic(fmt.Sprintf("failed to get mapped port: %v", err))
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true",
		dbUser, dbPassword, host, mappedPort.Port(), dbName,
	)

	return &MySQLContainer{
		Container: container,
		DSN:       dsn,
		Host:      host,
		Port:      mappedPort.Port(),
	}
}

func (mc *MySQLContainer) Teardown(ctx context.Context, t *testing.T) {
	t.Helper()

	if err := mc.Container.Terminate(ctx); err != nil {
		t.Logf("failed to terminate mysql container: %v", err)
	}
}
