package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type GormGroupFunnelRepository struct {
	db *gorm.DB
}

func NewGormGroupFunnelRepository(db *gorm.DB) *GormGroupFunnelRepository {
	return &GormGroupFunnelRepository{db: db}
}

func (r *GormGroupFunnelRepository) CreateOrUpdate(ctx context.Context, gf *domain.GroupFunnel) error {
	model := groupFunnelToModel(gf)
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *GormGroupFunnelRepository) FindByGroupID(ctx context.Context, groupID string) ([]domain.GroupFunnel, error) {
	var models []groupFunnelModel
	if err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.GroupFunnel, len(models))
	for i := range models {
		result[i] = *groupFunnelToDomain(&models[i])
	}
	return result, nil
}

func (r *GormGroupFunnelRepository) FindByFunnelID(ctx context.Context, funnelID string) ([]domain.GroupFunnel, error) {
	var models []groupFunnelModel
	if err := r.db.WithContext(ctx).
		Where("funnel_id = ?", funnelID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.GroupFunnel, len(models))
	for i := range models {
		result[i] = *groupFunnelToDomain(&models[i])
	}
	return result, nil
}

func (r *GormGroupFunnelRepository) FindByFunnelAndColumn(ctx context.Context, funnelID, columnID string) ([]domain.GroupFunnel, error) {
	var models []groupFunnelModel
	if err := r.db.WithContext(ctx).
		Where("funnel_id = ?", funnelID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	var result []domain.GroupFunnel
	for i := range models {
		gf := groupFunnelToDomain(&models[i])
		if gf.CoversColumn(columnID) {
			result = append(result, *gf)
		}
	}
	return result, nil
}

func (r *GormGroupFunnelRepository) Delete(ctx context.Context, groupID, funnelID string) error {
	result := r.db.WithContext(ctx).
		Where("group_id = ? AND funnel_id = ?", groupID, funnelID).
		Delete(&groupFunnelModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrGroupFunnelNotFound
	}
	return nil
}
