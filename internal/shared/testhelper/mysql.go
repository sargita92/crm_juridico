package testhelper

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
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
		WaitingFor: wait.ForAll(
			wait.ForLog("ready for connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
			wait.ForSQL("3306/tcp", "mysql", func(host string, port nat.Port) string {
				return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True",
					dbUser, dbPassword, host, port.Port(), dbName,
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

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
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

func (mc *MySQLContainer) Teardown(ctx context.Context, t *testing.T) {
	t.Helper()

	if err := mc.Container.Terminate(ctx); err != nil {
		t.Logf("failed to terminate mysql container: %v", err)
	}
}
