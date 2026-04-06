package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type GormFunnelRepository struct {
	db *gorm.DB
}

func NewGormFunnelRepository(db *gorm.DB) *GormFunnelRepository {
	return &GormFunnelRepository{db: db}
}

func (r *GormFunnelRepository) Create(ctx context.Context, funnel *domain.Funnel) error {
	model := funnelToModel(funnel)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormFunnelRepository) FindByID(ctx context.Context, id string) (*domain.Funnel, error) {
	var model funnelModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFunnelNotFound
		}
		return nil, err
	}
	return funnelToDomain(&model), nil
}

func (r *GormFunnelRepository) Update(ctx context.Context, funnel *domain.Funnel) error {
	model := funnelToModel(funnel)
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrFunnelNotFound
	}
	return nil
}

func (r *GormFunnelRepository) FindByTenantID(ctx context.Context, tenantID string) ([]domain.Funnel, error) {
	var models []funnelModel
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	funnels := make([]domain.Funnel, len(models))
	for i := range models {
		funnels[i] = *funnelToDomain(&models[i])
	}
	return funnels, nil
}

func (r *GormFunnelRepository) FindDefaultByTenantID(ctx context.Context, tenantID string) (*domain.Funnel, error) {
	var model funnelModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_default = ? AND active = ?", tenantID, true, true).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFunnelNotFound
		}
		return nil, err
	}
	return funnelToDomain(&model), nil
}
