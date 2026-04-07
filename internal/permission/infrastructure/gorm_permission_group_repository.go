package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type GormPermissionGroupRepository struct {
	db *gorm.DB
}

func NewGormPermissionGroupRepository(db *gorm.DB) *GormPermissionGroupRepository {
	return &GormPermissionGroupRepository{db: db}
}

func (r *GormPermissionGroupRepository) Create(ctx context.Context, group *domain.PermissionGroup) error {
	model := permissionGroupToModel(group)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormPermissionGroupRepository) FindByID(ctx context.Context, id string) (*domain.PermissionGroup, error) {
	var model permissionGroupModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrGroupNotFound
		}
		return nil, err
	}
	return permissionGroupToDomain(&model), nil
}

func (r *GormPermissionGroupRepository) Update(ctx context.Context, group *domain.PermissionGroup) error {
	model := permissionGroupToModel(group)
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrGroupNotFound
	}
	return nil
}

func (r *GormPermissionGroupRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&permissionGroupModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrGroupNotFound
	}
	return nil
}

func (r *GormPermissionGroupRepository) FindByTenantID(ctx context.Context, tenantID string) ([]domain.PermissionGroup, error) {
	var models []permissionGroupModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("name ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	groups := make([]domain.PermissionGroup, len(models))
	for i := range models {
		groups[i] = *permissionGroupToDomain(&models[i])
	}
	return groups, nil
}
