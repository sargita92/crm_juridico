package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

type userTenantModel struct {
	UserID     string `gorm:"primaryKey;column:user_id;type:char(36)"`
	TenantID   string `gorm:"primaryKey;column:tenant_id;type:char(36)"`
	IsOwner    bool   `gorm:"column:is_owner;type:tinyint(1);not null;default:0"`
	WhatsAppID string `gorm:"column:whatsapp_id;type:varchar(100)"`
}

func (userTenantModel) TableName() string { return "user_tenants" }

func userTenantToDomain(m *userTenantModel) *domain.UserTenant {
	return &domain.UserTenant{
		UserID:     m.UserID,
		TenantID:   m.TenantID,
		IsOwner:    m.IsOwner,
		WhatsAppID: m.WhatsAppID,
	}
}

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

func (r *GormUserTenantRepository) FindByTenantID(ctx context.Context, tenantID string) ([]*domain.UserTenant, error) {
	var models []userTenantModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	result := make([]*domain.UserTenant, len(models))
	for i := range models {
		result[i] = userTenantToDomain(&models[i])
	}
	return result, nil
}

func (r *GormUserTenantRepository) FindByUserAndTenant(ctx context.Context, userID, tenantID string) (*domain.UserTenant, error) {
	var model userTenantModel
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return userTenantToDomain(&model), nil
}

func (r *GormUserTenantRepository) UpdateIsOwner(ctx context.Context, userID, tenantID string, isOwner bool) error {
	return r.db.WithContext(ctx).
		Model(&userTenantModel{}).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Update("is_owner", isOwner).Error
}

func (r *GormUserTenantRepository) UpdateWhatsAppID(ctx context.Context, userID, tenantID string, whatsAppID string) error {
	return r.db.WithContext(ctx).
		Model(&userTenantModel{}).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Update("whatsapp_id", whatsAppID).Error
}

func (r *GormUserTenantRepository) RemoveFromTenant(ctx context.Context, userID, tenantID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Delete(&userTenantModel{}).Error
}

func (r *GormUserTenantRepository) IsOwner(ctx context.Context, userID, tenantID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&userTenantModel{}).
		Where("user_id = ? AND tenant_id = ? AND is_owner = 1", userID, tenantID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
