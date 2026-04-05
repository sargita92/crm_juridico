package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB, log *zap.Logger, migrationsPath ...string) error {
	source := "file://migrations"
	if len(migrationsPath) > 0 && migrationsPath[0] != "" {
		source = migrationsPath[0]
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	driver, err := mysql.WithInstance(sqlDB, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		source,
		"mysql",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		log.Info("migrations: no changes to apply")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Info("migrations: applied successfully")
	return nil
}
