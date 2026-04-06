package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type GormLeadMovementRepository struct {
	db *gorm.DB
}

func NewGormLeadMovementRepository(db *gorm.DB) *GormLeadMovementRepository {
	return &GormLeadMovementRepository{db: db}
}

func (r *GormLeadMovementRepository) Create(ctx context.Context, movement *domain.LeadMovement) error {
	model := movementToModel(movement)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormLeadMovementRepository) FindByLeadID(ctx context.Context, leadID string) ([]domain.LeadMovement, error) {
	var models []leadMovementModel
	if err := r.db.WithContext(ctx).
		Where("lead_id = ?", leadID).
		Order("moved_at DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	movements := make([]domain.LeadMovement, len(models))
	for i := range models {
		movements[i] = *movementToDomain(&models[i])
	}
	return movements, nil
}
