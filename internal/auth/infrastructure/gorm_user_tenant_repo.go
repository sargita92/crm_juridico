package infrastructure

import (
	"context"

	"gorm.io/gorm"
)

type userTenantModel struct {
	UserID   string `gorm:"primaryKey;column:user_id;type:char(36)"`
	TenantID string `gorm:"primaryKey;column:tenant_id;type:char(36)"`
}

func (userTenantModel) TableName() string { return "user_tenants" }

type GormUserTenantRepository struct {
	db *gorm.DB
}

func NewGormUserTenantRepository(db *gorm.DB) *GormUserTenantRepository {
	return &GormUserTenantRepository{db: db}
}

func (r *GormUserTenantRepository) Associate(ctx context.Context, userID, tenantID string) error {
	model := userTenantModel{UserID: userID, TenantID: tenantID}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *GormUserTenantRepository) FindTenantIDsByUserID(ctx context.Context, userID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).
		Model(&userTenantModel{}).
		Where("user_id = ?", userID).
		Pluck("tenant_id", &ids).Error
	return ids, err
}
