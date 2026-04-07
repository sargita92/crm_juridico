package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type GormPermissionRepository struct {
	db *gorm.DB
}

func NewGormPermissionRepository(db *gorm.DB) *GormPermissionRepository {
	return &GormPermissionRepository{db: db}
}

func (r *GormPermissionRepository) Create(ctx context.Context, p *domain.Permission) error {
	model := permissionToModel(p)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormPermissionRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&permissionModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrPermissionNotFound
	}
	return nil
}

func (r *GormPermissionRepository) FindByGroupID(ctx context.Context, groupID string) ([]domain.Permission, error) {
	var models []permissionModel
	if err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Permission, len(models))
	for i := range models {
		result[i] = *permissionToDomain(&models[i])
	}
	return result, nil
}

func (r *GormPermissionRepository) FindByUserID(ctx context.Context, userID string) ([]domain.Permission, error) {
	var models []permissionModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Permission, len(models))
	for i := range models {
		result[i] = *permissionToDomain(&models[i])
	}
	return result, nil
}

func (r *GormPermissionRepository) FindByGroupIDs(ctx context.Context, groupIDs []string) ([]domain.Permission, error) {
	if len(groupIDs) == 0 {
		return []domain.Permission{}, nil
	}
	var models []permissionModel
	if err := r.db.WithContext(ctx).
		Where("group_id IN ?", groupIDs).
		Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Permission, len(models))
	for i := range models {
		result[i] = *permissionToDomain(&models[i])
	}
	return result, nil
}

func (r *GormPermissionRepository) DeleteByGroupAndResource(ctx context.Context, groupID, resource string) error {
	return r.db.WithContext(ctx).
		Where("group_id = ? AND resource = ?", groupID, resource).
		Delete(&permissionModel{}).Error
}

func (r *GormPermissionRepository) DeleteByUserAndResource(ctx context.Context, userID, resource string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND resource = ?", userID, resource).
		Delete(&permissionModel{}).Error
}
