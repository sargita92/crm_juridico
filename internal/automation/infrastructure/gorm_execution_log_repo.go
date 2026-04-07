package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
)

// GormExecutionLogRepository is the GORM-backed implementation of domain.ExecutionLogRepository.
type GormExecutionLogRepository struct {
	db *gorm.DB
}

func NewGormExecutionLogRepository(db *gorm.DB) *GormExecutionLogRepository {
	return &GormExecutionLogRepository{db: db}
}

func (r *GormExecutionLogRepository) Create(ctx context.Context, log *domain.ExecutionLog) error {
	return r.db.WithContext(ctx).Create(executionLogToModel(log)).Error
}

func (r *GormExecutionLogRepository) FindByAutomationID(ctx context.Context, automationID string, limit, offset int) ([]domain.ExecutionLog, error) {
	var models []executionLogModel
	if err := r.db.WithContext(ctx).
		Where("automation_id = ?", automationID).
		Order("executed_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ExecutionLog, len(models))
	for i := range models {
		result[i] = *executionLogToDomain(&models[i])
	}
	return result, nil
}
