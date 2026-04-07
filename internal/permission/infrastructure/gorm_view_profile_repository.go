package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type GormViewProfileRepository struct {
	db *gorm.DB
}

func NewGormViewProfileRepository(db *gorm.DB) *GormViewProfileRepository {
	return &GormViewProfileRepository{db: db}
}

func (r *GormViewProfileRepository) CreateOrUpdate(ctx context.Context, vp *domain.ViewProfile) error {
	model := viewProfileToModel(vp)
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *GormViewProfileRepository) FindByGroupID(ctx context.Context, groupID string) ([]domain.ViewProfile, error) {
	var models []viewProfileModel
	if err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ViewProfile, len(models))
	for i := range models {
		result[i] = *viewProfileToDomain(&models[i])
	}
	return result, nil
}

func (r *GormViewProfileRepository) FindByGroupAndFunnel(ctx context.Context, groupID, funnelID string) (*domain.ViewProfile, error) {
	var model viewProfileModel
	if err := r.db.WithContext(ctx).
		Where("group_id = ? AND funnel_id = ?", groupID, funnelID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrViewProfileNotFound
		}
		return nil, err
	}
	return viewProfileToDomain(&model), nil
}

func (r *GormViewProfileRepository) FindByGroupIDs(ctx context.Context, groupIDs []string, funnelID string) ([]domain.ViewProfile, error) {
	if len(groupIDs) == 0 {
		return []domain.ViewProfile{}, nil
	}
	var models []viewProfileModel
	if err := r.db.WithContext(ctx).
		Where("group_id IN ? AND funnel_id = ?", groupIDs, funnelID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ViewProfile, len(models))
	for i := range models {
		result[i] = *viewProfileToDomain(&models[i])
	}
	return result, nil
}

func (r *GormViewProfileRepository) Delete(ctx context.Context, groupID, funnelID string) error {
	result := r.db.WithContext(ctx).
		Where("group_id = ? AND funnel_id = ?", groupID, funnelID).
		Delete(&viewProfileModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrViewProfileNotFound
	}
	return nil
}
