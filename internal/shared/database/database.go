package database

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/shared/config"
)

// New abre a conexão com o banco e configura o pool. O logger do Gorm é
// substituído por um logger respaldado em zap que registra erros e queries
// lentas (acima de cfg.SlowQueryThresholdMs), enriquecidas com os campos de
// contexto retornados por ctxFields (ex.: request_id, tenant_id). ctxFields
// pode ser nil.
func New(cfg config.DatabaseConfig, log *zap.Logger, ctxFields ContextFieldExtractor) (*gorm.DB, error) {
	slowThreshold := time.Duration(cfg.SlowQueryThresholdMs) * time.Millisecond

	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: newZapGormLogger(log, slowThreshold, ctxFields),
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
