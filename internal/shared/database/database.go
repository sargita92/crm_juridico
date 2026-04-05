package database

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/sasrgita/crm-juridico/internal/shared/config"
)

func New(cfg config.DatabaseConfig, log *zap.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	log.Info("database connected",
		zap.String("host", cfg.Host),
		zap.String("port", cfg.Port),
		zap.String("database", cfg.Name),
	)

	return db, nil
}

func Close(db *gorm.DB, log *zap.Logger) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Error("failed to get underlying sql.DB for close", zap.Error(err))
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Error("failed to close database", zap.Error(err))
	}
}
