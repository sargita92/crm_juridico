package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type GormUserGroupRepository struct {
	db *gorm.DB
}

func NewGormUserGroupRepository(db *gorm.DB) *GormUserGroupRepository {
	return &GormUserGroupRepository{db: db}
}

func (r *GormUserGroupRepository) Create(ctx context.Context, ug *domain.UserGroup) error {
	model := userGroupToModel(ug)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormUserGroupRepository) Delete(ctx context.Context, userID, groupID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND group_id = ?", userID, groupID).
		Delete(&userGroupModel{}).Error
}

func (r *GormUserGroupRepository) FindByGroupID(ctx context.Context, groupID string) ([]domain.UserGroup, error) {
	var models []userGroupModel
	if err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.UserGroup, len(models))
	for i := range models {
		result[i] = *userGroupToDomain(&models[i])
	}
	return result, nil
}

func (r *GormUserGroupRepository) FindByUserAndTenant(ctx context.Context, userID, tenantID string) ([]domain.UserGroup, error) {
	var models []userGroupModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.UserGroup, len(models))
	for i := range models {
		result[i] = *userGroupToDomain(&models[i])
	}
	return result, nil
}

func (r *GormUserGroupRepository) Exists(ctx context.Context, userID, groupID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&userGroupModel{}).
		Where("user_id = ? AND group_id = ?", userID, groupID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
